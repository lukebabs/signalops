package postgres

import (
	"context"
	"github.com/lukebabs/signalops/internal/storage"
	"time"
)

func (r *Repository) ListStorageMonitorSnapshots(ctx context.Context, store string, since time.Time, limit int) ([]storage.StorageMonitorSnapshot, error) {
	if limit <= 0 || limit > 3000 {
		limit = 3000
	}
	rows, err := r.db.QueryContext(ctx, `SELECT store_id,observed_at,used_bytes,capacity_bytes,free_bytes,status,detail FROM storage_monitor_snapshots WHERE ($1='' OR store_id=$1) AND observed_at >= $2 ORDER BY observed_at ASC LIMIT $3`, store, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.StorageMonitorSnapshot{}
	for rows.Next() {
		var x storage.StorageMonitorSnapshot
		if err = rows.Scan(&x.StoreID, &x.ObservedAt, &x.UsedBytes, &x.CapacityBytes, &x.FreeBytes, &x.Status, &x.DetailJSON); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
