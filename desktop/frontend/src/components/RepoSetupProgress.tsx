import { useState, useEffect } from "react";
import { Download, CheckCircle2, Loader2, Package, Terminal, AlertTriangle } from "lucide-react";

type Stage = "cloning" | "detecting" | "initializing" | "ready" | "error";

interface StageInfo {
  key: Stage;
  label: string;
  icon: typeof Download;
}

const stages: StageInfo[] = [
  { key: "cloning", label: "Cloning repository", icon: Download },
  { key: "detecting", label: "Detecting project type", icon: Package },
  { key: "initializing", label: "Running reasonix init", icon: Terminal },
  { key: "ready", label: "Ready", icon: CheckCircle2 },
];

interface Props {
  repoName: string;
  onComplete?: () => void;
}

export function RepoSetupProgress({ repoName, onComplete }: Props) {
  const [currentStage, setCurrentStage] = useState<Stage>("cloning");
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState("");
  const [projectType, setProjectType] = useState("");

  useEffect(() => {
    // Simulated progress — in production, this would subscribe to
    // clone progress events from the Go backend via Wails events.
    const timers: ReturnType<typeof setTimeout>[] = [];

    timers.push(setTimeout(() => setProgress(30), 500));
    timers.push(setTimeout(() => setProgress(60), 1200));
    timers.push(setTimeout(() => {
      setProgress(80);
      setCurrentStage("detecting");
      setProjectType("Go");
    }, 2000));
    timers.push(setTimeout(() => {
      setProgress(90);
      setCurrentStage("initializing");
    }, 3000));
    timers.push(setTimeout(() => {
      setProgress(100);
      setCurrentStage("ready");
      onComplete?.();
    }, 4000));

    return () => timers.forEach(clearTimeout);
  }, [onComplete]);

  return (
    <div className="repo-setup-progress p-6 max-w-md mx-auto" role="status" aria-label={`Setting up ${repoName}`}>
      <h2 className="text-lg font-semibold mb-1">Setting up {repoName}</h2>

      {/* Progress bar */}
      <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden mb-4 mt-3">
        <div
          className={`h-full rounded-full transition-all duration-500 ${error ? "bg-red-500" : "bg-blue-500"}`}
          style={{ width: `${progress}%` }}
          role="progressbar"
          aria-valuenow={progress}
          aria-valuemin={0}
          aria-valuemax={100}
        />
      </div>
      <p className="text-xs text-gray-400 mb-4">{progress}%</p>

      {/* Stage indicators */}
      <div className="space-y-2">
        {stages.map((stage) => {
          const stageIdx = stages.findIndex((s) => s.key === stage.key);
          const currentIdx = stages.findIndex((s) => s.key === currentStage);
          const isComplete = stageIdx < currentIdx;
          const isCurrent = stageIdx === currentIdx;
          const isError = error && isCurrent;

          return (
            <div key={stage.key} className="flex items-center gap-3">
              {isComplete ? (
                <CheckCircle2 className="w-5 h-5 text-green-500 flex-shrink-0" />
              ) : isError ? (
                <AlertTriangle className="w-5 h-5 text-red-500 flex-shrink-0" />
              ) : isCurrent ? (
                <Loader2 className="w-5 h-5 text-blue-500 animate-spin flex-shrink-0" />
              ) : (
                <div className="w-5 h-5 rounded-full border-2 border-gray-300 dark:border-gray-600 flex-shrink-0" />
              )}
              <span className={`text-sm ${isComplete ? "text-green-600" : isCurrent ? "text-blue-600 font-medium" : "text-gray-400"}`}>
                {stage.label}
              </span>
              {stage.key === "detecting" && projectType && isComplete && (
                <span className="text-xs px-1.5 py-0.5 rounded bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">
                  {projectType}
                </span>
              )}
            </div>
          );
        })}
      </div>

      {/* Error state */}
      {error && (
        <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800">
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
          <button
            onClick={() => { setError(""); setCurrentStage("cloning"); setProgress(0); }}
            className="mt-2 px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700"
          >
            Retry
          </button>
        </div>
      )}
    </div>
  );
}
