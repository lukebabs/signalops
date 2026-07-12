package dsm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func ExtractArtifacts(signal storage.SignalLedgerRecord) ([]storage.MarketOpsDSMArtifactRecord, error) {
	if signal.AppID != "marketops" || signal.Domain != "market_data" || signal.DetectorID == "" {
		return nil, nil
	}
	if len(signal.SemanticEvidenceJSON) == 0 {
		return nil, nil
	}
	var semanticItems []map[string]any
	if err := json.Unmarshal(signal.SemanticEvidenceJSON, &semanticItems); err != nil {
		return nil, fmt.Errorf("extract marketops dsm semantic evidence: %w", err)
	}
	records := []storage.MarketOpsDSMArtifactRecord{}
	for _, item := range semanticItems {
		if firstMapString(item, "type") != "dsm_artifact_proposal" {
			continue
		}
		artifactMap, ok := item["artifact"].(map[string]any)
		if !ok {
			continue
		}
		artifactID := firstMapString(item, "artifact_id")
		if artifactID == "" {
			artifactID = firstMapString(artifactMap, "artifact_id")
		}
		artifactType := firstMapString(artifactMap, "artifact_type")
		if artifactID == "" || artifactType == "" {
			continue
		}
		subjectSymbol := ""
		if subject, ok := artifactMap["subject"].(map[string]any); ok {
			subjectSymbol = firstMapString(subject, "symbol")
		}
		artifactJSON, err := json.Marshal(artifactMap)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm artifact: %w", err)
		}
		semanticJSON, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm semantic evidence: %w", err)
		}
		records = append(records, storage.MarketOpsDSMArtifactRecord{
			ArtifactID: artifactID, TenantID: signal.TenantID, AppID: signal.AppID, Domain: signal.Domain,
			UseCase: signal.UseCase, SourceID: signal.SourceID, SourceAdapter: signal.SourceAdapter,
			Dataset: signal.Dataset, SignalID: signal.SignalID, SignalType: signal.SignalType,
			DetectorID: signal.DetectorID, Severity: signal.Severity, Confidence: signal.Confidence,
			EventIDs: append([]string(nil), signal.EventIDs...), SubjectSymbol: subjectSymbol,
			ArtifactType: artifactType, ArtifactJSON: artifactJSON, SemanticEvidenceJSON: semanticJSON,
			GraphTargetsJSON: append([]byte(nil), signal.GraphTargetsJSON...), SupportingMetrics: append([]byte(nil), signal.SupportingMetrics...),
			QualityIssues: stringSlice(artifactMap["quality_issues"]),
		})
	}
	return records, nil
}

func ExtractGraphProposals(artifact storage.MarketOpsDSMArtifactRecord) ([]storage.MarketOpsDSMGraphProposalRecord, error) {
	if artifact.AppID != "marketops" || artifact.Domain != "market_data" || artifact.UseCase != "daily_market_surveillance" || artifact.ArtifactType != "marketops.dsm.signal_artifact.v1" {
		return nil, nil
	}
	if len(artifact.GraphTargetsJSON) == 0 {
		return nil, nil
	}
	var candidates []map[string]any
	if err := json.Unmarshal(artifact.GraphTargetsJSON, &candidates); err != nil {
		return nil, fmt.Errorf("extract marketops dsm graph targets: %w", err)
	}
	records := []storage.MarketOpsDSMGraphProposalRecord{}
	for _, candidate := range candidates {
		candidateType := firstMapString(candidate, "type")
		record := storage.MarketOpsDSMGraphProposalRecord{
			TenantID: artifact.TenantID, AppID: artifact.AppID, Domain: artifact.Domain, UseCase: artifact.UseCase,
			SourceID: artifact.SourceID, SourceAdapter: artifact.SourceAdapter, Dataset: artifact.Dataset,
			ArtifactID: artifact.ArtifactID, SignalID: artifact.SignalID, SignalType: artifact.SignalType,
			DetectorID: artifact.DetectorID, Severity: artifact.Severity, Confidence: artifact.Confidence,
			EventIDs: append([]string(nil), artifact.EventIDs...), SubjectSymbol: artifact.SubjectSymbol,
			CandidateType: candidateType, Labels: stringSliceOrEmpty(candidate["labels"]), Status: storage.MarketOpsDSMGraphProposalStatusProposed,
		}
		switch candidateType {
		case "node_candidate":
			record.NodeID = firstMapString(candidate, "node_id")
			if record.NodeID == "" {
				continue
			}
			record.ProposalID = StableGraphProposalID(artifact.ArtifactID, artifact.SignalID, candidateType, record.NodeID)
		case "relationship_candidate":
			record.FromNode = firstMapString(candidate, "from")
			record.Relationship = firstMapString(candidate, "relationship")
			record.ToNode = firstMapString(candidate, "to")
			if record.FromNode == "" || record.Relationship == "" || record.ToNode == "" {
				continue
			}
			record.ProposalID = StableGraphProposalID(artifact.ArtifactID, artifact.SignalID, candidateType, record.FromNode, record.Relationship, record.ToNode)
		default:
			continue
		}
		properties, ok := candidate["properties"].(map[string]any)
		if !ok {
			properties = map[string]any{}
		}
		propertiesJSON, err := json.Marshal(properties)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm graph proposal properties: %w", err)
		}
		rawCandidate, err := json.Marshal(candidate)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm graph proposal candidate: %w", err)
		}
		record.PropertiesJSON = propertiesJSON
		record.RawCandidate = rawCandidate
		records = append(records, record)
	}
	return records, nil
}

func StableGraphProposalID(parts ...string) string {
	h := sha256.New()
	for _, part := range append([]string{"marketops.dsm.graph_proposal_v1"}, parts...) {
		h.Write([]byte(strings.TrimSpace(part)))
		h.Write([]byte{0})
	}
	return "graphprop_marketops_dsm_v1_" + hex.EncodeToString(h.Sum(nil))[:24]
}

func firstMapString(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, item := range raw {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func stringSliceOrEmpty(value any) []string {
	items := stringSlice(value)
	if items == nil {
		return []string{}
	}
	return items
}
