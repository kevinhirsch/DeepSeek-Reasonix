# Dependency Graph

> What blocks what. Defines the critical path and parallelization opportunities.

## Issue → Blocks → Blocked By

```
16 Quality Architecture
    BLOCKS: 01, 02, 03, 04, 08, U2, U12, U7
    BLOCKED BY: —

01 Role Prompts
    BLOCKS: 05, G9
    BLOCKED BY: 16

02 DeepSeek Temp
    BLOCKS: —
    BLOCKED BY: 16

03 Cognitive Loop
    BLOCKS: —
    BLOCKED BY: 16

04 Subagent Messenger
    BLOCKS: 05
    BLOCKED BY: 16

05 Workflow Tool
    BLOCKS: 15, G2, G9, U1
    BLOCKED BY: 01, 04

06 Enriched Job View
    BLOCKS: 07
    BLOCKED BY: —

07 Background Panel
    BLOCKS: U5, U14, U8
    BLOCKED BY: 06

08 Effort Calibration
    BLOCKS: —
    BLOCKED BY: 01

09 Remote Bootstrap
    BLOCKS: —
    BLOCKED BY: —

10 Worker Package
    BLOCKS: 11, 12, 15
    BLOCKED BY: —

11 Worker Binary
    BLOCKS: —
    BLOCKED BY: 10

12 Work Queue
    BLOCKS: 15
    BLOCKED BY: 10

13 Docker Sandbox
    BLOCKS: —
    BLOCKED BY: —

14 Remotes Config
    BLOCKS: 15
    BLOCKED BY: —

15 Remote Target
    BLOCKS: —
    BLOCKED BY: 05, 10, 12, 14

G1 web_search
    BLOCKS: G2
    BLOCKED BY: —

G2 deep-research
    BLOCKS: —
    BLOCKED BY: 05, G1

G3 Monitor
    BLOCKS: —
    BLOCKED BY: —

G4 ReportFindings
    BLOCKS: —
    BLOCKED BY: —

G5 Cron/Schedule
    BLOCKS: —
    BLOCKED BY: —

G6 PushNotification
    BLOCKS: —
    BLOCKED BY: —

G7 Task System
    BLOCKS: —
    BLOCKED BY: —

G8 EnterWorktree
    BLOCKS: —
    BLOCKED BY: —

G9 Skills (9)
    BLOCKS: —
    BLOCKED BY: 05, 01

G10 AskUserQuestion
    BLOCKS: —
    BLOCKED BY: —

17 Repo Connection
    BLOCKS: —
    BLOCKED BY: —

18 Prompt Calibration
    BLOCKS: 01, G9, 19
    BLOCKED BY: 16

19 Structural Compensations
    BLOCKS: —
    BLOCKED BY: 18, 01, 04, 05

U1 Crash Recovery
    BLOCKS: —
    BLOCKED BY: 05

U2 Model Fallback
    BLOCKS: —
    BLOCKED BY: 16

U3 Backpressure
    BLOCKS: —
    BLOCKED BY: 05

U5 Debugging
    BLOCKS: —
    BLOCKED BY: 07

U6 Remote Security
    BLOCKS: —
    BLOCKED BY: 10, 12

U7 Config Migration
    BLOCKS: —
    BLOCKED BY: 16

U12 Testing
    BLOCKS: —
    BLOCKED BY: 16
```

## Critical Path

```
16 → 01 → 05 → 15 → (G2 + G9 + U1 + U3)
16 → 04 → 05 → 15
16 → 01 → 08
16 → 03, U2

Independent starters: 06, 09, 13, G1, G3, G4, G5, G6, G7, G8, G10, 17
```

## Parallelization Opportunities

| Group | Issues | Can Run Together? |
|---|---|---|
| Quality base | 16, 01, 02, 03, 08, U2 | 16 first, then 01+02+03+08 parallel |
| Subagent infra | 04, 06 | Parallel with quality base |
| Workflow | 05 | After 01+04 |
| Visibility | 07, 17 | Parallel with workflow |
| Remote | 09+13, then 10+11+12+14, then 15 | Sequential within remote, parallel with others |
| Tool gaps | G1-G10 | All independent of each other |
| Resilience | U1-U15 | Spread across phases as dependencies resolve |
