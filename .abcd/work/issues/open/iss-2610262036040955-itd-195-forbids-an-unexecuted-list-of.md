---
schema_version: 1
id: "iss-2610262036040955"
slug: "itd-195-forbids-an-unexecuted-list-of"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 621 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-195-a-claim-about-how-the-code-behaves-is.md:21"
remedy: "In 24-decide.md replace the list of which readers call recordid.CanonADRID and which carry their own derivation with a cited parity test over all derivations, or state only what the code in hand does."
related_intents: [itd-195]
finding_ref: cfk-5546cd9ed7243e31
---

itd-195 forbids an unexecuted list of which readers do what; 24-decide lists which readers call the ADR derivation and which carry their own, agreeing by inspection.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-195-a-claim-about-how-the-code-behaves-is.md:21` — "is backed by something that runs, or it is not written."
- End B: `.abcd/development/brief/04-surfaces/24-decide.md:144` — "Three readers still carry their own"

itd-195's rule is that a statement of how the codebase behaves is backed by something that runs, or it is not written, and it names a list of the readers and what each does as the forbidden shape. 24-decide enumerates the readers that call recordid.CanonADRID and recordid.ADRFileID, says three readers still carry their own derivation, and notes the derivations agree by inspection. The brief carries exactly the list the discipline says must not be written unless a test establishes it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 621.
