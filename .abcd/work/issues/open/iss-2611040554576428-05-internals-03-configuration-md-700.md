---
schema_version: 1
id: "iss-2611040554576428"
slug: "05-internals-03-configuration-md-700"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Limit the sentence to automatic draining and point at intent audit --owed for the on-demand path."
refines: [iss-2610262038166170]
---

05-internals/03-configuration.md:700 says no run seam ships, so nothing drains today. abcd intent audit --owed ships and its help says it lists or drains owed fidelity reviews; 05-internals/01-agents.md:153-154 says it drains the owed set. What is staged is automatic draining at the seam's iteration boundary.
