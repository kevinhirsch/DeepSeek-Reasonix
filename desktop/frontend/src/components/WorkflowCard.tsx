import { useState, useCallback } from "react";
import {
  BarChart3,
  GitBranch,
  Layers,
  CheckCircle2,
  Loader2,
  ChevronDown,
  ChevronRight,
  XCircle,
  Clock,
} from "lucide-react";

export type StageStatus = "running" | "completed" | "pending" | "failed";
export type VotingStrategy = "majority" | "unanimous" | "weighted";

export interface WorkflowStage {
  id: string;
  label: string;
  status: StageStatus;
  progress: number; // 0-100
  subagentCount: number;
  estimatedTokens: number;
  fanOut: number;
  votingStrategy: VotingStrategy;
}

export interface WorkflowData {
  active: boolean;
  name: string;
  stages: WorkflowStage[];
  totalSubagents: number;
  totalEstimatedTokens: number;
}

function stageStatusIcon(status: StageStatus) {
  switch (status) {
    case "running":
      return <Loader2 size={12} className="animate-spin text-blue-500" />;
    case "completed":
      return <CheckCircle2 size={12} className="text-green-500" />;
    case "failed":
      return <XCircle size={12} className="text-red-500" />;
    default:
      return <Clock size={12} className="text-gray-400" />;
  }
}

function stageStatusLabel(status: StageStatus): string {
  switch (status) {
    case "running":
      return "Running";
    case "completed":
      return "Completed";
    case "failed":
      return "Failed";
    default:
      return "Pending";
  }
}

function votingLabel(strategy: VotingStrategy): string {
  switch (strategy) {
    case "majority":
      return "Majority";
    case "unanimous":
      return "Unanimous";
    case "weighted":
      return "Weighted";
  }
}

interface MiniBarChartProps {
  stages: WorkflowStage[];
}

function MiniBarChart({ stages }: MiniBarChartProps) {
  const maxSubagents = Math.max(1, ...stages.map((s) => s.subagentCount));
  return (
    <div className="flex items-end gap-0.5 h-10">
      {stages.map((stage) => {
        const heightPct = (stage.subagentCount / maxSubagents) * 100;
        let bg = "bg-gray-300";
        if (stage.status === "completed") bg = "bg-green-400";
        else if (stage.status === "running") bg = "bg-blue-400";
        else if (stage.status === "failed") bg = "bg-red-400";
        return (
          <div
            key={stage.id}
            className="flex flex-col items-center gap-0.5"
            title={`${stage.label}: ${stage.subagentCount} subagents, ${stageStatusLabel(stage.status)}`}
          >
            <div
              className={`w-4 ${bg} rounded-t-sm transition-all duration-300`}
              style={{ height: `${Math.max(8, heightPct)}%` }}
            />
          </div>
        );
      })}
    </div>
  );
}

export function WorkflowCard({
  workflow,
  defaultOpen = false,
  open: controlledOpen,
  onOpenChange,
  className,
}: {
  workflow: WorkflowData;
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  className?: string;
}) {
  const [internalOpen, setInternalOpen] = useState(defaultOpen);
  const actualOpen = controlledOpen ?? internalOpen;

  const completedCount = workflow.stages.filter(
    (s) => s.status === "completed"
  ).length;
  const runningCount = workflow.stages.filter(
    (s) => s.status === "running"
  ).length;
  const failedCount = workflow.stages.filter(
    (s) => s.status === "failed"
  ).length;

  const toggle = useCallback(() => {
    const next = !actualOpen;
    if (controlledOpen === undefined) setInternalOpen(next);
    onOpenChange?.(next);
  }, [actualOpen, controlledOpen, onOpenChange]);

  return (
    <div
      className={`rounded-lg border border-gray-200 bg-white shadow-sm${
        className ? ` ${className}` : ""
      }`}
    >
      {/* Header */}
      <button
        type="button"
        className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-gray-50 transition-colors rounded-lg"
        onClick={toggle}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            e.preventDefault();
            toggle();
          }
        }}
        aria-expanded={actualOpen}
      >
        {/* Icon */}
        <div className="flex-shrink-0 w-8 h-8 rounded-md bg-indigo-50 flex items-center justify-center">
          <GitBranch size={16} className="text-indigo-600" />
        </div>

        {/* Name + summary */}
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm text-gray-900 truncate">
            {workflow.name}
          </div>
          <div className="flex items-center gap-2 mt-0.5 text-xs text-gray-500">
            <span className="flex items-center gap-1">
              <Layers size={10} />
              {workflow.stages.length} stages
            </span>
            <span>
              {completedCount} done / {runningCount} running
              {failedCount > 0 ? ` / ${failedCount} failed` : ""}
            </span>
          </div>
        </div>

        {/* Mini bar chart */}
        <div className="flex-shrink-0 hidden sm:block">
          <MiniBarChart stages={workflow.stages} />
        </div>

        {/* Agent & token totals */}
        <div className="flex-shrink-0 flex items-center gap-3 text-xs text-gray-500">
          <span className="flex items-center gap-1" title="Total subagents">
            <BarChart3 size={12} />
            {workflow.totalSubagents}
          </span>
          <span className="flex items-center gap-1" title="Estimated tokens">
            <span className="font-mono text-[10px]">T</span>
            {workflow.totalEstimatedTokens.toLocaleString()}
          </span>
        </div>

        {/* Chevron */}
        <div className="flex-shrink-0 text-gray-400">
          {actualOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </div>
      </button>

      {/* Expandable body */}
      {actualOpen && (
        <div className="border-t border-gray-100 px-4 py-3">
          <div className="space-y-2">
            {workflow.stages.map((stage) => (
              <div
                key={stage.id}
                className={`rounded-md border px-3 py-2 ${
                  stage.status === "running"
                    ? "border-blue-200 bg-blue-50/50"
                    : stage.status === "completed"
                    ? "border-green-200 bg-green-50/50"
                    : stage.status === "failed"
                    ? "border-red-200 bg-red-50/50"
                    : "border-gray-200 bg-gray-50/50"
                }`}
              >
                {/* Stage header row */}
                <div className="flex items-center gap-2">
                  {stageStatusIcon(stage.status)}
                  <span className="text-sm font-medium text-gray-800 flex-1">
                    {stage.label}
                  </span>
                  <span className="text-xs text-gray-500">
                    {stage.subagentCount} agents
                  </span>
                  <span className="text-xs text-gray-400 font-mono">
                    ~{stage.estimatedTokens.toLocaleString()} tok
                  </span>
                </div>

                {/* Progress bar */}
                {stage.status !== "pending" && (
                  <div className="mt-1.5 h-1.5 rounded-full bg-gray-200 overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-500 ${
                        stage.status === "completed"
                          ? "bg-green-500"
                          : stage.status === "failed"
                          ? "bg-red-500"
                          : "bg-blue-500"
                      }`}
                      style={{ width: `${Math.max(4, stage.progress)}%` }}
                    />
                  </div>
                )}

                {/* Fan-out + voting config */}
                <div className="flex items-center gap-3 mt-1.5 text-xs text-gray-500">
                  <span className="flex items-center gap-1">
                    <GitBranch size={10} />
                    Fan-out: {stage.fanOut}x
                  </span>
                  <span className="flex items-center gap-1">
                    <Layers size={10} />
                    Vote: {votingLabel(stage.votingStrategy)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
