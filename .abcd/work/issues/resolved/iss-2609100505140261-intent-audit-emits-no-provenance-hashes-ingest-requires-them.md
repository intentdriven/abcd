---
schema_version: 1
id: "iss-2609100505140261"
slug: "intent-audit-emits-no-provenance-hashes-ingest-requires-them"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (intent audit, intent audit ingest)"
resolution: "The request now carries host-computed provenance and the ingest verifies rather than shape-checks it. The rubric hash covers the judging contract this binary enforces, serialised from the very rules the validator applies and written verbatim into the request, so the auditor is handed the exact bytes that were hashed. The prompt hash covers the request document minus the provenance block, which is a pure function of facts the ingest holds, so the ingest recomputes it. A hash the host never issued is refused outright rather than dead-lettered, because it means this is not the answer to this question, and refusing leaves the owed marker parked so a re-emit stays open. An absent or malformed hash keeps its existing path. The measured scale was far worse than recorded: not three verdicts but thirty-six, across thirteen distinct rubric values, with two digests appearing under both field names, so the field name carried no meaning across the corpus. The thirty-six already ingested are left exactly as they are, deliberately: the ingest no-ops on them, and hand-editing committed audit notes would fabricate a second layer of provenance over the first."
impact: fix
---

`abcd intent audit <itd-N>` emits a fidelity-review request that carries no `policy.rubric_hash` and no `policy.prompt_hash`, but `abcd intent audit ingest` rejects a verdict whose `policy` hashes are empty. The two halves of the same verb disagree, and the gap lands on whoever writes the verdict.

Observed running all 13 owed reviews of a managed repository in one pass. Every one of the 13 independent auditor agents hit the same wall and each invented its own values to satisfy the non-empty rule. They did not agree on what to hash: some used the SHA-256 of the intent file, some the request file, some the bundled auditor agent definition, and some crossed the two fields over (rubric = request, prompt = intent). Eight of the thirteen said so explicitly in their reports and asked the host to substitute real values; the rest simply filled the fields.

Why this matters more than a missing flag. The fields exist to attest which rubric and which prompt produced a verdict, and `ingest` writes them into the shipped intent's Audit Notes, which is a permanent record. As it stands the verb cannot be completed without fabricating that attestation, so the record gains a provenance claim that looks verified and is not. A reader cannot tell a host-computed hash from an invented one, and the disagreement between auditors means the same field means a different thing on different intents. This is the false-green shape: the gate is satisfied and attests nothing.

A reporting session refined this afterwards and the refinement matters, because the first account was harsher than the facts. The values are not random: each auditor computed a defensible hash by a stated convention, usually the request file and the bundled auditor definition, and said in its report which convention it had used and that the host should substitute its own. The defect is not fabrication by the auditor. It is that the ingest accepts provenance the host never issued and cannot distinguish a conventional self-computed value from an arbitrary one, so the attestation attests only that some agent chose something.

This repository has the same condition and acquired it knowingly. Three verdicts were ingested here on 2026-09-10, each carrying hashes its auditor had computed itself and disclosed as such, and the ingest was performed anyway on the reasoning that the validator would object if the values were wrong. The validator checks the SHAPE of a hash and never its value, so it objected to nothing. Three permanent Audit Notes in this repository therefore carry self-issued provenance, and a fourth managed repository holds three more verdicts uningested for the same reason, with its handover recording the condition so that whoever ingests them does so knowingly. Six verdicts across two repositories is enough to say this is the normal outcome of the verb rather than an incident.

Needed: `intent audit` should emit the two hashes in the request it hands the auditor, so the verdict echoes back values the host itself computed and `ingest` can check rather than trust. Failing that, `ingest` should either accept empty hashes and record them as absent, or refuse with a message naming what the host expects to be hashed. Any of the three is better than a required field with no supported way to fill it.

Workaround: none that preserves the record's integrity. The verdicts were left uningested pending a decision, because ingesting them would write 13 fabricated attestations into the durable record.

Distinct from the sibling finding about the delivered DIFF RANGE the same request asks the host to supply: that one is about the range, this one is about the hashes, and fixing either leaves the other standing.

## Grounds

- pursued: we expect a hash the ingest can recompute to be the only kind worth requiring, because a hash it cannot recompute is unverifiable and that unverifiability is the defect; it is shown wrong if a legitimate re-audit is blocked by staleness more often than by a real mismatch, which is now load-bearing by design since editing criteria between emit and ingest moves the prompt hash
