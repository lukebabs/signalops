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

FROM gcr.io/distroless/static-debian12:nonroot AS gateway

COPY --from=build /out/signalops-gateway /signalops-gateway

EXPOSE 8080

ENTRYPOINT ["/signalops-gateway"]

FROM gcr.io/distroless/static-debian12:nonroot AS massive-puller

COPY --from=build /out/signalops-massive-puller /signalops-massive-puller

ENTRYPOINT ["/signalops-massive-puller"]


FROM gcr.io/distroless/static-debian12:nonroot AS massive-scheduler

COPY --from=build /out/signalops-massive-scheduler /signalops-massive-scheduler

ENTRYPOINT ["/signalops-massive-scheduler"]
