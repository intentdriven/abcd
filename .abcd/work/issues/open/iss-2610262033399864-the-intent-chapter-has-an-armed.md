---
schema_version: 1
id: "iss-2610262033399864"
slug: "the-intent-chapter-has-an-armed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 427 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:511"
remedy: "none (filed automatically)"
finding_ref: cfk-212c7a84451f35e8
---

The intent chapter has an armed record-lint rule that checks the promote back-link, while the lint contract lists that check as an unbuilt full-corpus tier.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:511` — "`related_issues_backlink` flags an intent's `related_issues` entry whose issue's `related_intents` omits the intent"
- End B: `.abcd/development/brief/05-internals/06-lint.md:54` — "for example a promote back-link between an intent and its source issue, which needs both files in view"

05-intent names related_issues_backlink as an armed record-lint rule. It flags an intent's related_issues entry when that issue's related_intents leaves the intent out. 06-lint gives that same promote back-link as its example of a check for a full-corpus tier that is a design target and not yet built. The check is either armed in the record gate or not yet built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 427.
