---
schema_version: 1
id: "iss-2608220150157507"
slug: "private-banlist-unanchored-pattern-false"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "abcdev-site facilitation session 2026-08-22"
found_at: ".abcd/.work.local/private-names.txt (machine-local; key only, content withheld by design)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): The sources-sync emitter lives outside this binary: bring it into abcd so anchoring can be fixed here, or close this record here?"
resolution: "Already fixed at the lane base. The emitter now lives in abcd: abcd source sync-banlist (itd-76) writes the generated sources block of the private store, and every phrase it emits is neighbour-bounded (internal/core/banlist/generated.go:97-102 and :132, landed in 92a7a1a76, carried by v0.11.1): a match needs a line edge or a byte that is not an ASCII letter or digit on both sides, so a short name can no longer match inside an unrelated longer surname. The pre-commit guard refreshes that block from this binary (.githooks/pre-commit, the itd-76 section), so the external generator the record blamed no longer writes it. Committing the site prototype into research/abcdev-site/ was waiting on this and is now a separate follow-up act."
impact: fix
shipped_in: v0.11.1
resolved_by:
  commit: "92a7a1a76"
---

The machine-local private name-guard blocked a commit on the entry its hook reports as entry-14: the unanchored pattern matches inside an unrelated academic author's surname in rendered bibliography content (the site prototype), a false positive that keeps the behavioural spec out of the shared tree. The fix site is the generator, not the file: the pattern sits inside the generated abcd-sources block of the private-names store ("do not edit between markers" — a hand-anchored edit is overwritten by the next sources sync), so the sources-sync that emits the block must word-boundary-anchor every short name pattern it writes at emit time. Until that ships the prototype lives in the local scratch tier and the plan documents the detour (decision confirmed 2026-08-22: status quo, no guard weakening, no hand-edits to the generated block; the prototype is committed into research/abcdev-site/ in a follow-up once the anchoring lands). Fix the detector, not the finding

## Grounds

- pursued: a generated sources block carries only neighbour-bounded patterns, so the entry-14 false positive cannot recur; shown wrong if a pattern sync-banlist emits matches inside a longer word
