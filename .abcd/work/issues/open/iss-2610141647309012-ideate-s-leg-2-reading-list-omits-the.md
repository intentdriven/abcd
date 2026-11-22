---
schema_version: 1
id: "iss-2610141647309012"
slug: "ideate-s-leg-2-reading-list-omits-the"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "lab lab-261014142726-9c4d759 (a hand-run of /abcd:ideate on one external write-up, 2026-10-14)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Deliver itd-2610141646456128: leg 2 reads the private verdict store, and a private verdict is citable by its key from another private verdict."
related_intents: [itd-2610141646456128]
---

ideate's leg-2 reading list omits the private verdict store (~/.abcd.noindex/ideas/<root-sha>/), and ideate record resolves repository ids only, so a directly relevant prior private verdict can be neither found by the list nor cited (lab finding F-5).

## Evidence 2026-11-22

Read at the default branch on 22 November 2026, this issue is half fixed.

- Fixed: the reading list. Leg 2 of `commands/ideate.md` now reads this
  machine's private verdicts in `~/.abcd.noindex/ideas/<root-sha>/`, and tells the
  reader never to quote one into a public verdict.
- Still open: citing. A hit is cited by record id, and a private verdict has no
  id the page lets another private verdict cite it by. That half is
  itd-2610141646456128, which is still a draft.

So the issue is not stale and is not resolved here: its remedy stands for the
citing half alone.
