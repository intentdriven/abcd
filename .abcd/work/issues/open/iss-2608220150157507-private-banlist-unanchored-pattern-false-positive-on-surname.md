---
schema_version: 1
id: "iss-2608220150157507"
slug: "private-banlist-unanchored-pattern-false-positive-on-surname"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "abcdev-site facilitation session 2026-08-22"
found_at: ".abcd/.work.local/private-names.txt (machine-local; key only, content withheld by design)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): The sources-sync emitter lives outside this binary: bring it into abcd so anchoring can be fixed here, or close this record here?"
---

The machine-local private name-guard blocked a commit on the entry its hook reports as entry-14: the unanchored pattern matches inside an unrelated academic author's surname in rendered bibliography content (the site prototype), a false positive that keeps the behavioural spec out of the shared tree. The fix site is the generator, not the file: the pattern sits inside the generated abcd-sources block of the private-names store ("do not edit between markers" — a hand-anchored edit is overwritten by the next sources sync), so the sources-sync that emits the block must word-boundary-anchor every short name pattern it writes at emit time. Until that ships the prototype lives in the local scratch tier and the plan documents the detour (decision confirmed 2026-08-22: status quo, no guard weakening, no hand-edits to the generated block; the prototype is committed into research/abcdev-site/ in a follow-up once the anchoring lands). Fix the detector, not the finding