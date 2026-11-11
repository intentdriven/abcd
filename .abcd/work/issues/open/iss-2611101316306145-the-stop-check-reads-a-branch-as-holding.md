---
schema_version: 1
id: "iss-2611101316306145"
slug: "the-stop-check-reads-a-branch-as-holding"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Have land set the branch's upstream to the remote branch it pushed, and have the owed read (unpushed in internal/core/autonomy/owedread.go) count a branch as pushed when the same-named remote branch holds its head, whatever its upstream says; a test cuts a branch tracking the default branch, pushes it by name and finds it not owed."
---

The stop check reads a branch as holding unpushed commits by comparing it with its configured upstream, so a branch cut tracking origin/main stays owed after land has pushed it under its own name, and land pushes without setting the upstream. On 10 Nov at 07:09 the check still named lane-p-checklist as having commits not pushed after land had pushed it, until the upstream was set to origin/lane-p-checklist by hand.
