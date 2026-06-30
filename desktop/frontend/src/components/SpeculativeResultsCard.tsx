import { useState } from "react";
import {
  Sigma,
  GitCompare,
  CheckCircle2,
  HelpCircle,
  XCircle,
  ChevronDown,
  ChevronRight,
  DollarSign,
} from "lucide-react";

export type Verdict = "confirmed" | "plausible" | "refuted" | "uncertain";

export interface SpeculativeRun {
  id: string;
  agentId: string;
  verdict: Verdict;
  confidence: number; // 0-100
  tokensUsed: number;
  cost: number;
  durationMs: number;
  summary: string;
  consensus: boolean;
}

export interface SpeculativeResults {
  confirmedRuns: SpeculativeRun[];
  uncertainRuns: SpeculativeRun[];
  totalRuns: number;
  totalCost: number;
  singleExplorerCost: number;
  totalTokens: number;
}

function verdictIcon(verdict: Verdict) {
  switch (verdict) {
    case "confirmed":
      return <CheckCircle2 size={14} className="text-green-500 flex-shrink-0" />;
    case "plausible":
      return <HelpCircle size={14} className="text-yellow-500 flex-shrink-0" />;
    case "refuted":
      return <XCircle size={14} className="text-red-500 flex-shrink-0" />;
    case "uncertain":
      return <HelpCircle size={14} className="text-gray-400 flex-shrink-0" />;
  }
}

function verdictLabel(verdict: Verdict): string {
  switch (verdict) {
    case "confirmed":
      return "CONFIRMED";
    case "plausible":
      return "PLAUSIBLE";
    case "refuted":
      return "REFUTED";
    case "uncertain":
      return "UNCERTAIN";
  }
}

function verdictColorClass(verdict: Verdict): string {
  switch (verdict) {
    case "confirmed":
      return "text-green-700 bg-green-100";
    case "plausible":
      return "text-yellow-700 bg-yellow-100";
    case "refuted":
      return "text-red-700 bg-red-100";
    case "uncertain":
      return "text-gray-600 bg-gray-100";
  }
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function RunDetailRow({ run }: { run: SpeculativeRun }) {
  return (
    <div className="flex items-start gap-3 py-2 border-b border-gray-100 last:border-0">
      {verdictIcon(run.verdict)}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-medium text-gray-800">
            {run.agentId}
          </span>
          <span
            className={`text-[10px] font-semibold px-1.5 py-0.5 rounded uppercase ${verdictColorClass(
              run.verdict
            )}`}
          >
            {verdictLabel(run.verdict)}
          </span>
          {run.consensus && (
            <span className="text-[10px] text-green-600 font-medium">
              IN CONSENSUS
            </span>
          )}
        </div>
        <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">
          {run.summary}
        </p>
        <div className="flex items-center gap-3 mt-1 text-[11px] text-gray-400">
          <span>{formatDuration(run.durationMs)}</span>
          <span>{run.tokensUsed.toLocaleString()} tok</span>
          <span>${run.cost.toFixed(4)}</span>
          <span>{run.confidence}% confidence</span>
        </div>
      </div>
    </div>
  );
}

function RunSection({
  title,
  runs,
  verdict,
  subtitle,
}: {
  title: string;
  runs: SpeculativeRun[];
  verdict: Verdict;
  subtitle: string;
}) {
  const [expanded, setExpanded] = useState(true);

  if (runs.length === 0) return null;

  return (
    <div className="rounded-md border border-gray-200 overflow-hidden">
      <button
        type="button"
        className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-gray-50 transition-colors"
        onClick={() => setExpanded((v) => !v)}
      >
        {verdictIcon(verdict)}
        <span className="text-sm font-medium text-gray-800 flex-1">
          {title}
        </span>
        <span className="text-xs text-gray-500">{subtitle}</span>
        {expanded ? (
          <ChevronDown size={14} className="text-gray-400" />
        ) : (
          <ChevronRight size={14} className="text-gray-400" />
        )}
      </button>
      {expanded && (
        <div className="border-t border-gray-100 px-3 py-1">
          {runs.map((run) => (
            <RunDetailRow key={run.id} run={run} />
          ))}
        </div>
      )}
    </div>
  );
}

export function SpeculativeResultsCard({
  results,
  className,
}: {
  results: SpeculativeResults;
  className?: string;
}) {
  const confirmedCount = results.confirmedRuns.length;
  const uncertainCount = results.uncertainRuns.length;
  const costDelta = results.totalCost - results.singleExplorerCost;
  const costDeltaPct = results.singleExplorerCost > 0
    ? ((costDelta / results.singleExplorerCost) * 100).toFixed(0)
    : "0";

  return (
    <div
      className={`rounded-lg border border-gray-200 bg-white shadow-sm${
        className ? ` ${className}` : ""
      }`}
    >
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3 border-b border-gray-100">
        <div className="w-8 h-8 rounded-md bg-amber-50 flex items-center justify-center">
          <Sigma size={16} className="text-amber-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm text-gray-900">
              Speculative Execution
            </span>
            <GitCompare size={14} className="text-gray-400" />
            <span className="text-xs text-gray-500">
              {results.totalRuns} parallel runs
            </span>
          </div>
        </div>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-2 gap-3 px-4 py-3 border-b border-gray-100">
        <div className="rounded-md bg-green-50 px-3 py-2">
          <div className="text-xs text-green-600 font-medium">
            CONFIRMED
          </div>
          <div className="text-lg font-bold text-green-700">
            {confirmedCount}/{results.totalRuns} runs
          </div>
        </div>
        <div className="rounded-md bg-gray-50 px-3 py-2">
          <div className="text-xs text-gray-500 font-medium">
            UNCERTAIN
          </div>
          <div className="text-lg font-bold text-gray-600">
            {uncertainCount}/{results.totalRuns} runs
          </div>
        </div>
      </div>

      {/* Cost comparison */}
      <div className="px-4 py-2.5 border-b border-gray-100 flex items-center gap-4 text-xs">
        <div className="flex items-center gap-1.5 text-gray-600">
          <DollarSign size={12} className="text-gray-400" />
          <span>
            Total: <span className="font-mono font-medium">${results.totalCost.toFixed(4)}</span>
          </span>
        </div>
        <div className="flex items-center gap-1.5 text-gray-600">
          <span>
            vs Single:{" "}
            <span className="font-mono font-medium">${results.singleExplorerCost.toFixed(4)}</span>
          </span>
        </div>
        <div
          className={`flex items-center gap-0.5 font-medium ${
            costDelta < 0 ? "text-green-600" : "text-amber-600"
          }`}
        >
          <span>
            {costDelta >= 0 ? "+" : ""}
            {costDeltaPct}%
          </span>
        </div>
        <div className="text-gray-400">
          {results.totalTokens.toLocaleString()} total tok
        </div>
      </div>

      {/* Run sections */}
      <div className="px-4 py-3 space-y-2">
        <RunSection
          title="CONFIRMED Runs"
          runs={results.confirmedRuns}
          verdict="confirmed"
          subtitle={`${confirmedCount}/${results.totalRuns} in consensus`}
        />
        <RunSection
          title="UNCERTAIN Runs"
          runs={results.uncertainRuns}
          verdict="uncertain"
          subtitle={`${uncertainCount}/${results.totalRuns} need review`}
        />
      </div>
    </div>
  );
}
