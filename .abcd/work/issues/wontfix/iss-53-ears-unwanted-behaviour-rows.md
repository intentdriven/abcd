---
schema_version: 1
id: "iss-53"
slug: "ears-unwanted-behaviour-rows"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
found_at: ".abcd/development/brief/04-surfaces"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Adopt the EARS If/Then unwanted-behaviour clause on mutating verb surface rows?"
wontfix_reason: "Superseded by itd-2609212113220149 (shipped; ruled 2026-09-21): every verb and sub-verb opens with one sentence naming what it does, what it writes and when it refuses, rendered byte-identically onto the command list, --help and the plugin page, and Phase 8 requires every surface chapter to say what the surface refuses to do. The unwanted-behaviour clause the record asked for is in force in that form; the EARS If/Then syntax was not adopted and would restate the same refusal a second time."
---

Every mutating verb's brief surface row carries at least one If/Then unwanted-behaviour clause in the EARS pattern (If <unwanted condition>, then the system shall <refusal or recovery>), making the refusal a first-class, testable spec item before any code exists. The convention serves specification completeness: unwanted-behaviour handling is the dominant omission class in specs, and a row that only describes the happy path leaves the fail-closed behaviour to be invented at implementation time. This feeds the iss-29 fail-closed test convention, which needs a spec-level clause to test against. Detector/acceptance: a surface-row lint flags any mutating verb whose row lacks an If/Then clause, and each clause maps to at least one fail-closed test.

## Grounds

- declined: the refusal is already a mandatory, generated spec line per verb; this would be wrong if a mutating verb could ship with no refusal stated on its surface
