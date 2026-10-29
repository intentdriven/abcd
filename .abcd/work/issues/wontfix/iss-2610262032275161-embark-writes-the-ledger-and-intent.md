---
schema_version: 1
id: "iss-2610262032275161"
slug: "embark-writes-the-ledger-and-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 270 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:47"
remedy: "none (filed automatically)"
finding_ref: cfk-152ecf844e942d9f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The two sentences govern different things: capture resolves which ledger a verb addresses, embark takes a path to unpack into, and the CLI defaults it to the working directory by design (filepath.Abs of '.')."
---

Embark writes the ledger and intent store under a target defaulting to the working directory; capture holds that the root is resolved, never taken to be the working directory

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:47` — "takes the lifeboat directory and an optional target, which defaults to the working directory"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:366` — "the repository root is **resolved** from the working directory, never taken to be it"

Embark's unpack takes an optional target that defaults to the working directory and writes the issue ledger, intent store and spec store beneath it. Capture says the repository root is resolved from the working directory and never taken to be it, because taking it verbatim minted a second ledger under a subdirectory, out of reach of every gate that reads the real one. An embark run from a subdirectory of a checkout lays exactly the stray store capture describes as the defect.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 270.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The two sentences govern different things: capture resolves which ledger a verb addresses, embark takes a path to unpack into, and the CLI defaults it to the working directory by design (filepath.Abs of '.').
