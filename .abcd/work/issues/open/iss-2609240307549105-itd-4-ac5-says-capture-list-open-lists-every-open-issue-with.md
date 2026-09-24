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
---

itd-4 AC5 says `capture list --open` lists every open issue with id, slug, severity and a one-line summary. The human render prints id, status, severity and slug and no summary (internal/surface/cli/cli.go, the list render's Fprintf); only `--json` carries the body. spc-6's AC5 pin, TestCaptureListOpenRendersIssueFields, asserts the --json shape alone, so the shipped criterion is covered on one of its two surfaces. Found during the fidelity audit of itd-4 (receipt rcp-2662745d5344), criterion ac-5.
