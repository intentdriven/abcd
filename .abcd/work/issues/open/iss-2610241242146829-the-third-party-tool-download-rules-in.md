---
schema_version: 1
id: "iss-2610241242146829"
slug: "the-third-party-tool-download-rules-in"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "setup redesign build, tools lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/fetch.go"
remedy: "move the shared download policy into one package both import (e.g. a small internal/core/fetchpolicy), and have update and tools use it"
---

the third-party tool download rules in internal/core/tools/fetch.go are a second copy of the updater's in internal/core/update (allowed hosts, size and time limits, no proxy, CA overrides ignored), because update imports ahoy which imports tools; two copies of a trust policy drift apart.
