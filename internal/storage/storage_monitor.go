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
type StorageComponentSnapshot struct {
	StoreID, ComponentKind, ComponentName, AppID, Domain, AttributionMethod string
	ObservedAt                                                              time.Time
	PhysicalBytes, AttributedBytes                                          int64
	MetadataJSON                                                            []byte
}
type StorageMonitorRepository interface {
	ListStorageMonitorSnapshots(context.Context, string, time.Time, int) ([]StorageMonitorSnapshot, error)
}
type StorageAnalysisRepository interface {
	ListStorageComponentSnapshots(context.Context, time.Time, int) ([]StorageComponentSnapshot, error)
}
