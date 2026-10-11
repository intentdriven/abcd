---
schema_version: 1
id: "iss-2610020726165588"
slug: "bare-abcd-ahoy-remote-prints-the-sub"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/consolidate_test.go"
remedy: "No code change: keep bare ahoy remote listing its sub-verbs with exit 0, as the BT1/H6 rulings, iss-2609251324599468's resolution and the v0.12.0 upgrade guide direct, and close this record as wontfix citing them; restoring an exit-2 answer would re-break the shipped, documented v0.12.0 surface."
wontfix_reason: "The behaviour is the documented one: itd-2609212130136102's scope bounds the moved-spelling answers to one release, rulings BT1 and H6 removed them in the breaking v0.12.0 cut, iss-2609251324599468 resolved that removal (bare ahoy remote lists its sub-verbs, commit 0c1b6b863), docs/how-to/upgrade-to-v0.12.0.md documents exit 0, and TestBareIdentityAndAhoyRemoteListTheirSubVerbs pins it. An exit-2 answer would break the shipped, documented surface."
---

Bare abcd ahoy remote prints the sub-verb's help and exits 0, while the shipped itd-2609212130136102 acceptance criterion ac-1 says bare ahoy remote answers naming --remote and exits non-zero, and its audit evidence still cites the markMoved stub in internal/surface/cli/cli.go that the v0.12.0 cycle removed. Read against the record this is not a regression: the intent's scope bounds the moved-spelling answers to one release, the product thinker's rulings BT1 and H6 (2026-09-29) removed them in the breaking v0.12.0 cut, iss-2609251324599468's resolution says bare ahoy remote lists its sub-verbs instead of answering with a successor, docs/how-to/upgrade-to-v0.12.0.md documents that it lists them and exits 0, and TestBareIdentityAndAhoyRemoteListTheirSubVerbs pins it.

## Grounds

- declined: the shipped v0.12.0 surface is the ruled one; a ruling or upgrade guide that keeps a bare answer naming --remote past v0.11.x would show it wrong
