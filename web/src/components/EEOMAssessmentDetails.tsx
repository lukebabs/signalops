const componentLabel = (name: string) =>
  ({
    vc: "VC",
    dosm: "DOSM",
    risk_reward: "Risk/Reward",
    event_materiality: "Event materiality",
  })[name] ?? name.replace(/_/g, " ");
export function EEOMAssessmentGuide() {
  return (
    <details className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-600">
      <summary className="cursor-pointer font-medium text-gray-800">
        Assessment outcomes
      </summary>
      <div className="mt-2 grid gap-2 md:grid-cols-2">
        <p>
          <strong>Priority A</strong> — setup quality ≥8.0 with a non-mixed
          posture; highest research priority.
        </p>
        <p>
          <strong>Priority B</strong> — setup quality ≥6.5; strong enough for
          analyst review.
        </p>
        <p>
          <strong>Distressed inflection</strong> — setup quality 5.5–6.49;
          strategic/fundamental context merits deeper research.
        </p>
        <p>
          <strong>Await validation</strong> — mixed directional evidence within
          five calendar days of earnings.
        </p>
        <p>
          <strong>Avoid</strong> — setup quality below 3.5; low-quality
          available evidence, not a trade instruction.
        </p>
        <p>
          <strong>Informational only</strong> — setup quality 3.5–5.49 without a
          higher-priority condition, or core evidence is withheld. It remains
          visible as context, not a review recommendation.
        </p>
      </div>
    </details>
  );
}
export function EEOMSignalDetails({ row }: { row: any }) {
  const result = row.trace?.result ?? {};
  const components = result.components ?? {};
  return (
    <details className="group">
      <summary className="cursor-pointer capitalize text-brand-700 hover:underline">
        {String(row.classification).replace(/_/g, " ")}
      </summary>
      <div className="mt-2 w-80 space-y-2 rounded border border-brand-100 bg-brand-50 p-2 text-[11px] text-gray-700 shadow-sm">
        <div>
          <strong>Observed signals</strong> · {row.posture} posture ·{" "}
          {row.evidence_quality} evidence
        </div>
        <div className="grid grid-cols-2 gap-x-3 gap-y-1">
          {Object.entries(components).map(([name, value]: any) => (
            <span key={name}>
              <span className="capitalize text-gray-500">
                {componentLabel(name)}:{" "}
              </span>
              <strong>
                {value.available
                  ? `${Number(value.score).toFixed(0)}/100 · ${Number(value.effective_weight || 0).toFixed(1)}%`
                  : "withheld"}
              </strong>
            </span>
          ))}
        </div>
        {result.withheld_inputs?.length ? (
          <div>
            <span className="font-medium">Withheld:</span>{" "}
            {result.withheld_inputs.join(", ").replace(/_/g, " ")}
          </div>
        ) : null}
        <p className="text-gray-500">
          Component values are normalized evidence inputs; EEOM setup quality is
          the weighted 0–10 analyst score.
        </p>
      </div>
    </details>
  );
}
export function EEOMSelectedSignals({ row }: { row: any }) {
  const result = row.trace?.result ?? {};
  const components = result.components ?? {};
  return (
    <div className="space-y-3 rounded border border-brand-200 bg-brand-50 p-3 text-xs text-gray-700">
      <div>
        <strong className="text-brand-900">
          {row.ticker} observed evidence
        </strong>
        <span className="ml-2 text-gray-500">
          {row.earnings_date} ·{" "}
          {row.event?.days_to_event != null ? ` days to earnings · ` : ""}
          {String(row.posture).replace(/_/g, " ")} posture ·{" "}
          {String(row.evidence_quality).replace(/_/g, " ")} evidence
        </span>
      </div>
      {row.event ? (
        <div className="rounded border border-brand-100 bg-white p-2 text-[11px]">
          <strong>Next earnings event</strong> · {row.event.event_date} ·{" "}
          {row.event.status?.replace(/_/g, " ")} · {row.event.source}. Timing is
          not supplied by FMP; this is awareness context, not a score modifier.
        </div>
      ) : null}
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {Object.entries(components).map(([name, value]: any) => (
          <div
            key={name}
            className="rounded border border-gray-200 bg-white p-2"
          >
            <div className="font-medium capitalize text-gray-700">
              {componentLabel(name)}
            </div>
            <div className="mt-1 text-gray-600">
              {value.available
                ? `${Number(value.score).toFixed(0)}/100 · ${Number(value.effective_weight || 0).toFixed(1)}% effective weight`
                : "Withheld"}
            </div>
            {value.reason ? (
              <div className="mt-1 text-[11px] text-gray-500">
                {value.reason}
              </div>
            ) : null}
          </div>
        ))}
      </div>
      {result.withheld_inputs?.length ? (
        <p className="text-gray-500">
          <strong>Withheld inputs:</strong>{" "}
          {result.withheld_inputs.join(", ").replace(/_/g, " ")}
        </p>
      ) : null}
      <p className="text-gray-500">
        Component values are persisted, normalized inputs. Setup quality is the
        reweighted 0–10 analyst score.
      </p>
    </div>
  );
}
