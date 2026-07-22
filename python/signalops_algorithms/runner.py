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


def main() -> int:
    try:
        request = json.load(sys.stdin)
        if request.get("schema_version") != SCHEMA:
            raise ValueError("unsupported execution schema")
        points = sorted(request.get("points", []), key=lambda item: item["observation_time"])
        if not points:
            raise ValueError("points are required")
        handlers = {
            "signalops.algorithms.river_anomaly_v1": river,
            "signalops.algorithms.ruptures_change_point_v1": ruptures,
            "signalops.algorithms.statsmodels_forecast_v1": statsmodels,
        }
        handler = handlers.get(request.get("algorithm_id"))
        if handler is None:
            raise ValueError("unsupported algorithm_id")
        print(json.dumps({"schema_version": SCHEMA, "results": handler(request, points)}, separators=(",", ":")))
        return 0
    except Exception as exc:
        print(json.dumps({"error": str(exc)}), file=sys.stderr)
        return 1

if __name__ == "__main__":
    raise SystemExit(main())
