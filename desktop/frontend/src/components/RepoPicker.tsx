import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { GitBranch, Search, Star, Lock, Clock, Download, FolderOpen, Loader2, AlertCircle } from "lucide-react";
import { useT } from "../lib/i18n";
import { app } from "../lib/bridge";
import type { ReactNode } from "react";

// ── Local types (mirror the Go-side RepoPickerItem / ClonedRepoView) ──

interface RepoInfo {
  name: string;
  fullName: string;
  description: string;
  private: boolean;
  updatedAt: string;
  cloneUrl: string;
  htmlUrl: string;
  language: string;
}

interface ClonedRepo {
  path: string;
  name: string;
  fullName: string;
  clonedAt: string;
}

type FilterTab = "mine" | "starred" | "search";

// ── Helpers ──

function timeAgo(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso.slice(0, 10);
  const now = Date.now();
  const diff = now - d.getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

// ── Skeleton row ──

function SkeletonRow() {
  return (
    <div className="repo-picker__item repo-picker__item--skeleton" aria-hidden="true">
      <span className="repo-picker__skeleton-icon" />
      <span className="repo-picker__skeleton-body">
        <span className="repo-picker__skeleton-line repo-picker__skeleton-line--name" />
        <span className="repo-picker__skeleton-line repo-picker__skeleton-line--desc" />
      </span>
    </div>
  );
}

// ── Main component ──

export function RepoPicker({
  workspaceRoot,
  onClose: _onClose,
  onRepoOpened,
}: {
  workspaceRoot: string;
  onClose: () => void;
  onRepoOpened: (path: string) => void;
}) {
  const t = useT();

  const [tab, setTab] = useState<FilterTab>("mine");
  const [query, setQuery] = useState("");
  const [repos, setRepos] = useState<RepoInfo[]>([]);
  const [clonedRepos, setClonedRepos] = useState<ClonedRepo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cloningFullName, setCloningFullName] = useState<string | null>(null);

  // Track whether the component is mounted for safe async state updates.
  const mountedRef = useRef(true);
  useEffect(() => () => { mountedRef.current = false; }, []);

  // ── Data fetching ──

  const loadRepos = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const filter = tab === "search" ? query : tab;
      const result = await app.OpenRepoPicker(filter);
      if (mountedRef.current) setRepos(result);
    } catch (err) {
      if (mountedRef.current) {
        setError(String(err));
        setRepos([]);
      }
    } finally {
      if (mountedRef.current) setLoading(false);
    }
  }, [tab, query]);

  const loadClonedRepos = useCallback(async () => {
    try {
      const result = await app.ListClonedRepos();
      if (mountedRef.current) setClonedRepos(result);
    } catch {
      // non-fatal — cloned-repo detection is best-effort
    }
  }, []);

  useEffect(() => {
    void loadRepos();
  }, [loadRepos]);

  useEffect(() => {
    void loadClonedRepos();
  }, [loadClonedRepos]);

  // Debounced search: only fire when the user stops typing.
  useEffect(() => {
    if (tab !== "search") return;
    const id = window.setTimeout(() => {
      if (query.trim()) void loadRepos();
    }, 400);
    return () => window.clearTimeout(id);
  }, [query, tab, loadRepos]);

  // ── Derived state ──

  const clonedMap = useMemo(() => {
    const m = new Map<string, string>(); // fullName → localPath
    for (const r of clonedRepos) {
      m.set(r.fullName, r.path);
    }
    return m;
  }, [clonedRepos]);

  // ── Actions ──

  const handleClone = async (repo: RepoInfo) => {
    setCloningFullName(repo.fullName);
    setError(null);
    try {
      const dir = workspaceRoot
        ? `${workspaceRoot.replace(/[/\\]+$/, "")}/${repo.name}`
        : "";
      const result = await app.CloneRepo(repo.cloneUrl, "", dir);
      if (mountedRef.current) {
        await loadClonedRepos();
        onRepoOpened(result.dir);
      }
    } catch (err) {
      if (mountedRef.current) setError(String(err));
    } finally {
      if (mountedRef.current) setCloningFullName(null);
    }
  };

  const handleOpen = async (fullName: string) => {
    const path = clonedMap.get(fullName);
    if (path) onRepoOpened(path);
  };

  // ── Language badge colour map ──

  const langClass = (lang: string): string => {
    const key = lang?.toLowerCase() || "";
    if (key === "go") return "repo-picker__badge--go";
    if (key === "typescript" || key === "javascript") return "repo-picker__badge--ts";
    if (key === "python") return "repo-picker__badge--py";
    if (key === "rust") return "repo-picker__badge--rust";
    return "";
  };

  // ── Render helpers ──

  const tabLabel = (t: FilterTab): string => {
    switch (t) {
      case "mine": return "My Repos";
      case "starred": return "Starred";
      case "search": return "Search";
    }
  };

  const tabIcon = (t: FilterTab): ReactNode => {
    switch (t) {
      case "mine": return <GitBranch size={14} />;
      case "starred": return <Star size={14} />;
      case "search": return <Search size={14} />;
    }
  };

  // ── Render ──

  return (
    <div className="repo-picker" role="dialog" aria-label="Repository picker">
      {/* Header */}
      <div className="repo-picker__header">
        <div className="repo-picker__tabs" role="tablist">
          {(["mine", "starred", "search"] as FilterTab[]).map((t) => (
            <button
              key={t}
              role="tab"
              aria-selected={tab === t}
              className={`repo-picker__tab${tab === t ? " repo-picker__tab--active" : ""}`}
              onClick={() => { setTab(t); setError(null); }}
            >
              {tabIcon(t)}
              <span>{tabLabel(t)}</span>
            </button>
          ))}
        </div>
        {tab === "search" && (
          <div className="repo-picker__search">
            <Search size={16} className="repo-picker__search-icon" aria-hidden="true" />
            <input
              type="text"
              className="repo-picker__search-input"
              placeholder={t("repoPicker.searchPlaceholder")}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              aria-label={t("repoPicker.searchLabel")}
            />
          </div>
        )}
      </div>

      {/* Body */}
      <div className="repo-picker__body">
        {/* Loading skeleton */}
        {loading && (
          <div className="repo-picker__list">
            {Array.from({ length: 5 }).map((_, i) => (
              <SkeletonRow key={i} />
            ))}
          </div>
        )}

        {/* Error */}
        {!loading && error && (
          <div className="repo-picker__error" role="alert">
            <AlertCircle size={18} aria-hidden="true" />
            <span className="repo-picker__error-text">{error}</span>
          </div>
        )}

        {/* Empty state */}
        {!loading && !error && repos.length === 0 && (
          <div className="repo-picker__empty">
            <GitBranch size={32} className="repo-picker__empty-icon" aria-hidden="true" />
            <p className="repo-picker__empty-title">
              {tab === "search" && query.trim()
                ? t("repoPicker.noSearchResults")
                : t("repoPicker.noRepos")}
            </p>
            <p className="repo-picker__empty-hint">
              {tab === "mine"
                ? t("repoPicker.connectHint")
                : ""}
            </p>
          </div>
        )}

        {/* Repo list */}
        {!loading && repos.length > 0 && (
          <div className="repo-picker__list">
            {repos.map((repo) => {
              const isCloned = clonedMap.has(repo.fullName);
              const isCloning = cloningFullName === repo.fullName;

              return (
                <div key={repo.fullName} className="repo-picker__item">
                  <GitBranch size={16} className="repo-picker__item-icon" aria-hidden="true" />

                  <div className="repo-picker__item-main">
                    <div className="repo-picker__item-top">
                      <span className="repo-picker__item-name" title={repo.fullName}>
                        {repo.fullName}
                      </span>
                      {repo.private && (
                        <Lock size={12} className="repo-picker__private" aria-label="Private repository" />
                      )}
                      {repo.language && (
                        <span className={`repo-picker__badge ${langClass(repo.language)}`}>
                          {repo.language}
                        </span>
                      )}
                    </div>

                    {repo.description && (
                      <p className="repo-picker__item-desc">{repo.description}</p>
                    )}

                    <span className="repo-picker__item-meta">
                      <Clock size={12} aria-hidden="true" />
                      <span>{timeAgo(repo.updatedAt)}</span>
                    </span>
                  </div>

                  <div className="repo-picker__item-actions">
                    {isCloned ? (
                      <button
                        className="repo-picker__action repo-picker__action--open"
                        onClick={() => handleOpen(repo.fullName)}
                        title={t("repoPicker.openRepo")}
                      >
                        <FolderOpen size={14} />
                        <span>{t("repoPicker.open")}</span>
                      </button>
                    ) : (
                      <button
                        className="repo-picker__action repo-picker__action--clone"
                        onClick={() => handleClone(repo)}
                        disabled={isCloning}
                        title={t("repoPicker.cloneRepo")}
                      >
                        {isCloning ? (
                          <Loader2 size={14} className="repo-picker__spin" />
                        ) : (
                          <Download size={14} />
                        )}
                        <span>{isCloning ? t("repoPicker.cloning") : t("repoPicker.clone")}</span>
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
