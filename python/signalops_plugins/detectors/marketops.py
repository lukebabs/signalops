from __future__ import annotations

import hashlib
from typing import Mapping, Sequence

from signalops_plugins.detectors.base import (
    DetectionResult,
    DetectorPlugin,
    EmittedSignal,
    Explanation,
    FeatureContext,
    RuntimeContext,
)


class MarketOpsEODPriceDetector(DetectorPlugin):
    detector_id = "marketops.dsm.eod_price_v1"
    detector_version = "0.1.0"
    model_version = "deterministic-v0"

    app_id = "marketops"
    domain = "market_data"
    source_adapter = "market_data.massive"
    dataset = "equity_eod_prices"
    use_case = "daily_market_surveillance"

    move_threshold_pct = 3.0
    range_threshold_pct = 5.0
    daily_return_threshold_pct = 4.0

    def __init__(self) -> None:
        self.initialized = False
        self.runtime_context: RuntimeContext | None = None

    def initialize(
        self,
        config: Mapping[str, object],
        model_registry: object | None,
        runtime_context: RuntimeContext,
    ) -> None:
        self.initialized = True
        self.runtime_context = runtime_context

    def detect(
        self,
        normalized_events: Sequence[Mapping[str, object]],
        feature_context: FeatureContext,
    ) -> DetectionResult:
        if len(normalized_events) != 1:
            return self._no_match("marketops eod detector expects one event", {})

        event = normalized_events[0]
        if not self._is_marketops_eod_event(event):
            return self._no_match("event is outside MarketOps Massive equity EOD scope", {})

        payload = _object(event.get("normalized_payload"))
        symbol = _symbol(payload, event)
        features = self._features(payload)
        quality_issues = self._quality_issues(features)
        metadata = {
            "event_id": _string(event.get("event_id")),
            "symbol": symbol,
            "features": features,
            "quality_issues": quality_issues,
        }

        if quality_issues:
            return DetectionResult(
                detector_id=self.detector_id,
                detector_version=self.detector_version,
                matched=True,
                score=0.99,
                reason=f"price quality exception for {symbol}",
                metadata={**metadata, "signal_type": "marketops.dsm.price_quality_exception"},
            )

        scores = [
            _ratio_abs(features.get("open_close_move_pct"), self.move_threshold_pct),
            _ratio_abs(features.get("intraday_range_pct"), self.range_threshold_pct),
            _ratio_abs(features.get("daily_return_pct"), self.daily_return_threshold_pct),
        ]
        score = max(scores)
        if score < 1:
            return self._no_match("no MarketOps DSM EOD threshold crossed", metadata)

        confidence = min(0.95, max(0.65, 0.55 + (score * 0.12)))
        return DetectionResult(
            detector_id=self.detector_id,
            detector_version=self.detector_version,
            matched=True,
            score=confidence,
            reason=f"volatility expansion threshold crossed for {symbol}",
            metadata={**metadata, "signal_type": "marketops.dsm.volatility_expansion"},
        )

    def explain(self, detection_result: DetectionResult) -> Explanation:
        metadata = dict(detection_result.metadata)
        features = _object(metadata.get("features"))
        symbol = _string(metadata.get("symbol")) or "unknown"
        summary = detection_result.reason
        evidence = [
            {
                "type": "computed_features",
                "ref": symbol,
                "summary": summary,
                "metadata": {
                    "open_close_move_pct": features.get("open_close_move_pct"),
                    "intraday_range_pct": features.get("intraday_range_pct"),
                    "vwap_distance_pct": features.get("vwap_distance_pct"),
                    "daily_return_pct": features.get("daily_return_pct"),
                    "quality_issues": metadata.get("quality_issues", []),
                },
            }
        ]
        return Explanation(summary=summary, evidence=evidence)

    def emit_signal(
        self,
        detection_result: DetectionResult,
        explanation: Explanation,
    ) -> EmittedSignal | None:
        if not detection_result.matched:
            return None

        metadata = dict(detection_result.metadata)
        signal_type = _string(metadata.get("signal_type"))
        event_id = _string(metadata.get("event_id"))
        symbol = _string(metadata.get("symbol")) or "UNKNOWN"
        features = _object(metadata.get("features"))
        quality_issues = _string_list(metadata.get("quality_issues"))
        severity = _severity(signal_type, features, quality_issues)
        artifact_id = _stable_artifact_id(signal_type, event_id, symbol)
        artifact = _dsm_artifact(
            artifact_id=artifact_id,
            signal_type=signal_type,
            event_id=event_id,
            symbol=symbol,
            features=features,
            quality_issues=quality_issues,
            confidence=detection_result.score,
            severity=severity,
            summary=explanation.summary,
        )
        graph_targets = _graph_proposals(
            artifact_id=artifact_id,
            signal_type=signal_type,
            symbol=symbol,
            confidence=detection_result.score,
            severity=severity,
        )

        return EmittedSignal(
            signal_id=_stable_signal_id(
                self.detector_id,
                signal_type,
                event_id,
                symbol,
                _string(features.get("observation_date")),
            ),
            signal_type=signal_type,
            confidence=detection_result.score,
            severity=severity,
            payload={
                "event_ids": [event_id] if event_id else [],
                "artifact_ids": [artifact_id],
                "entities": [
                    {
                        "type": "ticker",
                        "id": f"ticker:{symbol}",
                        "external_id": symbol,
                        "confidence": 1.0,
                    }
                ],
                "supporting_metrics": {
                    "open_close_move_pct": features.get("open_close_move_pct"),
                    "intraday_range_pct": features.get("intraday_range_pct"),
                    "vwap_distance_pct": features.get("vwap_distance_pct"),
                    "daily_return_pct": features.get("daily_return_pct"),
                    "volume": features.get("volume"),
                    "open_interest": features.get("open_interest"),
                    "volume_open_interest_ratio": features.get("volume_open_interest_ratio"),
                    "days_to_expiration": features.get("days_to_expiration"),
                    "moneyness_pct": features.get("moneyness_pct"),
                    "contract_type": features.get("contract_type"),
                    "quality_issue_count": len(quality_issues),
                    "detector_score": detection_result.score,
                },
                "graph_targets": graph_targets,
                "semantic_evidence": [
                    {
                        "type": "dsm_artifact_proposal",
                        "artifact_id": artifact_id,
                        "summary": explanation.summary,
                        "quality_issues": quality_issues,
                        "computed_features": {
                            "open_close_move_pct": features.get("open_close_move_pct"),
                            "intraday_range_pct": features.get("intraday_range_pct"),
                            "vwap_distance_pct": features.get("vwap_distance_pct"),
                            "daily_return_pct": features.get("daily_return_pct"),
                        },
                        "artifact": artifact,
                    }
                ],
                "evidence": [
                    {
                        "type": "normalized_event",
                        "ref": event_id,
                        "summary": explanation.summary,
                    }
                ],
                "recommendation": {
                    "action": "review_marketops_signal",
                    "summary": "Review the normalized Massive EOD record, DSM artifact proposal, and graph target candidates.",
                    "artifact_ids": [artifact_id],
                    "graph_target_count": len(graph_targets),
                },
            },
        )

    def _is_marketops_eod_event(self, event: Mapping[str, object]) -> bool:
        return (
            _string(event.get("app_id")) == self.app_id
            and _string(event.get("domain")) == self.domain
            and _string(event.get("source_adapter")) == self.source_adapter
            and _string(event.get("dataset")) == self.dataset
            and _string(event.get("use_case")) == self.use_case
        )

    def _features(self, payload: Mapping[str, object]) -> dict[str, object]:
        open_price = _number_field(payload, "open", "open_price")
        high = _number_field(payload, "high", "high_price")
        low = _number_field(payload, "low", "low_price")
        close = _number_field(payload, "close", "close_price")
        previous_close = _number_field(payload, "previous_close")
        vwap = _number_field(payload, "vwap")

        features: dict[str, object] = {
            "symbol": _string(payload.get("symbol") or payload.get("ticker")),
            "observation_date": _string(payload.get("observation_date")),
            "open": open_price,
            "high": high,
            "low": low,
            "close": close,
            "previous_close": previous_close,
            "vwap": vwap,
            "volume": _number_field(payload, "volume"),
        }
        if _positive(open_price) and close is not None:
            features["open_close_move_pct"] = _round_pct((close - open_price) / open_price)
        if _positive(open_price) and high is not None and low is not None:
            features["intraday_range_pct"] = _round_pct((high - low) / open_price)
        if _positive(vwap) and close is not None:
            features["vwap_distance_pct"] = _round_pct((close - vwap) / vwap)
        if _positive(previous_close) and close is not None:
            features["daily_return_pct"] = _round_pct((close - previous_close) / previous_close)
        return features

    def _quality_issues(self, features: Mapping[str, object]) -> list[str]:
        issues: list[str] = []
        for field in ("open", "high", "low", "close"):
            value = features.get(field)
            if value is None:
                issues.append(f"missing_{field}")
            elif not _positive(value):
                issues.append(f"non_positive_{field}")

        high = _number(features.get("high"))
        low = _number(features.get("low"))
        open_price = _number(features.get("open"))
        close = _number(features.get("close"))
        previous_close = _number(features.get("previous_close"))
        vwap = _number(features.get("vwap"))
        if high is not None and low is not None and high < low:
            issues.append("high_below_low")
        if high is not None and low is not None and open_price is not None:
            if open_price < low or open_price > high:
                issues.append("open_outside_daily_range")
        if high is not None and low is not None and close is not None:
            if close < low or close > high:
                issues.append("close_outside_daily_range")
        if previous_close is not None and not _positive(previous_close):
            issues.append("non_positive_previous_close")
        if vwap is not None and not _positive(vwap):
            issues.append("non_positive_vwap")
        return issues

    def _no_match(
        self, reason: str, metadata: Mapping[str, object]
    ) -> DetectionResult:
        return DetectionResult(
            detector_id=self.detector_id,
            detector_version=self.detector_version,
            matched=False,
            score=0.0,
            reason=reason,
            metadata=metadata,
        )



class MarketOpsDSMTaxonomyDetector(MarketOpsEODPriceDetector):
    detector_id = "marketops.dsm.taxonomy_v1"
    detector_version = "0.1.0"

    equity_dataset = "equity_eod_prices"
    option_dataset = "options_contracts_daily"
    datasets = {equity_dataset, option_dataset}

    def detect(
        self,
        normalized_events: Sequence[Mapping[str, object]],
        feature_context: FeatureContext,
    ) -> DetectionResult:
        if len(normalized_events) != 1:
            return self._no_match("marketops taxonomy detector expects one event", {})

        event = normalized_events[0]
        if not self._is_marketops_taxonomy_event(event):
            return self._no_match("event is outside MarketOps DSM taxonomy scope", {})

        payload = _object(event.get("normalized_payload"))
        dataset = _string(event.get("dataset"))
        symbol = _symbol(payload, event)
        features = self._features(payload)
        quality_issues = self._quality_issues(features) if dataset == self.equity_dataset else []
        metadata = {
            "event_id": _string(event.get("event_id")),
            "symbol": symbol,
            "features": features,
            "quality_issues": quality_issues,
        }

        if quality_issues:
            return DetectionResult(
                detector_id=self.detector_id,
                detector_version=self.detector_version,
                matched=True,
                score=0.99,
                reason=f"price quality exception for {symbol}",
                metadata={**metadata, "signal_type": "marketops.dsm.price_quality_exception"},
            )

        signal_type, score, reason = self._taxonomy_signal(dataset, payload, features, symbol)
        if not signal_type:
            return self._no_match("no MarketOps DSM taxonomy threshold crossed", metadata)

        confidence = min(0.95, max(0.62, score))
        return DetectionResult(
            detector_id=self.detector_id,
            detector_version=self.detector_version,
            matched=True,
            score=confidence,
            reason=reason,
            metadata={**metadata, "signal_type": signal_type},
        )

    def _is_marketops_taxonomy_event(self, event: Mapping[str, object]) -> bool:
        return (
            _string(event.get("app_id")) == self.app_id
            and _string(event.get("domain")) == self.domain
            and _string(event.get("source_adapter")) == self.source_adapter
            and _string(event.get("dataset")) in self.datasets
            and _string(event.get("use_case")) == self.use_case
        )

    def _features(self, payload: Mapping[str, object]) -> dict[str, object]:
        features = super()._features(payload)
        enriched = _object(payload.get("features"))
        for key, value in enriched.items():
            if key not in features or features[key] is None:
                features[key] = value
        for key in (
            "option_ticker",
            "underlying_symbol",
            "contract_type",
            "strike_price",
            "expiration_date",
            "days_to_expiration",
            "open_interest",
            "volume_open_interest_ratio",
            "moneyness_pct",
        ):
            if key not in features and key in payload:
                features[key] = payload[key]
        return features

    def _taxonomy_signal(
        self,
        dataset: str,
        payload: Mapping[str, object],
        features: Mapping[str, object],
        symbol: str,
    ) -> tuple[str, float, str]:
        if dataset == self.equity_dataset:
            divergence = self._equity_divergence(features)
            if divergence[0]:
                return divergence
            accumulation = self._equity_accumulation(features, symbol)
            if accumulation[0]:
                return accumulation

            scores = [
                _ratio_abs(features.get("open_close_move_pct"), self.move_threshold_pct),
                _ratio_abs(features.get("intraday_range_pct"), self.range_threshold_pct),
                _ratio_abs(features.get("daily_return_pct"), self.daily_return_threshold_pct),
            ]
            score = max(scores)
            if score >= 1:
                confidence = min(0.95, max(0.65, 0.55 + (score * 0.12)))
                return (
                    "marketops.dsm.volatility_expansion",
                    confidence,
                    f"volatility expansion threshold crossed for {symbol}",
                )
            return "", 0.0, ""

        if dataset == self.option_dataset:
            return self._option_taxonomy(payload, features, symbol)
        return "", 0.0, ""

    def _equity_accumulation(
        self, features: Mapping[str, object], symbol: str
    ) -> tuple[str, float, str]:
        daily_return = _number(features.get("daily_return_pct")) or 0.0
        open_close = _number(features.get("open_close_move_pct")) or 0.0
        vwap_distance = _number(features.get("vwap_distance_pct")) or 0.0
        volume = _number(features.get("volume")) or 0.0
        if daily_return >= 2.0 and open_close >= 1.0 and vwap_distance >= 0.5 and volume >= 1_000_000:
            score = min(0.92, 0.64 + min(daily_return / 20.0, 0.18) + min(vwap_distance / 20.0, 0.10))
            return "marketops.dsm.accumulation", score, f"accumulation pressure detected for {symbol}"
        return "", 0.0, ""

    def _equity_divergence(
        self, features: Mapping[str, object]
    ) -> tuple[str, float, str]:
        daily_return = _number(features.get("daily_return_pct")) or 0.0
        open_close = _number(features.get("open_close_move_pct")) or 0.0
        vwap_distance = _number(features.get("vwap_distance_pct")) or 0.0
        symbol = _string(features.get("symbol")) or "UNKNOWN"
        if abs(daily_return) >= 3.0 and ((daily_return > 0 and vwap_distance < -1.0) or (daily_return < 0 and vwap_distance > 1.0)):
            score = min(0.9, 0.66 + min(abs(daily_return) / 30.0, 0.16) + min(abs(vwap_distance) / 20.0, 0.08))
            return "marketops.dsm.divergence", score, f"price/VWAP divergence detected for {symbol}"
        if daily_return * open_close < 0 and abs(daily_return) >= 2.5 and abs(open_close) >= 1.5:
            score = min(0.88, 0.64 + min(abs(daily_return - open_close) / 30.0, 0.18))
            return "marketops.dsm.divergence", score, f"session/return divergence detected for {symbol}"
        return "", 0.0, ""

    def _option_taxonomy(
        self,
        payload: Mapping[str, object],
        features: Mapping[str, object],
        symbol: str,
    ) -> tuple[str, float, str]:
        contract_type = _string(features.get("contract_type") or payload.get("contract_type")).lower()
        volume = _number(features.get("volume")) or 0.0
        open_interest = _number(features.get("open_interest")) or 0.0
        volume_oi = _number(features.get("volume_open_interest_ratio")) or 0.0
        days = _number(features.get("days_to_expiration"))
        moneyness = abs(_number(features.get("moneyness_pct")) or 999.0)

        if days is not None and days <= 7 and moneyness <= 1.0 and open_interest >= 1000:
            return "marketops.dsm.pinning_risk", 0.84, f"pinning risk detected for {symbol}"
        if contract_type == "call" and volume >= 1000 and volume_oi >= 0.3 and (days is None or days <= 45):
            return "marketops.dsm.speculative_call_pressure", min(0.9, 0.66 + min(volume_oi / 3.0, 0.20)), f"speculative call pressure detected for {symbol}"
        if contract_type == "put" and volume >= 1000 and volume_oi >= 0.3 and (days is None or days <= 45):
            return "marketops.dsm.speculative_put_pressure", min(0.9, 0.66 + min(volume_oi / 3.0, 0.20)), f"speculative put pressure detected for {symbol}"
        if open_interest >= 3000 and volume_oi >= 0.25:
            return "marketops.dsm.hedging_pressure", min(0.9, 0.68 + min(volume_oi / 4.0, 0.18)), f"hedging pressure detected for {symbol}"
        return "", 0.0, ""



def _stable_artifact_id(signal_type: str, event_id: str, symbol: str) -> str:
    key = "|".join(
        part.strip()
        for part in ("marketops.dsm.artifact_v1", signal_type, event_id, symbol)
        if part.strip()
    )
    digest = hashlib.sha256(key.encode("utf-8")).hexdigest()[:20]
    return f"artifact_marketops_dsm_v1_{digest}"


def _dsm_artifact(
    *,
    artifact_id: str,
    signal_type: str,
    event_id: str,
    symbol: str,
    features: Mapping[str, object],
    quality_issues: Sequence[str],
    confidence: float,
    severity: str,
    summary: str,
) -> Mapping[str, object]:
    return {
        "artifact_id": artifact_id,
        "artifact_type": "marketops.dsm.signal_artifact.v1",
        "signal_type": signal_type,
        "source_event_id": event_id,
        "subject": {"type": "ticker", "id": f"ticker:{symbol}", "symbol": symbol},
        "severity": severity,
        "confidence": confidence,
        "summary": summary,
        "features": {
            "open_close_move_pct": features.get("open_close_move_pct"),
            "intraday_range_pct": features.get("intraday_range_pct"),
            "vwap_distance_pct": features.get("vwap_distance_pct"),
            "daily_return_pct": features.get("daily_return_pct"),
            "volume": features.get("volume"),
            "open_interest": features.get("open_interest"),
            "volume_open_interest_ratio": features.get("volume_open_interest_ratio"),
            "days_to_expiration": features.get("days_to_expiration"),
            "moneyness_pct": features.get("moneyness_pct"),
            "contract_type": features.get("contract_type"),
        },
        "quality_issues": list(quality_issues),
    }


def _graph_proposals(
    *,
    artifact_id: str,
    signal_type: str,
    symbol: str,
    confidence: float,
    severity: str,
) -> list[Mapping[str, object]]:
    ticker_id = f"ticker:{symbol}"
    signal_node_id = f"signal_type:{signal_type}"
    artifact_node_id = f"artifact:{artifact_id}"
    return [
        {
            "type": "node_candidate",
            "node_id": ticker_id,
            "labels": ["MarketAsset", "Ticker"],
            "properties": {"symbol": symbol, "app_id": "marketops"},
            "confidence": 1.0,
        },
        {
            "type": "node_candidate",
            "node_id": signal_node_id,
            "labels": ["DSMSignalType"],
            "properties": {"signal_type": signal_type, "domain": "market_data"},
            "confidence": 1.0,
        },
        {
            "type": "node_candidate",
            "node_id": artifact_node_id,
            "labels": ["DSMArtifact"],
            "properties": {"artifact_id": artifact_id, "severity": severity},
            "confidence": confidence,
        },
        {
            "type": "relationship_candidate",
            "from": ticker_id,
            "relationship": "EXHIBITS_SIGNAL",
            "to": signal_node_id,
            "properties": {"severity": severity},
            "confidence": confidence,
        },
        {
            "type": "relationship_candidate",
            "from": signal_node_id,
            "relationship": "SUPPORTED_BY_ARTIFACT",
            "to": artifact_node_id,
            "properties": {"artifact_id": artifact_id},
            "confidence": confidence,
        },
    ]

def _symbol(payload: Mapping[str, object], event: Mapping[str, object]) -> str:
    symbol = _string(payload.get("symbol") or payload.get("ticker"))
    if symbol:
        return symbol.upper()
    for entity in _object_sequence(event.get("entities")):
        if _string(entity.get("type")) == "ticker":
            value = _string(entity.get("external_id") or entity.get("id"))
            if value:
                return value.removeprefix("ticker:").upper()
    return "UNKNOWN"


def _stable_signal_id(*parts: str) -> str:
    key = "|".join(part.strip() for part in parts if part.strip())
    digest = hashlib.sha256(key.encode("utf-8")).hexdigest()[:20]
    prefix = "sig_marketops_dsm_taxonomy_v1" if parts and parts[0] == "marketops.dsm.taxonomy_v1" else "sig_marketops_dsm_eod_price_v1"
    return f"{prefix}_{digest}"


def _severity(
    signal_type: str, features: Mapping[str, object], quality_issues: Sequence[str]
) -> str:
    if signal_type == "marketops.dsm.price_quality_exception":
        return "high" if any(issue.startswith("missing_") for issue in quality_issues) else "medium"
    if signal_type in {"marketops.dsm.hedging_pressure", "marketops.dsm.pinning_risk"}:
        return "high"
    if signal_type in {"marketops.dsm.accumulation", "marketops.dsm.speculative_call_pressure", "marketops.dsm.speculative_put_pressure", "marketops.dsm.divergence"}:
        return "medium"

    move = abs(_number(features.get("open_close_move_pct")) or 0.0)
    range_pct = abs(_number(features.get("intraday_range_pct")) or 0.0)
    daily = abs(_number(features.get("daily_return_pct")) or 0.0)
    if max(move, range_pct, daily) >= 12:
        return "critical"
    if move >= 6 or range_pct >= 8 or daily >= 7:
        return "high"
    return "medium"


def _ratio_abs(value: object, threshold: float) -> float:
    number = _number(value)
    if number is None:
        return 0.0
    return abs(number) / threshold


def _number_field(payload: Mapping[str, object], *names: str) -> float | None:
    for name in names:
        value = _number(payload.get(name))
        if value is not None:
            return value
    return None


def _number(value: object) -> float | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        return float(value)
    if isinstance(value, str):
        try:
            return float(value.strip())
        except ValueError:
            return None
    return None


def _positive(value: object) -> bool:
    number = _number(value)
    return number is not None and number > 0


def _round_pct(value: float) -> float:
    return round(value * 100, 4)


def _string(value: object) -> str:
    if isinstance(value, str):
        return value.strip()
    return ""


def _string_list(value: object) -> list[str]:
    if not isinstance(value, Sequence) or isinstance(value, (str, bytes, bytearray)):
        return []
    return [item.strip() for item in value if isinstance(item, str) and item.strip()]


def _object(value: object) -> Mapping[str, object]:
    if isinstance(value, Mapping):
        return dict(value)
    return {}


def _object_sequence(value: object) -> list[Mapping[str, object]]:
    if not isinstance(value, Sequence) or isinstance(value, (str, bytes, bytearray)):
        return []
    return [dict(item) for item in value if isinstance(item, Mapping)]
