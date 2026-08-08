package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// ResolveSignalValidationContract implements the v1.1 deterministic registry
// rule: exact algorithm/version wins, generic fallback is valid only when it
// is the sole applicable candidate.
func (r *Repository) ResolveSignalValidationContract(ctx context.Context, event storage.SignalAssuranceEligibleEvent) (storage.SignalValidationContractRecord, error) {
	rows, err := r.db.QueryContext(ctx, signalValidationContractSelect+`
WHERE active=true AND signal_type=$1 AND direction=$2
 AND (contract_id=$3 OR signal_type || ':' || contract_version=$3)
 AND (algorithm IS NULL OR algorithm=$4)
 AND (algorithm_version IS NULL OR algorithm_version=$5)
ORDER BY CASE WHEN algorithm=$4 AND algorithm_version=$5 THEN 0 ELSE 1 END, contract_id
LIMIT 2`, event.SignalType, event.Direction, event.ValidationContractRef, event.Algorithm, event.AlgorithmVersion)
	if err != nil {
		return storage.SignalValidationContractRecord{}, fmt.Errorf("resolve signal validation contract: %w", err)
	}
	defer rows.Close()
	candidates := []storage.SignalValidationContractRecord{}
	for rows.Next() {
		x, scanErr := scanSignalValidationContract(rows)
		if scanErr != nil {
			return storage.SignalValidationContractRecord{}, scanErr
		}
		candidates = append(candidates, x)
	}
	if err := rows.Err(); err != nil {
		return storage.SignalValidationContractRecord{}, err
	}
	if len(candidates) == 0 {
		return storage.SignalValidationContractRecord{}, fmt.Errorf("no active validation contract for %s", strings.TrimSpace(event.ValidationContractRef))
	}
	if len(candidates) > 1 {
		exact := candidates[0].Algorithm == event.Algorithm && candidates[0].AlgorithmVersion == event.AlgorithmVersion
		if !exact {
			return storage.SignalValidationContractRecord{}, fmt.Errorf("ambiguous validation contract for %s", strings.TrimSpace(event.ValidationContractRef))
		}
	}
	return candidates[0], nil
}
