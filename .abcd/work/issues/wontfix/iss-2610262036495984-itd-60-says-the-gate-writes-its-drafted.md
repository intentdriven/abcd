---
schema_version: 1
id: "iss-2610262036495984"
slug: "itd-60-says-the-gate-writes-its-drafted"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 676 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:104"
remedy: "none (filed automatically)"
related_intents: [itd-60]
finding_ref: cfk-f3016389b5be6acb
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent's ac-5 and its Divergences list say exactly this, and internal/core/docfidelity/docfidelity.go proposes the edit and refuses until Apply runs."
---

itd-60 says the gate writes its drafted brief edit into the shipping change itself; the docs chapter says the gate only proposes the edit and still refuses

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:104` — "The gate proposes that edit and still refuses."
- End B: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:130` — "When the gate finds the brief lagging, the pass writes the brief edit into the shipping change itself"

In itd-60, when the brief lags, the pass writes the brief edit into the shipping change and flags it, so the change is not held up by the paragraph. The docs chapter says a drafted replacement is only proposed by the gate, which 'still refuses'. Only a separate apply form, or the autonomous form, writes the sentence and lets the change through. One says drift is fixed and the change proceeds by default; the other says drift refuses by default until a second invocation applies the fix.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 676.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent's ac-5 and its Divergences list say exactly this, and internal/core/docfidelity/docfidelity.go proposes the edit and refuses until Apply runs.
