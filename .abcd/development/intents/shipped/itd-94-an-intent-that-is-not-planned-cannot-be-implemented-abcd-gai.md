---
id: itd-94
supersedes: [itd-27]
slug: an-intent-that-is-not-planned-cannot-be-implemented-abcd-gai
spec_id: spc-9
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# An intent that is not planned cannot be implemented: abcd gains a machine-checkable implement-readiness gate that refuses with a remedy and offers the planning interview

## Press Release

You ask your agent to implement itd-93. Instead of improvising against a draft
whose acceptance criteria nobody confirmed, it runs `abcd intent ready itd-93`,
reads the NOT READY verdict, and tells you plainly: "itd-93 is not specced, so
it cannot be implemented yet — want to plan it together?" The interview that
follows is yours: you confirm the press release, resolve the open questions,
and accept, edit, or strike every acceptance criterion. Only then does the
agent run `abcd intent plan` — your sign-off act — and the minted spec becomes
the design record it builds against. Autonomous drain runs get the same gate
for free: exit 1 is a journaled SKIP, never a guess.

## Why This Matters

Today "is this intent ready to implement" is re-derived from prose by every
run (the drain-run skip filter) and by every human conversation. iss-83 already
records the failure mode: a missing plan precondition invites the agent to
synthesize one, re-introducing the guess-the-gate risk that fail-closed design
exists to prevent. Facilitator-seeded acceptance criteria look real but carry
no maintainer sign-off; nothing machine-checkable distinguishes them. A single
read-only verb with a strict exit-code contract closes both holes: humans get
a refusal with a remedy and an offered interview, runs get a deterministic
step-0 gate.

## Acceptance Criteria

- Given an intent in `drafts/` (regardless of how good its seeded acceptance
  criteria look), when `abcd intent ready <itd-N>` runs, then it reports NOT
  READY with a `bucket` failure whose remedy names the exact next step
  (`abcd intent plan <itd-N>` after maintainer confirmation, or the planning
  interview when acceptance criteria are missing) and the process exits 1.
- Given a planned intent whose linked spec body still carries the minted
  `_Draft:` stub, when `abcd intent ready` runs, then the `spec_body` check
  fails with a write-the-spec-body remedy and the process exits 1.
- Given a planned intent with a linked spec whose reciprocal `intent:` matches
  and whose body is written, when `abcd intent ready` runs, then all four
  checks (`bucket`, `acceptance_criteria`, `spec_link`, `spec_body`) pass and
  the process exits 0.
- Given a structural fault (malformed id, unknown intent, unreadable record),
  when `abcd intent ready` runs, then it exits 2 with a one-line diagnostic —
  distinguishable by exit code from a NOT READY verdict (exit 1).
- Given the plugin surface `/abcd:intent`, when a user asks the host to
  implement an intent for which `ready` exits 1, then the surface instructs
  the host to refuse, present each failing check's detail and remedy, and
  offer the planning interview — and forbids authoring acceptance criteria or
  running `abcd intent plan` without the human's explicit in-session sign-off.
- Given the run protocol, when an unattended run picks an intent-backed item,
  then its step 0 is `abcd intent ready <itd-N>` and a nonzero exit is a
  journaled SKIP, never an improvised plan.

## Open Questions

_None recorded yet — the scaffold surface questions that block itd-93 do not
apply here; this gate is self-contained (read-only reporter + docs)._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-20c38dc58b6c -->
Fidelity review — receipt rcp-20c38dc58b6c (verifier abcd:intent-fidelity-reviewer claude-fable-5).

Provenance: abcd:intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:dd8ae17ee7f52cb06ade6cbb844a346a5a1b582f2490e82480face9765c74a5f · prompt_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86
Input attestations: diff:a87dfb9892243e9e069249d512912c63081937a8 (PR #103, merge 3b42d4e) plus current tree on docs/itd-94-record-catchup: internal/core/intent/ready.go, internal/core/intent/ready_test.go, internal/surface/cli/cli.go, commands/abcd/intent.md, .abcd/development/plans/2026-07-12-abcd-run-protocol.md@-; rubric:.abcd/.work.local/reviews/rcp-20c38dc58b6c.request.md@sha256:dd8ae17ee7f52cb06ade6cbb844a346a5a1b582f2490e82480face9765c74a5f;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Live run on draft itd-97 exits 1 with a failing bucket check whose remedy names `abcd intent plan itd-97` after maintainer confirmation; the no-criteria branch remedies via the planning interview at ready.go:114 and is asserted at ready_test.go:80.
  evidence: cmd: go run ./cmd/abcd intent ready itd-97 (exit 1) — "[fail] bucket: itd-97 is a draft — an intent that is not planned cannot be implemented / remedy: confirm the Acceptance Criteria with the maintainer, then run `abcd intent plan itd-97`"
  evidence: internal/core/intent/ready.go:111-115 — "if acCount > 0 { … `abcd intent plan %s` } else { … run the planning interview (/abcd:intent) …"
  evidence: internal/core/intent/ready_test.go:80 — "if !strings.Contains(bucket.Remedy, \"planning interview\")"
- ac-2 — MET: No live stub-spec subject exists on the branch (grep of specs/open found no `_Draft:`; host forbade manufacturing one), so judged from the shipped detector and its watched test: specBodyCheck fails on spec.BodyIsStub with a write-the-spec-body remedy, asserted by TestReadyPlannedStubSpecBody, and any not-ready result maps to exit 1 at cli.go:1196-1198.
  evidence: internal/core/intent/ready.go:199-201 — "if spec.BodyIsStub(string(data)) { … Remedy = fmt.Sprintf(\"write the spec body at %s, then re-run `abcd intent ready %s`\""
  evidence: internal/core/spec/spec.go:135-140 — "const stubMarker = \"_Draft: describe what \" … func BodyIsStub(content string) bool"
  evidence: internal/core/intent/ready_test.go:164-166 — "body.OK || !strings.Contains(body.Remedy, \"write the spec body\")"
  evidence: internal/surface/cli/cli.go:1196-1198 — "if !res.Ready { return &exitError{Code: 1} }"
- ac-3 — MET: Live run on planned itd-88 (bidirectionally linked to spc-3, body written) reports all four named checks passing and exits 0.
  evidence: cmd: go run ./cmd/abcd intent ready itd-88 (exit 0) — "abcd intent ready — itd-88 READY (planned) / [ ok ] bucket / [ ok ] acceptance_criteria / [ ok ] spec_link: linked to spc-3 (bidirectional) / [ ok ] spec_body: … is written"
  evidence: internal/core/intent/ready.go:13-18 — "CheckBucket = \"bucket\" … CheckAcceptanceCriteria … CheckSpecLink … CheckSpecBody"
- ac-4 — MET: Unknown id itd-9999 and malformed id not-an-id each produce a one-line stderr diagnostic and exit 2 (observed as `exit status 2` from go run), via the structural-fault mapping at cli.go:1174 — distinct from the exit-1 not-ready path. <!-- record-lint: illustrative -->
  evidence: cmd: go run ./cmd/abcd intent ready itd-9999 (exit 2) — "abcd: abcd intent ready: intent: itd-9999 not found in any bucket / exit status 2" <!-- record-lint: illustrative -->
  evidence: cmd: go run ./cmd/abcd intent ready not-an-id (exit 2) — "abcd: abcd intent ready: intent: id \"not-an-id\" must match ^itd-[0-9]+$ / exit status 2"
  evidence: internal/surface/cli/cli.go:1173-1174 — "return &exitError{Code: 2, Msg: \"abcd intent ready: \" + err.Error()}"
- ac-5 — MET: The committed plugin surface carries THE RULE: on exit 1 the host must refuse, present each failing check's detail and remedy, offer the planning interview, and is forbidden from improvising acceptance criteria or running `abcd intent plan` without the human's explicit in-session sign-off.
  evidence: commands/abcd/intent.md:49-54 — "Exit 1 (not ready): DO NOT IMPLEMENT. Do not improvise acceptance criteria … do not run `abcd intent plan` on your own authority … present each failing check's `detail` and `remedy` … offer the planning interview"
  evidence: commands/abcd/intent.md:70-72 — "the human accepts, edits, or strikes each … Seeded criteria are proposals, never approvals."
  evidence: commands/abcd/intent.md:80-81 — "This invocation IS the maintainer's sign-off act — never run it unattended or infer consent."
- ac-6 — MET: The run protocol's Step 0 for intent-backed items is `abcd intent ready <itd-N>`; nonzero exit is a journaled SKIP, and unattended planning/criteria-authoring/spec-synthesis is explicitly forbidden.
  evidence: .abcd/development/plans/2026-07-12-abcd-run-protocol.md:65-70 — "Step 0 for any intent-backed item: run `abcd intent ready <itd-N>`. A nonzero exit is a SKIP — journal the rendered findings and move on. Never run `abcd intent plan`, author acceptance criteria, or synthesize a spec in an unattended run"
  evidence: commands/abcd/intent.md:90-92 — "exit 1 from `ready` is a SKIP: journal the rendered findings and move to the next item."

Gap audit:
- honoured:
  - A single read-only verb with a strict exit-code contract (0 ready / 1 not ready / 2 fault) gates implementation
    evidence: internal/surface/cli/cli.go:1159-1162 — "Exit codes are the machine seam an autonomous run gates on: 0 ready, 1 not ready … 2 structural fault."
    evidence: cmd: go run ./cmd/abcd intent ready itd-97|itd-88|itd-9999 — "observed exits 1 / 0 / 2" <!-- record-lint: illustrative -->
  - Refusal tells the user plainly the intent is not specced and offers the planning interview; `abcd intent plan` is the human sign-off act
    evidence: commands/abcd/intent.md:52-54 — "\"`<itd-N>` is not specced, so it cannot be implemented yet\" … offer the planning interview"
    evidence: commands/abcd/intent.md:80 — "This invocation IS the maintainer's sign-off act"
  - Autonomous drain runs get the same gate for free: exit 1 is a journaled SKIP, never a guess
    evidence: .abcd/development/plans/2026-07-12-abcd-run-protocol.md:65-70 — "a missing gate is a STOP, never an invitation to improvise one"
  - Readiness re-uses the same acceptance-criteria parser as Plan and the fidelity review, so the gates cannot disagree
    evidence: internal/core/intent/ready.go:131-133 — "through the same parser Plan and the fidelity review use (countAcceptanceCriteria)"
- diverged:
  - The intent title promises the gate "refuses with a remedy"; terminal buckets (shipped, disciplines, superseded) deliberately refuse without one — remedies attach only to fixable states
    evidence: internal/core/intent/ready.go:101-102 — "Terminal buckets carry no remedy: there is nothing to fix, the answer is simply no."
- missing: (none)