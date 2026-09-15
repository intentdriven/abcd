---
schema_version: 1
id: "iss-2608292038186663"
slug: "nine-show-toplevel-wrappers-one-validated"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "v0.6.9-security-pass"
found_at: "internal/gitutil/repo.go"
related_issues: ["iss-2608291924452604"]
---

v0.6.9 combined ruthless review: nine wrappers around 'git rev-parse --show-toplevel' exist and only one validates the answer. internal/core/site/outdir.go checkoutRoot holds the output to a single absolute line that contains the invoking directory; the other eight — internal/surface/cli/cli.go, internal/surface/cli/banlist.go, internal/core/capture/roots.go, internal/core/ahoy/remote.go, internal/core/banlist/worktree.go, cmd/record-lint and cmd/scaffold-sync, plus the site one — each re-derive the call. Proposal: one gitutil.Toplevel(dir) carrying the single-absolute-line-containing-the-invoking-directory rule, with every wrapper calling it. This REFINES iss-2608291924452604 (the ledger has no typed link, so the relation is stated here and in related_issues).
