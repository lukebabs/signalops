# Massive Market Data Adapter Seeds

This directory contains the first market-data universe seed for the Massive
(formerly Polygon.io) adapter.

## Top 50 Megacap Seed

Source text:

```text
top50megacap.txt
```

Normalized DB-seed artifact:

```text
top50megacap.normalized.csv
```

Parser:

```text
megacap_seed.go
```

The parser exposes `TopMegacapCompanies()`, returning records with:

- `rank`
- `ticker`
- `ticker_key`
- `company`
- `company_key`
- `sector`
- `sector_key`
- `industry`
- `industry_key`

Normalization rules:

- Tickers are uppercased for display/storage.
- `ticker_key` lowercases tickers and converts exchange/class separators such
  as `.` and `-` to `_`.
- Company, sector, and industry keys are lowercase snake-case strings.
- Lines with `Sector / Industry` are split into primary sector and industry.
- Lines with only a sector leave industry blank.
