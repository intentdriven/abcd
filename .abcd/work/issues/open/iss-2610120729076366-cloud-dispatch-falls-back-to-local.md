---
schema_version: 1
id: "iss-2610120729076366"
slug: "cloud-dispatch-falls-back-to-local"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "release preparation with cloud sessions, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "cloud and remote lane dispatch"
remedy: "Make any abcd path that dispatches work to a cloud or remote runner check that the runner is reachable and refuse (or ask) instead of starting locally; and have abcd's guidance for agent worktrees name the machine store, so a harness default under the checkout is not used"
---

A request to run lanes as cloud sessions fell back to local sub-agents without saying so: on 2026-10-12 three lanes meant for the cloud started on this machine, inside the requesting checkout (under its .claude/worktrees/), and other sessions only learned of it when the requesting session noticed and corrected its own announcement. A run that cannot reach the place it was asked to run in must say so before it starts (principle loud-staging).
