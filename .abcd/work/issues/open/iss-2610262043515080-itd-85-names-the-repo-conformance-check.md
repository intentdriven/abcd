---
schema_version: 1
id: "iss-2610262043515080"
slug: "itd-85-names-the-repo-conformance-check"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1166 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-85, itd-124]
finding_ref: cfk-77efaaa86cc3708b
---

itd-85 names the repo-conformance check `abcd audit`; itd-124 says the conformance check is `abcd lint` and `audit` is reserved for something else

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:16` — "a read-only check that tells you, in one command, whether a repository follows the working conventions"
- End B: `.abcd/development/intents/shipped/itd-124-the-conformance-check-calls-itself-lint.md:18` — "The conformance check calls itself what it is. `abcd lint` checks a repo"

The draft itd-85 promises a verb `abcd audit` whose job is checking a repository against the working conventions. The shipped itd-124 says exactly that job is `abcd lint`, and that the word `audit` returns to meaning 'did we do what we said', with `/abcd:audit` reserved for hash-chain fidelity checks. One name is given to two different concepts: building itd-85 as written would put the conformance check back under the name itd-124 took it away from.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1166.
