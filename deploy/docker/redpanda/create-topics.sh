#!/bin/sh
set -eu

BROKERS="${SIGNALOPS_BROKERS:-redpanda:9092}"
PARTITIONS="${SIGNALOPS_TOPIC_PARTITIONS:-3}"
REPLICAS="${SIGNALOPS_TOPIC_REPLICAS:-1}"
ENVIRONMENT="${SIGNALOPS_ENV:-local}"

topics="
signalops.${ENVIRONMENT}.raw.v1
signalops.${ENVIRONMENT}.normalized.v1
signalops.${ENVIRONMENT}.signal.v1
signalops.${ENVIRONMENT}.artifact.v1
signalops.${ENVIRONMENT}.graph_mutation.v1
signalops.${ENVIRONMENT}.insight_candidate.v1
signalops.${ENVIRONMENT}.retry.algorithm.v1
signalops.${ENVIRONMENT}.dlq.algorithm.v1
"

for topic in $topics; do
  rpk topic create "$topic" \
    --brokers "$BROKERS" \
    --partitions "$PARTITIONS" \
    --replicas "$REPLICAS" \
    --if-not-exists
done

rpk topic list --brokers "$BROKERS"
