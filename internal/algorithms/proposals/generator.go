package proposals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	DefaultGeneratorVersion = "g111.v1"
	DefaultLimit            = 100
)

type Config struct {
	TenantID           string
	AlgorithmID        string
	ExecutionRequestID string
	AlgorithmResultID  string
	ResultType         string
	Severity           string
	CorrelationID      string
	MinConfidence      float64
	Limit              int
	CreatedBy          string
	GeneratorVersion   string
}

type Result struct {
	Scanned   int                                     `json:"scanned"`
	Proposed  int                                     `json:"proposed"`
	Skipped   int                                     `json:"skipped"`
	Proposals []storage.AlgorithmSignalProposalRecord `json:"proposals"`
}

func Generate(ctx context.Context, repo storage.AlgorithmRepository, cfg Config) (Result, error) {
	cfg = cfg.withDefaults()
	if strings.TrimSpace(cfg.TenantID) == "" {
		return Result{}, errors.New("tenant id is required")
	}
	results, err := repo.ListAlgorithmResults(ctx, storage.AlgorithmResultFilter{TenantID: cfg.TenantID, AlgorithmID: cfg.AlgorithmID, ExecutionRequestID: cfg.ExecutionRequestID, ResultType: cfg.ResultType, Severity: cfg.Severity, CorrelationID: cfg.CorrelationID, Limit: cfg.Limit})
	if err != nil {
		return Result{}, err
	}
	out := Result{Scanned: len(results), Proposals: []storage.AlgorithmSignalProposalRecord{}}
	for _, result := range results {
		if strings.TrimSpace(cfg.AlgorithmResultID) != "" && result.AlgorithmResultID != cfg.AlgorithmResultID {
			out.Skipped++
			continue
		}
		if result.Confidence < cfg.MinConfidence {
			out.Skipped++
			continue
		}
		if !passesQualityGate(result) {
			out.Skipped++
			continue
		}
		proposal, ok, err := ProposalFromResult(result, cfg)
		if err != nil {
			return out, err
		}
		if !ok {
			out.Skipped++
			continue
		}
		inserted, err := repo.InsertAlgorithmSignalProposal(ctx, proposal)
		if err != nil {
			return out, err
		}
		if !inserted {
			out.Skipped++
			continue
		}
		out.Proposed++
		out.Proposals = append(out.Proposals, proposal)
	}
	return out, nil
}

func passesQualityGate(result storage.AlgorithmResultRecord) bool {
	payload := resultPayloadMap(result)
	dataset := strings.TrimSpace(stringFromPayload(payload, "dataset"))
	feature := strings.TrimSpace(stringFromPayload(payload, "feature"))
	if dataset != "options_distribution_daily" || feature != "call_put_open_interest_ratio" {
		return true
	}
	return strings.EqualFold(stringFromPayload(payload, "call_put_oi_ratio_quality"), "usable")
}

func resultPayloadMap(result storage.AlgorithmResultRecord) map[string]any {
	payload := map[string]any{}
	if len(result.ResultPayloadJSON) == 0 {
		return payload
	}
	_ = json.Unmarshal(result.ResultPayloadJSON, &payload)
	return payload
}

func stringFromPayload(payload map[string]any, key string) string {
	value, ok := payload[key].(string)
	if !ok {
		return ""
	}
	return value
}

func ProposalFromResult(result storage.AlgorithmResultRecord, cfg Config) (storage.AlgorithmSignalProposalRecord, bool, error) {
	signalType := proposedSignalType(result)
	if signalType == "" {
		return storage.AlgorithmSignalProposalRecord{}, false, nil
	}
	payload := map[string]any{
		"schema_version":       "algorithm_signal_proposal.v1",
		"generator_version":    cfg.withDefaults().GeneratorVersion,
		"proposed_signal_type": signalType,
		"algorithm_result": map[string]any{
			"algorithm_result_id":  result.AlgorithmResultID,
			"algorithm_id":         result.AlgorithmID,
			"algorithm_version":    result.AlgorithmVersion,
			"execution_request_id": result.ExecutionRequestID,
			"result_type":          result.ResultType,
			"score":                result.Score,
			"confidence":           result.Confidence,
			"severity":             result.Severity,
			"payload":              rawJSON(result.ResultPayloadJSON),
		},
		"quality_gate": map[string]any{
			"passed": true,
			"policy": "g131.options_distribution_quality.v1",
		},
		"lineage": map[string]any{
			"source_event_ids":  result.SourceEventIDs,
			"feature_value_ids": result.FeatureValueIDs,
			"evidence_refs":     result.EvidenceRefs,
		},
	}
	rationale := map[string]any{
		"reason":                       "algorithm_result_selected_for_operator_review",
		"status":                       storage.AlgorithmSignalProposalStatusProposed,
		"no_production_signal_written": true,
		"review_required":              true,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.AlgorithmSignalProposalRecord{}, false, err
	}
	rationaleJSON, err := json.Marshal(rationale)
	if err != nil {
		return storage.AlgorithmSignalProposalRecord{}, false, err
	}
	return storage.AlgorithmSignalProposalRecord{
		ProposalID: stableProposalID(result, signalType), TenantID: result.TenantID,
		AlgorithmResultID: result.AlgorithmResultID, AlgorithmID: result.AlgorithmID,
		AlgorithmVersion: result.AlgorithmVersion, ExecutionRequestID: result.ExecutionRequestID,
		ProposedSignalType: signalType, Status: storage.AlgorithmSignalProposalStatusProposed,
		Score: result.Score, Confidence: result.Confidence, Severity: result.Severity,
		ProposalPayloadJSON: payloadJSON, RationaleJSON: rationaleJSON,
		SourceEventIDs: result.SourceEventIDs, EvidenceRefs: result.EvidenceRefs,
		CorrelationID: result.CorrelationID, CreatedBy: cfg.withDefaults().CreatedBy,
	}, true, nil
}

func proposedSignalType(result storage.AlgorithmResultRecord) string {
	switch strings.TrimSpace(result.ResultType) {
	case "z_score", "anomaly_score", "online_anomaly_score", "isolation_score":
		return "signalops.algorithm.anomaly_candidate"
	case "change_point_score":
		return "signalops.algorithm.change_point_candidate"
	case "forecast_residual":
		return "signalops.algorithm.forecast_deviation_candidate"
	case "classifier_label":
		return "signalops.algorithm.classification_candidate"
	default:
		return ""
	}
}

func stableProposalID(result storage.AlgorithmResultRecord, signalType string) string {
	return "algsigprop_" + stableHash(fmt.Sprintf("%s|%s|%s|%s", result.TenantID, result.AlgorithmResultID, signalType, DefaultGeneratorVersion))[:24]
}

func stableHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func rawJSON(raw []byte) any {
	if len(raw) == 0 || !json.Valid(raw) {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return map[string]any{}
	}
	return value
}

func (cfg Config) withDefaults() Config {
	if cfg.Limit <= 0 {
		cfg.Limit = DefaultLimit
	}
	if strings.TrimSpace(cfg.CreatedBy) == "" {
		cfg.CreatedBy = "operator-local"
	}
	if strings.TrimSpace(cfg.GeneratorVersion) == "" {
		cfg.GeneratorVersion = DefaultGeneratorVersion
	}
	return cfg
}
