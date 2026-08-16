FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-gateway ./cmd/gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-storage-monitor ./cmd/storage-monitor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-administration-notification-recorder ./cmd/administration-notification-recorder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-massive-puller ./cmd/massive-puller
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-massive-scheduler ./cmd/massive-scheduler
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-normalizer ./cmd/normalizer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-cyberops-connect-persister ./cmd/cyberops-connect-persister
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-cyberops-connect-outbox ./cmd/cyberops-connect-outbox
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-cyberops-normalizer ./cmd/cyberops-normalizer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-cyberops-detector ./cmd/cyberops-detector
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-cyberops-iot-anomaly ./cmd/cyberops-iot-anomaly
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-cyberops-hourly-feature-materializer ./cmd/cyberops-hourly-feature-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-cyberops-daily-feature-materializer ./cmd/cyberops-daily-feature-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-retention-governor ./cmd/retention-governor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-signal-persister ./cmd/signal-persister
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-signal-assurance-registrar ./cmd/marketops-signal-assurance-registrar
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-signal-assurance-worker ./cmd/marketops-signal-assurance-worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-signal-assurance-outbox ./cmd/marketops-signal-assurance-outbox
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-replay-worker ./cmd/replay-worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-backtest ./cmd/marketops-backtest
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-hypothesis-backtest ./cmd/marketops-hypothesis-backtest
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-feature-materializer ./cmd/marketops-options-feature-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-chain-ingestor ./cmd/marketops-options-chain-ingestor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-distribution-backfill ./cmd/marketops-options-distribution-backfill
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-coverage-runner ./cmd/marketops-options-coverage-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-state-materializer ./cmd/marketops-state-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-hypothesis-evaluator ./cmd/marketops-hypothesis-evaluator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-intelligence-graph-mapper ./cmd/marketops-intelligence-graph-mapper
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-intelligence-cohort-runner ./cmd/marketops-intelligence-cohort-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-syncratic-intelligence-runner ./cmd/marketops-syncratic-intelligence-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-hypothesis-proposal-generator ./cmd/marketops-hypothesis-proposal-generator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-opportunity-builder ./cmd/marketops-opportunity-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-outcome-materializer ./cmd/marketops-outcome-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-history-runner ./cmd/marketops-history-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-sri-runner ./cmd/marketops-sri-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-sri-holdings-runner ./cmd/marketops-sri-holdings-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-algorithm-evaluator ./cmd/marketops-algorithm-evaluator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-algorithm-evaluation-backfill ./cmd/marketops-algorithm-evaluation-backfill
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-intraday-monitor ./cmd/marketops-intraday-monitor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-valuation-runner ./cmd/marketops-valuation-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-tactical-valuation-runner ./cmd/marketops-tactical-valuation-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-eroc-runner ./cmd/marketops-eroc-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-eeom-runner ./cmd/marketops-eeom-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-algorithm-runner ./cmd/algorithm-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-algorithm-adjudicator ./cmd/marketops-algorithm-adjudicator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-asset-backfill-worker ./cmd/marketops-asset-backfill-worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-algorithm-proposal-generator ./cmd/algorithm-proposal-generator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-subscriber-global-marketops-parity-manifest ./cmd/subscriber-global-marketops-parity-manifest
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-subscriber-global-marketops-evidence-materializer ./cmd/subscriber-global-marketops-evidence-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-subscriber-global-eod-history-materializer ./cmd/subscriber-global-eod-history-materializer

FROM python:3.12-slim AS gateway

WORKDIR /app

COPY --from=build /out/signalops-gateway /usr/local/bin/signalops-gateway
COPY python ./python
COPY contracts ./contracts

ENV PYTHONPATH=/app/python

EXPOSE 8080

ENTRYPOINT ["signalops-gateway"]

FROM gcr.io/distroless/static-debian12:nonroot AS massive-puller

COPY --from=build /out/signalops-massive-puller /signalops-massive-puller

ENTRYPOINT ["/signalops-massive-puller"]


FROM gcr.io/distroless/static-debian12:nonroot AS massive-scheduler

COPY --from=build /out/signalops-massive-scheduler /signalops-massive-scheduler

ENTRYPOINT ["/signalops-massive-scheduler"]

FROM gcr.io/distroless/static-debian12:nonroot AS normalizer

COPY --from=build /out/signalops-normalizer /signalops-normalizer

ENTRYPOINT ["/signalops-normalizer"]

FROM gcr.io/distroless/static-debian12:nonroot AS cyberops-connect-persister

COPY --from=build /out/signalops-cyberops-connect-persister /signalops-cyberops-connect-persister

ENTRYPOINT ["/signalops-cyberops-connect-persister"]

FROM gcr.io/distroless/static-debian12:nonroot AS cyberops-connect-outbox

COPY --from=build /out/signalops-cyberops-connect-outbox /signalops-cyberops-connect-outbox

ENTRYPOINT ["/signalops-cyberops-connect-outbox"]

FROM gcr.io/distroless/static-debian12:nonroot AS cyberops-normalizer

COPY --from=build /out/signalops-cyberops-normalizer /signalops-cyberops-normalizer

ENTRYPOINT ["/signalops-cyberops-normalizer"]

FROM gcr.io/distroless/static-debian12:nonroot AS cyberops-detector

COPY --from=build /out/signalops-cyberops-detector /signalops-cyberops-detector

ENTRYPOINT ["/signalops-cyberops-detector"]

FROM gcr.io/distroless/static-debian12:nonroot AS cyberops-iot-anomaly

COPY --from=build /out/signalops-cyberops-iot-anomaly /signalops-cyberops-iot-anomaly

ENTRYPOINT ["/signalops-cyberops-iot-anomaly"]

FROM gcr.io/distroless/static-debian12:nonroot AS cyberops-hourly-feature-materializer

COPY --from=build /out/signalops-cyberops-hourly-feature-materializer /signalops-cyberops-hourly-feature-materializer

ENTRYPOINT ["/signalops-cyberops-hourly-feature-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS signal-persister

COPY --from=build /out/signalops-signal-persister /signalops-signal-persister

ENTRYPOINT ["/signalops-signal-persister"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-signal-assurance-registrar
COPY --from=build /out/signalops-marketops-signal-assurance-registrar /signalops-marketops-signal-assurance-registrar
ENTRYPOINT ["/signalops-marketops-signal-assurance-registrar"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-signal-assurance-worker
COPY --from=build /out/signalops-marketops-signal-assurance-worker /signalops-marketops-signal-assurance-worker
ENTRYPOINT ["/signalops-marketops-signal-assurance-worker"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-signal-assurance-outbox
COPY --from=build /out/signalops-marketops-signal-assurance-outbox /signalops-marketops-signal-assurance-outbox
ENTRYPOINT ["/signalops-marketops-signal-assurance-outbox"]

FROM gcr.io/distroless/static-debian12:nonroot AS replay-worker

COPY --from=build /out/signalops-replay-worker /signalops-replay-worker

ENTRYPOINT ["/signalops-replay-worker"]

FROM python:3.12-slim AS marketops-backtest

WORKDIR /app

COPY --from=build /out/signalops-marketops-backtest /usr/local/bin/signalops-marketops-backtest
COPY python ./python
COPY contracts ./contracts

ENV PYTHONPATH=/app/python

ENTRYPOINT ["signalops-marketops-backtest"]

FROM python:3.12-slim AS algorithm-runner

WORKDIR /app
COPY python/requirements-worker.txt ./python/requirements-worker.txt
RUN pip install --no-cache-dir -r ./python/requirements-worker.txt
COPY --from=build /out/signalops-algorithm-runner /usr/local/bin/signalops-algorithm-runner
COPY python ./python
ENV PYTHONPATH=/app/python SIGNALOPS_PYTHON_ALGORITHM_RUNTIME=true
ENTRYPOINT ["signalops-algorithm-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-algorithm-adjudicator
COPY --from=build /out/signalops-marketops-algorithm-adjudicator /signalops-marketops-algorithm-adjudicator
ENTRYPOINT ["/signalops-marketops-algorithm-adjudicator"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-options-feature-materializer

COPY --from=build /out/signalops-marketops-options-feature-materializer /signalops-marketops-options-feature-materializer

ENTRYPOINT ["/signalops-marketops-options-feature-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-options-chain-ingestor

COPY --from=build /out/signalops-marketops-options-chain-ingestor /signalops-marketops-options-chain-ingestor

ENTRYPOINT ["/signalops-marketops-options-chain-ingestor"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-asset-backfill-worker
WORKDIR /app
COPY --from=build /out/signalops-marketops-asset-backfill-worker /signalops-marketops-asset-backfill-worker
USER nonroot
ENTRYPOINT ["/signalops-marketops-asset-backfill-worker"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-options-distribution-backfill

COPY --from=build /out/signalops-marketops-options-distribution-backfill /signalops-marketops-options-distribution-backfill

ENTRYPOINT ["/signalops-marketops-options-distribution-backfill"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-options-coverage-runner

COPY --from=build /out/signalops-marketops-options-coverage-runner /signalops-marketops-options-coverage-runner

ENTRYPOINT ["/signalops-marketops-options-coverage-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-hypothesis-backtest

COPY --from=build /out/signalops-marketops-hypothesis-backtest /signalops-marketops-hypothesis-backtest

ENTRYPOINT ["/signalops-marketops-hypothesis-backtest"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-state-materializer

COPY --from=build /out/signalops-marketops-state-materializer /signalops-marketops-state-materializer

ENTRYPOINT ["/signalops-marketops-state-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-hypothesis-evaluator

COPY --from=build /out/signalops-marketops-hypothesis-evaluator /signalops-marketops-hypothesis-evaluator

ENTRYPOINT ["/signalops-marketops-hypothesis-evaluator"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-hypothesis-proposal-generator

COPY --from=build /out/signalops-marketops-hypothesis-proposal-generator /signalops-marketops-hypothesis-proposal-generator

ENTRYPOINT ["/signalops-marketops-hypothesis-proposal-generator"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-opportunity-builder

COPY --from=build /out/signalops-marketops-opportunity-builder /signalops-marketops-opportunity-builder

ENTRYPOINT ["/signalops-marketops-opportunity-builder"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-outcome-materializer

COPY --from=build /out/signalops-marketops-outcome-materializer /signalops-marketops-outcome-materializer

ENTRYPOINT ["/signalops-marketops-outcome-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-intelligence-graph-mapper

COPY --from=build /out/signalops-marketops-intelligence-graph-mapper /signalops-marketops-intelligence-graph-mapper

ENTRYPOINT ["/signalops-marketops-intelligence-graph-mapper"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-history-runner

COPY --from=build /out/signalops-marketops-history-runner /signalops-marketops-history-runner

ENTRYPOINT ["/signalops-marketops-history-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-sri-runner

COPY --from=build /out/signalops-marketops-sri-runner /signalops-marketops-sri-runner

ENTRYPOINT ["/signalops-marketops-sri-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-sri-holdings-runner

COPY --from=build /out/signalops-marketops-sri-holdings-runner /signalops-marketops-sri-holdings-runner

ENTRYPOINT ["/signalops-marketops-sri-holdings-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-algorithm-evaluator

COPY --from=build /out/signalops-marketops-algorithm-evaluator /signalops-marketops-algorithm-evaluator

ENTRYPOINT ["/signalops-marketops-algorithm-evaluator"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-algorithm-evaluation-backfill

COPY --from=build /out/signalops-marketops-algorithm-evaluation-backfill /signalops-marketops-algorithm-evaluation-backfill
COPY --from=build /out/signalops-massive-puller /usr/local/bin/signalops-massive-puller
ENV SIGNALOPS_MASSIVE_PULLER_BIN=/usr/local/bin/signalops-massive-puller

ENTRYPOINT ["/signalops-marketops-algorithm-evaluation-backfill"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-intraday-monitor

COPY --from=build /out/signalops-marketops-intraday-monitor /signalops-marketops-intraday-monitor

ENTRYPOINT ["/signalops-marketops-intraday-monitor"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-valuation-runner

COPY --from=build /out/signalops-marketops-valuation-runner /signalops-marketops-valuation-runner

ENTRYPOINT ["/signalops-marketops-valuation-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-tactical-valuation-runner

COPY --from=build /out/signalops-marketops-tactical-valuation-runner /signalops-marketops-tactical-valuation-runner

ENTRYPOINT ["/signalops-marketops-tactical-valuation-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-eroc-runner
COPY --from=build /out/signalops-marketops-eroc-runner /signalops-marketops-eroc-runner
ENTRYPOINT ["/signalops-marketops-eroc-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-eeom-runner
COPY --from=build /out/signalops-marketops-eeom-runner /signalops-marketops-eeom-runner
ENTRYPOINT ["/signalops-marketops-eeom-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS algorithm-proposal-generator

COPY --from=build /out/signalops-algorithm-proposal-generator /signalops-algorithm-proposal-generator

ENTRYPOINT ["/signalops-algorithm-proposal-generator"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-syncratic-intelligence-runner
COPY --from=build /out/signalops-marketops-syncratic-intelligence-runner /signalops-marketops-syncratic-intelligence-runner
ENTRYPOINT ["/signalops-marketops-syncratic-intelligence-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-intelligence-cohort-runner

COPY --from=build /out/signalops-marketops-intelligence-cohort-runner /usr/local/bin/signalops-marketops-intelligence-cohort-runner
COPY --from=build /out/signalops-marketops-state-materializer /usr/local/bin/signalops-marketops-state-materializer
COPY --from=build /out/signalops-marketops-hypothesis-evaluator /usr/local/bin/signalops-marketops-hypothesis-evaluator
COPY --from=build /out/signalops-marketops-opportunity-builder /usr/local/bin/signalops-marketops-opportunity-builder
COPY --from=build /out/signalops-marketops-outcome-materializer /usr/local/bin/signalops-marketops-outcome-materializer
COPY --from=build /out/signalops-marketops-hypothesis-proposal-generator /usr/local/bin/signalops-marketops-hypothesis-proposal-generator

ENTRYPOINT ["/usr/local/bin/signalops-marketops-intelligence-cohort-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS storage-monitor
COPY --from=build /out/signalops-storage-monitor /signalops-storage-monitor
ENTRYPOINT ["/signalops-storage-monitor"]

FROM gcr.io/distroless/static-debian12:nonroot AS cyberops-daily-feature-materializer
COPY --from=build /out/signalops-cyberops-daily-feature-materializer /signalops-cyberops-daily-feature-materializer
ENTRYPOINT ["/signalops-cyberops-daily-feature-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS retention-governor
COPY --from=build /out/signalops-retention-governor /signalops-retention-governor
ENTRYPOINT ["/signalops-retention-governor"]

FROM gcr.io/distroless/static-debian12:nonroot AS administration-notification-recorder
COPY --from=build /out/signalops-administration-notification-recorder /signalops-administration-notification-recorder
ENTRYPOINT ["/signalops-administration-notification-recorder"]


FROM gcr.io/distroless/static-debian12:nonroot AS subscriber-global-marketops-parity-manifest
COPY --from=build /out/signalops-subscriber-global-marketops-parity-manifest /signalops-subscriber-global-marketops-parity-manifest
ENTRYPOINT ["/signalops-subscriber-global-marketops-parity-manifest"]

FROM gcr.io/distroless/static-debian12:nonroot AS subscriber-global-marketops-evidence-materializer
COPY --from=build /out/signalops-subscriber-global-marketops-evidence-materializer /signalops-subscriber-global-marketops-evidence-materializer
ENTRYPOINT ["/signalops-subscriber-global-marketops-evidence-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS subscriber-global-eod-history-materializer
COPY --from=build /out/signalops-subscriber-global-eod-history-materializer /signalops-subscriber-global-eod-history-materializer
ENTRYPOINT ["/signalops-subscriber-global-eod-history-materializer"]
