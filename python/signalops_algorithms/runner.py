from __future__ import annotations

import json
import sys

SCHEMA = "signalops.platform_algorithm_execution.v1"


def severity(score: float, threshold: float) -> str:
    if score >= threshold * 1.5:
        return "high"
    if score >= threshold:
        return "medium"
    if score >= threshold * 0.5:
        return "low"
    return "info"


def result(request: dict, point: dict, result_type: str, score: float, extra: dict) -> dict:
    threshold = float(request.get("config", {}).get("score_threshold", 3.0))
    score = round(max(0.0, float(score)), 6)
    payload = {
        "algorithm_id": request["algorithm_id"], "runtime": "python_platform",
        "schema_version": SCHEMA, "dataset": request.get("dataset", ""),
        "feature": request.get("feature", ""), "symbol": point.get("symbol", ""),
        "value": point["value"], "observation_time": point["observation_time"],
        "window_start": request.get("window_start", ""), "window_end": request.get("window_end", ""),
        **extra,
    }
    return {"source_event_id": point["event_id"], "result_type": result_type,
            "score": score, "confidence": round(min(1.0, score / (threshold * 1.5)), 6),
            "severity": severity(score, threshold), "payload": payload}


def river(request: dict, points: list[dict]) -> list[dict]:
    from river import anomaly
    model = anomaly.HalfSpaceTrees(n_trees=25, height=8, window_size=256)
    out = []
    prior = None
    for index, point in enumerate(points):
        score = float(model.score_one({"value": point["value"]})) if index >= 2 else 0.0
        direction = "up" if prior is not None and point["value"] > prior else "down" if prior is not None and point["value"] < prior else ""
        model.learn_one({"value": point["value"]})
        out.append(result(request, point, "online_anomaly_score", score,
                          {"library": "river", "training_samples_before_event": index, "direction": direction}))
        prior = point["value"]
    return out


def ruptures(request: dict, points: list[dict]) -> list[dict]:
    import numpy as np
    import ruptures as rpt
    if len(points) < 6:
        return []
    values = np.asarray([point["value"] for point in points], dtype=float)
    breaks = set(rpt.Binseg(model="l2", min_size=3).fit(values).predict(n_bkps=max(1, min(3, len(values) // 6)))[:-1])
    stddev = float(np.std(values)) or 1.0
    out = []
    for index, point in enumerate(points):
        if index + 1 in breaks:
            prior = values[index - 1] if index else values[index]
            out.append(result(request, point, "change_point_score", abs(values[index] - prior) / stddev,
                              {"library": "ruptures", "segment_boundary_index": index + 1, "direction": "up" if values[index] > prior else "down" if values[index] < prior else ""}))
    return out


def statsmodels(request: dict, points: list[dict]) -> list[dict]:
    import numpy as np
    from statsmodels.tsa.ar_model import AutoReg
    if len(points) < 6:
        return []
    values = np.asarray([point["value"] for point in points], dtype=float)
    fit = AutoReg(values, lags=1, trend="c").fit()
    residuals = fit.resid
    stddev = float(np.std(residuals)) or 1.0
    return [result(request, point, "forecast_residual", abs(float(residuals[index - 1])) / stddev,
                   {"library": "statsmodels", "predicted_value": round(float(fit.fittedvalues[index - 1]), 6), "residual": round(float(residuals[index - 1]), 6), "direction": "up" if residuals[index - 1] > 0 else "down" if residuals[index - 1] < 0 else ""})
            for index, point in enumerate(points) if index > 0]


def risk_reward(request: dict, points: list[dict]) -> list[dict]:
    """Research-only technical posture; speculative put/call evidence never sets direction."""
    out = []
    for point in points:
        values = point.get("features", {})
        required = ("range_position_252d", "rsi_14", "return_5d", "volume_ratio_10d", "distance_sma_50_pct", "distance_sma_200_pct", "sma_50_slope_20d_pct", "atr_14_pct")
        usable = {name: values.get(name) for name in required if isinstance(values.get(name), (int, float))}
        factors, signed = [], 0.0
        def add(name: str, direction: str, weight: float, value: float, detail: str) -> None:
            nonlocal signed
            signed += weight if direction == "bullish" else -weight if direction == "bearish" else 0.0
            factors.append({"key": name, "direction": direction, "weight": weight, "value": round(value, 6), "detail": detail})
        if "range_position_252d" in usable:
            v = usable["range_position_252d"]; add("range_position_252d", "bearish" if v >= 90 else "bullish" if v <= 10 else "neutral", 25, v, "252-session range position")
        if "rsi_14" in usable:
            v = usable["rsi_14"]; add("rsi_14", "bearish" if v >= 60 else "bullish" if v <= 40 else "neutral", 20, v, "14-session RSI")
        if "return_5d" in usable and "volume_ratio_10d" in usable:
            ret, volume = usable["return_5d"], usable["volume_ratio_10d"]
            direction = "bearish" if ret > 0 and volume < .8 else "bullish" if ret < 0 and volume > 1.2 else "neutral"
            add("volume_price_divergence", direction, 15, volume, "5-session price move versus 10-session volume ratio")
        if all(name in usable for name in ("distance_sma_50_pct", "distance_sma_200_pct", "sma_50_slope_20d_pct")):
            d50, d200, slope = usable["distance_sma_50_pct"], usable["distance_sma_200_pct"], usable["sma_50_slope_20d_pct"]
            direction = "bullish" if d50 > 0 and d200 > 0 and slope > 0 else "bearish" if d50 < 0 and d200 < 0 and slope < 0 else "neutral"
            add("trend_regime", direction, 25, slope, "50/200-session price structure and 50-session slope")
        if "return_5d" in usable:
            v = usable["return_5d"]; add("price_trend", "bearish" if v >= 5 else "bullish" if v <= -5 else "neutral", 15, v, "5-session price trend")
        atr = usable.get("atr_14_pct")
        risk_level = "high" if atr is not None and atr >= 3 else "medium" if atr is not None and atr >= 1.5 else "low" if atr is not None else "unavailable"
        technical_score = max(-100.0, min(100.0, signed))
        direction = "bullish" if technical_score >= 25 else "bearish" if technical_score <= -25 else "neutral"
        pcr, deviation = values.get("put_call_volume_ratio"), values.get("put_call_volume_ratio_10d_deviation_pct")
        speculative = "unavailable"
        if isinstance(pcr, (int, float)) and isinstance(deviation, (int, float)):
            speculative = "bearish" if pcr > 1 and deviation >= 10 else "bullish" if pcr < 1 and deviation <= -10 else "neutral"
        agreement = speculative if speculative in ("bullish", "bearish") and speculative != direction and direction != "neutral" else "aligned" if speculative == direction and direction != "neutral" else "neutral"
        confidence = min(1.0, len(usable) / len(required))
        if risk_level == "unavailable": confidence *= .85
        payload = {"algorithm_id": request["algorithm_id"], "runtime": "python_platform", "schema_version": "signalops.platform_algorithm_execution.v2", "symbol": point.get("symbol", ""), "observation_time": point["observation_time"], "technical_direction": direction, "technical_score": round(technical_score, 6), "risk_level": risk_level, "confidence_basis": {"usable_technical_inputs": len(usable), "required_technical_inputs": len(required)}, "technical_factors": factors, "speculative_corroboration": {"direction": speculative, "agreement": agreement, "put_call_volume_ratio": pcr, "deviation_pct": deviation}, "feature_value_ids": point.get("feature_value_ids", []), "evidence_refs": point.get("evidence_refs", []), "research_only": True}
        score = abs(technical_score) / 100
        severity_value = "high" if abs(technical_score) >= 65 else "medium" if abs(technical_score) >= 40 else "low" if abs(technical_score) >= 25 else "info"
        out.append({"source_event_id": point["event_id"], "result_type": "risk_reward_temporal", "score": score, "confidence": round(confidence, 6), "severity": severity_value, "payload": payload})
    return out


def main() -> int:
    try:
        request = json.load(sys.stdin)
        if request.get("schema_version") not in (SCHEMA, "signalops.platform_algorithm_execution.v2"):
            raise ValueError("unsupported execution schema")
        points = sorted(request.get("points", []), key=lambda item: item["observation_time"])
        if not points:
            raise ValueError("points are required")
        handlers = {
            "signalops.algorithms.river_anomaly_v1": river,
            "signalops.algorithms.ruptures_change_point_v1": ruptures,
            "signalops.algorithms.statsmodels_forecast_v1": statsmodels,
            "signalops.algorithms.risk_reward_temporal_v1": risk_reward,
        }
        handler = handlers.get(request.get("algorithm_id"))
        if handler is None:
            raise ValueError("unsupported algorithm_id")
        print(json.dumps({"schema_version": request.get("schema_version"), "results": handler(request, points)}, separators=(",", ":")))
        return 0
    except Exception as exc:
        print(json.dumps({"error": str(exc)}), file=sys.stderr)
        return 1

if __name__ == "__main__":
    raise SystemExit(main())
