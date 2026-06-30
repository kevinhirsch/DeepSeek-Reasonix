import { useState, useEffect, useCallback } from "react";
import { Server, Wifi, WifiOff, Activity, Shield, Plus, Trash2, Globe, Zap } from "lucide-react";
import { app } from "../lib/bridge";

interface RemoteEntry {
  name: string;
  url: string;
  maxConcurrent: number;
  preferFor: string[];
  online?: boolean;
  load?: number;
  latency?: number;
}

export function RemotesPanel() {
  const [remotes, setRemotes] = useState<RemoteEntry[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchRemotes = useCallback(async () => {
    setLoading(true);
    try {
      const result = await app.ListRemotes();
      if (Array.isArray(result)) {
        setRemotes(result as RemoteEntry[]);
      }
    } catch {
      // silently ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchRemotes(); }, [fetchRemotes]);

  if (loading) {
    return (
      <div className="p-4 space-y-3 animate-pulse">
        {[1, 2].map((i) => (
          <div key={i} className="h-16 bg-gray-200 dark:bg-gray-700 rounded" />
        ))}
      </div>
    );
  }

  if (remotes.length === 0) {
    return (
      <div className="p-8 text-center text-gray-400">
        <Server className="w-8 h-8 mx-auto mb-2 opacity-50" />
        <p className="text-sm">No remote workers configured</p>
        <p className="text-xs mt-1">Add [[remotes]] to reasonix.toml or use the bootstrap command</p>
      </div>
    );
  }

  return (
    <div className="remotes-panel" role="region" aria-label="Remote workers">
      <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
        <h3 className="text-sm font-medium">Remote Workers</h3>
        <button
          className="flex items-center gap-1 px-2 py-0.5 text-xs bg-blue-600 text-white rounded hover:bg-blue-700"
          aria-label="Add remote worker"
        >
          <Plus className="w-3 h-3" /> Add
        </button>
      </div>
      <div className="divide-y divide-gray-100 dark:divide-gray-800">
        {remotes.map((r) => (
          <div key={r.name} className="px-3 py-2.5 flex items-center gap-3">
            <Globe className={`w-4 h-4 ${r.online !== false ? "text-green-500" : "text-gray-400"}`} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium truncate">{r.name}</span>
                {r.online !== false ? (
                  <Wifi className="w-3 h-3 text-green-500" title="Online" />
                ) : (
                  <WifiOff className="w-3 h-3 text-red-500" title="Offline" />
                )}
                <Shield className={`w-3 h-3 ${r.online !== false ? "text-green-500" : "text-gray-400"}`} title="TLS secured" />
              </div>
              <div className="flex items-center gap-3 mt-0.5 text-xs text-gray-400">
                <span>{r.url}</span>
                {r.load !== undefined && (
                  <span className="flex items-center gap-1"><Activity className="w-3 h-3" />{r.load}/{r.maxConcurrent}</span>
                )}
                {r.latency !== undefined && (
                  <span className="flex items-center gap-1"><Zap className="w-3 h-3" />{r.latency}ms</span>
                )}
              </div>
              {r.preferFor && r.preferFor.length > 0 && (
                <div className="flex gap-1 mt-1">
                  {r.preferFor.map((tag) => (
                    <span key={tag} className="px-1.5 py-0.5 text-[10px] rounded bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300">
                      {tag}
                    </span>
                  ))}
                </div>
              )}
            </div>
            <button className="p-1 text-gray-400 hover:text-red-500" aria-label={`Remove ${r.name}`}>
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
