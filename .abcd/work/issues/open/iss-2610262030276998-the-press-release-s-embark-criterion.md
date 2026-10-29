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
remedy: "Rewrite the embark acceptance criterion (line 39) and the 'Unpack the lifeboat' scope bullet to the shipped unpack: embark verifies the lifeboat, refuses any conflict, writes the five record families verbatim and re-injects the marker block, with no interview in the write path and no brief README written; the press-release interview that frames the voyage is the host-run step fed by embark lessons."
finding_ref: cfk-406b78614bbcb598
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The press release's embark criterion promises an interview, a brief README and memory/docs/terminology; the embark chapter has no interview and writes only record families and the marker block

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:39` — "the user is presented with the press-release interview as the first interaction, the amended press release becomes the new repo's `.abcd/development/brief/README.md`, and all canonical-location files (specs, ADRs, memory, docs, terminology, principles) land at the locations specified in"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:100` — "Embark is a deterministic Go run: it reads the lifeboat, plans, refuses on any conflict, then writes the record families plus the marker block. No interactive scaffolder and no model sit in the write path."

The press release's acceptance criterion has embark open with the press-release interview, write the amended press release to `.abcd/development/brief/README.md`, and land memory, docs and terminology. The embark chapter says the run is deterministic, with no interactive scaffolder and no model in the write path, and that it writes the record families plus the marker block. The criterion as written cannot pass against the surface as designed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 2.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
