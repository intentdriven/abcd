---
schema_version: 1
id: "iss-2610102021042657"
slug: "once-the-drain-lane-for-iss-2610090642392144-lands-this"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "abcd-60 drain run 2026-10-10, lane for iss-2610090642392144"
origin: researcher-authored
production_mode: hand-written
found_at: "the bundled default rules (internal/core/rules/defaults)"
remedy: "Waits on a person's ruling: should managed repositories be taught the outside-every-working-tree rule too? If yes, update the bundled CONCURRENCY rule and the-users-directory-is-theirs.md to match AGENTS.md, and have the itd-193 and adr-2609091014087993 archive examples name the scratchpad or the abcd home's store as the target."
---

Once the drain lane for iss-2610090642392144 lands, this repository's AGENTS.md and .abcd/rules.json say a verifier's copy goes outside every working tree, never under .abcd/.work.local/scratch/. The bundled default CONCURRENCY rule every managed repository receives and the principle the-users-directory-is-theirs.md still allow it in the checkout's .abcd/.work.local/ tier, and itd-193 and adr-2609091014087993 quote the archive command without saying where the copy goes. Changing the bundled rule changes what every managed repository is taught, so that lane left it alone.
