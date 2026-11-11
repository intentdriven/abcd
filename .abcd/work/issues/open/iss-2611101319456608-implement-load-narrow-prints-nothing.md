---
schema_version: 1
id: "iss-2611101319456608"
slug: "implement-load-narrow-prints-nothing"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_narrow.go"
remedy: "Print a still-running line every few minutes with the elapsed time and the package in progress; a test runs a slow command through the gate and finds a progress line."
---

implement load --narrow prints nothing while its command runs, so an agent cannot tell a slow run from a hung one. On 9 Nov a lane's narrow gate run showed nothing for 33 minutes.
