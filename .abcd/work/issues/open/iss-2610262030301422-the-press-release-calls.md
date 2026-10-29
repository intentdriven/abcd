---
schema_version: 1
id: "iss-2610262030301422"
slug: "the-press-release-calls"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 9 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "Reword the press release's last sentence in its opening quote (line 5): most verbs render status when bare and mutations take an explicit sub-verb or operand, with the exceptions enumerated in 04-surfaces/README.md under Bare invocation; drop 'any command' and 'the universal abcd convention'. One edit settles the duplicates filed against this sentence."
finding_ref: cfk-c053753d683dafb4
---

The press release calls bare-shows-status a universal convention; the naming page says conformance is partial and several verbs refuse or act instead

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "Bare invocation of any command shows status + suggested next actions; state-mutating actions require an explicit sub-verb (the universal abcd convention)."
- End B: `.abcd/development/brief/02-constraints/04-naming.md:105` — "**Conformance is partial, and the gap is the shipped surface's rather than the discipline's.** Several parents print usage with no state, several verbs refuse or act instead of rendering"

The press release says bare invocation of any command shows status and next actions, and that state-mutating actions require a sub-verb, as the universal abcd convention. The naming page says conformance is partial: several parents print usage with no state, and several verbs refuse or act instead of rendering. Both cannot describe the shipped surface.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 9.
