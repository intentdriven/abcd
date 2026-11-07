---
schema_version: 1
id: "iss-2611060823132319"
slug: "a-stop-removed-by-an-age-cleaner-on"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runstop/runstop.go"
remedy: "Say in the refusal that the stop's file is gone rather than lifted, and suggest a shared folder on Linux that no age cleaner tends."
---

A stop removed by an age cleaner on Linux reads as no stop, so new runs can open as well as a run it had not halted, and a halted run's refusal says the stop has lifted though no one lifted it. The v0.16.2 changelog names the run it had not halted and not the new runs.
