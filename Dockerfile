FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/signalops-gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot AS gateway

COPY --from=build /out/signalops-gateway /signalops-gateway

EXPOSE 8080

ENTRYPOINT ["/signalops-gateway"]

