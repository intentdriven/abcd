---
schema_version: 1
id: "iss-53"
slug: "ears-unwanted-behaviour-rows"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
found_at: ".abcd/development/brief/04-surfaces"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: adopt EARS If/Then unwanted-behaviour clause on mutating verb surface rows). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Every mutating verb's brief surface row carries at least one If/Then unwanted-behaviour clause in the EARS pattern (If <unwanted condition>, then the system shall <refusal or recovery>), making the refusal a first-class, testable spec item before any code exists. The convention serves specification completeness: unwanted-behaviour handling is the dominant omission class in specs, and a row that only describes the happy path leaves the fail-closed behaviour to be invented at implementation time. This feeds the iss-29 fail-closed test convention, which needs a spec-level clause to test against. Detector/acceptance: a surface-row lint flags any mutating verb whose row lacks an If/Then clause, and each clause maps to at least one fail-closed test.