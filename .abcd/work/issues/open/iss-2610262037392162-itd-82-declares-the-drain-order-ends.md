---
schema_version: 1
id: "iss-2610262037392162"
slug: "itd-82-declares-the-drain-order-ends"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 739 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-82-drain-ledger-triage.md:49"
remedy: "none (filed automatically)"
related_intents: [itd-82]
finding_ref: cfk-3dee3f366ce2a5e2
---

itd-82 declares the drain order ends oldest first; 35-drain orders each group by a rendezvous hash on origin, with file paths ahead of directories

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:49` — "then `nitpick` before `minor`, then oldest first; the run's summary states the rule"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:156` — "drain on `origin` takes each part in its own order instead: a rendezvous"

itd-82 fixes the order as category, then nitpick before minor, then oldest first, and declares it so that no implementer decides it alone. 35-drain adds a rule that puts issues whose found_at names a file before those naming a directory or no path. It keeps oldest first only in a checkout with no origin. On origin each part is taken in a per-holder, per-copy rendezvous-hash order instead. The two documents give different orders for the same eligible set, and a summary that states the itd-82 rule would describe an order the shipped drain does not follow.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 739.
