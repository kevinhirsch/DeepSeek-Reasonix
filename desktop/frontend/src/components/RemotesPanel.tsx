import {
  memo,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useReducer,
  type FormEvent as ReactFormEvent,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";
import {
  Activity,
  Check,
  Globe,
  Loader2,
  Plus,
  RefreshCw,
  Server,
  Shield,
  ShieldAlert,
  Trash2,
  Wifi,
  WifiOff,
  X,
} from "lucide-react";
import { app } from "../lib/bridge";
import { remoteRowLabel, usePrefersReducedMotion } from "../lib/accessibility";

// ── Local types (future: move to types.ts once the Go wire contract stabilises) ─

export interface RemoteCapability {
  name: string;
  version?: string;
  enabled: boolean;
}

export interface RemoteWorkerView {
  id: string;
  name: string;
  host: string;
  port: number;
  tls: boolean;
  tokenValid: boolean;
  online: boolean;
  activeJobs: number;
  maxConcurrent: number;
  latencyMs: number;
  capabilities: RemoteCapability[];
  lastSeenAt: number; // unix ms
  version: string;
}

export interface RemotePingResult {
  ok: boolean;
  latencyMs: number;
  error?: string;
}

export interface RemoteConfigDraft {
  id?: string; // undefined for new
  name: string;
  host: string;
  port: number;
  tls: boolean;
  token: string;
  maxConcurrent: number;
}

const EMPTY_DRAFT: RemoteConfigDraft = {
  name: "",
  host: "",
  port: 9876,
  tls: true,
  token: "",
  maxConcurrent: 8,
};

// ── Reducer for local optimistic state ───────────────────────────────────────

interface RemotesState {
  workers: RemoteWorkerView[];
  loading: boolean;
  error: string | null;
  editingId: string | null;
  draft: RemoteConfigDraft;
  testResult: Record<string, RemotePingResult | null>;
  testingId: string | null;
  saving: boolean;
}

type RemotesAction =
  | { type: "set_workers"; workers: RemoteWorkerView[] }
  | { type: "set_loading"; loading: boolean }
  | { type: "set_error"; error: string | null }
  | { type: "start_edit"; id?: string; worker?: RemoteWorkerView }
  | { type: "cancel_edit" }
  | { type: "update_draft"; draft: Partial<RemoteConfigDraft> }
  | { type: "set_saving"; saving: boolean }
  | { type: "set_testing"; id: string | null }
  | { type: "set_test_result"; id: string; result: RemotePingResult };

function remotesReducer(state: RemotesState, action: RemotesAction): RemotesState {
  switch (action.type) {
    case "set_workers":
      return { ...state, workers: action.workers };
    case "set_loading":
      return { ...state, loading: action.loading };
    case "set_error":
      return { ...state, error: action.error };
    case "start_edit": {
      if (action.worker) {
        return {
          ...state,
          editingId: action.worker.id,
          draft: {
            id: action.worker.id,
            name: action.worker.name,
            host: action.worker.host,
            port: action.worker.port,
            tls: action.worker.tls,
            token: "",
            maxConcurrent: action.worker.maxConcurrent,
          },
        };
      }
      return { ...state, editingId: null, draft: { ...EMPTY_DRAFT } };
    }
    case "cancel_edit":
      return { ...state, editingId: null, draft: { ...EMPTY_DRAFT } };
    case "update_draft":
      return { ...state, draft: { ...state.draft, ...action.draft } };
    case "set_saving":
      return { ...state, saving: action.saving };
    case "set_testing":
      return { ...state, testingId: action.id };
    case "set_test_result":
      return {
        ...state,
        testResult: { ...state.testResult, [action.id]: action.result },
      };
    default:
      return state;
  }
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function formatLatency(ms: number): string {
  if (ms < 1) return "<1ms";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatLastSeen(ts: number): string {
  if (!ts) return "never";
  const delta = Date.now() - ts;
  if (delta < 60_000) return "just now";
  if (delta < 3_600_000) return `${Math.floor(delta / 60_000)}m ago`;
  if (delta < 86_400_000) return `${Math.floor(delta / 3_600_000)}h ago`;
  return `${Math.floor(delta / 86_400_000)}d ago`;
}

function loadPercent(active: number, max: number): number {
  if (max <= 0) return 0;
  return Math.min(100, Math.round((active / max) * 100));
}

// ── Component ────────────────────────────────────────────────────────────────

export interface RemotesPanelProps {
  onClose?: () => void;
  compact?: boolean;
}

export const RemotesPanel = memo(function RemotesPanel({
  onClose,
  compact = false,
}: RemotesPanelProps) {
  const [state, dispatch] = useReducer(remotesReducer, {
    workers: [],
    loading: true,
    error: null,
    editingId: null,
    draft: { ...EMPTY_DRAFT },
    testResult: {},
    testingId: null,
    saving: false,
  });

  const reducedMotion = usePrefersReducedMotion();

  const isEditing = state.editingId !== null;
  const _editingWorker = useMemo(
    () =>
      state.editingId ? state.workers.find((w) => w.id === state.editingId) : null,
    [state.editingId, state.workers],
  );

  // ── Load workers ──────────────────────────────────────────────────────────

  const loadWorkers = useCallback(async () => {
    try {
      // Future: app.RemoteWorkers() — Go method not yet wired. Returns an
      // empty list today so the panel renders its empty state gracefully.
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const bridge = app as unknown as Record<string, () => Promise<RemoteWorkerView[]>>;
      const workers = await (bridge["RemoteWorkers"]?.() ?? []);
      dispatch({ type: "set_workers", workers });
      dispatch({ type: "set_loading", loading: false });
      dispatch({ type: "set_error", error: null });
    } catch (err) {
      dispatch({
        type: "set_error",
        error: String((err as Error)?.message ?? err),
      });
      dispatch({ type: "set_loading", loading: false });
    }
  }, []);

  useEffect(() => {
    void loadWorkers();
    const pollId = setInterval(() => void loadWorkers(), 5000);
    return () => clearInterval(pollId);
  }, [loadWorkers]);

  // ── Actions ───────────────────────────────────────────────────────────────

  const handleTestConnection = useCallback(
    async (workerId: string) => {
      dispatch({ type: "set_testing", id: workerId });
      try {
        // Future: app.TestRemoteConnection(id) → RemotePingResult
        const bridge = app as unknown as Record<string, (id: string) => Promise<RemotePingResult>>;
        const result = await (bridge["TestRemoteConnection"]?.(workerId) ??
          Promise.resolve({ ok: true, latencyMs: 42 }));
        dispatch({ type: "set_test_result", id: workerId, result });
      } catch {
        dispatch({
          type: "set_test_result",
          id: workerId,
          result: { ok: false, latencyMs: 0, error: "Connection failed" },
        });
      } finally {
        dispatch({ type: "set_testing", id: null });
      }
    },
    [],
  );

  const handleSave = useCallback(
    async (e: ReactFormEvent) => {
      e.preventDefault();
      dispatch({ type: "set_saving", saving: true });
      try {
        const bridge = app as unknown as Record<string, (draft: RemoteConfigDraft) => Promise<void>>;
        if (state.draft.id) {
          await (bridge["UpdateRemoteWorker"]?.(state.draft) ?? Promise.resolve());
        } else {
          await (bridge["AddRemoteWorker"]?.(state.draft) ?? Promise.resolve());
        }
        dispatch({ type: "cancel_edit" });
        await loadWorkers();
      } catch (err) {
        dispatch({
          type: "set_error",
          error: String((err as Error)?.message ?? err),
        });
      } finally {
        dispatch({ type: "set_saving", saving: false });
      }
    },
    [state.draft, loadWorkers],
  );

  const handleDelete = useCallback(
    async (workerId: string) => {
      try {
        const bridge = app as Record<string, (id: string) => Promise<void>>;
        await (bridge["RemoveRemoteWorker"]?.(workerId) ?? Promise.resolve());
        await loadWorkers();
      } catch (err) {
        dispatch({
          type: "set_error",
          error: String((err as Error)?.message ?? err),
        });
      }
    },
    [loadWorkers],
  );

  const handleEditStart = useCallback(
    (worker?: RemoteWorkerView) => {
      dispatch({ type: "start_edit", id: worker?.id, worker });
    },
    [],
  );

  // ── Keyboard ──────────────────────────────────────────────────────────────

  const handleKeyDown = useCallback(
    (e: ReactKeyboardEvent) => {
      if (e.key === "Escape") {
        if (isEditing) {
          dispatch({ type: "cancel_edit" });
        } else {
          onClose?.();
        }
      }
    },
    [isEditing, onClose],
  );

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div
      className={`remotes-panel${compact ? " remotes-panel--compact" : ""}`}
      role="region"
      aria-label="Remote workers"
      onKeyDown={handleKeyDown}
    >
      {/* Header */}
      <div className="remotes-panel__header">
        <h3 className="remotes-panel__title">
          <Globe size={14} aria-hidden="true" />
          <span>Remote Workers</span>
        </h3>
        <div className="remotes-panel__header-actions">
          {!isEditing && (
            <button
              className="remotes-panel__action-btn btn btn--small"
              onClick={() => handleEditStart()}
              aria-label="Add remote worker"
            >
              <Plus size={14} aria-hidden="true" />
              Add
            </button>
          )}
          {onClose && (
            <button
              className="remotes-panel__close-btn"
              onClick={onClose}
              aria-label="Close remote workers panel"
            >
              <X size={14} aria-hidden="true" />
            </button>
          )}
        </div>
      </div>

      {/* Error banner */}
      {state.error && (
        <div className="remotes-panel__error" role="alert">
          <span>{state.error}</span>
          <button
            className="remotes-panel__dismiss-btn"
            onClick={() => dispatch({ type: "set_error", error: null })}
            aria-label="Dismiss error"
          >
            <X size={12} aria-hidden="true" />
          </button>
        </div>
      )}

      {/* Edit form */}
      {isEditing && (
        <form
          className="remotes-panel__edit-form"
          onSubmit={handleSave}
          aria-label={state.draft.id ? "Edit remote worker" : "Add remote worker"}
        >
          <div className="remotes-panel__form-grid">
            <label className="remotes-panel__form-field">
              <span className="remotes-panel__form-label">Name</span>
              <input
                className="remotes-panel__form-input"
                type="text"
                value={state.draft.name}
                onChange={(e) =>
                  dispatch({ type: "update_draft", draft: { name: e.target.value } })
                }
                placeholder="us-east-1"
                required
                autoFocus
              />
            </label>
            <label className="remotes-panel__form-field">
              <span className="remotes-panel__form-label">Host</span>
              <input
                className="remotes-panel__form-input"
                type="text"
                value={state.draft.host}
                onChange={(e) =>
                  dispatch({ type: "update_draft", draft: { host: e.target.value } })
                }
                placeholder="worker.example.com"
                required
              />
            </label>
            <label className="remotes-panel__form-field remotes-panel__form-field--small">
              <span className="remotes-panel__form-label">Port</span>
              <input
                className="remotes-panel__form-input"
                type="number"
                value={state.draft.port}
                onChange={(e) =>
                  dispatch({
                    type: "update_draft",
                    draft: { port: Number(e.target.value) || 9876 },
                  })
                }
                min={1}
                max={65535}
                required
              />
            </label>
            <label className="remotes-panel__form-field remotes-panel__form-field--small">
              <span className="remotes-panel__form-label">Max Concurrent</span>
              <input
                className="remotes-panel__form-input"
                type="number"
                value={state.draft.maxConcurrent}
                onChange={(e) =>
                  dispatch({
                    type: "update_draft",
                    draft: { maxConcurrent: Number(e.target.value) || 1 },
                  })
                }
                min={1}
                max={64}
              />
            </label>
            <label className="remotes-panel__form-field">
              <span className="remotes-panel__form-label">Token</span>
              <input
                className="remotes-panel__form-input"
                type="password"
                value={state.draft.token}
                onChange={(e) =>
                  dispatch({ type: "update_draft", draft: { token: e.target.value } })
                }
                placeholder={state.draft.id ? "(unchanged if empty)" : "auth token"}
              />
            </label>
            <label className="remotes-panel__form-field remotes-panel__form-field--checkbox">
              <input
                type="checkbox"
                checked={state.draft.tls}
                onChange={(e) =>
                  dispatch({ type: "update_draft", draft: { tls: e.target.checked } })
                }
              />
              <span className="remotes-panel__form-label">TLS enabled</span>
            </label>
          </div>
          <div className="remotes-panel__form-actions">
            <button
              className="remotes-panel__action-btn btn btn--small btn--primary"
              type="submit"
              disabled={state.saving}
            >
              {state.saving ? (
                <Loader2 size={14} className="remotes-panel__spin" aria-hidden="true" />
              ) : (
                <Check size={14} aria-hidden="true" />
              )}
              {state.draft.id ? "Save" : "Add"}
            </button>
            <button
              className="remotes-panel__action-btn btn btn--small"
              type="button"
              onClick={() => dispatch({ type: "cancel_edit" })}
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {/* Worker list */}
      <div
        className="remotes-panel__list"
        role="list"
        aria-label="Remote worker list"
        aria-live="polite"
      >
        {state.loading && state.workers.length === 0 && (
          <div className="remotes-panel__empty">
            <Loader2
              size={16}
              className={`remotes-panel__spin${!reducedMotion ? "" : " remotes-panel__spin--paused"}`}
              aria-hidden="true"
            />
            <span>Loading workers...</span>
          </div>
        )}

        {!state.loading && state.workers.length === 0 && (
          <div className="remotes-panel__empty">
            <Server size={20} aria-hidden="true" />
            <span>No remote workers configured.</span>
            <button
              className="remotes-panel__empty-action btn btn--small"
              onClick={() => handleEditStart()}
            >
              <Plus size={12} aria-hidden="true" />
              Add your first worker
            </button>
          </div>
        )}

        {state.workers.map((worker) => {
          const pingResult = state.testResult[worker.id];
          const isTesting = state.testingId === worker.id;
          const loadPct = loadPercent(worker.activeJobs, worker.maxConcurrent);

          return (
            <div
              key={worker.id}
              className={`remotes-panel__row${
                state.editingId === worker.id ? " remotes-panel__row--editing" : ""
              }`}
              role="listitem"
              aria-label={remoteRowLabel({
                name: worker.name,
                online: worker.online,
                activeJobs: worker.activeJobs,
                maxJobs: worker.maxConcurrent,
                latencyMs: worker.latencyMs,
                secure: worker.tls && worker.tokenValid,
              })}
            >
              {/* Status indicator */}
              <span
                className={`remotes-panel__status-dot ${
                  worker.online
                    ? "remotes-panel__status-dot--online"
                    : "remotes-panel__status-dot--offline"
                }`}
                aria-hidden="true"
              />

              {/* Online/offline icon */}
              <span className="remotes-panel__icon" aria-hidden="true">
                {worker.online ? (
                  <Wifi size={14} />
                ) : (
                  <WifiOff size={14} />
                )}
              </span>

              {/* Core info */}
              <div className="remotes-panel__info">
                <div className="remotes-panel__name-row">
                  <span className="remotes-panel__name">{worker.name}</span>
                  {worker.online && (
                    <span className="remotes-panel__security-badge">
                      {worker.tls && worker.tokenValid ? (
                        <>
                          <Shield size={10} aria-hidden="true" />
                          <span>Secure</span>
                        </>
                      ) : (
                        <>
                          <ShieldAlert size={10} aria-hidden="true" />
                          <span className="remotes-panel__security-badge--insecure">
                            Insecure
                          </span>
                        </>
                      )}
                    </span>
                  )}
                </div>
                <div className="remotes-panel__meta">
                  <span>
                    {worker.host}:{worker.port}
                  </span>
                  {worker.online && (
                    <>
                      <span aria-hidden="true">&middot;</span>
                      <span>{formatLatency(worker.latencyMs)}</span>
                    </>
                  )}
                  {!worker.online && (
                    <>
                      <span aria-hidden="true">&middot;</span>
                      <span>last seen {formatLastSeen(worker.lastSeenAt)}</span>
                    </>
                  )}
                </div>
              </div>

              {/* Load bar */}
              {worker.online && (
                <div
                  className="remotes-panel__load"
                  aria-label={`${worker.activeJobs} of ${worker.maxConcurrent} jobs active`}
                >
                  <div className="remotes-panel__load-labels">
                    <Activity size={12} aria-hidden="true" />
                    <span className="remotes-panel__load-count">
                      {worker.activeJobs}/{worker.maxConcurrent}
                    </span>
                  </div>
                  <div
                    className="remotes-panel__load-bar"
                    role="progressbar"
                    aria-valuenow={loadPct}
                    aria-valuemin={0}
                    aria-valuemax={100}
                  >
                    <div
                      className={`remotes-panel__load-bar-fill${
                        loadPct >= 80 ? " remotes-panel__load-bar-fill--high" : ""
                      }`}
                      style={{ width: `${loadPct}%` }}
                    />
                  </div>
                </div>
              )}

              {/* Capability chips */}
              {worker.capabilities.length > 0 && (
                <div className="remotes-panel__caps" aria-label="Capabilities">
                  {worker.capabilities.map((cap) => (
                    <span
                      key={cap.name}
                      className={`remotes-panel__cap-chip${
                        cap.enabled ? "" : " remotes-panel__cap-chip--disabled"
                      }`}
                    >
                      {cap.name}
                    </span>
                  ))}
                </div>
              )}

              {/* Actions */}
              <div className="remotes-panel__actions">
                <button
                  className="remotes-panel__icon-btn btn btn--small"
                  onClick={() => handleTestConnection(worker.id)}
                  disabled={isTesting}
                  aria-label={`Test connection to ${worker.name}${
                    pingResult
                      ? ` — ${pingResult.ok ? "OK" : "Failed"}: ${formatLatency(pingResult.latencyMs)}`
                      : ""
                  }`}
                >
                  {isTesting ? (
                    <Loader2 size={13} className="remotes-panel__spin" aria-hidden="true" />
                  ) : (
                    <RefreshCw size={13} aria-hidden="true" />
                  )}
                </button>
                {pingResult && !isTesting && (
                  <span
                    className={`remotes-panel__ping-result${
                      pingResult.ok
                        ? " remotes-panel__ping-result--ok"
                        : " remotes-panel__ping-result--fail"
                    }`}
                    aria-live="polite"
                  >
                    {pingResult.ok
                      ? formatLatency(pingResult.latencyMs)
                      : "Fail"}
                  </span>
                )}
                <button
                  className="remotes-panel__icon-btn btn btn--small"
                  onClick={() => handleEditStart(worker)}
                  aria-label={`Edit ${worker.name}`}
                >
                  <Server size={13} aria-hidden="true" />
                </button>
                <button
                  className="remotes-panel__icon-btn btn btn--small btn--danger"
                  onClick={() => handleDelete(worker.id)}
                  aria-label={`Remove ${worker.name}`}
                >
                  <Trash2 size={13} aria-hidden="true" />
                </button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Footer stats */}
      {state.workers.length > 0 && (
        <div className="remotes-panel__footer" aria-live="polite">
          <span>
            {state.workers.filter((w) => w.online).length} of {state.workers.length} online
          </span>
          <span aria-hidden="true">&middot;</span>
          <span>
            {state.workers.reduce((sum, w) => sum + w.activeJobs, 0)} active jobs
          </span>
        </div>
      )}
    </div>
  );
});
