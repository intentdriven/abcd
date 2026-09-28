---
schema_version: 1
id: "iss-2609280932480608"
slug: "an-abcd-owned-dangling-path-entry-that-sits-earlier-on-path"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/store.go"
---

An abcd-owned dangling PATH entry that sits EARLIER on PATH than the owned copy ahoy adopts is never repaired: install acts only on the adopted target (effectiveBinTarget skips dangling entries), so the symlink.dangling gap the earlier entry raises stays in Remaining on every run and ahoy install can never report clean. The install-time shadow note for it is also wrong in two ways: it says the dangling link 'is what runs when you type abcd' (a link that resolves to nothing runs nothing; the shell skips it) and ends 'abcd never clobbers a binary it does not own' about an entry it classifies as its own. Reproduced (lane drainH, 2026-09-28): a stranded plugin-update link in one PATH directory ahead of a one-liner-installed owned copy in ~/.local/bin, on a warm cache and on a cold one; the warm-cache half is the same at base 0f9d652a. Needed: decide whether install removes an owned dangling entry that is not its target (it destroys nothing, and uninstall already removes owned entries), and give the shadow note owned-entry and dangling-entry wording.
