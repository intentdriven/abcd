---
schema_version: 1
id: "iss-2610241242226828"
slug: "at-abcd-ahoy-remote-apply-and-site-setup"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "setup redesign build, tools lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/release.go"
remedy: "apply the tool's compatible range at every gh call site, and offer abcd's copy (or say which copy runs) when the PATH gh is out of range"
---

at abcd ahoy remote apply and site setup any gh on PATH counts as present: only a missing gh is offered abcd's verified copy, so an incompatible older gh is used there without a word, unlike gitleaks at ahoy install.
