import { useCallback, useEffect, useRef, useState } from "react";
import { Download, CheckCircle2, Loader2, Package, Terminal, AlertTriangle, RefreshCw } from "lucide-react";
import { useT } from "../lib/i18n";
import { app } from "../lib/bridge";

// ── Types ──

/** Stage names match the "stage" field emitted from Go's RepoCloneProgress. */
type Stage = "cloning" | "detecting" | "init" | "ready" | "error";

interface StageDef {
  key: Stage;
  labelKey: string;   // i18n key
  icon: typeof Download;
}

/** Progress event shape — must match Go-side RepoCloneProgress. */
interface CloneProgressEvent {
  stage: Stage;
  progress: number;  // 0.0 – 1.0
  message: string;
  repoUrl: string;
  dir: string;
}

// ── Stage definitions ──

const STAGES: StageDef[] = [
  { key: "cloning",   labelKey: "repoSetup.stageCloning",   icon: Download },
  { key: "detecting", labelKey: "repoSetup.stageDetecting", icon: Package },
  { key: "init",      labelKey: "repoSetup.stageInit",      icon: Terminal },
  { key: "ready",     labelKey: "repoSetup.stageReady",     icon: CheckCircle2 },
];

const PROGRESS_CHANNEL = "repo:clone-progress";

// ── Helpers ──

function stageIndex(key: Stage): number {
  return STAGES.findIndex((s) => s.key === key);
}

/** Subscribe to clone progress events from the Wails runtime. */
function subscribeProgress(cb: (e: CloneProgressEvent) => void): () => void {
  if (typeof window !== "undefined" && window.runtime?.EventsOn) {
    return window.runtime.EventsOn(PROGRESS_CHANNEL, (payload) => {
      try {
        const evt = (typeof payload === "string" ? JSON.parse(payload) : payload) as CloneProgressEvent;
        cb(evt);
      } catch {
        /* ignore malformed events */
      }
    });
  }
  return () => {};
}

// ── Component ──

export function RepoSetupProgress({
  repoUrl,
  repoName,
  branch,
  targetDir,
  onComplete,
  onError,
}: {
  repoUrl: string;
  repoName: string;
  branch?: string;
  targetDir?: string;
  onComplete?: (dir: string) => void;
  onError?: (err: string) => void;
}) {
  const t = useT();

  const [activeStage, setActiveStage] = useState<Stage>("cloning");
  const [_progress, setProgress] = useState(0);
  // Percentage display value (0-100) derived from the 0-1 progress float.
  const [pct, setPct] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [projectType, setProjectType] = useState("");
  const [cloneDir, setCloneDir] = useState("");

  const mountedRef = useRef(true);
  const startedRef = useRef(false);

  useEffect(() => () => { mountedRef.current = false; }, []);

  // ── Run the clone ──

  const runClone = useCallback(async () => {
    if (startedRef.current) return;
    startedRef.current = true;

    setActiveStage("cloning");
    setProgress(0);
    setPct(0);
    setError(null);

    // Start listening for progress events before kicking off the clone,
    // so we don't miss the first emission.
    const unsub = subscribeProgress((evt) => {
      if (!mountedRef.current) return;
      setActiveStage(evt.stage);
      setProgress(evt.progress);
      setPct(Math.round(evt.progress * 100));
      if (evt.message && evt.stage === "detecting") {
        setProjectType(evt.message);
      }
      if (evt.dir) setCloneDir(evt.dir);
    });

    try {
      const result = await app.CloneRepo(repoUrl, branch || "", targetDir || "");
      unsub();

      if (!mountedRef.current) return;

      if (result.error) {
        setActiveStage("error");
        setError(result.error);
        onError?.(result.error);
      } else {
        setActiveStage("ready");
        setProgress(1);
        setPct(100);
        onComplete?.(result.dir);
      }
    } catch (err) {
      unsub();
      if (!mountedRef.current) return;
      const msg = String(err);
      setActiveStage("error");
      setError(msg);
      onError?.(msg);
    }
  }, [repoUrl, branch, targetDir, onComplete, onError]);

  useEffect(() => {
    void runClone();
  }, [runClone]);

  // ── Retry ──

  const handleRetry = () => {
    startedRef.current = false;
    void runClone();
  };

  // ── Render ──

  const currentIdx = stageIndex(activeStage);

  return (
    <div
      className="repo-setup"
      role="status"
      aria-label={t("repoSetup.settingUp", { name: repoName })}
    >
      <h2 className="repo-setup__title">
        {t("repoSetup.settingUp", { name: repoName })}
      </h2>

      {/* Progress bar */}
      <div className="repo-setup__bar" aria-hidden="true">
        <div
          className={`repo-setup__bar-fill${error ? " repo-setup__bar-fill--err" : ""}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className="repo-setup__pct" aria-live="polite">
        {pct}%
      </p>

      {/* Stage indicators */}
      <div className="repo-setup__stages">
        {STAGES.map((stage) => {
          const idx = stageIndex(stage.key);
          const isComplete = currentIdx >= 0 && idx < currentIdx;
          const isCurrent = idx === currentIdx && !error;
          const isFailed = error && idx === currentIdx;

          let statusClass = "repo-setup__stage--pending";
          if (isComplete) statusClass = "repo-setup__stage--done";
          else if (isFailed) statusClass = "repo-setup__stage--failed";
          else if (isCurrent) statusClass = "repo-setup__stage--current";

          return (
            <div key={stage.key} className={`repo-setup__stage ${statusClass}`}>
              <span className="repo-setup__stage-icon">
                {isComplete ? (
                  <CheckCircle2 size={18} aria-hidden="true" />
                ) : isFailed ? (
                  <AlertTriangle size={18} aria-hidden="true" />
                ) : isCurrent ? (
                  <Loader2 size={18} className="repo-setup__spin" aria-hidden="true" />
                ) : (
                  <stage.icon size={18} aria-hidden="true" />
                )}
              </span>
              <span className="repo-setup__stage-label">
                {t(stage.labelKey as any)}
              </span>
              {stage.key === "detecting" && projectType && isComplete && (
                <span className="repo-setup__stage-tag">{projectType}</span>
              )}
            </div>
          );
        })}
      </div>

      {/* Error state */}
      {error && (
        <div className="repo-setup__error" role="alert">
          <p className="repo-setup__error-msg">{error}</p>
          <button
            className="repo-setup__retry"
            onClick={handleRetry}
            type="button"
          >
            <RefreshCw size={14} aria-hidden="true" />
            <span>{t("repoSetup.retry")}</span>
          </button>
        </div>
      )}

      {/* Clone directory (informational, shown when ready) */}
      {cloneDir && activeStage === "ready" && (
        <p className="repo-setup__dest">
          {t("repoSetup.clonedTo")} {cloneDir}
        </p>
      )}
    </div>
  );
}
