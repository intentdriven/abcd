---
schema_version: 1
id: "iss-165"
slug: "the-grill-interview-capability-should-follow-the-basics-buil"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-105 grill session"
found_at: "commands/abcd"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Should grill delegate to an installed external interview skill when one is present, falling back to the built-in?"
---

The grill/interview capability should follow the basics-built-in, SOTA-delegated pattern: when an external state-of-the-art interviewing skill is installed in the host (the maintainer names Matt Pocock's grill skill), abcd's grill delegates to it; when absent, abcd offers its own comprehensive built-in interview instead. Same stance as the oracle boundary: the external skill is an opt-in dependency, never a requirement.