---
schema_version: 1
id: "iss-2610262030530560"
slug: "the-press-release-s-criterion-files-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 57 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:43"
remedy: "Rewrite the capture acceptance criterion (line 43) to include the required remedy: running /abcd:capture with the quoted text and a remedy leaves an issue under open/ with frontmatter populated and provenance recorded; a filing without a remedy is refused and writes nothing."
finding_ref: cfk-74b0b9cf959612c5
---

The press release's criterion files an issue from text alone; the capture chapter refuses any new filing without a remedy

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:43` — "a structured issue at `.abcd/work/issues/open/iss-N-<slug>.md` exists with frontmatter populated and source provenance recorded"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:66` — "The remedy has no default and is required: every new issue names the fix it proposes"

01-press-release's criterion says that running /abcd:capture "< text>" leaves a structured issue under open/, while 06-capture says the remedy has no default and is required, so a filing without one is refused with nothing written. Run as the criterion states it, the command writes nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 57.
