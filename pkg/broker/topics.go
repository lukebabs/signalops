package broker

import (
	"fmt"
	"strings"
)

const (
	DefaultEnvironment = "local"

	RawTopic              = "raw"
	NormalizedTopic       = "normalized"
	SignalTopic           = "signal"
	ArtifactTopic         = "artifact"
	GraphMutationTopic    = "graph_mutation"
	InsightCandidateTopic = "insight_candidate"
	RetryAlgorithmTopic   = "retry.algorithm"
	DLQAlgorithmTopic     = "dlq.algorithm"
)

var durableTopicNames = []string{
	RawTopic,
	NormalizedTopic,
	SignalTopic,
	ArtifactTopic,
	GraphMutationTopic,
	InsightCandidateTopic,
	RetryAlgorithmTopic,
	DLQAlgorithmTopic,
}

// TopicName returns the durable SignalOps topic name for an environment.
func TopicName(environment, name string) string {
	environment = strings.TrimSpace(environment)
	if environment == "" {
		environment = DefaultEnvironment
	}

	return fmt.Sprintf("signalops.%s.%s.v1", environment, name)
}

// DurableTopics returns all durable SignalOps topics for an environment.
func DurableTopics(environment string) []string {
	topics := make([]string, 0, len(durableTopicNames))
	for _, name := range durableTopicNames {
		topics = append(topics, TopicName(environment, name))
	}
	return topics
}
