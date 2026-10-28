---
schema_version: 1
id: "iss-2610121056562293"
slug: "when-git-will-not-name-the-repository"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "fix for iss-2610110432159652, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question.go"
remedy: "In questionGate, when a labelled question passes the limits but the root lookup failed in a repo-shaped tree, write one stderr line saying the badge and the open-question mark were not recorded, keeping the exit code the limits decided"
---

When git will not name the repository root inside a checkout, a question that carries abcd's label is still checked against the field limits, but if it passes, the guard's question check cannot set the status-line badge or mark the question open, and it says nothing about that: the person and the transcript cannot tell the bookkeeping was skipped. The unlabelled case now says so (iss-2610110432159652); the labelled case is still silent.
