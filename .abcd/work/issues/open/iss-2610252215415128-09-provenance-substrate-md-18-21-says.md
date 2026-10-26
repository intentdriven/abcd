---
schema_version: 1
id: "iss-2610252215415128"
slug: "09-provenance-substrate-md-18-21-says"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/09-provenance-substrate.md"
remedy: "Name memory.detectLicence and say the Go port reads package.json and LICENSE files but no TOML manifest."
---

09-provenance-substrate.md:18-21 says licence detection is 'implemented by internal/core/provenance detect_licence' and reads Cargo.toml and pyproject.toml; the detector is memory.detectLicence (internal/core/memory/provenance.go:308), internal/core/provenance is the disclosure vocabulary, and the code says TOML manifests are not parsed (provenance.go:305)
