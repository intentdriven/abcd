---
schema_version: 1
id: "iss-2609280932480608"
slug: "an-abcd-owned-dangling-path-entry-that"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/store.go"
resolution: "ahoy install removes every abcd-owned dangling PATH entry other than its target, with its record, once the target is a working entry of abcd's own (clearStrandedEntries, gated on entryAnswers), so the gap a stranded link ahead of the adopted copy raised no longer stays in Remaining; the shadow note and the dangling gap say a dangling link runs nothing — the shell skips it — instead of 'is what runs', 'a binary it does not own' or 'shadows every later PATH entry'. Unowned dangling links are never removed by this step."
impact: fix
resolved_by:
  commit: "7711d1677"
---

An abcd-owned dangling PATH entry that sits EARLIER on PATH than the owned copy ahoy adopts is never repaired: install acts only on the adopted target (effectiveBinTarget skips dangling entries), so the symlink.dangling gap the earlier entry raises stays in Remaining on every run and ahoy install can never report clean. The install-time shadow note for it is also wrong in two ways: it says the dangling link 'is what runs when you type abcd' (a link that resolves to nothing runs nothing; the shell skips it) and ends 'abcd never clobbers a binary it does not own' about an entry it classifies as its own. Reproduced (lane drainH, 2026-09-28): a stranded plugin-update link in one PATH directory ahead of a one-liner-installed owned copy in ~/.local/bin, on a warm cache and on a cold one; the warm-cache half is the same at base 0f9d652a. Needed: decide whether install removes an owned dangling entry that is not its target (it destroys nothing, and uninstall already removes owned entries), and give the shadow note owned-entry and dangling-entry wording.

## Grounds

- pursued: an install over a stranded owned link ahead of a working owned entry now finishes with nothing Remaining, cold or warm; a run that leaves the stranded link or its record behind, removes an unowned dangling link, or removes one while nothing working stands at the target would show it wrong
