import {
  memo,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useReducer,
  useRef,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";
import {
  Activity,
  CheckCircle2,
  ChevronRight,
  Clock,
  Globe,
  Loader2,
  MessageSquare,
  X,
  XCircle,
} from "lucide-react";
import { app } from "../lib/bridge";
import {
  a11yId,
  announceToScreenReader,
  formatShortcutForScreenReader,
  taskRowLabel,
  useHighContrast,
  usePrefersReducedMotion,
  type ShortcutDef,
} from "../lib/accessibility";
import type { JobView } from "../lib/types";

// ── Extended task model ─────────────────────────────────────────────────────

export type TaskStatus = "pending" | "waiting" | "running" | "done" | "failed";

export interface BackgroundTaskView extends JobView {
  taskStatus: TaskStatus;
  toolCount: number;
  lastTool: string;
  reasoningTail: string;
  remote: boolean;
  remoteWorker?: string;
  completedAt?: number;
}

// ── Keyboard hint definitions ───────────────────────────────────────────────

const PEEK_SHORTCUT: ShortcutDef = { key: "Enter" };
const SEND_SHORTCUT: ShortcutDef = { key: "s" };
const KILL_SHORTCUT: ShortcutDef = { key: "k" };

// ── State ───────────────────────────────────────────────────────────────────

interface TasksState {
  tasks: BackgroundTaskView[];
  loading: boolean;
  error: string | null;
  focusedIndex: number;
  peekId: string | null;
  visible: boolean;
}

type TasksAction =
  | { type: "set_tasks"; tasks: BackgroundTaskView[] }
  | { type: "set_loading"; loading: boolean }
  | { type: "set_error"; error: string | null }
  | { type: "focus_index"; index: number }
  | { type: "focus_next" }
  | { type: "focus_prev" }
  | { type: "set_peek"; id: string | null }
  | { type: "toggle_peek"; id: string }
  | { type: "set_visible"; visible: boolean };

function tasksReducer(state: TasksState, action: TasksAction): TasksState {
  switch (action.type) {
    case "set_tasks": {
      const hasActive = action.tasks.some(
        (t) => t.taskStatus === "running" || t.taskStatus === "waiting",
      );
      return {
        ...state,
        tasks: action.tasks,
        visible: hasActive || state.peekId !== null,
        focusedIndex:
          state.focusedIndex >= action.tasks.length
            ? Math.max(0, action.tasks.length - 1)
            : state.focusedIndex,
      };
    }
    case "set_loading":
      return { ...state, loading: action.loading };
    case "set_error":
      return { ...state, error: action.error };
    case "focus_index":
      return {
        ...state,
        focusedIndex: Math.max(0, Math.min(action.index, state.tasks.length - 1)),
      };
    case "focus_next":
      return {
        ...state,
        focusedIndex: Math.min(state.focusedIndex + 1, state.tasks.length - 1),
      };
    case "focus_prev":
      return { ...state, focusedIndex: Math.max(state.focusedIndex - 1, 0) };
    case "set_peek":
      return { ...state, peekId: action.id };
    case "toggle_peek":
      return { ...state, peekId: state.peekId === action.id ? null : action.id };
    case "set_visible":
      return { ...state, visible: action.visible };
    default:
      return state;
  }
}

// ── Helpers ─────────────────────────────────────────────────────────────────

const STATUS_ICONS: Record<TaskStatus, { icon: string; label: string }> = {
  pending: { icon: "○", label: "pending" },
  waiting: { icon: "◐", label: "waiting" },
  running: { icon: "●", label: "running" },
  done: { icon: "✓", label: "done" },
  failed: { icon: "✗", label: "failed" },
};

function statusDotClass(status: TaskStatus): string {
  switch (status) {
    case "running":
      return "bg-tasks__status-dot--running";
    case "waiting":
      return "bg-tasks__status-dot--waiting";
    case "done":
      return "bg-tasks__status-dot--done";
    case "failed":
      return "bg-tasks__status-dot--failed";
    default:
      return "bg-tasks__status-dot--pending";
  }
}

function statusAriaLabel(status: TaskStatus, remote: boolean): string {
  const base = STATUS_ICONS[status].label;
  return remote ? `${base} (remote)` : base;
}

function formatTimeAgo(ms: number): string {
  const delta = Date.now() - ms;
  if (delta < 60_000) return `${Math.floor(delta / 1000)}s`;
  if (delta < 3_600_000) return `${Math.floor(delta / 60_000)}m`;
  if (delta < 86_400_000) return `${Math.floor(delta / 3_600_000)}h`;
  return `${Math.floor(delta / 86_400_000)}d`;
}

function truncateTail(tail: string, maxLen: number = 120): string {
  if (tail.length <= maxLen) return tail;
  return tail.slice(-maxLen).replace(/^\S+/, (m) => `...${m}`);
}

// ── Component ────────────────────────────────────────────────────────────────

export interface BackgroundTasksPanelProps {
  onClose?: () => void;
  compact?: boolean;
}

export const BackgroundTasksPanel = memo(function BackgroundTasksPanel({
  onClose,
  compact = false,
}: BackgroundTasksPanelProps) {
  const [state, dispatch] = useReducer(tasksReducer, {
    tasks: [],
    loading: true,
    error: null,
    focusedIndex: 0,
    peekId: null,
    visible: true,
  });

  const panelId = useId();
  const listRef = useRef<HTMLDivElement>(null);
  const reducedMotion = usePrefersReducedMotion();
  const highContrast = useHighContrast();
  const autoHideTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const liveRegionId = useMemo(() => a11yId("tasks-live"), []);

  // ── Screen reader shortcut labels ────────────────────────────────────────

  const srEnterLabel = useMemo(
    () => formatShortcutForScreenReader(PEEK_SHORTCUT),
    [],
  );
  const srSLabel = useMemo(
    () => formatShortcutForScreenReader(SEND_SHORTCUT),
    [],
  );
  const srKLabel = useMemo(
    () => formatShortcutForScreenReader(KILL_SHORTCUT),
    [],
  );

  // ── Poll tasks at 500ms via bridge Jobs() ─────────────────────────────────

  const loadTasks = useCallback(async () => {
    try {
      const jobs: JobView[] = await app.Jobs();

      // Merge with local state to preserve reasoning tail and tool counts
      // that aren't yet in the Go-side JobView wire type.
      const tasks: BackgroundTaskView[] = jobs.map((job) => {
        const existing = state.tasks.find((t) => t.id === job.id);
        if (existing) {
          return {
            ...existing,
            ...job,
            taskStatus: existing.taskStatus,
            toolCount: existing.toolCount,
            lastTool: existing.lastTool,
            reasoningTail: existing.reasoningTail,
            remote: existing.remote,
            remoteWorker: existing.remoteWorker,
            completedAt: existing.completedAt,
          };
        }
        return {
          ...job,
          taskStatus: "running" as TaskStatus,
          toolCount: 0,
          lastTool: "",
          reasoningTail: "",
          remote: false,
        };
      });

      dispatch({ type: "set_tasks", tasks });
      dispatch({ type: "set_error", error: null });
    } catch (err) {
      dispatch({
        type: "set_error",
        error: String((err as Error)?.message ?? err),
      });
    } finally {
      dispatch({ type: "set_loading", loading: false });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    void loadTasks();
    const pollId = setInterval(() => void loadTasks(), 500);
    return () => clearInterval(pollId);
  }, [loadTasks]);

  // ── Auto-hide when empty ──────────────────────────────────────────────────

  const hasActiveTasks = useMemo(
    () =>
      state.tasks.some(
        (t) => t.taskStatus === "running" || t.taskStatus === "waiting",
      ),
    [state.tasks],
  );

  useEffect(() => {
    if (!hasActiveTasks && state.peekId === null && state.visible) {
      autoHideTimer.current = setTimeout(() => {
        dispatch({ type: "set_visible", visible: false });
        announceToScreenReader(liveRegionId, "All background tasks completed");
      }, 1500);
    }
    return () => {
      if (autoHideTimer.current) clearTimeout(autoHideTimer.current);
    };
  }, [hasActiveTasks, state.peekId, state.visible, liveRegionId]);

  // Re-show when tasks become active again
  useEffect(() => {
    if (hasActiveTasks && !state.visible) {
      dispatch({ type: "set_visible", visible: true });
      announceToScreenReader(
        liveRegionId,
        `${state.tasks.filter((t) => t.taskStatus === "running").length} background tasks running`,
      );
    }
  }, [hasActiveTasks, state.visible, state.tasks, liveRegionId]);

  // ── Auto-collapse completed tasks after 3s ───────────────────────────────

  const completedRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    for (const task of state.tasks) {
      if (
        (task.taskStatus === "done" || task.taskStatus === "failed") &&
        !completedRef.current.has(task.id)
      ) {
        completedRef.current.add(task.id);
        setTimeout(() => {
          // The marker lets CSS handle the collapse transition.
          // After 3s the row receives bg-tasks__row--collapsed.
        }, 3000);
        // Timer cleanup handled per-effect; ref tracking prevents duplicates.
      }
    }
  }, [state.tasks]);

  // ── Keyboard navigation ──────────────────────────────────────────────────

  const handleKeyDown = useCallback(
    (e: ReactKeyboardEvent) => {
      const { key } = e;

      if (key === "Escape") {
        e.preventDefault();
        if (state.peekId !== null) {
          dispatch({ type: "set_peek", id: null });
        } else {
          onClose?.();
        }
        return;
      }

      if (key === "j" || key === "ArrowDown") {
        e.preventDefault();
        dispatch({ type: "focus_next" });
        return;
      }
      if (key === "k" || key === "ArrowUp") {
        e.preventDefault();
        dispatch({ type: "focus_prev" });
        return;
      }

      if (key === "Enter" && state.focusedIndex >= 0) {
        e.preventDefault();
        const task = state.tasks[state.focusedIndex];
        if (task) dispatch({ type: "toggle_peek", id: task.id });
        return;
      }

      if (key === "s" && state.focusedIndex >= 0) {
        e.preventDefault();
        // Future: open send-to-subagent composing mode
        return;
      }

      if (key === "k" && state.focusedIndex >= 0) {
        const task = state.tasks[state.focusedIndex];
        if (task && task.taskStatus === "running") {
          e.preventDefault();
          // Future: app.CancelJob(task.id)
        }
        return;
      }
    },
    [state.focusedIndex, state.peekId, state.tasks, onClose],
  );

  // Scroll focused row into view
  useEffect(() => {
    if (!listRef.current || state.focusedIndex < 0) return;
    const rows = listRef.current.querySelectorAll(".bg-tasks__row");
    const row = rows[state.focusedIndex] as HTMLElement | undefined;
    row?.scrollIntoView({ block: "nearest" });
  }, [state.focusedIndex]);

  // ── Render ────────────────────────────────────────────────────────────────

  if (!state.visible && !compact) return null;

  const activeCount = state.tasks.filter(
    (t) => t.taskStatus === "running" || t.taskStatus === "waiting",
  ).length;
  const doneCount = state.tasks.filter((t) => t.taskStatus === "done").length;
  const failedCount = state.tasks.filter((t) => t.taskStatus === "failed").length;

  return (
    <div
      className={`bg-tasks${compact ? " bg-tasks--compact" : ""}${
        !state.visible ? " bg-tasks--hidden" : ""
      }`}
      role="region"
      aria-label={`Background tasks — ${activeCount} active`}
      aria-live="polite"
      onKeyDown={handleKeyDown}
      id={panelId}
    >
      {/* Hidden live region for screen reader announcements */}
      <span id={liveRegionId} className="sr-only" aria-live="assertive" />

      {/* Header */}
      <div className="bg-tasks__header">
        <h3 className="bg-tasks__title">
          <Activity size={14} aria-hidden="true" />
          <span>Background Tasks</span>
          {activeCount > 0 && (
            <span className="bg-tasks__badge">{activeCount}</span>
          )}
        </h3>
        <div className="bg-tasks__header-meta">
          <span className="bg-tasks__stats">
            <span className="bg-tasks__stat bg-tasks__stat--done">
              <CheckCircle2 size={11} aria-hidden="true" />
              {doneCount}
            </span>
            <span className="bg-tasks__stat bg-tasks__stat--failed">
              <XCircle size={11} aria-hidden="true" />
              {failedCount}
            </span>
          </span>
          {onClose && (
            <button
              className="bg-tasks__close-btn"
              onClick={onClose}
              aria-label="Close background tasks panel"
            >
              <X size={14} aria-hidden="true" />
            </button>
          )}
        </div>
      </div>

      {/* Error banner */}
      {state.error && (
        <div className="bg-tasks__error" role="alert">
          <span>{state.error}</span>
          <button
            className="bg-tasks__dismiss-btn"
            onClick={() => dispatch({ type: "set_error", error: null })}
            aria-label="Dismiss error"
          >
            <X size={12} aria-hidden="true" />
          </button>
        </div>
      )}

      {/* Keyboard hints */}
      <div className="bg-tasks__hints" aria-hidden="true">
        <span className="bg-tasks__key-hint">j/k</span>
        <span>navigate</span>
        <span className="bg-tasks__key-hint">{PEEK_SHORTCUT.key}</span>
        <span>peek</span>
        <span className="bg-tasks__key-hint">{SEND_SHORTCUT.key}</span>
        <span>send</span>
        <span className="bg-tasks__key-hint">{KILL_SHORTCUT.key}</span>
        <span>kill</span>
        <span className="bg-tasks__key-hint">Esc</span>
        <span>close</span>
      </div>

      {/* Task list */}
      <div
        className="bg-tasks__list"
        role="list"
        aria-label="Background task list"
        ref={listRef}
      >
        {state.loading && state.tasks.length === 0 && (
          <div className="bg-tasks__empty">
            <Loader2
              size={16}
              className={`bg-tasks__spin${!reducedMotion ? "" : " bg-tasks__spin--paused"}`}
              aria-hidden="true"
            />
            <span>Watching for tasks...</span>
          </div>
        )}

        {!state.loading && state.tasks.length === 0 && (
          <div className="bg-tasks__empty">
            <Activity size={20} aria-hidden="true" />
            <span>No background tasks running.</span>
          </div>
        )}

        {state.tasks.map((task, idx) => {
          const isFocused = idx === state.focusedIndex;
          const isPeeked = state.peekId === task.id;
          const isTerminal =
            task.taskStatus === "done" || task.taskStatus === "failed";
          const completedAgo =
            task.completedAt && isTerminal
              ? formatTimeAgo(task.completedAt)
              : null;
          const label = taskRowLabel({
            label: task.label,
            status: statusAriaLabel(task.taskStatus, task.remote),
            toolCount: task.toolCount,
            latestTool: task.lastTool,
          });

          return (
            <div
              key={task.id}
              className={`bg-tasks__row${
                isFocused ? " bg-tasks__row--focused" : ""
              }${isTerminal ? " bg-tasks__row--terminal" : ""}${
                completedAgo ? " bg-tasks__row--collapsed" : ""
              }${isPeeked ? " bg-tasks__row--peeked" : ""}${
                highContrast ? " bg-tasks__row--hc" : ""
              }`}
              role="listitem"
              aria-label={label}
              aria-selected={isFocused}
              tabIndex={0}
              onFocus={() => dispatch({ type: "focus_index", index: idx })}
              onClick={() => dispatch({ type: "focus_index", index: idx })}
              onDoubleClick={() => dispatch({ type: "toggle_peek", id: task.id })}
            >
              {/* Status indicator */}
              <span
                className={`bg-tasks__status-dot ${statusDotClass(task.taskStatus)}`}
                aria-label={statusAriaLabel(task.taskStatus, task.remote)}
                title={statusAriaLabel(task.taskStatus, task.remote)}
              >
                {STATUS_ICONS[task.taskStatus].icon}
              </span>

              {/* Remote badge */}
              {task.remote && (
                <span
                  className="bg-tasks__remote-badge"
                  aria-label={`Running on ${task.remoteWorker ?? "remote worker"}`}
                >
                  <Globe size={11} aria-hidden="true" />
                </span>
              )}

              {/* Core info */}
              <div className="bg-tasks__info">
                <div className="bg-tasks__name-row">
                  <span className="bg-tasks__name">{task.label}</span>
                  {task.remote && task.remoteWorker && (
                    <span className="bg-tasks__worker-tag" aria-hidden="true">
                      {task.remoteWorker}
                    </span>
                  )}
                  {isTerminal && completedAgo && (
                    <span className="bg-tasks__completed-ago" aria-hidden="true">
                      <Clock size={10} />
                      {completedAgo}
                    </span>
                  )}
                </div>
                <div className="bg-tasks__meta">
                  {task.toolCount > 0 && (
                    <span className="bg-tasks__tool-count">
                      {task.toolCount} tool{task.toolCount !== 1 ? "s" : ""}
                    </span>
                  )}
                  {task.lastTool && (
                    <>
                      <span aria-hidden="true">&middot;</span>
                      <span className="bg-tasks__last-tool">{task.lastTool}</span>
                    </>
                  )}
                  {task.reasoningTail && (
                    <span className="bg-tasks__reasoning-tail">
                      {truncateTail(task.reasoningTail)}
                    </span>
                  )}
                </div>
              </div>

              {/* Quick actions */}
              <div className="bg-tasks__actions">
                <button
                  className="bg-tasks__action-btn bg-tasks__peek-btn btn btn--small"
                  onClick={(e) => {
                    e.stopPropagation();
                    dispatch({ type: "toggle_peek", id: task.id });
                  }}
                  aria-label={`Peek ${task.label} (${srEnterLabel})`}
                >
                  <ChevronRight
                    size={13}
                    className={`bg-tasks__chevron${
                      isPeeked ? " bg-tasks__chevron--open" : ""
                    }`}
                    aria-hidden="true"
                  />
                </button>
                <button
                  className="bg-tasks__action-btn bg-tasks__send-btn btn btn--small"
                  onClick={(e) => {
                    e.stopPropagation();
                    // Future: send message to subagent
                  }}
                  aria-label={`Send message to ${task.label} (${srSLabel})`}
                  disabled={isTerminal}
                >
                  <MessageSquare size={13} aria-hidden="true" />
                </button>
                {task.taskStatus === "running" && (
                  <button
                    className="bg-tasks__action-btn bg-tasks__kill-btn btn btn--small btn--danger"
                    onClick={(e) => {
                      e.stopPropagation();
                      // Future: app.CancelJob(task.id)
                    }}
                    aria-label={`Kill ${task.label} (${srKLabel})`}
                  >
                    <X size={13} aria-hidden="true" />
                  </button>
                )}
              </div>

              {/* Peek detail (expanded) */}
              {isPeeked && (
                <div
                  className="bg-tasks__peek"
                  role="region"
                  aria-label={`Details for ${task.label}`}
                >
                  <div className="bg-tasks__peek-grid">
                    <div className="bg-tasks__peek-cell">
                      <span className="bg-tasks__peek-label">Status</span>
                      <span className="bg-tasks__peek-value">
                        <span className={`bg-tasks__status-dot ${statusDotClass(task.taskStatus)}`}>
                          {STATUS_ICONS[task.taskStatus].icon}
                        </span>
                        {STATUS_ICONS[task.taskStatus].label}
                        {task.remote ? " (remote)" : ""}
                      </span>
                    </div>
                    <div className="bg-tasks__peek-cell">
                      <span className="bg-tasks__peek-label">Tools called</span>
                      <span className="bg-tasks__peek-value">{task.toolCount}</span>
                    </div>
                    <div className="bg-tasks__peek-cell">
                      <span className="bg-tasks__peek-label">Last tool</span>
                      <span className="bg-tasks__peek-value bg-tasks__peek-value--code">
                        {task.lastTool || "-"}
                      </span>
                    </div>
                    <div className="bg-tasks__peek-cell">
                      <span className="bg-tasks__peek-label">Started</span>
                      <span className="bg-tasks__peek-value">
                        {formatTimeAgo(task.startedAt)} ago
                      </span>
                    </div>
                    {task.remoteWorker && (
                      <div className="bg-tasks__peek-cell">
                        <span className="bg-tasks__peek-label">Worker</span>
                        <span className="bg-tasks__peek-value">
                          <Globe size={11} aria-hidden="true" />
                          {task.remoteWorker}
                        </span>
                      </div>
                    )}
                    {task.completedAt && (
                      <div className="bg-tasks__peek-cell">
                        <span className="bg-tasks__peek-label">Duration</span>
                        <span className="bg-tasks__peek-value">
                          {((task.completedAt - task.startedAt) / 1000).toFixed(1)}s
                        </span>
                      </div>
                    )}
                  </div>
                  {task.reasoningTail && (
                    <div className="bg-tasks__peek-reasoning">
                      <span className="bg-tasks__peek-label">Reasoning</span>
                      <pre className="bg-tasks__peek-reasoning-text">
                        {task.reasoningTail}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Footer */}
      {state.tasks.length > 0 && (
        <div className="bg-tasks__footer" aria-live="polite">
          <span>
            {activeCount} active, {doneCount} done, {failedCount} failed
          </span>
          <span aria-hidden="true">&middot;</span>
          <span>polling 500ms</span>
        </div>
      )}
    </div>
  );
});
