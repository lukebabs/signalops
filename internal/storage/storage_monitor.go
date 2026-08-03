package storage

import (
	"context"
	"time"
)

type StorageMonitorSnapshot struct {
	StoreID                             string
	ObservedAt                          time.Time
	UsedBytes, CapacityBytes, FreeBytes int64
	Status                              string
	DetailJSON                          []byte
}
type StorageMonitorRepository interface {
	ListStorageMonitorSnapshots(context.Context, string, time.Time, int) ([]StorageMonitorSnapshot, error)
}
