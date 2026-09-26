---
schema_version: 1
id: "iss-2609261636083314"
slug: "the-build-loop-s-brief-step-reads-the-intent-the-spec-agents"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-loop2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/brief.go"
---

The build loop's brief step reads the intent, the spec, AGENTS.md, the decision log and the cited ADRs from the lane worktree's working tree while the brief labels them as read at the lane's base commit, so a brief re-rendered after the implementer edited or committed in the worktree quotes the implementer's tree under the base's label. They should be read from the base commit's objects, size-capped, so the brief always renders the base it names.
