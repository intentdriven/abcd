---
schema_version: 1
id: "iss-2610262031389232"
slug: "itd-142-pre-populates-a-brownfield"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 157 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:68"
remedy: "In itd-142 (title, press release, scope and acceptance criteria) replace `embark probe` with `disembark probe <source-repo>`, the verb that reports coverage for a repository; `embark probe` takes a lifeboat directory."
related_intents: [itd-142]
finding_ref: cfk-8ff623e5c7d325fb
---

itd-142 pre-populates a brownfield repository from embark probe coverage, but embark probe reads a lifeboat, not a repository

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:68` — "Common entry for both modes: brownfield pre-population from `embark probe`"
- End B: `.abcd/development/brief/01-product/02-context.md:9` — "the `probe <lifeboat-dir>` sub-verb reports what an unpack would write, coverage blanks first, and writes nothing"

itd-142's common entry draws brownfield pre-population from embark probe coverage. The context chapter defines embark's probe as taking a lifeboat directory and reporting what an unpack would write. A brownfield repository with no lifeboat gives embark probe nothing to read, so the entry depends on a verb that cannot supply the coverage it needs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 157.
