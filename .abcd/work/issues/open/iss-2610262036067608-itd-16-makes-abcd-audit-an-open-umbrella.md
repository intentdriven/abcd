---
schema_version: 1
id: "iss-2610262036067608"
slug: "itd-16-makes-abcd-audit-an-open-umbrella"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 623 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:65"
remedy: "In itd-16 drop 'schema audit' and 'prompt-quality audit' as applications that register under `/abcd:audit` (out-of-scope bullet and the header comment): under adr-40 a check of form is a lint and an `abcd lint` target; keep the umbrella open only to audits in adr-40's sense, a reality-against-commitment comparison such as the lifeboat integrity check."
related_intents: [itd-16]
finding_ref: cfk-376c55f7b4e7c2ca
---

itd-16 makes /abcd:audit an open umbrella a schema or prompt-quality audit can join; 16-lint reserves it for hash-chain fidelity and calls a check of form a lint.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:65` — "schema audit, prompt-quality audit, third-party-vendor audit"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:20` — "`/abcd:audit` stays reserved for itd-16's hash-chain fidelity surface."

itd-16 names schema audit, prompt-quality audit and third-party-vendor audit as applications that may register as their own intents under the /abcd:audit umbrella. 16-lint says a verb applying rules about form is, in adr-40's vocabulary, a lint, and that /abcd:audit stays reserved for itd-16's hash-chain fidelity surface. A schema check is an audit sub-verb in one document and a lint in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 623.
