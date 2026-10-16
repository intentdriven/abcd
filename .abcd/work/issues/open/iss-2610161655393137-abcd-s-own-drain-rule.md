---
schema_version: 1
id: "iss-2610161655393137"
slug: "abcd-s-own-drain-rule"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "product thinker interview 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when.md"
remedy: "Amend abcd's own drain record to drain_severities nitpick, minor, major, critical and drain_security take. Update invariant 19's sentence and the test that holds abcd at the baseline: the bundled baseline stays strict for other projects, while abcd's own record is now loosened, loudly named by the dry run and start. Order each move's eligible issues most urgent first (security, then critical, major, minor, nitpick) ahead of the per-account spread. Tests: a major bug with a remedy is taken first, and a security issue is taken before it."
---

abcd's own drain rule (adr-2609291342092738) holds abcd at the strict baseline, taking nitpick and minor and handing every security issue back, and a test pins it there, so urgent fixes that need no decision wait for a person. The product thinker ruled on 2026-10-16 that abcd's own repository takes every severity and security, most urgent first.
