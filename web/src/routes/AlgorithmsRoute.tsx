import { useState } from 'react';
import { Cpu, GitPullRequestArrow, Zap } from 'lucide-react';
import {
  useAlgorithmDefinitions,
  useAlgorithmExecutionRequests,
  useAlgorithmExecutionSummary,
  useAlgorithmSignalProposals,
  useAlgorithmSignalProposal,
  useAlgorithmSignalProposalSummary,
  useAlgorithmSignalMaterializationPreflight,
  useAlgorithmSignalMaterializations,
  useMaterializeAlgorithmSignalProposal,
  useDecideAlgorithmSignalProposal,
} from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { CopyButton } from '../components/CopyButton';
import { OptionsQualityBadge } from '../components/OptionsQualityBadge';
import { formatUtc, formatPercent } from '../lib/format';
import {
  summarizeAlgorithmDefinition,
  summarizeAlgorithmExecutionRequest,
  summarizeAlgorithmResult,
  summarizeAlgorithmExecutionSummary,
  summarizeAlgorithmSignalProposal,
  summarizeAlgorithmSignalProposalSummary,
  summarizeAlgorithmSignalMaterializationPreflight,
  summarizeAlgorithmSignalMaterialization,
  algorithmDefinitionStatusStyle,
  algorithmExecutionStatusStyle,
  algorithmSeverityStyle,
  algorithmProposalStatusStyle,
  algorithmPreflightStatusStyle,
  algorithmMaterializationStatusStyle,
  findPreflightItemByProposalId,
  describeMaterializationEligibility,
  type AlgorithmSignalProposalSummary,
  type AlgorithmSignalProposalSummaryView,
  type AlgorithmSignalMaterializationPreflightView,
  type AlgorithmSignalMaterializationPreflightItemView,
  type AlgorithmSignalMaterializationView,
  type AlgorithmCountEntry,
} from '../lib/algorithms';
import {
  summarizeAlgorithmResultQuality,
  summarizeProposalQualityGate,
  formatZeroRate,
  isOptionsRatioQualityUsable,
} from '../lib/optionsQuality';
import { useTenant, useCanMutateLifecycle } from '../auth/session';
import type {
  AlgorithmDefinition,
  AlgorithmResult,
  AlgorithmSignalProposal,
  AlgorithmSignalProposalStatus,
} from '../types';

// G109 algorithm execution visibility (read-only). Mirrors the dense, restrained
// table layout of the other MarketOps workbenches. No Run/Tune/Promote/Deploy/
// Convert controls — this surface only renders backend algorithm state.

const DEFINITION_STATUSES = ['', 'draft', 'active', 'disabled', 'deprecated'];
const RUNTIME_TYPES = ['', 'python_plugin', 'container_plugin', 'http_plugin'];
const EXECUTION_STATUSES = ['', 'queued', 'running', 'succeeded', 'failed', 'canceled'];
const LIMITS = [25, 50, 100, 200];

// G113/G114 signal proposal review. The filter dropdowns include an empty "any"
// option; the review selector lists all four reviewable statuses (no `accepted`).
// Default list filter is status=proposed, limit=50 per the spec.
const PROPOSAL_STATUSES = ['', 'proposed', 'reviewed', 'rejected', 'superseded'];
const REVIEW_STATUSES: AlgorithmSignalProposalStatus[] = ['proposed', 'reviewed', 'rejected', 'superseded'];
const PROPOSAL_SEVERITIES = ['', 'critical', 'high', 'medium', 'low', 'info'];

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

// Review-only proposal status badge. `reviewed` is positive/complete tone but
// must never read as accepted/deployed — see algorithmProposalStatusStyle.
function ProposalStatusBadge({ status }: { status: string }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${algorithmProposalStatusStyle(status)}`}
    >
      {status || '—'}
    </span>
  );
}

// Read-only materialization preflight status badge (G119). `eligible` is neutral
// tone only — it must never read as accepted/deployed/materialized. See
// algorithmPreflightStatusStyle.
function PreflightStatusBadge({ status }: { status: string }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${algorithmPreflightStatusStyle(status)}`}
    >
      {status || '—'}
    </span>
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

  // G113/G114 signal proposal review filters + selection. Default status=proposed
  // and limit=50 per the spec; propExecReqId is carried over from a selected
  // execution request (selectExecutionRequest) but stays operator-editable so the
  // tenant-wide "all recent proposals" view is one clear away.
  const [propStatus, setPropStatus] = useState('proposed');
  const [propSeverity, setPropSeverity] = useState('');
  const [propAlgorithmId, setPropAlgorithmId] = useState('');
  const [propExecReqId, setPropExecReqId] = useState('');
  const [propCorrelationId, setPropCorrelationId] = useState('');
  const [propLimit, setPropLimit] = useState(50);
  const [selectedProposalId, setSelectedProposalId] = useState<string | null>(null);

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

  // Signal proposal list + selected-proposal detail. The list runs tenant-wide
  // with the operator filters; detail falls back to the matched list row while
  // the detail GET resolves. No polling — a decision invalidates both prefixes.
  const proposalsQ = useAlgorithmSignalProposals({
    tenant_id: TENANT_ID,
    algorithm_id: propAlgorithmId || undefined,
    execution_request_id: propExecReqId || undefined,
    status: (propStatus || undefined) as AlgorithmSignalProposalStatus | undefined,
    severity: propSeverity || undefined,
    correlation_id: propCorrelationId || undefined,
    limit: propLimit,
  });
  const proposals = proposalsQ.data?.algorithm_signal_proposals ?? [];
  // G116 review-coverage summary. Couples to the same filters as the list except
  // limit (the summary aggregates the whole matched slice). Runs independently so
  // its loading/error/empty states never block the list/detail workflow.
  const proposalSummaryQ = useAlgorithmSignalProposalSummary({
    tenant_id: TENANT_ID,
    algorithm_id: propAlgorithmId || undefined,
    execution_request_id: propExecReqId || undefined,
    status: (propStatus || undefined) as AlgorithmSignalProposalStatus | undefined,
    severity: propSeverity || undefined,
    correlation_id: propCorrelationId || undefined,
  });
  const proposalSummaryView = proposalSummaryQ.data
    ? summarizeAlgorithmSignalProposalSummary(proposalSummaryQ.data.algorithm_signal_proposal_summary)
    : null;
  // G119 read-only materialization preflight. Couples to the same proposal
  // filters as the list (including limit); min_reviewed_ratio defaults to 1 and
  // policy_version to materialization_preflight.v1 in the API client. Runs
  // independently so its loading/error/empty states never block the list/detail.
  const preflightQ = useAlgorithmSignalMaterializationPreflight({
    tenant_id: TENANT_ID,
    algorithm_id: propAlgorithmId || undefined,
    execution_request_id: propExecReqId || undefined,
    status: (propStatus || undefined) as AlgorithmSignalProposalStatus | undefined,
    severity: propSeverity || undefined,
    correlation_id: propCorrelationId || undefined,
    limit: propLimit,
  });
  const preflightView = preflightQ.data
    ? summarizeAlgorithmSignalMaterializationPreflight(
        preflightQ.data.algorithm_signal_materialization_preflight,
      )
    : null;
  const proposalDetailQ = useAlgorithmSignalProposal(selectedProposalId, TENANT_ID);
  const selectedProposal =
    proposalDetailQ.data?.algorithm_signal_proposal ??
    proposals.find((p) => p.proposal_id === selectedProposalId) ??
    null;
  // G123: the materialize action is gated on the selected proposal's preflight
  // item (eligible + would_write) and on any global preflight blockers. The
  // preflight slice follows the proposal-list filters, so a selected proposal
  // outside the current filter has no item and the action stays disabled.
  const selectedPreflightItem = findPreflightItemByProposalId(preflightView, selectedProposalId ?? '');
  const preflightGlobalBlockingActive = !!preflightView && preflightView.globalBlockingReasons.length > 0;

  function selectAlgorithm(id: string) {
    setSelectedAlgorithmId(id);
    setSelectedExecutionRequestId(null);
    setSelectedResultId(null);
  }
  function selectExecutionRequest(id: string) {
    setSelectedExecutionRequestId(id);
    setSelectedResultId(null);
    // Scope the proposals review view to this execution request per the spec.
    // The field stays editable so operators can clear it for the tenant-wide view.
    setPropExecReqId(id);
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
                    <th className="whitespace-nowrap px-3 py-2">Created</th>
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
                        <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{formatUtc(s.createdAt)}</td>
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

              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-600">
                <span>
                  <span className="text-gray-500">Requested by: </span>
                  {summaryView.executionRequest.requestedBy || '—'}
                </span>
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
                        <th className="whitespace-nowrap px-3 py-2">Quality</th>
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
                          <td className="whitespace-nowrap px-3 py-2">
                            {r.isOptionsRatio && r.optionsRatioQuality ? (
                              <OptionsQualityBadge quality={r.optionsRatioQuality} />
                            ) : (
                              <span className="text-xs text-gray-400">—</span>
                            )}
                          </td>
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
            const q = summarizeAlgorithmResultQuality(selectedResult.result_payload);
            if (!q.isOptionsRatio) return null;
            const usable = isOptionsRatioQualityUsable(q.ratioQuality);
            return (
              <div className={`mb-2 rounded border p-2 text-xs ${usable ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-amber-200 bg-amber-50 text-amber-700'}`}>
                <div className="flex flex-wrap items-center gap-2">
                  <OptionsQualityBadge quality={q.ratioQuality} label={`Ratio ${q.ratioQuality}`} />
                  <span>OI quality <strong>{q.quality.openInterestQuality || '—'}</strong></span>
                  <span>Zero rate <strong>{formatZeroRate(q.quality.openInterestZeroRate)}</strong></span>
                  <span>Zero/positive <strong>{q.quality.openInterestZeroCount ?? 0}/{q.quality.openInterestPositiveCount ?? 0}</strong></span>
                </div>
                <p className="mt-1 text-[11px]">
                  {usable
                    ? 'Usable evidence — eligible for the G131 proposal gate.'
                    : 'Not eligible for the G131 proposal gate — skipped from the proposal queue on quality grounds (still an algorithm result, not a vanished row).'}
                </p>
              </div>
            );
          })()}
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

      {/* G113/G114 signal proposals review surface (review-only). Renders the
          tenant-wide proposal ledger with operator filters; selecting a row
          opens evidence detail + bounded review controls. No materialization. */}
      <div className="rounded border border-gray-200 bg-white p-3">
        <div className="mb-2 flex items-center gap-2">
          <GitPullRequestArrow size={16} className="text-brand-700" />
          <div>
            <div className="text-xs font-semibold text-gray-700">Signal Proposals</div>
            <p className="text-[11px] text-gray-500">
              Review-only ledger of algorithm-derived candidate signal proposals · no production signal is materialized
            </p>
          </div>
        </div>

        <div className="mb-2 flex flex-wrap items-center gap-2">
          <select
            value={propStatus}
            onChange={(e) => setPropStatus(e.target.value)}
            className={inputCls}
            aria-label="Filter proposals by status"
          >
            {PROPOSAL_STATUSES.map((s) => (
              <option key={s} value={s}>{s || 'any status'}</option>
            ))}
          </select>
          <select
            value={propSeverity}
            onChange={(e) => setPropSeverity(e.target.value)}
            className={inputCls}
            aria-label="Filter proposals by severity"
          >
            {PROPOSAL_SEVERITIES.map((s) => (
              <option key={s} value={s}>{s || 'any severity'}</option>
            ))}
          </select>
          <input
            value={propAlgorithmId}
            onChange={(e) => setPropAlgorithmId(e.target.value)}
            className={inputCls}
            aria-label="Filter by algorithm id"
            placeholder="algorithm id"
          />
          <input
            value={propExecReqId}
            onChange={(e) => setPropExecReqId(e.target.value)}
            className={inputCls}
            aria-label="Filter by execution request id"
            placeholder="execution request id"
          />
          <input
            value={propCorrelationId}
            onChange={(e) => setPropCorrelationId(e.target.value)}
            className={inputCls}
            aria-label="Filter by correlation id"
            placeholder="correlation id"
          />
          <select
            value={propLimit}
            onChange={(e) => setPropLimit(Number(e.target.value))}
            className={inputCls}
            aria-label="Proposal page limit"
          >
            {LIMITS.map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </div>

        <ProposalSummaryPanel
          view={proposalSummaryView}
          isLoading={proposalSummaryQ.isLoading}
          isError={proposalSummaryQ.isError}
          error={proposalSummaryQ.error}
        />

        <MaterializationPreflightPanel
          view={preflightView}
          isLoading={preflightQ.isLoading}
          isError={preflightQ.isError}
          error={preflightQ.error}
        />

        {proposalsQ.isLoading ? (
          <LoadingState label="Loading algorithm signal proposals..." />
        ) : proposalsQ.isError ? (
          <ErrorState error={proposalsQ.error} />
        ) : proposals.length ? (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="whitespace-nowrap px-3 py-2">Proposal</th>
                  <th className="whitespace-nowrap px-3 py-2">Signal type</th>
                  <th className="whitespace-nowrap px-3 py-2">Status</th>
                  <th className="whitespace-nowrap px-3 py-2">Severity</th>
                  <th className="whitespace-nowrap px-3 py-2">Score</th>
                  <th className="whitespace-nowrap px-3 py-2">Conf.</th>
                  <th className="whitespace-nowrap px-3 py-2">Algorithm</th>
                  <th className="whitespace-nowrap px-3 py-2">Execution request</th>
                  <th className="whitespace-nowrap px-3 py-2">Result</th>
                  <th className="whitespace-nowrap px-3 py-2">Correlation</th>
                  <th className="whitespace-nowrap px-3 py-2">Reviewed by</th>
                  <th className="whitespace-nowrap px-3 py-2">Decided</th>
                  <th className="whitespace-nowrap px-3 py-2">Updated</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {proposals.map((p) => {
                  const s = summarizeAlgorithmSignalProposal(p);
                  return (
                    <tr
                      key={p.proposal_id}
                      onClick={() => setSelectedProposalId(p.proposal_id)}
                      className={`cursor-pointer align-top hover:bg-gray-50 ${selectedProposalId === p.proposal_id ? 'bg-brand-50' : ''}`}
                    >
                      <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{s.proposalId || '—'}</code></td>
                      <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{s.proposedSignalType || '—'}</code></td>
                      <td className="px-3 py-2"><ProposalStatusBadge status={s.status} /></td>
                      <td className="px-3 py-2"><SeverityBadge severity={s.severity} /></td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800">{s.score.toFixed(3)}</td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800">{s.confidence.toFixed(2)}</td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{s.algorithmId || '—'}</td>
                      <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{s.executionRequestId || '—'}</code></td>
                      <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{s.algorithmResultId || '—'}</code></td>
                      <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{s.correlationId || '—'}</code></td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{s.reviewedBy || '—'}</td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{s.decidedAt ? formatUtc(s.decidedAt) : '—'}</td>
                      <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{formatUtc(s.updatedAt)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState message="No algorithm signal proposals found." />
        )}
      </div>

      {/* Selected proposal detail + review controls */}
      {selectedProposal ? (
        <ProposalDetail
          key={selectedProposal.proposal_id}
          proposal={selectedProposal}
          tenantId={TENANT_ID}
          detailLoading={proposalDetailQ.isLoading && !!selectedProposalId}
          detailError={proposalDetailQ.isError ? proposalDetailQ.error : null}
          preflightItem={selectedPreflightItem}
          preflightLoading={preflightQ.isLoading}
          preflightFailed={preflightQ.isError}
          globalBlockingActive={preflightGlobalBlockingActive}
        />
      ) : (
        <div className="rounded border border-gray-200 bg-white p-3">
          <EmptyState message="Select a proposal to inspect its evidence." />
        </div>
      )}
    </div>
  );
}

// Selected algorithm signal proposal evidence detail. Renders all spec-required
// fields, lineage id lists, and the collapsible proposal_payload / rationale
// JSON, then the bounded review controls, the G123 materialization action, and
// the materialization ledger. Read-only except for the review + materialize forms.
function ProposalDetail({
  proposal,
  tenantId,
  detailLoading,
  detailError,
  preflightItem,
  preflightLoading,
  preflightFailed,
  globalBlockingActive,
}: {
  proposal: AlgorithmSignalProposal;
  tenantId: string;
  detailLoading: boolean;
  detailError: unknown;
  preflightItem: AlgorithmSignalMaterializationPreflightItemView | null;
  preflightLoading: boolean;
  preflightFailed: boolean;
  globalBlockingActive: boolean;
}) {
  const p = summarizeAlgorithmSignalProposal(proposal);
  return (
    <div className="rounded border border-gray-200 bg-white p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <div className="text-xs font-semibold text-gray-700">Proposal Detail</div>
          <ProposalStatusBadge status={p.status} />
          <code className="break-all text-[11px] text-gray-500">{p.proposedSignalType}</code>
          {detailLoading ? <span className="text-[11px] text-gray-400">refreshing…</span> : null}
        </div>
        <CopyButton value={p.proposalId} />
      </div>

      {detailError ? (
        <p className="mb-2 rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-700">
          Detail refresh unavailable; showing the latest list row.
        </p>
      ) : null}

      <div className="space-y-3">
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div><div className="text-xs text-gray-500">Proposal id</div><code className="break-all text-xs text-gray-700">{p.proposalId || '—'}</code></div>
          <div><div className="text-xs text-gray-500">Tenant id</div><code className="break-all text-xs text-gray-700">{p.tenantId || '—'}</code></div>
          <div><div className="text-xs text-gray-500">Proposed signal type</div><code className="break-all text-xs text-gray-700">{p.proposedSignalType || '—'}</code></div>
          <div><div className="text-xs text-gray-500">Status</div><div><ProposalStatusBadge status={p.status} /></div></div>
          <div><div className="text-xs text-gray-500">Score / confidence</div><div className="text-xs">{p.score.toFixed(3)} / {p.confidence.toFixed(2)}</div></div>
          <div><div className="text-xs text-gray-500">Severity</div><div><SeverityBadge severity={p.severity} /></div></div>
          <div><div className="text-xs text-gray-500">Algorithm</div><div className="break-all text-xs">{p.algorithmId || '—'} <span className="text-gray-500">v{p.algorithmVersion || '—'}</span></div></div>
          <div><div className="text-xs text-gray-500">Execution request</div><code className="break-all text-xs text-gray-700">{p.executionRequestId || '—'}</code></div>
          <div><div className="text-xs text-gray-500">Algorithm result</div><code className="break-all text-xs text-gray-700">{p.algorithmResultId || '—'}</code></div>
          <div><div className="text-xs text-gray-500">Correlation id</div><code className="break-all text-xs text-gray-700">{p.correlationId || '—'}</code></div>
          <div><div className="text-xs text-gray-500">Created by</div><div className="break-all text-xs">{p.createdBy || '—'}</div></div>
          <div><div className="text-xs text-gray-500">Reviewed by</div><div className="break-all text-xs">{p.reviewedBy || '—'}</div></div>
          <div><div className="text-xs text-gray-500">Decided at</div><div className="text-xs">{p.decidedAt ? formatUtc(p.decidedAt) : '—'}</div></div>
          <div><div className="text-xs text-gray-500">Created / updated</div><div className="text-xs">{formatUtc(p.createdAt)} / {formatUtc(p.updatedAt)}</div></div>
          <div className="col-span-2"><div className="text-xs text-gray-500">Decision note</div><div className="break-all text-xs">{p.decisionNote || '—'}</div></div>
        </div>

        <div className="space-y-2">
          <IdRefList label="Source event ids" ids={p.sourceEventIds} />
          <IdRefList label="Evidence refs" ids={p.evidenceRefs} />
        </div>

        {(() => {
          // G131 evidence-quality gate + nested options ratio quality.
          const g = summarizeProposalQualityGate(p.proposalPayload);
          if (!g.present && !g.isOptionsRatio) return null;
          return (
            <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs">
              <div className="mb-1 font-medium text-gray-600">Evidence quality</div>
              <div className="flex flex-wrap items-center gap-2">
                {g.present ? (
                  <span
                    className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[11px] font-medium ${g.passed ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-amber-200 bg-amber-50 text-amber-700'}`}
                  >
                    Gate {g.passed === null ? '—' : g.passed ? 'passed' : 'not passed'}
                  </span>
                ) : null}
                {g.policy ? (
                  <span className="text-gray-600">policy <code>{g.policy}</code></span>
                ) : null}
                {g.isOptionsRatio ? <OptionsQualityBadge quality={g.ratioQuality} label={`Ratio ${g.ratioQuality}`} /> : null}
                {g.openInterestQuality ? (
                  <span className="text-gray-600">OI <strong>{g.openInterestQuality}</strong></span>
                ) : null}
                {g.isOptionsRatio ? (
                  <span className="text-gray-600">Zero rate <strong>{formatZeroRate(g.quality.openInterestZeroRate)}</strong></span>
                ) : null}
              </div>
              {!g.present ? (
                <p className="mt-1 text-[11px] text-gray-400">No G131 quality gate recorded for this proposal.</p>
              ) : null}
            </div>
          );
        })()}

        <JsonViewer label="Proposal payload JSON" value={p.proposalPayload} />
        <JsonViewer label="Rationale JSON" value={p.rationale} />

        <ProposalReviewControls proposal={p} tenantId={tenantId} />

        <ProposalMaterializationSection
          proposal={p}
          tenantId={tenantId}
          preflightItem={preflightItem}
          preflightLoading={preflightLoading}
          preflightFailed={preflightFailed}
          globalBlockingActive={globalBlockingActive}
        />
      </div>
    </div>
  );
}

// Bounded review form for a single algorithm signal proposal. Records one review
// decision (reviewed / rejected / superseded / restore to proposed) with an
// optional note that is required for rejected and superseded. The POST only
// updates review metadata — it materializes no production signal.
function ProposalReviewControls({
  proposal,
  tenantId,
}: {
  proposal: AlgorithmSignalProposalSummary;
  tenantId: string;
}) {
  const [status, setStatus] = useState<AlgorithmSignalProposalStatus>(proposal.status || 'proposed');
  const [note, setNote] = useState(proposal.decisionNote || '');
  const mutation = useDecideAlgorithmSignalProposal();

  const requiresNote = status === 'rejected' || status === 'superseded';
  const noteMissing = requiresNote && note.trim() === '';
  const canSubmit = !mutation.isPending && !noteMissing;

  function submit() {
    if (!canSubmit) return;
    mutation.mutate({
      proposalId: proposal.proposalId,
      tenantId,
      request: {
        tenant_id: tenantId,
        status,
        note: note.trim() || undefined,
      },
    });
  }

  const errorMessage = mutation.isError
    ? isApiError(mutation.error)
      ? mutation.error.message
      : 'Review update failed.'
    : '';

  return (
    <div className="rounded border border-gray-200 bg-gray-50 p-2">
      <div className="mb-2 text-xs font-medium text-gray-600">Review Decision</div>
      <div className="grid grid-cols-1 gap-2 md:grid-cols-3">
        <div>
          <label className="mb-0.5 block text-[11px] text-gray-500" htmlFor="proposal-review-status">Status</label>
          <select
            id="proposal-review-status"
            value={status}
            onChange={(e) => setStatus(e.target.value as AlgorithmSignalProposalStatus)}
            className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
            disabled={mutation.isPending}
          >
            {REVIEW_STATUSES.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </div>
        <div className="md:col-span-2">
          <label className="mb-0.5 block text-[11px] text-gray-500" htmlFor="proposal-review-note">Note</label>
          <textarea
            id="proposal-review-note"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={requiresNote ? 'Required note for this decision' : 'Optional review note'}
            className="h-16 w-full resize-none rounded border border-gray-300 px-2 py-1 text-xs text-gray-700 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            disabled={mutation.isPending}
          />
        </div>
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit}
          className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Save review
        </button>
        {noteMissing ? <span className="text-[11px] text-red-700">A note is required for {status}.</span> : null}
        {mutation.isPending ? <span className="text-[11px] text-gray-400">Saving…</span> : null}
        {mutation.isSuccess ? <span className="text-[11px] text-emerald-700">Review saved.</span> : null}
        {errorMessage ? <span className="text-[11px] text-red-700">{errorMessage}</span> : null}
      </div>
      <p className="mt-1 text-[11px] text-gray-400">Review records operator metadata only; no production signal is materialized.</p>
    </div>
  );
}

// Truncate a string for dense table cells; full value stays available via title.
function truncate(value: string, max: number): string {
  if (value.length <= max) return value;
  return `${value.slice(0, max)}…`;
}

// Materialization status badge (G121/G123). Mirrors the preflight/proposal badge
// shape; tones per algorithmMaterializationStatusStyle.
function MaterializationStatusBadge({ status }: { status: string }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${algorithmMaterializationStatusStyle(status)}`}
    >
      {status || '—'}
    </span>
  );
}

// G123 single-proposal materialization action + ledger. The Materialize button is
// gated by describeMaterializationEligibility (reviewed + eligible preflight item
// + would_write + operator/admin role + no global blockers + not already
// materialized). A click expands an inline confirmation (no modal in this app);
// confirming POSTs the G122 mutation. Result rendering branches on
// materialization_status — succeeded/duplicate/blocked come back as 2xx
// envelopes, so they are read from mutation.data, not mutation.error. This is the
// only control on the surface that writes a production signal.
function ProposalMaterializationSection({
  proposal,
  tenantId,
  preflightItem,
  preflightLoading,
  preflightFailed,
  globalBlockingActive,
}: {
  proposal: AlgorithmSignalProposalSummary;
  tenantId: string;
  preflightItem: AlgorithmSignalMaterializationPreflightItemView | null;
  preflightLoading: boolean;
  preflightFailed: boolean;
  globalBlockingActive: boolean;
}) {
  const canMutate = useCanMutateLifecycle();
  const materializationsQ = useAlgorithmSignalMaterializations({
    tenant_id: tenantId,
    proposal_id: proposal.proposalId,
    limit: 50,
  });
  const mutation = useMaterializeAlgorithmSignalProposal();
  const [confirming, setConfirming] = useState(false);
  const [note, setNote] = useState('');

  const rows = (materializationsQ.data?.algorithm_signal_materializations ?? []).map(summarizeAlgorithmSignalMaterialization);
  const hasRecordedMaterialization = rows.some(
    (r) => r.materializationStatus === 'succeeded' || r.materializationStatus === 'duplicate',
  );
  const eligibility = describeMaterializationEligibility({
    proposalStatus: proposal.status,
    preflightItem,
    preflightLoading,
    preflightFailed,
    globalBlockingActive,
    canMutate,
    mutationPending: mutation.isPending,
    hasRecordedMaterialization,
  });

  function confirm() {
    if (mutation.isPending) return;
    mutation.mutate(
      {
        proposalId: proposal.proposalId,
        request: {
          tenant_id: tenantId,
          policy_version: 'algorithm_materialization.v1',
          metadata: note.trim() ? { note: note.trim() } : {},
        },
      },
      { onSuccess: () => setConfirming(false) },
    );
  }
  function cancel() {
    setConfirming(false);
    mutation.reset();
  }

  const resultView = mutation.data
    ? summarizeAlgorithmSignalMaterialization(mutation.data.algorithm_signal_materialization)
    : null;
  const btnCls = 'inline-flex items-center gap-1 rounded border px-2 py-1 text-xs font-medium';

  return (
    <div className="space-y-2">
      <div className="rounded border border-gray-200 bg-gray-50 p-2">
        <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
          <div className="text-xs font-medium text-gray-600">Materialization</div>
          <span className="text-[11px] text-gray-400">Writes one production signal · no alerts/insights/graph</span>
        </div>
        {!confirming ? (
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setConfirming(true)}
              disabled={!eligibility.canMaterialize}
              title={!eligibility.canMaterialize ? eligibility.reason : undefined}
              className={`${btnCls} border-gray-300 bg-white text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50`}
            >
              <Zap size={14} /> Materialize
            </button>
            {!eligibility.canMaterialize ? (
              <span className="text-[11px] text-gray-500">{eligibility.reason}</span>
            ) : null}
          </div>
        ) : (
          <div className="space-y-2 rounded border border-amber-200 bg-amber-50 p-2">
            <div className="text-xs font-medium text-amber-800">Confirm materialization</div>
            <div className="grid grid-cols-2 gap-1 text-[11px] text-gray-700">
              <div className="col-span-2">Proposal: <code className="break-all">{proposal.proposalId || '—'}</code></div>
              <div>Signal type: <code className="break-all">{proposal.proposedSignalType || '—'}</code></div>
              <div>Algorithm: <span className="break-all">{proposal.algorithmId || '—'} <span className="text-gray-500">v{proposal.algorithmVersion || '—'}</span></span></div>
              <div>Severity: <SeverityBadge severity={proposal.severity} /></div>
              <div>Confidence: {proposal.confidence.toFixed(2)}</div>
              <div>Source events: {proposal.sourceEventIds.length}</div>
              <div>Policy: <code>algorithm_materialization.v1</code></div>
            </div>
            <p className="text-[11px] text-amber-800">
              One production signal may be created. Alerts, insights, and graph proposals are not directly created by this action.
            </p>
            <div>
              <label className="mb-0.5 block text-[11px] text-gray-500" htmlFor="materialize-note">Note (optional)</label>
              <textarea
                id="materialize-note"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Optional operator note recorded in materialization metadata"
                className="h-14 w-full resize-none rounded border border-gray-300 px-2 py-1 text-xs text-gray-700 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
                disabled={mutation.isPending}
              />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={confirm}
                disabled={mutation.isPending}
                className={`${btnCls} border-amber-400 bg-amber-100 text-amber-900 hover:bg-amber-200 disabled:cursor-not-allowed disabled:opacity-50`}
              >
                <Zap size={14} /> Confirm materialization
              </button>
              <button
                type="button"
                onClick={cancel}
                disabled={mutation.isPending}
                className={`${btnCls} border-gray-300 bg-white text-gray-600 hover:bg-gray-50 disabled:opacity-50`}
              >
                Cancel
              </button>
              {mutation.isPending ? <span className="text-[11px] text-gray-500">Materializing…</span> : null}
            </div>
          </div>
        )}

        <MaterializationResultBanner view={resultView} error={mutation.isError ? mutation.error : null} />

        <p className="mt-1 text-[11px] text-gray-400">
          The backend remains authoritative; eligibility here only controls the button. The action is idempotent — repeating it returns the existing record.
        </p>
      </div>

      <MaterializationLedgerPanel
        rows={rows}
        isLoading={materializationsQ.isLoading}
        isError={materializationsQ.isError}
        error={materializationsQ.error}
      />
    </div>
  );
}

// Render the latest mutation outcome. succeeded/duplicate/blocked arrive as 2xx
// envelopes (read from mutation.data); only not-found/auth/server failures throw
// and surface via the error branch. Sanitized — no tokens/headers/stack.
function MaterializationResultBanner({
  view,
  error,
}: {
  view: AlgorithmSignalMaterializationView | null;
  error: unknown;
}) {
  if (error) {
    const msg = isApiError(error) ? `${error.code}: ${error.message}` : 'Materialization failed.';
    return (
      <p className="mt-2 rounded border border-red-200 bg-red-50 px-2 py-1 text-xs text-red-700">{msg}</p>
    );
  }
  if (!view) return null;
  const status = view.materializationStatus;
  if (status === 'succeeded') {
    return (
      <div className="mt-2 flex flex-wrap items-center gap-2 rounded border border-emerald-200 bg-emerald-50 px-2 py-1 text-xs text-emerald-700">
        <span>Materialized as signal <code className="break-all">{view.signalId || '—'}</code></span>
        {view.signalId ? <CopyButton value={view.signalId} /> : null}
        {view.signalId ? <a className="underline hover:text-emerald-800" href="/signals">view in Signals</a> : null}
      </div>
    );
  }
  if (status === 'duplicate') {
    return (
      <p className="mt-2 rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-700">
        Already exists as signal <code className="break-all">{view.duplicateOfSignalId || '—'}</code>; no new signal created.
      </p>
    );
  }
  if (status === 'blocked') {
    return (
      <p className="mt-2 rounded border border-orange-200 bg-orange-50 px-2 py-1 text-xs text-orange-700">
        Blocked: {view.errorMessage || view.errorCode || 'blocked'}
      </p>
    );
  }
  if (status === 'failed') {
    return (
      <p className="mt-2 rounded border border-red-200 bg-red-50 px-2 py-1 text-xs text-red-700">
        Failed: {view.errorMessage || view.errorCode || 'failed'}
      </p>
    );
  }
  if (status === 'requested' || status === 'running') {
    return (
      <p className="mt-2 rounded border border-blue-200 bg-blue-50 px-2 py-1 text-xs text-blue-700">
        Materialization in progress…
      </p>
    );
  }
  if (status === 'superseded') {
    return (
      <p className="mt-2 rounded border border-gray-200 bg-gray-100 px-2 py-1 text-xs text-gray-500">Superseded.</p>
    );
  }
  return null;
}

// G121 materialization ledger for the selected proposal. Dense rows (status,
// policy, signal ids, requester, timestamps, error, idempotency key) with the
// three JSON fields collapsed via JsonViewer. Scoped loading/error/empty.
function MaterializationLedgerPanel({
  rows,
  isLoading,
  isError,
  error,
}: {
  rows: AlgorithmSignalMaterializationView[];
  isLoading: boolean;
  isError: boolean;
  error: unknown;
}) {
  return (
    <div className="rounded border border-gray-200 bg-white p-2">
      <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500">Materialization Records</div>
      {isLoading ? (
        <div className="text-xs text-gray-500">Loading materialization records…</div>
      ) : isError ? (
        <div className="text-xs text-red-700">
          Materialization records unavailable{isApiError(error) ? `: ${error.message}` : ''}.
        </div>
      ) : rows.length === 0 ? (
        <div className="text-xs text-gray-500">No materialization records for this proposal.</div>
      ) : (
        <div className="space-y-2">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="whitespace-nowrap px-3 py-2">Materialization</th>
                  <th className="whitespace-nowrap px-3 py-2">Status</th>
                  <th className="whitespace-nowrap px-3 py-2">Policy</th>
                  <th className="whitespace-nowrap px-3 py-2">Signal</th>
                  <th className="whitespace-nowrap px-3 py-2">Duplicate of</th>
                  <th className="whitespace-nowrap px-3 py-2">Requested by</th>
                  <th className="whitespace-nowrap px-3 py-2">Requested</th>
                  <th className="whitespace-nowrap px-3 py-2">Completed</th>
                  <th className="whitespace-nowrap px-3 py-2">Error</th>
                  <th className="whitespace-nowrap px-3 py-2">Idempotency key</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {rows.map((r) => (
                  <tr key={r.materializationId} className="align-top">
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{r.materializationId || '—'}</code></td>
                    <td className="px-3 py-2"><MaterializationStatusBadge status={r.materializationStatus} /></td>
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{r.materializationPolicyVersion || '—'}</code></td>
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{r.signalId || '—'}</code></td>
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{r.duplicateOfSignalId || '—'}</code></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{r.requestedBy || '—'}</td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{r.requestedAt ? formatUtc(r.requestedAt) : '—'}</td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{(r.completedAt || r.failedAt) ? formatUtc(r.completedAt || r.failedAt) : '—'}</td>
                    <td className="px-3 py-2">
                      <code className="break-all text-[11px] text-gray-600" title={r.errorMessage || r.errorCode || ''}>
                        {r.errorCode || r.errorMessage
                          ? `${r.errorCode || ''}${r.errorCode && r.errorMessage ? ': ' : ''}${truncate(r.errorMessage, 60)}`
                          : '—'}
                      </code>
                    </td>
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-500" title={r.idempotencyKey}>{truncate(r.idempotencyKey, 24)}</code></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {rows.map((r) => (
            <div key={`${r.materializationId}-json`} className="grid grid-cols-1 gap-2 md:grid-cols-3">
              <JsonViewer label="Request metadata" value={r.requestMetadata} />
              <JsonViewer label="Preflight snapshot" value={r.preflightSnapshot} />
              <JsonViewer label="Signal payload preview" value={r.signalPayloadPreview} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// G116 compact review-coverage summary for the currently filtered proposal
// slice. Dense metrics strip + small breakdown chip lists. Loading / error /
// empty states are scoped to this panel and never block the list or detail.
// `reviewed` is shown as coverage only — it never implies accepted or deployed.
function ProposalSummaryPanel({
  view,
  isLoading,
  isError,
  error,
}: {
  view: AlgorithmSignalProposalSummaryView | null;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
}) {
  const hasHighCriticalUnreviewed = (view?.highCriticalUnreviewedCount ?? 0) > 0;
  return (
    <div className="mb-2 rounded border border-gray-200 bg-gray-50 p-2">
      <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500">Review Coverage</div>
      {isLoading ? (
        <div className="text-xs text-gray-500">Loading proposal summary…</div>
      ) : isError ? (
        <div className="text-xs text-red-700">
          Proposal summary unavailable{isApiError(error) ? `: ${error.message}` : ''}.
        </div>
      ) : !view || view.totalProposals === 0 ? (
        <div className="text-xs text-gray-500">No proposal summary for this filter.</div>
      ) : (
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-700">
            <SummaryStat label="Total" value={view.totalProposals} />
            <SummaryStat label="Reviewed" value={formatPercent(view.reviewedRatio)} />
            <SummaryStat label="Proposed" value={view.proposedCount} />
            <SummaryStat label="Reviewed #" value={view.reviewedCount} />
            <SummaryStat label="Rejected" value={view.rejectedCount} />
            <SummaryStat label="Superseded" value={view.supersededCount} />
            {hasHighCriticalUnreviewed ? (
              <span className="inline-flex items-center gap-1 rounded border border-red-300 bg-red-50 px-1.5 py-0.5 font-medium text-red-700">
                High/critical unreviewed <strong>{view.highCriticalUnreviewedCount}</strong>
              </span>
            ) : (
              <SummaryStat label="High/critical unreviewed" value={view.highCriticalUnreviewedCount} />
            )}
          </div>
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
            <SummaryCountChips label="Status" entries={view.statusCounts} chipClassName={algorithmProposalStatusStyle} />
            <SummaryCountChips
              label="Severity"
              entries={view.severityCounts}
              chipClassName={(s) => `border-gray-200 bg-white ${algorithmSeverityStyle(s)}`}
            />
            <SummaryCountChips label="Signal type" entries={view.proposedSignalTypeCounts} />
            <SummaryCountChips label="Algorithm" entries={view.algorithmIdCounts} />
            <SummaryCountChips label="Reviewer" entries={view.reviewerCounts} />
          </div>
        </div>
      )}
    </div>
  );
}

// G119 compact read-only materialization preflight panel. Dense metrics strip +
// prominent global-blocker warnings + reason/global-blocker chip lists + a small
// per-proposal preflight table. Loading / error / empty states are scoped to
// this panel and never block the list, detail, or review workflow. `eligible` and
// `would_write` are forecast-only — this panel materializes nothing and never
// says proposals are accepted, deployed, or production signals.
function MaterializationPreflightPanel({
  view,
  isLoading,
  isError,
  error,
}: {
  view: AlgorithmSignalMaterializationPreflightView | null;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
}) {
  const coverageBelow = view ? !view.reviewCoverageSatisfied : false;
  const hasHighCriticalUnreviewed = (view?.highCriticalUnreviewedCount ?? 0) > 0;
  return (
    <div className="mb-2 rounded border border-gray-200 bg-gray-50 p-2">
      <div className="mb-1 flex flex-wrap items-center justify-between gap-2">
        <div className="text-[11px] font-semibold uppercase tracking-wide text-gray-500">Materialization Preflight</div>
        <span className="text-[11px] text-gray-400">Read-only preflight · no signal is materialized</span>
      </div>
      {isLoading ? (
        <div className="text-xs text-gray-500">Loading materialization preflight…</div>
      ) : isError ? (
        <div className="text-xs text-red-700">
          Materialization preflight unavailable{isApiError(error) ? `: ${error.message}` : ''}.
        </div>
      ) : !view || view.totalProposals === 0 ? (
        <div className="text-xs text-gray-500">No materialization preflight rows for this filter.</div>
      ) : (
        <div className="space-y-2">
          {(coverageBelow || hasHighCriticalUnreviewed) && (
            <div className="flex flex-wrap gap-1.5">
              {coverageBelow && (
                <span className="inline-flex items-center gap-1 rounded border border-amber-300 bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700">
                  Review coverage below threshold
                </span>
              )}
              {hasHighCriticalUnreviewed && (
                <span className="inline-flex items-center gap-1 rounded border border-red-300 bg-red-50 px-1.5 py-0.5 text-[11px] font-medium text-red-700">
                  High/critical unreviewed <strong>{view.highCriticalUnreviewedCount}</strong>
                </span>
              )}
            </div>
          )}

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-700">
            <SummaryStat label="Total" value={view.totalProposals} />
            <SummaryStat label="Eligible" value={view.eligibleCount} />
            <SummaryStat label="Duplicate risk" value={view.duplicateRiskCount} />
            <SummaryStat label="Blocked" value={view.blockedCount} />
            <SummaryStat label="Invalid" value={view.invalidCount} />
            <SummaryStat label="Would write" value={view.wouldWriteCount} />
            <SummaryStat label="Reviewed" value={formatPercent(view.reviewedRatio)} />
            <SummaryStat label="Min reviewed" value={formatPercent(view.minReviewedRatio)} />
            <SummaryStat label="High/critical unreviewed" value={view.highCriticalUnreviewedCount} />
          </div>

          <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
            <SummaryCountChips
              label="Global blockers"
              entries={view.globalBlockingReasons}
              chipClassName={() => 'border-amber-200 bg-amber-50 text-amber-700'}
            />
            <SummaryCountChips label="Reason breakdown" entries={view.itemReasonCounts} />
          </div>

          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="whitespace-nowrap px-3 py-2">Proposal</th>
                  <th className="whitespace-nowrap px-3 py-2">Signal type</th>
                  <th className="whitespace-nowrap px-3 py-2">Preflight</th>
                  <th className="whitespace-nowrap px-3 py-2">Review</th>
                  <th className="whitespace-nowrap px-3 py-2">Severity</th>
                  <th className="whitespace-nowrap px-3 py-2">Conf.</th>
                  <th className="whitespace-nowrap px-3 py-2">Would write</th>
                  <th className="whitespace-nowrap px-3 py-2">Reasons</th>
                  <th className="whitespace-nowrap px-3 py-2">Dup signals</th>
                  <th className="whitespace-nowrap px-3 py-2">Algorithm</th>
                  <th className="whitespace-nowrap px-3 py-2">Execution request</th>
                  <th className="whitespace-nowrap px-3 py-2">Result</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {view.items.map((it) => (
                  <tr key={it.proposalId || `${it.algorithmResultId}-${it.proposedSignalType}`} className="align-top">
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{it.proposalId || '—'}</code></td>
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{it.proposedSignalType || '—'}</code></td>
                    <td className="px-3 py-2"><PreflightStatusBadge status={it.preflightStatus} /></td>
                    <td className="px-3 py-2"><ProposalStatusBadge status={it.status} /></td>
                    <td className="px-3 py-2"><SeverityBadge severity={it.severity} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800">{it.confidence.toFixed(2)}</td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{it.wouldWrite ? 'Yes' : 'No'}</td>
                    <td className="px-3 py-2">
                      <code className="break-all text-[11px] text-gray-600" title={it.reasons.join(', ')}>
                        {it.reasons.length ? it.reasons.join(', ') : '—'}
                      </code>
                    </td>
                    <td className="px-3 py-2">
                      {it.duplicateSignalIds.length ? (
                        <span className="inline-flex items-center gap-1">
                          <span className="rounded border border-gray-200 px-1 text-[11px] text-gray-600">{it.duplicateSignalIds.length}</span>
                          <code className="break-all text-[10px] text-gray-500" title={it.duplicateSignalIds.join(', ')}>
                            {it.duplicateSignalIds.join(', ')}
                          </code>
                        </span>
                      ) : (
                        <span className="text-xs text-gray-400">—</span>
                      )}
                    </td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700">{it.algorithmId || '—'}</td>
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{it.executionRequestId || '—'}</code></td>
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-600">{it.algorithmResultId || '—'}</code></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <p className="text-[11px] text-gray-400">
            Preflight forecasts materialization eligibility only; would-write counts are hypothetical and perform no action.
          </p>
        </div>
      )}
    </div>
  );
}

function SummaryStat({ label, value }: { label: string; value: number | string }) {
  return (
    <span>
      <span className="text-gray-500">{label}: </span>
      <strong className="text-gray-800">{value}</strong>
    </span>
  );
}

// Compact `key count` chip list for a summary breakdown. chipClassName returns
// the full chip tone (status uses proposal status style; severity layers the
// severity text color over a neutral chip); omitted falls back to neutral.
// Empty maps render as `None`, never raw `{}`.
function SummaryCountChips({
  label,
  entries,
  chipClassName,
}: {
  label: string;
  entries: AlgorithmCountEntry[];
  chipClassName?: (key: string) => string;
}) {
  const base = 'inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[11px]';
  return (
    <div>
      <div className="mb-0.5 text-[11px] font-medium text-gray-600">{label}</div>
      {entries.length ? (
        <div className="flex flex-wrap gap-1">
          {entries.map((e) => (
            <span
              key={e.key}
              className={`${base} ${chipClassName ? chipClassName(e.key) : 'border-gray-200 bg-white text-gray-700'}`}
            >
              <span className="break-all">{e.key}</span>
              <strong>{e.count}</strong>
            </span>
          ))}
        </div>
      ) : (
        <span className="text-[11px] text-gray-400">None</span>
      )}
    </div>
  );
}
