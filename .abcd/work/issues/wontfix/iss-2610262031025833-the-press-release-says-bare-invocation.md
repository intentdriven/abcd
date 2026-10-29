---
schema_version: 1
id: "iss-2610262031025833"
slug: "the-press-release-says-bare-invocation"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 78 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "none (filed automatically)"
finding_ref: cfk-de1ef7a5443ba34a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Bare report opens an editor and files what is saved, already enumerated as an exception; the same press release sentence is the one to fix."
duplicates: [iss-2610262030301422]
---

The press release says bare invocation only shows status, but bare `abcd report` opens an editor and files a report into the inbox

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "Bare invocation of any command shows status + suggested next actions; state-mutating actions require an explicit sub-verb (the universal abcd convention)."
- End B: `.abcd/development/brief/04-surfaces/29-report.md:35` — "Bare `abcd report` opens the skeleton in `$VISUAL` or `$EDITOR` when both ends of the session are a terminal, from a private temporary file outside both repositories, and files what is saved."

The press release says bare invocation of any command shows status and that state-mutating actions need an explicit sub-verb. The report chapter registers no sub-verb, and its bare form opens the skeleton in an editor and files what is saved into the machine inbox, a state-mutating act with no sub-verb. The two documents disagree on what a bare command may do.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 78.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Bare report opens an editor and files what is saved, already enumerated as an exception; the same press release sentence is the one to fix.
