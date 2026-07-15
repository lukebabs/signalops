import { useState } from 'react';
import { Cpu } from 'lucide-react';
import {
  useAlgorithmDefinitions,
  useAlgorithmExecutionRequests,
  useAlgorithmExecutionSummary,
} from '../api/queries';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { CopyButton } from '../components/CopyButton';
import { formatUtc } from '../lib/format';
import {
  summarizeAlgorithmDefinition,
  summarizeAlgorithmExecutionRequest,
  summarizeAlgorithmResult,
  summarizeAlgorithmExecutionSummary,
  algorithmDefinitionStatusStyle,
  algorithmExecutionStatusStyle,
  algorithmSeverityStyle,
} from '../lib/algorithms';
import { useTenant } from '../auth/session';
import type { AlgorithmDefinition, AlgorithmResult } from '../types';

// G109 algorithm execution visibility (read-only). Mirrors the dense, restrained
// table layout of the other MarketOps workbenches. No Run/Tune/Promote/Deploy/
// Convert controls — this surface only renders backend algorithm state.

const DEFINITION_STATUSES = ['', 'draft', 'active', 'disabled', 'deprecated'];
const RUNTIME_TYPES = ['', 'python_plugin', 'container_plugin', 'http_plugin'];
const EXECUTION_STATUSES = ['', 'queued', 'running', 'succeeded', 'failed', 'canceled'];
const LIMITS = [25, 50, 100, 200];

function DefinitionStatusBadge({ status }: { status: string }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${algorithmDefinitionStatusStyle(status)}`}
    >
      {status || '—'}
    </span>
  );
}

function ExecutionStatusBadge({ status }: { status: string }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${algorithmExecutionStatusStyle(status)}`}
    >
      {status || '—'}
    </span>
  );
}

function SeverityBadge({ severity }: { severity: string }) {
  return (
    <span className={`text-xs font-medium ${algorithmSeverityStyle(severity)}`}>{severity || '—'}</span>
  );
}

// Compact read-only id reference list with a copy-all button. Never renders
// anything when there are no ids so the detail panel stays uncluttered.
function IdRefList({ label, ids }: { label: string; ids: string[] }) {
  if (!ids.length) return null;
  return (
    <div>
      <div className="mb-0.5 flex items-center gap-2">
        <span className="text-xs font-medium text-gray-600">{label}</span>
        <span className="rounded border border-gray-200 px-1.5 text-[11px] text-gray-600">{ids.length}</span>
        <CopyButton value={ids.join(', ')} />
      </div>
      <code className="break-all text-xs text-gray-700">{ids.join(', ')}</code>
    </div>
  );
}

export function AlgorithmsRoute() {
  const TENANT_ID = useTenant();

  // Definition list filters.
  const [defStatus, setDefStatus] = useState('');
  const [defType, setDefType] = useState('');
  const [defRuntime, setDefRuntime] = useState('');
  const [limit, setLimit] = useState(50);

  // Drilldown selection state.
  const [selectedAlgorithmId, setSelectedAlgorithmId] = useState<string | null>(null);
  const [execStatus, setExecStatus] = useState('');
  const [correlationId, setCorrelationId] = useState('');
  const [selectedExecutionRequestId, setSelectedExecutionRequestId] = useState<string | null>(null);
  const [selectedResultId, setSelectedResultId] = useState<string | null>(null);

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';

  const definitions = useAlgorithmDefinitions({
    tenant_id: TENANT_ID,
    algorithm_type: defType || undefined,
    runtime_type: defRuntime || undefined,
    status: defStatus || undefined,
    limit,
  });
  const defData: AlgorithmDefinition[] = definitions.data?.algorithm_definitions ?? [];

  const executionRequests = useAlgorithmExecutionRequests({
    tenant_id: TENANT_ID,
    algorithm_id: selectedAlgorithmId || undefined,
    status: execStatus || undefined,
    correlation_id: correlationId || undefined,
    limit,
  });
  const execData = executionRequests.data?.algorithm_execution_requests ?? [];

  const summaryQuery = useAlgorithmExecutionSummary(selectedExecutionRequestId, TENANT_ID, 10);
  const summaryRaw = summaryQuery.data?.algorithm_execution_summary ?? null;
  const summaryView = summaryRaw ? summarizeAlgorithmExecutionSummary(summaryRaw) : null;
  const selectedResult: AlgorithmResult | null =
    summaryRaw?.top_results.find((r) => r.algorithm_result_id === selectedResultId) ?? null;

  function selectAlgorithm(id: string) {
    setSelectedAlgorithmId(id);
    setSelectedExecutionRequestId(null);
    setSelectedResultId(null);
  }
  function selectExecutionRequest(id: string) {
    setSelectedExecutionRequestId(id);
    setSelectedResultId(null);
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Cpu size={18} className="text-brand-700" />
        <div>
          <h1 className="text-lg font-semibold">Algorithms</h1>
          <p className="text-xs text-gray-500">
            Read-only visibility for algorithm definitions, execution requests, and result evidence · tenant {TENANT_ID}
          </p>
        </div>
      </div>

      {/* Algorithm definitions */}
      <div className="rounded border border-gray-200 bg-white p-3">
        <div className="mb-2 text-xs font-semibold text-gray-700">Algorithm Definitions</div>
        <div className="mb-2 flex flex-wrap items-center gap-2">
          <select
            value={defStatus}
            onChange={(e) => setDefStatus(e.target.value)}
            className={inputCls}
            aria-label="Filter definitions by status"
          >
            {DEFINITION_STATUSES.map((s) => (
              <option key={s} value={s}>{s || 'any status'}</option>
            ))}
          </select>
          <input
            value={defType}
            onChange={(e) => setDefType(e.target.value)}
            className={inputCls}
            aria-label="Filter by algorithm type"
            placeholder="algorithm type"
          />
          <select
            value={defRuntime}
            onChange={(e) => setDefRuntime(e.target.value)}
            className={inputCls}
            aria-label="Filter by runtime type"
          >
            {RUNTIME_TYPES.map((r) => (
              <option key={r} value={r}>{r || 'any runtime'}</option>
            ))}
          </select>
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className={inputCls}
            aria-label="Page limit"
          >
            {LIMITS.map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </div>
        {definitions.isLoading ? (
          <LoadingState label="Loading algorithm definitions..." />
        ) : definitions.isError ? (
          <ErrorState error={definitions.error} />
        ) : defData.length ? (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="whitespace-nowrap px-3 py-2">Algorithm</th>
                  <th className="whitespace-nowrap px-3 py-2">Type</th>
                  <th className="whitespace-nowrap px-3 py-2">Runtime</th>
                  <th className="whitespace-nowrap px-3 py-2">Version</th>
                  <th className="whitespace-nowrap px-3 py-2">Status</th>
                  <th className="whitespace-nowrap px-3 py-2">Input features</th>
                  <th className="whitespace-nowrap px-3 py-2">Input event types</th>
                  <th className="whitespace-nowrap px-3 py-2">Updated</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {defData.map((d) => {
                  const s = summarizeAlgorithmDefinition(d);
                  return (
                    <tr
                      key={d.algorithm_id}
                      onClick={() => selectAlgorithm(d.algorithm_id)}
                      className={`cursor-pointer align-top hover:bg-gray-50 ${selectedAlgorithmId === d.algorithm_id ? 'bg-brand-50' : ''}`}
                    >
                      <td className="px-3 py-2">
                        <div className="max-w-[16rem] truncate text-xs font-medium text-gray-800" title={s.name || s.algorithmId}>
                          {s.name || s.algorithmId || '—'}
                        </div>
                        <code className="break-all text-[11px] text-gray-500">{s.algorithmId}</code>
                      </td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{s.algorithmType || '—'}</td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{s.runtimeType || '—'}</td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{s.version || '—'}</td>
                      <td className="px-3 py-2"><DefinitionStatusBadge status={s.status} /></td>
                      <td className="px-3 py-2">
                        <code className="break-all text-[11px] text-gray-700" title={s.inputFeatures.join(', ')}>
                          {s.inputFeatures.length ? s.inputFeatures.join(', ') : '—'}
                        </code>
                      </td>
                      <td className="px-3 py-2">
                        <code className="break-all text-[11px] text-gray-700" title={s.inputEventTypes.join(', ')}>
                          {s.inputEventTypes.length ? s.inputEventTypes.join(', ') : '—'}
                        </code>
                      </td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{formatUtc(s.updatedAt)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState message="No algorithm definitions found." />
        )}
      </div>

      {/* Execution requests for the selected algorithm */}
      {selectedAlgorithmId && (
        <div className="rounded border border-gray-200 bg-white p-3">
          <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
            <div className="text-xs font-semibold text-gray-700">
              Execution Requests · <code className="text-[11px] text-gray-500">{selectedAlgorithmId}</code>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <select
                value={execStatus}
                onChange={(e) => setExecStatus(e.target.value)}
                className={inputCls}
                aria-label="Filter execution requests by status"
              >
                {EXECUTION_STATUSES.map((s) => (
                  <option key={s} value={s}>{s || 'any status'}</option>
                ))}
              </select>
              <input
                value={correlationId}
                onChange={(e) => setCorrelationId(e.target.value)}
                className={inputCls}
                aria-label="Filter by correlation id"
                placeholder="correlation id"
              />
            </div>
          </div>
          {executionRequests.isLoading ? (
            <LoadingState label="Loading execution requests..." />
          ) : executionRequests.isError ? (
            <ErrorState error={executionRequests.error} />
          ) : execData.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="whitespace-nowrap px-3 py-2">Execution request</th>
                    <th className="whitespace-nowrap px-3 py-2">Status</th>
                    <th className="whitespace-nowrap px-3 py-2">Version</th>
                    <th className="whitespace-nowrap px-3 py-2">Requested by</th>
                    <th className="whitespace-nowrap px-3 py-2">Correlation</th>
                    <th className="whitespace-nowrap px-3 py-2">Window ref</th>
                    <th className="whitespace-nowrap px-3 py-2">Feature refs</th>
                    <th className="whitespace-nowrap px-3 py-2">Updated</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {execData.map((r) => {
                    const s = summarizeAlgorithmExecutionRequest(r);
                    return (
                      <tr
                        key={r.execution_request_id}
                        onClick={() => selectExecutionRequest(r.execution_request_id)}
                        className={`cursor-pointer align-top hover:bg-gray-50 ${selectedExecutionRequestId === r.execution_request_id ? 'bg-brand-50' : ''}`}
                      >
                        <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{s.executionRequestId || '—'}</code></td>
                        <td className="px-3 py-2"><ExecutionStatusBadge status={s.status} /></td>
                        <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{s.algorithmVersion || '—'}</td>
                        <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{s.requestedBy || '—'}</td>
                        <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{s.correlationId || '—'}</code></td>
                        <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{s.windowRef || '—'}</code></td>
                        <td className="px-3 py-2">
                          <code className="break-all text-[11px] text-gray-700" title={s.featureRefs.join(', ')}>
                            {s.featureRefs.length ? s.featureRefs.join(', ') : '—'}
                          </code>
                        </td>
                        <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{formatUtc(s.updatedAt)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No execution requests found." />
          )}
        </div>
      )}

      {/* Execution summary + top results */}
      {selectedExecutionRequestId && (
        <div className="rounded border border-gray-200 bg-white p-3">
          <div className="mb-2 text-xs font-semibold text-gray-700">
            Execution Summary · <code className="text-[11px] text-gray-500">{selectedExecutionRequestId}</code>
          </div>
          {summaryQuery.isLoading ? (
            <LoadingState label="Loading execution summary..." />
          ) : summaryQuery.isError ? (
            <ErrorState error={summaryQuery.error} />
          ) : summaryView ? (
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
                <MetricTile label="Results" value={summaryView.resultCount} />
                <MetricTile label="Max score" value={summaryView.maxScore.toFixed(3)} />
                <MetricTile label="Max confidence" value={summaryView.maxConfidence.toFixed(2)} />
                <MetricTile label="Status" value={summaryView.executionRequest.status || '—'} />
              </div>

              {summaryView.severityCounts.length > 0 && (
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="text-xs text-gray-500">Severity counts:</span>
                  {summaryView.severityCounts.map((c) => (
                    <span
                      key={c.severity}
                      className={`inline-flex items-center gap-1 rounded border border-gray-200 bg-gray-50 px-2 py-0.5 text-xs ${algorithmSeverityStyle(c.severity)}`}
                    >
                      {c.severity} <span className="font-semibold">{c.count}</span>
                    </span>
                  ))}
                </div>
              )}

              {summaryView.executionRequest.errorMessage && (
                <p className="rounded border border-red-200 bg-red-50 px-2 py-1 text-xs text-red-700">
                  {summaryView.executionRequest.errorMessage}
                </p>
              )}

              {/* Config + result JSON, collapsed by default (JsonViewer). */}
              <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
                {summaryRaw && <JsonViewer label="Config JSON" value={summaryRaw.execution_request.config} />}
                {summaryRaw && <JsonViewer label="Result JSON" value={summaryRaw.execution_request.result} />}
              </div>

              {/* Top results table */}
              <div className="text-xs font-semibold text-gray-700">Top results</div>
              {summaryView.topResults.length ? (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                      <tr>
                        <th className="whitespace-nowrap px-3 py-2">Result</th>
                        <th className="whitespace-nowrap px-3 py-2">Type</th>
                        <th className="whitespace-nowrap px-3 py-2">Score</th>
                        <th className="whitespace-nowrap px-3 py-2">Conf.</th>
                        <th className="whitespace-nowrap px-3 py-2">Severity</th>
                        <th className="whitespace-nowrap px-3 py-2">Created</th>
                        <th className="whitespace-nowrap px-3 py-2">Source events</th>
                        <th className="whitespace-nowrap px-3 py-2">Feature values</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {summaryView.topResults.map((r) => (
                        <tr
                          key={r.algorithmResultId}
                          onClick={() => setSelectedResultId(r.algorithmResultId)}
                          className={`cursor-pointer align-top hover:bg-gray-50 ${selectedResultId === r.algorithmResultId ? 'bg-brand-50' : ''}`}
                        >
                          <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{r.algorithmResultId || '—'}</code></td>
                          <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{r.resultType || '—'}</td>
                          <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800">{r.score.toFixed(3)}</td>
                          <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800">{r.confidence.toFixed(2)}</td>
                          <td className="px-3 py-2"><SeverityBadge severity={r.severity} /></td>
                          <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{formatUtc(r.createdAt)}</td>
                          <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{r.sourceEventIds.length}</td>
                          <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{r.featureValueIds.length}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <EmptyState message="No algorithm results found." />
              )}
            </div>
          ) : null}
        </div>
      )}

      {/* Result detail */}
      {selectedResult && (
        <div className="rounded border border-gray-200 bg-white p-3">
          <div className="mb-2 flex items-center justify-between gap-2">
            <div className="text-xs font-semibold text-gray-700">Result Detail</div>
            <CopyButton value={selectedResult.algorithm_result_id} />
          </div>
          {(() => {
            const r = summarizeAlgorithmResult(selectedResult);
            return (
              <div className="space-y-3">
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div><div className="text-xs text-gray-500">Result id</div><code className="break-all text-xs text-gray-700">{r.algorithmResultId || '—'}</code></div>
                  <div><div className="text-xs text-gray-500">Result type</div><div className="text-xs">{r.resultType || '—'}</div></div>
                  <div><div className="text-xs text-gray-500">Score / confidence</div><div className="text-xs">{r.score.toFixed(3)} / {r.confidence.toFixed(2)}</div></div>
                  <div><div className="text-xs text-gray-500">Severity</div><div><SeverityBadge severity={r.severity} /></div></div>
                  <div><div className="text-xs text-gray-500">Algorithm</div><div className="break-all text-xs">{r.algorithmId || '—'} <span className="text-gray-500">v{r.algorithmVersion || '—'}</span></div></div>
                  <div><div className="text-xs text-gray-500">Execution request</div><code className="break-all text-xs text-gray-700">{r.executionRequestId || '—'}</code></div>
                  <div><div className="text-xs text-gray-500">Correlation id</div><code className="break-all text-xs text-gray-700">{r.correlationId || '—'}</code></div>
                  <div><div className="text-xs text-gray-500">Created</div><div className="text-xs">{formatUtc(r.createdAt)}</div></div>
                </div>

                <JsonViewer label="Result payload JSON" value={selectedResult.result_payload} />

                <div className="space-y-2">
                  <IdRefList label="Source event ids" ids={r.sourceEventIds} />
                  <IdRefList label="Feature value ids" ids={r.featureValueIds} />
                  <IdRefList label="Evidence refs" ids={r.evidenceRefs} />
                </div>
              </div>
            );
          })()}
        </div>
      )}
    </div>
  );
}
