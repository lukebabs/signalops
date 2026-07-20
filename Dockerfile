FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-gateway ./cmd/gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-massive-puller ./cmd/massive-puller
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-massive-scheduler ./cmd/massive-scheduler
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-normalizer ./cmd/normalizer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-signal-persister ./cmd/signal-persister
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-replay-worker ./cmd/replay-worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-backtest ./cmd/marketops-backtest
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-hypothesis-backtest ./cmd/marketops-hypothesis-backtest
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-feature-materializer ./cmd/marketops-options-feature-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-chain-ingestor ./cmd/marketops-options-chain-ingestor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-distribution-backfill ./cmd/marketops-options-distribution-backfill
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-options-coverage-runner ./cmd/marketops-options-coverage-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-state-materializer ./cmd/marketops-state-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-hypothesis-evaluator ./cmd/marketops-hypothesis-evaluator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-opportunity-builder ./cmd/marketops-opportunity-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-outcome-materializer ./cmd/marketops-outcome-materializer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-marketops-history-runner ./cmd/marketops-history-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-algorithm-runner ./cmd/algorithm-runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-algorithm-proposal-generator ./cmd/algorithm-proposal-generator

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

FROM gcr.io/distroless/static-debian12:nonroot AS signal-persister

COPY --from=build /out/signalops-signal-persister /signalops-signal-persister

ENTRYPOINT ["/signalops-signal-persister"]

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

FROM gcr.io/distroless/static-debian12:nonroot AS algorithm-runner

COPY --from=build /out/signalops-algorithm-runner /signalops-algorithm-runner

ENTRYPOINT ["/signalops-algorithm-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-options-feature-materializer

COPY --from=build /out/signalops-marketops-options-feature-materializer /signalops-marketops-options-feature-materializer

ENTRYPOINT ["/signalops-marketops-options-feature-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-options-chain-ingestor

COPY --from=build /out/signalops-marketops-options-chain-ingestor /signalops-marketops-options-chain-ingestor

ENTRYPOINT ["/signalops-marketops-options-chain-ingestor"]

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

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-opportunity-builder

COPY --from=build /out/signalops-marketops-opportunity-builder /signalops-marketops-opportunity-builder

ENTRYPOINT ["/signalops-marketops-opportunity-builder"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-outcome-materializer

COPY --from=build /out/signalops-marketops-outcome-materializer /signalops-marketops-outcome-materializer

ENTRYPOINT ["/signalops-marketops-outcome-materializer"]

FROM gcr.io/distroless/static-debian12:nonroot AS marketops-history-runner

COPY --from=build /out/signalops-marketops-history-runner /signalops-marketops-history-runner

ENTRYPOINT ["/signalops-marketops-history-runner"]

FROM gcr.io/distroless/static-debian12:nonroot AS algorithm-proposal-generator

COPY --from=build /out/signalops-algorithm-proposal-generator /signalops-algorithm-proposal-generator

ENTRYPOINT ["/signalops-algorithm-proposal-generator"]
