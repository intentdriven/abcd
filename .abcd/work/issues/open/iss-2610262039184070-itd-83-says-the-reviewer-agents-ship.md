---
schema_version: 1
id: "iss-2610262039184070"
slug: "itd-83-says-the-reviewer-agents-ship"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 852 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:84"
remedy: "In itd-83 drop the version number from 'They ship as-is at `0.1.0`' (say 'They ship as they are')."
related_intents: [itd-83]
finding_ref: cfk-3a1e986ef09eb554
---

itd-83 says the reviewer agents ship as-is at 0.1.0, but the prompt-quality page says they entered the changelog at 0.2.0

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:84` — "They ship as-is at `0.1.0`."
- End B: `.abcd/development/brief/05-internals/05-prompt-quality.md:109` — "`sota-researcher` enter the changelog at `0.2.0` instead, the bump that first"

itd-83 leaves the reviewer agents it names unchanged at `0.1.0`. The prompt-quality page says `ruthless-reviewer`, `security-reviewer`, `docs-currency-reviewer` and `sota-researcher` entered the prompt-version log at `0.2.0`. None of them carries the version itd-83 names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 852.
