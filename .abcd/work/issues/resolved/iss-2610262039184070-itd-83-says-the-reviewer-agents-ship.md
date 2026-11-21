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
resolution: "itd-83's non-goal now says the reviewer agents' prompts ship as they are, naming no version, so it no longer contradicts the prompt-version log's 0.2.0; a lint-package test holds the draft to that while it stays in drafts/."
impact: internal
resolved_by:
  commit: "735fd6a52bbf417c22dcd8fc55b713716e85dea7"
---

itd-83 says the reviewer agents ship as-is at 0.1.0, but the prompt-quality page says they entered the changelog at 0.2.0

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:84` — "They ship as-is at `0.1.0`."
- End B: `.abcd/development/brief/05-internals/05-prompt-quality.md:109` — "`sota-researcher` enter the changelog at `0.2.0` instead, the bump that first"

itd-83 leaves the reviewer agents it names unchanged at `0.1.0`. The prompt-quality page says `ruthless-reviewer`, `security-reviewer`, `docs-currency-reviewer` and `sota-researcher` entered the prompt-version log at `0.2.0`. None of them carries the version itd-83 names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 852.

## Grounds

- pursued: itd-83 line 84 names no agent version and says 'They ship as they are.'; TestReviewBarDraftNamesNoAgentVersion failing, or the draft carrying 'as-is at `0.1.0`' again, would show it wrong
