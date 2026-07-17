// Pure display helpers for the G128 MarketOps asset options intelligence panel.
// Coverage / distribution / chain JSON arrives already-parsed from the gateway
// (typed `unknown` on flexible fields). Narrow with type guards only; never
// JSON.parse. Missing/malformed scalars collapse to 0 / '' / null and must never
// throw. These power a read-only analyst surface — they never trigger ingestion,
// live-preview, algorithm execution, or signal materialization.

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function asNumber(v: unknown): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return 0;
}

function asString(v: unknown): string {
  return typeof v === 'string' ? v : '';
}

function asStringArray(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
}

// Server-nullable numerics (omitempty pointers on the chain DTO) -> number | null
// so the UI can render absent values as `—` rather than 0 (critical for
// open_interest / volume, which weaken call/put ratio interpretation).
function asNullableNumber(v: unknown): number | null {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return null;
}

// Known bucket key order (backend-owned). Unknown future keys append after the
// known ones so the display stays stable and forward-compatible.
export const MONEYNESS_BUCKET_ORDER = ['<90%', '90-95%', '95-100%', '100-105%', '105-110%', '>110%', 'unknown'] as const;
export const EXPIRATION_BUCKET_ORDER = ['0-7d', '8-30d', '31-60d', '61d+'] as const;

export interface MarketOpsOptionsBucketEntry {
  key: string;
  callOpenInterest: number;
  putOpenInterest: number;
  callVolume: number;
  putVolume: number;
  contractCount: number;
}

function bucketNumber(bucket: unknown, field: string): number {
  if (!isRecord(bucket)) return 0;
  return asNumber(bucket[field]);
}

// Coerce a backend bucket map (Record<string, bucket totals>) into stable,
// ordered entries. Known keys render first (per `order`), then any unknown keys.
// Missing per-bucket fields collapse to 0. Never throws.
export function optionsBucketEntries(map: unknown, order: readonly string[]): MarketOpsOptionsBucketEntry[] {
  if (!isRecord(map)) return [];
  const present = Object.keys(map);
  const seen = new Set<string>();
  const ordered: string[] = [];
  for (const k of order) {
    if (present.includes(k)) {
      ordered.push(k);
      seen.add(k);
    }
  }
  for (const k of present) {
    if (!seen.has(k)) ordered.push(k);
  }
  return ordered.map((key) => {
    const b = map[key];
    return {
      key,
      callOpenInterest: bucketNumber(b, 'call_open_interest'),
      putOpenInterest: bucketNumber(b, 'put_open_interest'),
      callVolume: bucketNumber(b, 'call_volume'),
      putVolume: bucketNumber(b, 'put_volume'),
      contractCount: bucketNumber(b, 'contract_count'),
    };
  });
}

export interface MarketOpsOptionsCoverageView {
  symbol: string;
  tradeDayCount: number;
  contractCount: number;
  firstTradeDate: string;
  lastTradeDate: string;
  lastUpdatedAt: string;
}

const EMPTY_COVERAGE: MarketOpsOptionsCoverageView = {
  symbol: '',
  tradeDayCount: 0,
  contractCount: 0,
  firstTradeDate: '',
  lastTradeDate: '',
  lastUpdatedAt: '',
};

export function summarizeMarketOpsOptionsCoverage(c: unknown): MarketOpsOptionsCoverageView {
  if (!isRecord(c)) return { ...EMPTY_COVERAGE };
  return {
    symbol: asString(c.symbol),
    tradeDayCount: asNumber(c.trade_day_count),
    contractCount: asNumber(c.contract_count),
    firstTradeDate: asString(c.first_trade_date),
    lastTradeDate: asString(c.last_trade_date),
    lastUpdatedAt: asString(c.last_updated_at),
  };
}

export interface MarketOpsOptionsDistributionView {
  tradeDate: string;
  windowName: string;
  provider: string;
  sourceId: string;
  tradeDays: number;
  contractCount: number;
  callContractCount: number;
  putContractCount: number;
  totalCallOpenInterest: number;
  totalPutOpenInterest: number;
  totalCallVolume: number;
  totalPutVolume: number;
  missingOpenInterestCount: number;
  callPutOpenInterestRatio: number;
  callPutVolumeRatio: number;
  ratioDelta: number;
  ratioChangePct: number;
  ratioZScore: number;
  changePointScore: number;
  confidence: number;
  moneynessBuckets: MarketOpsOptionsBucketEntry[];
  expirationBuckets: MarketOpsOptionsBucketEntry[];
  metrics: unknown;
  sourceTradeDates: string[];
  updatedAt: string;
}

const EMPTY_DISTRIBUTION: MarketOpsOptionsDistributionView = {
  tradeDate: '',
  windowName: '',
  provider: '',
  sourceId: '',
  tradeDays: 0,
  contractCount: 0,
  callContractCount: 0,
  putContractCount: 0,
  totalCallOpenInterest: 0,
  totalPutOpenInterest: 0,
  totalCallVolume: 0,
  totalPutVolume: 0,
  missingOpenInterestCount: 0,
  callPutOpenInterestRatio: 0,
  callPutVolumeRatio: 0,
  ratioDelta: 0,
  ratioChangePct: 0,
  ratioZScore: 0,
  changePointScore: 0,
  confidence: 0,
  moneynessBuckets: [],
  expirationBuckets: [],
  metrics: {},
  sourceTradeDates: [],
  updatedAt: '',
};

export function summarizeMarketOpsOptionsDistribution(d: unknown): MarketOpsOptionsDistributionView {
  if (!isRecord(d)) return { ...EMPTY_DISTRIBUTION };
  return {
    tradeDate: asString(d.trade_date),
    windowName: asString(d.window_name),
    provider: asString(d.provider),
    sourceId: asString(d.source_id),
    tradeDays: asNumber(d.trade_days),
    contractCount: asNumber(d.contract_count),
    callContractCount: asNumber(d.call_contract_count),
    putContractCount: asNumber(d.put_contract_count),
    totalCallOpenInterest: asNumber(d.total_call_open_interest),
    totalPutOpenInterest: asNumber(d.total_put_open_interest),
    totalCallVolume: asNumber(d.total_call_volume),
    totalPutVolume: asNumber(d.total_put_volume),
    missingOpenInterestCount: asNumber(d.missing_open_interest_count),
    callPutOpenInterestRatio: asNumber(d.call_put_open_interest_ratio),
    callPutVolumeRatio: asNumber(d.call_put_volume_ratio),
    ratioDelta: asNumber(d.ratio_delta),
    ratioChangePct: asNumber(d.ratio_change_pct),
    ratioZScore: asNumber(d.ratio_zscore),
    changePointScore: asNumber(d.change_point_score),
    confidence: asNumber(d.confidence),
    moneynessBuckets: optionsBucketEntries(d.moneyness_distribution, MONEYNESS_BUCKET_ORDER),
    expirationBuckets: optionsBucketEntries(d.expiration_distribution, EXPIRATION_BUCKET_ORDER),
    metrics: d.metrics ?? {},
    sourceTradeDates: asStringArray(d.source_trade_dates),
    updatedAt: asString(d.updated_at),
  };
}

export interface MarketOpsOptionsChainRowView {
  optionTicker: string;
  contractType: string;
  tradeDate: string;
  expirationDate: string;
  strikePrice: number;
  underlyingClose: number | null;
  moneyness: number | null;
  volume: number | null;
  openInterest: number | null;
  impliedVolatility: number | null;
  delta: number | null;
  gamma: number | null;
  theta: number | null;
  vega: number | null;
  provider: string;
  sourceId: string;
  ingestionRunId: string;
  payloadHash: string;
  updatedAt: string;
  rawPayload: unknown;
}

const EMPTY_CHAIN_ROW: MarketOpsOptionsChainRowView = {
  optionTicker: '',
  contractType: '',
  tradeDate: '',
  expirationDate: '',
  strikePrice: 0,
  underlyingClose: null,
  moneyness: null,
  volume: null,
  openInterest: null,
  impliedVolatility: null,
  delta: null,
  gamma: null,
  theta: null,
  vega: null,
  provider: '',
  sourceId: '',
  ingestionRunId: '',
  payloadHash: '',
  updatedAt: '',
  rawPayload: {},
};

export function summarizeMarketOpsOptionsChainRow(r: unknown): MarketOpsOptionsChainRowView {
  if (!isRecord(r)) return { ...EMPTY_CHAIN_ROW };
  return {
    optionTicker: asString(r.option_ticker),
    contractType: asString(r.contract_type),
    tradeDate: asString(r.trade_date),
    expirationDate: asString(r.expiration_date),
    strikePrice: asNumber(r.strike_price),
    underlyingClose: asNullableNumber(r.underlying_close),
    moneyness: asNullableNumber(r.moneyness),
    volume: asNullableNumber(r.volume),
    openInterest: asNullableNumber(r.open_interest),
    impliedVolatility: asNullableNumber(r.implied_volatility),
    delta: asNullableNumber(r.delta),
    gamma: asNullableNumber(r.gamma),
    theta: asNullableNumber(r.theta),
    vega: asNullableNumber(r.vega),
    provider: asString(r.provider),
    sourceId: asString(r.source_id),
    ingestionRunId: asString(r.ingestion_run_id),
    payloadHash: asString(r.payload_hash),
    updatedAt: asString(r.updated_at),
    rawPayload: r.raw_payload ?? {},
  };
}

// Extract the YYYY-MM-DD date-only portion from an RFC3339 timestamp for the
// chain trade_date query param (the gateway validates it as a calendar date).
export function marketOpsOptionsDateOnly(iso: string): string {
  return asString(iso).slice(0, 10);
}
