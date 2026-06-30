import { useState, useEffect, useCallback } from "react";
import { Activity, Loader2, CheckCircle2, XCircle, Clock, Globe, MessageSquare, X, ChevronRight, AlertTriangle } from "lucide-react";
import { app } from "../lib/bridge";

interface TaskView {
  id: string;
  kind: string;
  label: string;
  status: string;
  toolCalls: number;
  lastTool: string;
  lastReasoning: string;
  model: string;
  effort: string;
}

export function BackgroundTasksPanel() {
  const [tasks, setTasks] = useState<TaskView[]>([]);
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [visible, setVisible] = useState(false);

  const poll = useCallback(async () => {
    try {
      const result = await app.GetBackgroundTasks();
      if (Array.isArray(result)) {
        setTasks(result as TaskView[]);
        setVisible(result.length > 0);
        if (selectedIdx >= result.length) {
          setSelectedIdx(Math.max(0, result.length - 1));
        }
      }
    } catch {
      // silently ignore poll errors
    }
  }, [selectedIdx]);

  useEffect(() => {
    const interval = setInterval(poll, 500);
    return () => clearInterval(interval);
  }, [poll]);

  useEffect(() => {
    poll();
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (!visible) return;
    switch (e.key) {
      case "j":
      case "ArrowDown":
        e.preventDefault();
        setSelectedIdx((i) => Math.min(tasks.length - 1, i + 1));
        break;
      case "k":
      case "ArrowUp":
        e.preventDefault();
        setSelectedIdx((i) => Math.max(0, i - 1));
        break;
      case "Escape":
        e.preventDefault();
        setVisible(false);
        break;
    }
  }, [visible, tasks.length]);

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  if (!visible || tasks.length === 0) return null;

  const running = tasks.filter((t) => t.status === "running").length;
  const done = tasks.filter((t) => t.status === "done").length;
  const totalTokens = tasks.reduce((sum, t) => sum + (t.toolCalls || 0) * 100, 0);

  const statusIcon = (status: string) => {
    switch (status) {
      case "running": return <Activity className="w-3.5 h-3.5 text-green-500" />;
      case "done": return <CheckCircle2 className="w-3.5 h-3.5 text-green-600" />;
      case "failed": return <XCircle className="w-3.5 h-3.5 text-red-500" />;
      case "killed": return <AlertTriangle className="w-3.5 h-3.5 text-gray-400" />;
      default: return <Clock className="w-3.5 h-3.5 text-gray-400" />;
    }
  };

  return (
    <div className="bg-tasks-panel border-t border-gray-200 dark:border-gray-700 font-mono text-xs" role="region" aria-label="Background tasks">
      <div className="px-3 py-1.5 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between text-gray-500">
        <span>
          <Activity className="w-3 h-3 inline mr-1" />
          Background Tasks ({running} running · {done} done
          {totalTokens > 1000 ? ` · ${(totalTokens / 1000).toFixed(1)}K tokens` : ""})
        </span>
        <button onClick={() => setVisible(false)} className="hover:text-gray-700 dark:hover:text-gray-300" aria-label="Close panel">
          <X className="w-3 h-3" />
        </button>
      </div>
      <div className="max-h-48 overflow-y-auto">
        {tasks.map((task, i) => (
          <div
            key={task.id}
            className={`px-3 py-1.5 border-b border-gray-100 dark:border-gray-800 flex items-center gap-2 ${i === selectedIdx ? "bg-blue-50 dark:bg-blue-900/20" : ""}`}
            role="listitem"
          >
            <span className="flex-shrink-0">{statusIcon(task.status)}</span>
            <span className="flex-1 truncate font-medium">{task.label || task.kind}</span>
            <span className="text-gray-400">[{task.status}]</span>
            {task.toolCalls > 0 && (
              <span className="text-gray-400">{task.toolCalls} calls</span>
            )}
            {task.lastTool && (
              <span className="text-gray-500">{task.lastTool}</span>
            )}
            {task.model && (
              <span className="text-gray-400">{task.model}{task.effort ? `/${task.effort}` : ""}</span>
            )}
          </div>
        ))}
      </div>
      {tasks.length > 0 && (
        <div className="px-3 py-1 text-gray-400 flex gap-3">
          <span><kbd className="text-[10px]">j/k</kbd> navigate</span>
          <span><kbd className="text-[10px]">Enter</kbd> peek</span>
          <span><kbd className="text-[10px]">s</kbd> send</span>
          <span><kbd className="text-[10px]">Esc</kbd> close</span>
        </div>
      )}
    </div>
  );
}
