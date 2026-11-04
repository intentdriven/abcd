---
schema_version: 1
id: "iss-2611040554569602"
slug: "05-internals-03-configuration-md-574-578"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Document interview.redrafts beside interview.list, its layer and its refusal."
---

05-internals/03-configuration.md:574-578 calls the interview's settings one key, interview.list, and :9-10 and :669 speak of that key alone. internal/core/layered/interview.go:23,34 also reads interview.redrafts, how many times a refused question is redrafted, the repository's to set and refused from the machine layer. No file under .abcd/development/brief/ names it.
