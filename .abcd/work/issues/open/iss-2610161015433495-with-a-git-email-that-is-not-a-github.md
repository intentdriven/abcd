---
schema_version: 1
id: "iss-2610161015433495"
slug: "with-a-git-email-that-is-not-a-github"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/holder.go"
remedy: "Cache the resolved holder for the run"
related_intents: [itd-2610161016401115]
---

With a git email that is not a GitHub noreply address, every reservation act calls gh api user again, uncached.
