// i18n keys for repository management, background tasks, remote workers,
// workflow visualisation, speculative execution, and voting breakdown components.
//
// Keys are dotted by component area; values are the English defaults.
// Every key in this file must be added to locales/en.ts, and the zh / zh-TW
// files must mirror them (their `Record<DictKey, string>` annotation enforces
// this at compile time).
//
// Once merged into locales/en.ts, delete this file.

// ── RepoPicker ────────────────────────────────────────────────────────────────

export const repoPickerKeys = {
  "repoPicker.searchPlaceholder": "Search repositories...",
  "repoPicker.searchLabel": "Search repositories",
  "repoPicker.noSearchResults": "No matching repositories",
  "repoPicker.noRepos": "No repositories",
  "repoPicker.connectHint": "Connect a GitHub account to see your repositories",
  "repoPicker.openRepo": "Open repository",
  "repoPicker.open": "Open",
  "repoPicker.cloneRepo": "Clone repository",
  "repoPicker.cloning": "Cloning...",
  "repoPicker.clone": "Clone",
};

// ── RepoSetupProgress ─────────────────────────────────────────────────────────

export const repoSetupKeys = {
  "repoSetup.settingUp": "Setting up {name}",
  "repoSetup.stageCloning": "Cloning repository",
  "repoSetup.stageDetecting": "Detecting project type",
  "repoSetup.stageInit": "Initializing workspace",
  "repoSetup.stageReady": "Ready",
  "repoSetup.retry": "Retry",
  "repoSetup.clonedTo": "Cloned to",
};

// ── BackgroundTasksPanel ──────────────────────────────────────────────────────

export const backgroundTasksKeys = {
  "backgroundTasks.title": "Background Tasks",
  "backgroundTasks.regionLabel": "Background tasks — {count} active",
  "backgroundTasks.allCompleted": "All background tasks completed",
  "backgroundTasks.activeRunning": "{n} background tasks running",
  "backgroundTasks.watching": "Watching for tasks...",
  "backgroundTasks.empty": "No background tasks running",
  "backgroundTasks.closePanel": "Close background tasks panel",
  "backgroundTasks.dismissError": "Dismiss error",
  "backgroundTasks.taskList": "Background task list",
  "backgroundTasks.taskListLabel": "Background task list",
  "backgroundTasks.peekAction": "Peek {label}",
  "backgroundTasks.sendAction": "Send message to {label}",
  "backgroundTasks.killAction": "Kill {label}",
  "backgroundTasks.detailsLabel": "Details for {label}",
  "backgroundTasks.status": "Status",
  "backgroundTasks.statusPending": "pending",
  "backgroundTasks.statusWaiting": "waiting",
  "backgroundTasks.statusRunning": "running",
  "backgroundTasks.statusDone": "done",
  "backgroundTasks.statusFailed": "failed",
  "backgroundTasks.toolsCalled": "Tools called",
  "backgroundTasks.lastTool": "Last tool",
  "backgroundTasks.started": "Started",
  "backgroundTasks.startedAgo": "{time} ago",
  "backgroundTasks.worker": "Worker",
  "backgroundTasks.duration": "Duration",
  "backgroundTasks.durationSeconds": "{n}s",
  "backgroundTasks.reasoning": "Reasoning",
  "backgroundTasks.remoteLabel": " (remote)",
  "backgroundTasks.remoteWorker": "remote worker",
  "backgroundTasks.runningOn": "Running on {worker}",
  "backgroundTasks.toolCount": "{n} tool",
  "backgroundTasks.toolCountPlural": "{n} tools",
  "backgroundTasks.footerStatus": "{active} active, {done} done, {failed} failed",
  "backgroundTasks.polling": "polling {ms}ms",
  "backgroundTasks.hintNavigate": "navigate",
  "backgroundTasks.hintPeek": "peek",
  "backgroundTasks.hintSend": "send",
  "backgroundTasks.hintKill": "kill",
  "backgroundTasks.hintClose": "close",
};

// ── RemotesPanel ──────────────────────────────────────────────────────────────

export const remotesKeys = {
  "remotes.title": "Remote Workers",
  "remotes.addWorker": "Add remote worker",
  "remotes.closePanel": "Close remote workers panel",
  "remotes.dismissError": "Dismiss error",
  "remotes.editForm": "Edit remote worker",
  "remotes.addForm": "Add remote worker",
  "remotes.formName": "Name",
  "remotes.formHost": "Host",
  "remotes.formPort": "Port",
  "remotes.formMaxConcurrent": "Max Concurrent",
  "remotes.formToken": "Token",
  "remotes.formTls": "TLS enabled",
  "remotes.formTokenPlaceholder": "auth token",
  "remotes.formTokenUnchanged": "(unchanged if empty)",
  "remotes.save": "Save",
  "remotes.add": "Add",
  "remotes.cancel": "Cancel",
  "remotes.loadingWorkers": "Loading workers...",
  "remotes.empty": "No remote workers configured.",
  "remotes.addFirst": "Add your first worker",
  "remotes.workerList": "Remote worker list",
  "remotes.secure": "Secure",
  "remotes.insecure": "Insecure",
  "remotes.lastSeen": "last seen {time}",
  "remotes.jobsActive": "{active} of {max} jobs active",
  "remotes.capabilities": "Capabilities",
  "remotes.testConnection": "Test connection to {name}",
  "remotes.testOk": "OK",
  "remotes.testFailed": "Failed",
  "remotes.testResult": "{status}: {latency}",
  "remotes.edit": "Edit {name}",
  "remotes.remove": "Remove {name}",
  "remotes.footerOnline": "{online} of {total} online",
  "remotes.footerActiveJobs": "{count} active jobs",
  "remotes.connectionFailed": "Connection failed",
};

// ── WorkflowCard ──────────────────────────────────────────────────────────────

export const workflowKeys = {
  "workflow.stageStatusRunning": "Running",
  "workflow.stageStatusCompleted": "Completed",
  "workflow.stageStatusFailed": "Failed",
  "workflow.stageStatusPending": "Pending",
  "workflow.voteMajority": "Majority",
  "workflow.voteUnanimous": "Unanimous",
  "workflow.voteWeighted": "Weighted",
  "workflow.stagesCount": "{n} stages",
  "workflow.statusSummary": "{done} done / {running} running",
  "workflow.statusSummaryFailed": "{done} done / {running} running / {failed} failed",
  "workflow.totalSubagents": "Total subagents",
  "workflow.estimatedTokens": "Estimated tokens",
  "workflow.stageAgents": "{n} agents",
  "workflow.stageTokens": "~{n} tok",
  "workflow.fanOut": "Fan-out: {n}x",
  "workflow.vote": "Vote: {strategy}",
};

// ── SpeculativeResultsCard ────────────────────────────────────────────────────

export const speculativeKeys = {
  "speculative.title": "Speculative Execution",
  "speculative.parallelRuns": "{n} parallel runs",
  "speculative.confirmed": "CONFIRMED",
  "speculative.uncertain": "UNCERTAIN",
  "speculative.confirmedRuns": "{count}/{total} runs",
  "speculative.uncertainRuns": "{count}/{total} runs",
  "speculative.totalCost": "Total: ${cost}",
  "speculative.vsSingle": "vs Single: ${cost}",
  "speculative.totalTokens": "{n} total tok",
  "speculative.confirmedTitle": "CONFIRMED Runs",
  "speculative.uncertainTitle": "UNCERTAIN Runs",
  "speculative.inConsensus": "{count}/{total} in consensus",
  "speculative.needReview": "{count}/{total} need review",
  "speculative.verdictConfirmed": "CONFIRMED",
  "speculative.verdictPlausible": "PLAUSIBLE",
  "speculative.verdictRefuted": "REFUTED",
  "speculative.verdictUncertain": "UNCERTAIN",
  "speculative.inConsensusBadge": "IN CONSENSUS",
  "speculative.confidence": "{n}% confidence",
};

// ── VotingBreakdown ───────────────────────────────────────────────────────────

export const votingKeys = {
  "voting.title": "Voting Breakdown",
  "voting.totalVotes": "{n} total votes",
  "voting.thresholdType": "{type} threshold",
  "voting.thresholdLabel": "Threshold: {count} votes ({pct}%)",
  "voting.passed": "PASSED",
  "voting.notMet": "NOT MET",
  "voting.noVotes": "No votes recorded",
  "voting.votes": "{n} votes",
  "voting.legendThreshold": "threshold",
  "voting.majority": "majority",
  "voting.unanimous": "unanimous",
};

// ── WorkflowPanel ─────────────────────────────────────────────────────────────

export const workflowPanelKeys = {
  "workflowPanel.tabWorkflow": "Workflow",
  "workflowPanel.tabSpeculative": "Speculative",
  "workflowPanel.tabVoting": "Voting",
  "workflowPanel.autoRefresh": "Auto-refresh",
  "workflowPanel.noActiveWorkflow": "No active workflow",
  "workflowPanel.emptyDescription":
    "Start a multi-agent workflow with fan-out and voting to see progress, speculative results, and voting breakdowns here.",
  "workflowPanel.press": "Press",
  "workflowPanel.toOpen": "to open",
  "workflowPanel.noWorkflowData": "No workflow data available",
  "workflowPanel.noSpeculativeData": "No speculative results yet",
  "workflowPanel.noVotingData": "No voting data available",
};

// ── Aggregate: all keys flattened into one record (for merging into en.ts) ───

export const i18nRepo = {
  ...repoPickerKeys,
  ...repoSetupKeys,
  ...backgroundTasksKeys,
  ...remotesKeys,
  ...workflowKeys,
  ...speculativeKeys,
  ...votingKeys,
  ...workflowPanelKeys,
};
