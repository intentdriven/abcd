---
id: itd-2610071545369520
slug: when-abcd-s-banned-names-check-warns-on
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
refines: [iss-2610071538075997]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A person confirms the assistant's verdict on each warned banned-name line instead of judging it alone

## Press Release

> When abcd's banned-names check warns on a line (a listed name that is also an ordinary word, such as a game title like Portal), the assistant in the person's session proposes for each warned line whether it uses the protected name or the ordinary word, with a one-line reason, and the person approves or overrides. An approved protected use stays blocked; an approved ordinary use gets the allow marker, so the next run passes. Product thinker's ruling 2026-10-07 (interview on iss-2610071538075997, promoted from a managed repository's report): add the judge; the managed repository's further asks, a free local model judging silently into a private log and a review of that log's miss rate, are not taken on now. Open for planning: what identifies a line so a verdict still matches after the line moves; whether the allow marker records who decided and when.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [security-isolation SOTA note](../../research/notes/2026-10-23-security-isolation-sota.md).

- **Automation bias is robust and only partly mitigated**: Parasuraman and Manzey (Human Factors 2010) on complacency and bias as "overlapping automation-induced phenomena"; Goddard, Roudsari and Wyatt (JAMIA 2012, 74 studies) found trust, workload and time pressure raise it, and listed as mitigators training, emphasising user accountability, the position of advice on the screen, confidence levels shown with the output, and information rather than a recommendation. Whether training helps is where the two reviews differ.
- **A reason helps only when checking it is cheaper than deciding alone**: Vasconcelos et al. (CSCW 2023, N = 731 across five studies) found explanations reduce overreliance when they "lower the effort needed to verify"; Buçinca et al. (N = 199) found cognitive forcing "significantly reduced overreliance", at the cost of the least favourable ratings.
- **Line identity is a content hash, not a line number**: GitHub code scanning matches alerts across edits with `partialFingerprints.primaryLocationLineHash` plus a stable rule id and path; gitleaks' `.gitleaksignore` fingerprint ends in the line number and is "experimental", the shape to avoid (SARIF support for code scanning; gitleaks).
- **Who decided and why is recorded in the field**: GitHub keeps a dismissal comment "added to the alert timeline and can be used as justification during auditing"; detect-secrets preserves audit labels across rescans; Semgrep's `nosemgrep` takes a rule id but no reason (resolving code scanning alerts; detect-secrets; Semgrep ignores).
- **The originating evidence is one test**: the managed repository's lab, 280 held-out lines, frontier judge 97.8% with no miss, local model 90.0% with ten; "One test is not a track record" (iss-2610071538075997).

**Path 2: native, shaped to GitHub's line fingerprint for identity and its dismissal record for provenance, with the interaction built from the overreliance evidence**: verdict and reason beside the quoted line, one explicit answer per line with no accept-all, and a marker recording who, when and why. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) **Override rate**: the lab asked for a calibration signal, and the 2026-10-07 ruling declined a silent judge logging to a private file and a review of that log's miss rate. Is a count of overrides, read from the markers the person already writes, inside or outside that ruling? For the product thinker.
- (SOTA 2026-10-23) **Line identity**: a hash of the line's content with the rule id, as GitHub's fingerprint does, so a verdict follows a moved line and lapses when the line changes.
- (SOTA 2026-10-23) **Marker contents**: the field records decider, time and reason; whether abcd's allow marker carries all three or points at a record that does.
- (SOTA 2026-10-23) **No accept-all**: whether a batch approval is ever offered, given the evidence that forcing a per-item answer is what reduces overreliance.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
