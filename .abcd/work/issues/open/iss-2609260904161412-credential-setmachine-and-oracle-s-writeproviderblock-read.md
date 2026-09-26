---
schema_version: 1
id: "iss-2609260904161412"
slug: "credential-setmachine-and-oracle-s-writeproviderblock-read"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/credential.go"
---

credential.SetMachine and oracle's writeProviderBlock read, modify and rename ~/.abcd/credentials.json and ~/.abcd/config.json with no lock, so concurrent ahoy connect runs lose a key or a provider block while each reports it wrote the file.
