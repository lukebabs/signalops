package broker

import "testing"

func TestTopicNameDefaultsEnvironment(t *testing.T) {
	got := TopicName("", SignalTopic)
	want := "signalops.local.signal.v1"
	if got != want {
		t.Fatalf("TopicName() = %q, want %q", got, want)
	}
}

func TestDurableTopics(t *testing.T) {
	got := DurableTopics("dev")
	want := []string{
		"signalops.dev.raw.v1",
		"signalops.dev.normalized.v1",
		"signalops.dev.signal.v1",
		"signalops.dev.artifact.v1",
		"signalops.dev.graph_mutation.v1",
		"signalops.dev.insight_candidate.v1",
		"signalops.dev.retry.algorithm.v1",
		"signalops.dev.dlq.algorithm.v1",
	}

	if len(got) != len(want) {
		t.Fatalf("DurableTopics() length = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DurableTopics()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
