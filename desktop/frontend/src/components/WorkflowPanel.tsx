import { useState, useEffect, useCallback, useRef } from "react";
import {
  GitBranch,
  Sigma,
  BarChart4,
  Layers,
  Play,
  Keyboard,
} from "lucide-react";
import { WorkflowCard } from "./WorkflowCard";
import type { WorkflowData } from "./WorkflowCard";
import { SpeculativeResultsCard } from "./SpeculativeResultsCard";
import type { SpeculativeResults } from "./SpeculativeResultsCard";
import { VotingBreakdown } from "./VotingBreakdown";
import type { VotingData } from "./VotingBreakdown";

type PanelTab = "workflow" | "speculative" | "voting";

export interface WorkflowPanelData {
  workflow: WorkflowData | null;
  speculative: SpeculativeResults | null;
  voting: VotingData | null;
}

interface WorkflowPanelProps {
  data?: WorkflowPanelData;
  /** Polling interval in ms (default 500) */
  pollIntervalMs?: number;
  /** Called on each poll tick; the parent fetches fresh data */
  onPoll?: () => Promise<WorkflowPanelData | undefined>;
  className?: string;
}

const TABS: { id: PanelTab; label: string; icon: typeof GitBranch; shortcut: string }[] = [
  { id: "workflow", label: "Workflow", icon: Layers, shortcut: "1" },
  { id: "speculative", label: "Speculative", icon: Sigma, shortcut: "2" },
  { id: "voting", label: "Voting", icon: BarChart4, shortcut: "3" },
];

function KeyboardShortcutHint({ keys }: { keys: string[] }) {
  return (
    <span className="inline-flex items-center gap-0.5 text-[10px] text-gray-400 font-mono ml-2">
      {keys.map((k, i) => (
        <span
          key={i}
          className="inline-flex items-center justify-center px-1 py-0.5 rounded border border-gray-300 bg-gray-50 text-gray-500"
        >
          {k}
        </span>
      ))}
    </span>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
      <div className="w-12 h-12 rounded-full bg-gray-100 flex items-center justify-center mb-3">
        <Play size={20} className="text-gray-400" />
      </div>
      <div className="text-sm font-medium text-gray-500 mb-1">
        No active workflow
      </div>
      <div className="text-xs text-gray-400 max-w-xs">
        Start a multi-agent workflow with fan-out and voting to see progress,
        speculative results, and voting breakdowns here.
      </div>
      <div className="mt-3 flex items-center gap-2 text-[11px] text-gray-400">
        <Keyboard size={12} />
        <span>Press</span>
        <span className="inline-flex items-center gap-0.5 font-mono">
          <span className="px-1 py-0.5 rounded border border-gray-300 bg-gray-50">
            Ctrl
          </span>
          <span className="text-gray-400">+</span>
          <span className="px-1 py-0.5 rounded border border-gray-300 bg-gray-50">
            W
          </span>
        </span>
        <span>to open</span>
      </div>
    </div>
  );
}

export function WorkflowPanel({
  data,
  pollIntervalMs = 500,
  onPoll,
  className,
}: WorkflowPanelProps) {
  const [activeTab, setActiveTab] = useState<PanelTab>("workflow");
  const [liveData, setLiveData] = useState<WorkflowPanelData | undefined>(data);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Sync external data when prop changes
  useEffect(() => {
    if (data) setLiveData(data);
  }, [data]);

  // Polling
  useEffect(() => {
    if (!onPoll) return;

    const tick = async () => {
      try {
        const fresh = await onPoll();
        if (fresh) setLiveData((prev) => ({ ...prev, ...fresh }));
      } catch {
        // Silently ignore poll errors
      }
    };

    // Initial fetch
    tick();

    // Start interval
    pollRef.current = setInterval(tick, pollIntervalMs);
    return () => {
      if (pollRef.current) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
    };
  }, [onPoll, pollIntervalMs]);

  // Keyboard shortcuts
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Ctrl+W toggles visibility (handled by parent)
      if (e.ctrlKey && e.key === "w") {
        e.preventDefault();
        return;
      }
      // Number keys switch tabs when panel is focused
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
      const tab = TABS.find((t) => t.shortcut === e.key);
      if (tab) {
        e.preventDefault();
        setActiveTab(tab.id);
      }
    },
    []
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  const isEmpty =
    !liveData ||
    (!liveData.workflow?.active &&
      !liveData.speculative?.totalRuns &&
      (!liveData.voting || liveData.voting.totalVotes === 0));

  return (
    <div
      className={`rounded-lg border border-gray-200 bg-white shadow-sm flex flex-col${
        className ? ` ${className}` : ""
      }`}
    >
      {/* Tabs */}
      <div className="flex border-b border-gray-200 bg-gray-50 rounded-t-lg">
        {TABS.map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              type="button"
              className={`flex items-center gap-1.5 px-3 py-2 text-xs font-medium transition-colors border-b-2 -mb-px${
                isActive
                  ? "border-indigo-500 text-indigo-700 bg-white"
                  : "border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-100"
              }`}
              onClick={() => setActiveTab(tab.id)}
            >
              <Icon size={13} />
              {tab.label}
              <KeyboardShortcutHint keys={[tab.shortcut]} />
            </button>
          );
        })}

        {/* Refresh hint */}
        {onPoll && (
          <div className="ml-auto flex items-center gap-1 px-3 text-[10px] text-gray-400">
            <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
            Auto-refresh
          </div>
        )}
      </div>

      {/* Body */}
      <div className="flex-1 overflow-auto">
        {isEmpty ? (
          <EmptyState />
        ) : (
          <>
            {activeTab === "workflow" && liveData?.workflow && (
              <WorkflowCard workflow={liveData.workflow} />
            )}
            {activeTab === "speculative" && liveData?.speculative && (
              <SpeculativeResultsCard results={liveData.speculative} />
            )}
            {activeTab === "voting" && liveData?.voting && (
              <VotingBreakdown data={liveData.voting} />
            )}
            {activeTab === "workflow" && !liveData?.workflow?.active && (
              <div className="text-xs text-gray-400 text-center py-6">
                No workflow data available
              </div>
            )}
            {activeTab === "speculative" && !liveData?.speculative?.totalRuns && (
              <div className="text-xs text-gray-400 text-center py-6">
                No speculative results yet
              </div>
            )}
            {activeTab === "voting" &&
              (!liveData?.voting || liveData.voting.totalVotes === 0) && (
                <div className="text-xs text-gray-400 text-center py-6">
                  No voting data available
                </div>
              )}
          </>
        )}
      </div>
    </div>
  );
}
