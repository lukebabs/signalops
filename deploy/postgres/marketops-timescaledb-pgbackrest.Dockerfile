FROM timescale/timescaledb:2.17.2-pg16

RUN apk add --no-cache pgbackrest ca-certificates
