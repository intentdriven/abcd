---
schema_version: 1
id: "iss-2611251924251514"
slug: "seven-cli-tests-fail-in-a-verifier-copy"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "Skip the two with a named reason where the tree has no .git, and give the five their own routing configuration, so that a verifier copy passes as the checkout does. Grounds: AGENTS.md prescribes that copy for every verifier."
---

Seven tests of internal/surface/cli fail in the copy a verifier is told to work on (git archive HEAD unpacked outside every working tree), with no change involved: TestBootstrapFreshInstallSelfCheck and TestBootstrapReadsARealBuildsClaim run git archive HEAD and the copy has no .git; five TestDisembark tests get a routing line before their JSON. Run on 2026-11-25 by a spec reviewer over an export of this branch; the cause of the five was read and not confirmed by a run.
