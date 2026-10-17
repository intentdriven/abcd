---
schema_version: 1
id: "iss-2610120729087253"
slug: "cloud-session-work-trapped-without"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "release preparation with cloud sessions, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-push"
remedy: "Give a session a sanctioned way to hand back unmergeable work, e.g. pushing to a reserved work-in-progress namespace the pre-push hook admits without a receipt and the merge queue never accepts, or exporting a git bundle to a place abcd collects; prove the namespace can never reach main"
related_intents: [itd-2610120738075547]
---

A cloud session whose preflight cannot pass has no way to hand its commits back: the pre-push hook refuses a push without a preflight receipt, and skipping that hook is forbidden, so on 2026-10-12 a finished fix (two commits and a resolved record) stayed trapped in the cloud container and had to be redone locally.
