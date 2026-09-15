---
schema_version: 1
id: "iss-2609021816004079"
slug: "the-committed-preset-file-s-one-entry-per-position-is-keyed"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "itd-2609021003095168 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config/reading-presets.json"
wontfix_reason: "Stale within the same branch: the preset file moved to schema version 2 in the preset-windows change, where each position's entry stands directly and there is no preset-name level, so the key this record names no longer exists in the committed file; the residual question, what a version-1 file's single preset is called, needs no ruling because a version-1 file is an adopter's compatibility shape and not this repository's."
---

The committed preset file's one entry per position is keyed default, a name the implementer chose when the two-operand invocation retired the cold and warm tokens: neither adr-2609021016286571 nor spc-2609021004075744 names the surviving key, and the divergence register, which by its own terms lists every point the materials chose where the documents are open, carries no entry for it. The key is inert at the invocation (a file holding other than one preset is refused), so nothing depends on the word, but the record now carries a name the decision record did not decide. A register entry or a ruling settles it.

## Grounds

- declined: Stale within the same branch: the preset file moved to schema version 2 in the preset-windows change, where each position's entry stands directly and there is no preset-name level, so the key this record names no longer exists in the committed file; the residual question, what a version-1 file's single preset is called, needs no ruling because a version-1 file is an adopter's compatibility shape and not this repository's.
