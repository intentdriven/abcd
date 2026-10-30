---
schema_version: 1
id: "iss-2610230643534431"
slug: "a-cutter-whose-copy-pushes-through-an"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "release-hold round 4 (iss-2610222014550692)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawpush.go"
remedy: "Decide how a copy that pushes through an app takes part in the release hold: withdraw nothing and leave the gate to refuse, or vouch its withdrawals another way (a local-tier note of what this copy itself recorded), then build it"
---

A cutter whose copy pushes through an app (a hosted session's installation token) records every withdrawal on a commit an app pushed, so abcd can neither admit nor re-arm any of them: each is named app_pushed once and left for the person to re-arm by hand after the cut. Whether such a copy should withdraw at all, or how its withdrawals could be vouched for, needs a design decision.
