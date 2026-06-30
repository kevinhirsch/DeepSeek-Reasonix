import { useEffect, useState, useRef } from "react";
import { BarChart4, Percent, Minus } from "lucide-react";

export interface VoteEntry {
  verdict: "confirmed" | "plausible" | "refuted" | "uncertain";
  label: string;
  count: number;
  totalVotes?: number;
}

export interface VotingData {
  votes: VoteEntry[];
  totalVotes: number;
  threshold: number; // e.g. 50 for majority, 100 for unanimous
  thresholdType: "majority" | "unanimous";
}

function verdictBarColor(verdict: VoteEntry["verdict"]): string {
  switch (verdict) {
    case "confirmed":
      return "bg-green-500";
    case "plausible":
      return "bg-yellow-500";
    case "refuted":
      return "bg-red-500";
    case "uncertain":
      return "bg-gray-400";
  }
}

function verdictDotColor(verdict: VoteEntry["verdict"]): string {
  switch (verdict) {
    case "confirmed":
      return "bg-green-500";
    case "plausible":
      return "bg-yellow-500";
    case "refuted":
      return "bg-red-500";
    case "uncertain":
      return "bg-gray-400";
  }
}

function useAnimatedWidth(targetPct: number, duration = 600) {
  const [width, setWidth] = useState(0);
  const rafRef = useRef<number>(0);
  const startedRef = useRef(false);

  useEffect(() => {
    if (targetPct <= 0) {
      setWidth(0);
      startedRef.current = false;
      return;
    }
    // Reset animation when target changes
    const startTime = performance.now();
    const from = 0;
    const to = targetPct;
    startedRef.current = true;

    function frame(now: number) {
      const elapsed = now - startTime;
      const progress = Math.min(1, elapsed / duration);
      // Ease-out quad
      const eased = 1 - (1 - progress) * (1 - progress);
      setWidth(from + (to - from) * eased);
      if (progress < 1) {
        rafRef.current = requestAnimationFrame(frame);
      }
    }

    rafRef.current = requestAnimationFrame(frame);
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [targetPct, duration]);

  return width;
}

function VoteBar({
  entry,
  maxPct,
  thresholdPct,
}: {
  entry: VoteEntry;
  maxPct: number;
  thresholdPct: number;
}) {
  const pct = entry.totalVotes > 0 ? (entry.count / entry.totalVotes) * 100 : 0;
  const displayPct = pct.toFixed(1);
  const animatedPct = useAnimatedWidth(pct);
  const normalizedMax = Math.max(maxPct, 1);

  return (
    <div className="flex items-center gap-3">
      {/* Label */}
      <div className="flex items-center gap-1.5 w-28 flex-shrink-0">
        <div
          className={`w-2 h-2 rounded-full flex-shrink-0 ${verdictDotColor(
            entry.verdict
          )}`}
        />
        <span className="text-xs font-medium text-gray-700 truncate">
          {entry.label}
        </span>
      </div>

      {/* Bar track */}
      <div className="flex-1 relative h-6 rounded-md bg-gray-100 overflow-hidden">
        {/* Animated fill */}
        <div
          className={`h-full rounded-md transition-colors duration-300 ${verdictBarColor(
            entry.verdict
          )}`}
          style={{ width: `${(animatedPct / normalizedMax) * 100}%` }}
        />

        {/* Threshold marker */}
        <div
          className="absolute top-0 bottom-0 w-0.5 bg-red-400 z-10"
          style={{ left: `${(thresholdPct / normalizedMax) * 100}%` }}
          title={`${entry.totalVotes > 0 ? entry.totalVotes : 0}%`}
        />

        {/* Count label inside bar */}
        <span className="absolute inset-0 flex items-center px-2 text-xs font-medium text-gray-800">
          {entry.count} votes
        </span>
      </div>

      {/* Percentage */}
      <div className="w-14 flex-shrink-0 text-right flex items-center gap-1">
        <Percent size={10} className="text-gray-400" />
        <span className="text-xs font-mono text-gray-600">{displayPct}%</span>
      </div>
    </div>
  );
}

export function VotingBreakdown({
  data,
  className,
}: {
  data: VotingData;
  className?: string;
}) {
  const maxPct = Math.max(
    ...data.votes.map((v) =>
      data.totalVotes > 0 ? (v.count / data.totalVotes) * 100 : 0
    ),
    1
  );

  const thresholdPct = data.totalVotes > 0
    ? (data.threshold / data.totalVotes) * 100
    : data.threshold;

  const thresholdPassed = data.votes.some(
    (v) =>
      data.totalVotes > 0 &&
      (v.count / data.totalVotes) * 100 >= thresholdPct
  );

  return (
    <div
      className={`rounded-lg border border-gray-200 bg-white shadow-sm${
        className ? ` ${className}` : ""
      }`}
    >
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3 border-b border-gray-100">
        <div className="w-8 h-8 rounded-md bg-purple-50 flex items-center justify-center">
          <BarChart4 size={16} className="text-purple-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm text-gray-900">
            Voting Breakdown
          </div>
          <div className="text-xs text-gray-500">
            {data.totalVotes} total votes &middot; {data.thresholdType} threshold
          </div>
        </div>
      </div>

      {/* Threshold line */}
      <div className="px-4 pt-3 pb-1 flex items-center gap-2">
        <Minus size={12} className="text-red-400" />
        <span className="text-xs text-gray-500">
          Threshold: {data.threshold} votes ({thresholdPct.toFixed(0)}%) &mdash;{" "}
          {data.thresholdType}
        </span>
        {thresholdPassed && (
          <span className="text-[10px] font-semibold text-green-600 bg-green-100 px-1.5 py-0.5 rounded-full">
            PASSED
          </span>
        )}
        {!thresholdPassed && data.votes.length > 0 && (
          <span className="text-[10px] font-semibold text-amber-600 bg-amber-100 px-1.5 py-0.5 rounded-full">
            NOT MET
          </span>
        )}
      </div>

      {/* Vote bars */}
      <div className="px-4 py-3 space-y-3">
        {data.votes.length === 0 ? (
          <div className="text-xs text-gray-400 text-center py-4">
            No votes recorded
          </div>
        ) : (
          data.votes.map((entry) => (
            <VoteBar
              key={entry.verdict}
              entry={{ ...entry, totalVotes: data.totalVotes }}
              maxPct={maxPct}
              thresholdPct={thresholdPct}
            />
          ))
        )}
      </div>

      {/* Legend */}
      {data.votes.length > 0 && (
        <div className="border-t border-gray-100 px-4 py-2 flex items-center gap-4 flex-wrap">
          {data.votes.map((entry) => (
            <div
              key={entry.verdict}
              className="flex items-center gap-1 text-xs text-gray-500"
            >
              <div
                className={`w-2.5 h-2.5 rounded-sm ${verdictBarColor(
                  entry.verdict
                )}`}
              />
              {entry.label}
            </div>
          ))}
          <div className="flex items-center gap-1 text-xs text-gray-400 ml-auto">
            <div className="w-2.5 h-2.5 border-l border-red-400" />
            threshold
          </div>
        </div>
      )}
    </div>
  );
}
