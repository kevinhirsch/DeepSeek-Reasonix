import { useState, useEffect, useCallback } from "react";
import { Search, Star, Lock, Clock, GitBranch, Download, FolderOpen, Loader2, ExternalLink } from "lucide-react";
import { app } from "../lib/bridge";

interface RepoInfo {
  name: string;
  fullName: string;
  description: string;
  private: boolean;
  updatedAt: string;
  cloneUrl: string;
  htmlUrl: string;
  language: string;
  isCloned: boolean;
  localPath?: string;
}

type Tab = "my-repos" | "starred" | "search";

export function RepoPicker() {
  const [tab, setTab] = useState<Tab>("my-repos");
  const [repos, setRepos] = useState<RepoInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [cloning, setCloning] = useState<string | null>(null);

  const fetchRepos = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      if (tab === "my-repos") {
        const result = await app.ListGitHubRepos(30);
        setRepos(result || []);
      } else if (tab === "starred") {
        const result = await app.ListStarredRepos();
        setRepos(result || []);
      }
    } catch (e) {
      setError(String(e));
      setRepos([]);
    } finally {
      setLoading(false);
    }
  }, [tab]);

  useEffect(() => {
    if (tab !== "search") fetchRepos();
  }, [tab, fetchRepos]);

  const handleClone = async (repo: RepoInfo) => {
    setCloning(repo.fullName);
    try {
      await app.CloneRepo(repo.cloneUrl, "");
    } catch (e) {
      setError(String(e));
    } finally {
      setCloning(null);
    }
  };

  const handleOpen = async (repo: RepoInfo) => {
    if (repo.localPath) {
      await app.OpenClonedRepo(repo.localPath);
    }
  };

  const filtered = searchQuery
    ? repos.filter((r) =>
        r.fullName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        r.description?.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : repos;

  return (
    <div className="repo-picker flex flex-col h-full" role="dialog" aria-label="Repository picker">
      <div className="flex items-center gap-2 p-3 border-b border-gray-200 dark:border-gray-700">
        <div className="flex gap-1">
          {(["my-repos", "starred", "search"] as Tab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`px-3 py-1 text-sm rounded ${tab === t ? "bg-blue-600 text-white" : "text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800"}`}
              aria-pressed={tab === t}
            >
              {t === "my-repos" ? "My Repos" : t === "starred" ? "Starred" : "Search"}
            </button>
          ))}
        </div>
        {tab === "search" && (
          <div className="relative flex-1">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search repositories..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-8 pr-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-900"
              aria-label="Search repositories"
            />
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && (
          <div className="p-4 space-y-3">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="animate-pulse flex gap-3 p-3">
                <div className="w-5 h-5 bg-gray-200 dark:bg-gray-700 rounded" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
                  <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-2/3" />
                </div>
              </div>
            ))}
          </div>
        )}

        {error && (
          <div className="p-4 text-red-600 text-sm text-center" role="alert">{error}</div>
        )}

        {!loading && !error && filtered.length === 0 && (
          <div className="p-8 text-center text-gray-400">
            <GitBranch className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No repositories found</p>
            {tab === "my-repos" && (
              <p className="text-xs mt-1">Connect your GitHub account to see your repos</p>
            )}
          </div>
        )}

        {!loading && filtered.map((repo) => (
          <div
            key={repo.fullName}
            className="flex items-center gap-3 p-3 hover:bg-gray-50 dark:hover:bg-gray-800 border-b border-gray-100 dark:border-gray-800"
          >
            <GitBranch className="w-4 h-4 text-blue-500 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium truncate">{repo.fullName}</span>
                {repo.private && <Lock className="w-3 h-3 text-yellow-500" title="Private" />}
                {repo.language && (
                  <span className="text-xs px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300">
                    {repo.language}
                  </span>
                )}
              </div>
              {repo.description && (
                <p className="text-xs text-gray-500 truncate mt-0.5">{repo.description}</p>
              )}
              <div className="flex items-center gap-3 mt-1 text-xs text-gray-400">
                <span className="flex items-center gap-1"><Clock className="w-3 h-3" />{repo.updatedAt?.slice(0, 10)}</span>
              </div>
            </div>
            <div className="flex items-center gap-1 flex-shrink-0">
              <a
                href={repo.htmlUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                aria-label={`Open ${repo.fullName} on GitHub`}
              >
                <ExternalLink className="w-4 h-4" />
              </a>
              {repo.isCloned ? (
                <button
                  onClick={() => handleOpen(repo)}
                  className="flex items-center gap-1 px-3 py-1 text-sm bg-green-600 text-white rounded hover:bg-green-700"
                >
                  <FolderOpen className="w-3.5 h-3.5" /> Open
                </button>
              ) : (
                <button
                  onClick={() => handleClone(repo)}
                  disabled={cloning === repo.fullName}
                  className="flex items-center gap-1 px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                >
                  {cloning === repo.fullName ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Download className="w-3.5 h-3.5" />
                  )}
                  Clone
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
