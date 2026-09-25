---
schema_version: 1
id: "iss-2608270926037088"
slug: "graveyard-evidence-is-terminal-safe-but-not-markdown-safe-an"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/lifeboat/graveyard.go"
resolution: "Finding carries a typed notices field for the binary's cap, shadow and listing-truncation statements, with record paths inside a notice cleaned and Go-quoted; every record-drawn graveyard string passes through the shared prose cleaner (CleanProseLine). graveyard_notices_test.go holds both channels, including the crafted-text ok side."
impact: fix
resolved_by:
  commit: "b5483605"
---

graveyard evidence is terminal-safe but not markdown-safe and untyped: termsafe.Sanitize leaves CommonMark and raw-HTML openers for the layer-3 interpreter to read, and cap or shadow notices share the plain evidence string array with record-influenced text, so a crafted path can forge an omission notice — a typed field or CleanProse pass closes the channel

## Grounds

- pursued: a crafted path or bullet can no longer pass for an omission notice or carry live markup to the layer-3 interpreter; shown wrong if any graveyard builder writes record text into notices or bypasses gvText, which TestGraveyardRecordTextIsMarkdownSafe and TestGraveyardCraftedRecordTextStaysEvidence would catch
