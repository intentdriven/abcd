---
schema_version: 1
id: "iss-2610262037323830"
slug: "the-build-brief-says-the-no-host-runner"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 730 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:539"
remedy: "none (filed automatically)"
related_intents: [itd-2610222215485265]
finding_ref: cfk-f8f437dafca4cea6
---

The build brief says the no-host runner.fallback_host path waits on the ADR owed by the build intent's decision 6; the drive draft records the same reversal in an ADR of its own

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:539` — "is the process driver's reversal of the host-delegated boundary, and waits on the ADR decision 6 of the intent owes"
- End B: `.abcd/development/intents/drafts/itd-2610222215485265-abcd-drain-run-from-a-plain-terminal.md:78` — "The reversal of the host-delegated boundary it carries is recorded as adr-2610222222326104, proposed."

The build chapter says that sending a host-routed role to `runner.fallback_host` reverses the host-delegated boundary. It says that path waits on the ADR owed by decision 6 of the build intent, the `--auto-plan` record. The drive draft uses the same no-host path and records the same reversal as adr-2610222222326104, proposed. Two records now claim the decision that unlocks the no-host path, and the corpus does not say which one governs it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 730.
