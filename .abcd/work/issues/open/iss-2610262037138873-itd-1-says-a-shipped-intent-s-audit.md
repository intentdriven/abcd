---
schema_version: 1
id: "iss-2610262037138873"
slug: "itd-1-says-a-shipped-intent-s-audit"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 708 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "In itd-1 line 45 replace the deferral and 'populated only when /abcd:intent audit is run by hand' with the shipped flow: a spec close that ships an intent parks an owed review marker and emits the request, the build loop audits and ingests, and a provider-routed close sends and ingests it (commands/intent.md, 34-build.md)."
related_intents: [itd-1]
finding_ref: cfk-0a121e9d2d4adede
---

itd-1 says a shipped intent's Audit Notes come only from an audit run by hand, because the automatic audit at ship time is deferred; the build loop already runs that audit and ingests it when the intent ships.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "Until that lands, a shipped intent's `## Audit Notes` is populated only when `/abcd:intent audit` is run by hand."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:743` — "runs the spec's close (which ships the intent and parks the fidelity receipt) and ingests the verdict of the audit the lane took into that receipt"

itd-1 defers automatic invocation on the planned-to-shipped move and says that until then, Audit Notes are filled only when /abcd:intent audit is run by hand. In 34-build, the landing runs the spec close that ships the intent, then ingests the verdict of the intent-auditor the lane already ran. That is an automatic audit at exactly that move, so 'only when run by hand' cannot also hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 708.
