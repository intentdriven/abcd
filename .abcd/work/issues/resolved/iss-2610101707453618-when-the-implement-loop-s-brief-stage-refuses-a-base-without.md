---
schema_version: 1
id: "iss-2610101707453618"
slug: "when-the-implement-loop-s-brief-stage-refuses-a-base-without"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "v0.13.4 brief-drift pass, 15-prepare-this-repo finding"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/brief.go"
remedy: "Name /abcd:prepare-this-repo in the remedy through one shared constant used by both brief paths, with a test per path watched to fail first."
resolution: "both brief refusals name /abcd:prepare-this-repo through one shared remedy constant"
impact: fix
---

When the implement loop's brief stage refuses a base without AGENTS.md, its remedy tells the person to run `abcd prepare-this-repo`, a binary verb that does not exist (it exits 2 as an unknown command); prepare-this-repo is the plugin command /abcd:prepare-this-repo. Both the intent brief (brief.go) and the issue brief (issuebrief.go) carry the same wrong text.
