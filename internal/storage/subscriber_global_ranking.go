package storage

import (
	"context"
	"time"
)

type SubscriberGlobalRankingSnapshotEntry struct {
	SelectionRank   int
	SourceRank      int
	ProviderSymbol  string
	CompanyName     string
	MarketCapRaw    string
	RevenueRaw      string
	SourceRowSHA256 string
}

type SubscriberGlobalRankingSnapshotImport struct {
	RankingSnapshotID       string
	SourceLabel             string
	SourceSHA256            string
	AsOfDate                time.Time
	RequestedCapacity       int
	SourceRowsExamined      int
	DuplicateSymbolsSkipped int
	ImportedBy              string
	ProvenanceJSON          []byte
	Entries                 []SubscriberGlobalRankingSnapshotEntry
}

type SubscriberGlobalRankingSnapshotRepository interface {
	ImportSubscriberGlobalRankingSnapshot(context.Context, SubscriberGlobalRankingSnapshotImport) (SubscriberGlobalRankingSnapshotImport, error)
}
