package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	algorithmproposals "github.com/lukebabs/signalops/internal/algorithms/proposals"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("algorithm proposal generator failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	appCfg := config.Load()
	if strings.TrimSpace(appCfg.DatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}
	generatorCfg := loadCLIConfig()
	ctx := context.Background()
	repo, err := postgresstorage.Open(ctx, appCfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()

	logger.Info("algorithm signal proposal generation started", "tenant_id", generatorCfg.TenantID, "algorithm_id", generatorCfg.AlgorithmID, "execution_request_id", generatorCfg.ExecutionRequestID)
	result, err := algorithmproposals.Generate(ctx, repo, generatorCfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(map[string]any{"metrics": map[string]any{"scanned": result.Scanned, "proposed": result.Proposed, "skipped": result.Skipped}, "algorithm_signal_proposals": proposalOutputs(result.Proposals)}, "", "  ")
	fmt.Println(string(out))
	logger.Info("algorithm signal proposal generation completed", "scanned", result.Scanned, "proposed", result.Proposed, "skipped", result.Skipped)
	return nil
}

func loadCLIConfig() algorithmproposals.Config {
	cfg := algorithmproposals.Config{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.AlgorithmID, "algorithm-id", "", "optional algorithm id filter")
	flag.StringVar(&cfg.ExecutionRequestID, "execution-request-id", "", "optional execution request id filter")
	flag.StringVar(&cfg.AlgorithmResultID, "algorithm-result-id", "", "optional single algorithm result id filter")
	flag.StringVar(&cfg.ResultType, "result-type", "", "optional result type filter")
	flag.StringVar(&cfg.Severity, "severity", "", "optional severity filter")
	flag.StringVar(&cfg.CorrelationID, "correlation-id", "", "optional correlation id filter")
	flag.Float64Var(&cfg.MinConfidence, "min-confidence", 0, "minimum confidence required")
	flag.IntVar(&cfg.Limit, "limit", algorithmproposals.DefaultLimit, "maximum algorithm results to scan")
	flag.StringVar(&cfg.CreatedBy, "created-by", "operator-local", "operator or process creating proposals")
	flag.Parse()
	return cfg
}

type proposalOutput struct {
	ProposalID         string          `json:"proposal_id"`
	TenantID           string          `json:"tenant_id"`
	AlgorithmResultID  string          `json:"algorithm_result_id"`
	AlgorithmID        string          `json:"algorithm_id"`
	AlgorithmVersion   string          `json:"algorithm_version"`
	ExecutionRequestID string          `json:"execution_request_id"`
	ProposedSignalType string          `json:"proposed_signal_type"`
	Status             string          `json:"status"`
	Score              float64         `json:"score"`
	Confidence         float64         `json:"confidence"`
	Severity           string          `json:"severity"`
	ProposalPayload    json.RawMessage `json:"proposal_payload"`
	Rationale          json.RawMessage `json:"rationale"`
	SourceEventIDs     []string        `json:"source_event_ids"`
	EvidenceRefs       []string        `json:"evidence_refs"`
	CorrelationID      string          `json:"correlation_id"`
	CreatedBy          string          `json:"created_by"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func proposalOutputs(records []storage.AlgorithmSignalProposalRecord) []proposalOutput {
	out := make([]proposalOutput, 0, len(records))
	for _, record := range records {
		out = append(out, proposalOutput{ProposalID: record.ProposalID, TenantID: record.TenantID, AlgorithmResultID: record.AlgorithmResultID, AlgorithmID: record.AlgorithmID, AlgorithmVersion: record.AlgorithmVersion, ExecutionRequestID: record.ExecutionRequestID, ProposedSignalType: record.ProposedSignalType, Status: record.Status, Score: record.Score, Confidence: record.Confidence, Severity: record.Severity, ProposalPayload: json.RawMessage(jsonOrDefault(record.ProposalPayloadJSON)), Rationale: json.RawMessage(jsonOrDefault(record.RationaleJSON)), SourceEventIDs: record.SourceEventIDs, EvidenceRefs: record.EvidenceRefs, CorrelationID: record.CorrelationID, CreatedBy: record.CreatedBy, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt})
	}
	return out
}

func jsonOrDefault(raw []byte) []byte {
	if len(raw) == 0 || !json.Valid(raw) {
		return []byte(`{}`)
	}
	return raw
}
