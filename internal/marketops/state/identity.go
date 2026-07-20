package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type IdentityKind string

const (
	IdentityFeatureObservation     IdentityKind = "feature_observation"
	IdentityMarketState            IdentityKind = "market_state"
	IdentityStateTransition        IdentityKind = "state_transition"
	IdentityEvidence               IdentityKind = "evidence"
	IdentityHypothesisEvaluation   IdentityKind = "hypothesis_evaluation"
	IdentityOpportunity            IdentityKind = "opportunity"
	IdentitySignalProposal         IdentityKind = "signal_proposal"
	IdentityOpportunityDisposition IdentityKind = "opportunity_disposition"
	IdentityOutcome                IdentityKind = "outcome"
)

type Identity struct {
	ID               string
	DeterministicKey string
}

func NewIdentity(kind IdentityKind, components ...string) (Identity, error) {
	prefix, ok := identityPrefix(kind)
	if !ok {
		return Identity{}, fmt.Errorf("unsupported MarketOps identity kind %q", kind)
	}
	if len(components) == 0 {
		return Identity{}, errors.New("MarketOps identity requires at least one component")
	}

	hash := sha256.New()
	writeIdentityPart(hash, string(kind))
	for index, component := range components {
		component = strings.TrimSpace(component)
		if component == "" {
			return Identity{}, fmt.Errorf("MarketOps identity component %d is required", index)
		}
		writeIdentityPart(hash, component)
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	return Identity{
		ID:               prefix + digest[:32],
		DeterministicKey: "marketops." + string(kind) + ".v1:" + digest,
	}, nil
}

func CanonicalDimensions(raw []byte) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return "{}", nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var dimensions map[string]any
	if err := decoder.Decode(&dimensions); err != nil {
		return "", fmt.Errorf("decode MarketOps dimensions: %w", err)
	}
	if dimensions == nil {
		return "", errors.New("MarketOps dimensions must be a JSON object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return "", errors.New("MarketOps dimensions must contain one JSON object")
	}
	canonical, err := json.Marshal(dimensions)
	if err != nil {
		return "", fmt.Errorf("encode MarketOps dimensions: %w", err)
	}
	return string(canonical), nil
}

func identityPrefix(kind IdentityKind) (string, bool) {
	switch kind {
	case IdentityFeatureObservation:
		return "mfeat_", true
	case IdentityMarketState:
		return "mstate_", true
	case IdentityStateTransition:
		return "mtrans_", true
	case IdentityEvidence:
		return "mevidence_", true
	case IdentityHypothesisEvaluation:
		return "mhypeval_", true
	case IdentityOpportunity:
		return "mopp_", true
	case IdentitySignalProposal:
		return "msigprop_", true
	case IdentityOpportunityDisposition:
		return "moppdisp_", true
	case IdentityOutcome:
		return "moutcome_", true
	default:
		return "", false
	}
}

type identityWriter interface {
	Write([]byte) (int, error)
}

func writeIdentityPart(writer identityWriter, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = writer.Write([]byte(value))
}
