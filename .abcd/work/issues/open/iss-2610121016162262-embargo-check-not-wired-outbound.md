---
schema_version: 1
id: "iss-2610121016162262"
slug: "embargo-check-not-wired-outbound"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of abcd embargo, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/check.go"
remedy: "Wire embargo.Check into the commit-msg hook, lint outbound, launch ship and the implement loop's pull-request bodies in the next rung, and take the triage reasons, and the advisory ids where gh allows, from stdin rather than argv."
---

Nothing calls embargo.Check but the abcd embargo check command: the commit-msg hook, abcd lint outbound, abcd launch ship and the implement loop's pull-request bodies all publish text without running it, so an unpublished advisory's id, reserved CVE id or title reaches public history unless the person remembers to run the check by hand. Separately, advisory ids travel in gh's argv (the REST path names the advisory id), and the triage --reason, --exists-without and --cannot-help text travels in abcd's own argv, where any local account can read them from the process list while the call runs; the update bodies already go to gh on stdin.
