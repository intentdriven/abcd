---
schema_version: 1
id: "iss-2610241242295613"
slug: "a-tool-install-is-refused-when-the-bin"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "setup redesign build, tools lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/install.go"
remedy: "say which program occupies the name and offer the by-hand steps (ManualText) or an alternative folder the person chooses"
---

a tool install is refused when the bin folder already holds a program of that name that abcd did not install, with no other route offered.
