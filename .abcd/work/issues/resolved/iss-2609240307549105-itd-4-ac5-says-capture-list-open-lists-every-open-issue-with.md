---
schema_version: 1
id: "iss-2609240307549105"
slug: "itd-4-ac5-says-capture-list-open-lists-every-open-issue-with"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "The human render of capture list now ends each row with a one-line summary: the first non-blank line of the body, sanitised through termsafe and clipped to 80 runes (summaryNote in internal/surface/cli/cli.go). TestCaptureListOpenHumanRenderCarriesSummary pins the human surface alongside the existing --json pin, and was watched fail on the summary-less row before the change. The capture chapter's list criterion names the summary."
impact: fix
resolved_by:
  commit: "d12cc627c"
---

itd-4 AC5 says `capture list --open` lists every open issue with id, slug, severity and a one-line summary. The human render prints id, status, severity and slug and no summary (internal/surface/cli/cli.go, the list render's Fprintf); only `--json` carries the body. spc-6's AC5 pin, TestCaptureListOpenRendersIssueFields, asserts the --json shape alone, so the shipped criterion is covered on one of its two surfaces. Found during the fidelity audit of itd-4 (receipt rcp-2662745d5344), criterion ac-5.

## Grounds

- pursued: every human list row carries its record's first body line; a row with a non-empty body and no ' — ' tail, or a multi-line row, would show it wrong
