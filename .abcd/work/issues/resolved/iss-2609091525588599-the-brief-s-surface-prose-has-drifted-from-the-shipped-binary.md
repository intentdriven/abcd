---
schema_version: 1
id: "iss-2609091525588599"
slug: "the-brief-s-surface-prose-has-drifted-from-the-shipped-binary"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief"
resolution: "All 246 discrepancies the full-tier cross-check returned are corrected across the pinned chapters and the Direction B surfaces, verified against the code or the binary's own help rather than against another document, with four findings rejected on re-derivation as stale or wrong and the better finding recorded underneath each."
impact: internal
---

The release gate's semantic cross-check ran at full tier against the v0.8.0 content commit and returned 246 discrepancies between the brief's surface prose and what the binary and the tree actually do. All thirty-seven checkers reported: every pinned chapter and every real surface. The classes are one hundred and thirty-eight false claims, forty-two undocumented surfaces, thirty fictional layouts, twenty-seven stale counts and nine criterion violations, and the weight sits in the chapters that count or enumerate rather than describe: the verification matrix at thirty-seven, the configuration chapter at twenty-three, the naming constraints at eighteen, prompt quality at fourteen and the build sequence at twelve. The findings are specific rather than stylistic. The ahoy chapter states that every sub-verb also ships on the CLI, where the status sub-verb does not exist there and the chapter's own machine-checked table already omits it, so the prose contradicts the table beside it; it points a reader at an ahoy-state.json shape that exists nowhere in the tree and cites a chapter that documents no such shape; it says a remote apply that changed nothing exits non-zero, where two such statuses exit clean and one of them is the idempotent re-run the same paragraph promises. The merge-hygiene block is read from the forge, emitted in the JSON envelope and written into the committed mirror, and its name appears in no markdown file in the repository. Almost none of this arrived with the release under cut: it is accumulated drift, surfaced now because a breaking release is the impact class that requires both directions over the whole chapter list, and because nothing else runs this comparison. The maintainer has ruled that the release holds until the record matches the software. Detector: the cross-check at full tier returns no discrepancy for a chapter whose claims a reader can check against the binary, and the run that gates a release is the one that says so.

## Grounds

- pursued: we expect a chapter's surface claims to survive a reader checking them against the shipped binary, so the design record can be built from rather than merely read; a cross-check rerun that returns a discrepancy in a corrected chapter, or a correction that made a true claim false, would show it wrong.
