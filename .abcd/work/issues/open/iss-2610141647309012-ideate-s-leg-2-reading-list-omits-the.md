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
remedy: "Deliver itd-2610141646454347: leg 2 reads the private verdict store and a private verdict is citable by its own key."
---

ideate's leg-2 reading list omits the private verdict store (~/.abcd.noindex/ideas/<root-sha>/), and ideate record resolves repository ids only, so a directly relevant prior private verdict can be neither found by the list nor cited (lab finding F-5).
