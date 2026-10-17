---
schema_version: 1
id: "iss-2610142025250222"
slug: "receipts-issues-dir-not-contained"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "deferral discipline build"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasereceipts.go"
remedy: "Check issues_dir is a repository-relative path inside the checkout before CheckReleaseReceipts uses it, and refuse naming the setting otherwise."
resolution: "moot: the release's arming now pins the receipt gate's issues_dir to the canonical .abcd/work/issues, as it pins the severity, so the committed config's issues_dir never reaches CheckReleaseReceipts' git status pathspec or the gate's ledger read"
impact: internal
---

abcd launch receipts passes the configured issues_dir to git status as a pathspec without checking it stays inside the repository; the release gate checks containment itself, so an out-of-repository value surfaces only as a raw git error rather than a refusal naming the setting.
