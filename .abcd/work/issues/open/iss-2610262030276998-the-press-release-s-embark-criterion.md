---
schema_version: 1
id: "iss-2610262030276998"
slug: "the-press-release-s-embark-criterion"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 2 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:39"
remedy: "none (filed automatically)"
finding_ref: cfk-406b78614bbcb598
---

The press release's embark criterion promises an interview, a brief README and memory/docs/terminology; the embark chapter has no interview and writes only record families and the marker block

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:39` — "the user is presented with the press-release interview as the first interaction, the amended press release becomes the new repo's `.abcd/development/brief/README.md`, and all canonical-location files (specs, ADRs, memory, docs, terminology, principles) land at the locations specified in"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:100` — "Embark is a deterministic Go run: it reads the lifeboat, plans, refuses on any conflict, then writes the record families plus the marker block. No interactive scaffolder and no model sit in the write path."

The press release's acceptance criterion has embark open with the press-release interview, write the amended press release to `.abcd/development/brief/README.md`, and land memory, docs and terminology. The embark chapter says the run is deterministic, with no interactive scaffolder and no model in the write path, and that it writes the record families plus the marker block. The criterion as written cannot pass against the surface as designed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 2.
