---
id: itd-188
slug: machine-assistance-in-maintaining-the-ledger-without-any-con
spec_id: spc-66
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Machine assistance in maintaining the ledger, without any context that holds both ledger content and a reading

## Press Release

> **The scribe sees the ledger and nothing else.** Its access rule is the
> assembler's inverse: it reads ledger content, transcribes reading outputs
> and researcher dispositions, and authors nothing — and it never receives
> the shipped repository as an object of judgement. No session holds both a
> reading and the ledger; each is a distinct retained session, and session
> retention can show it.

## What's In Scope

- The scribe definition with the inverse access rule, and (when the verb
  lands) its ingest path.
- The fidelity-flag permission: the scribe may flag an internal
  inconsistency in what it is transcribing ("this disposition contradicts
  the ruling recorded earlier in the session") — that is transcription
  fidelity, not judgement, and it does not breach authors-nothing. What it
  may never do is propose a resolution.
- The contribution stamp: anything the scribe is explicitly asked to
  produce beyond formatting opens with a stamped attribution that travels
  with the material if adopted — the hand-run precursor of the `origin` /
  `scribe-transcribed` keys, in force until those keys ship. An unstamped
  contribution is never delivered.
- The protocol until then, documented and followed: entries are transcribed
  when the reading returns, not later — a protocol invented under time
  pressure is a protocol that gets skipped.

## Acceptance Criteria

- **Given** a scribe invocation, **when** its context is assembled,
  **then** it contains ledger content and no shipped-tree material.
- **Given** a reading run and a scribe run, **when** session retention is
  inspected, **then** each is a distinct retained session and no session
  holds both.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-71f5a9b044cb -->
Fidelity review — receipt rcp-71f5a9b044cb (verifier intent-auditor claude-fable-5).

Provenance: intent-auditor@claude-fable-5 · rubric_hash sha256:f3bba86a84b329fcbfbd3df64ec5d77d382dda9b4b1a61baf887e316bc41f38e · prompt_hash sha256:c07526fb6e333e1e2944acc40cd7f4272cab0c2b73e2352e27c956063e9a590f
Input attestations: diff:543aaf19...a1d3ba9a@sha256:52899f2afe5d9898d1472ed0a58814405373369784717f27086e6ea38f95d232;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The definition's inputs block is a positive allow list confined to .abcd/work/issues/ plus supplied text, its exclusions name the shipped tree and the transcript store, and TestScribeInputsAreLedgerOnly / TestScribeDeclaresNoTranscriptStoreAccess / TestScribePromptSatisfiesTheContract pass over the real file (go test, 6 cases PASS). Concern: no assembler runs for the scribe this cycle, so what is proved is that the DECLARATION names no shipped-tree path, not that any host-assembled context contained only ledger content; the criterion's 'when its context is assembled' is realised at declaration level only, which the spec states.
  evidence: agents/scribe.md:40 — "## Inputs (the allow list — anything not named here is excluded)"
  evidence: agents/scribe.md:46 — "`.abcd/work/issues/readings/`"
  evidence: agents/scribe.md:59 — "**The shipped repository as an object of judgement**"
  evidence: agents/scribe.md:68 — "**The session-transcript store.** You are not one of its enumerated consumers"
  evidence: internal/core/lint/scribecontract_test.go:42 — "const scribeLedgerRoot = \".abcd/work/issues/\""
  evidence: internal/core/lint/scribecontract_test.go:240 — "if !strings.HasPrefix(path.Clean(tok)+\"/\", scribeLedgerRoot)"
  evidence: internal/core/lint/scribecontract_test.go:350 — "func TestScribeInputsAreLedgerOnly(t *testing.T)"
  evidence: internal/core/lint/scribecontract_test.go:375 — "func TestScribeDeclaresNoTranscriptStoreAccess(t *testing.T)"
  evidence: internal/core/lint/scribecontract_test.go:516 — "func TestScribePromptSatisfiesTheContract(t *testing.T)"
  evidence: .abcd/development/specs/closed/spc-66-machine-assistance-in-maintaining-the-ledger-without-any-con.md:150 — "No assembler runs for the scribe this cycle, so what is pinned is the declaration"
- ac-2 — MET_WITH_CONCERNS: The protocol in the brief's agents chapter requires the reading run and the scribe run to be separate host sessions each retained under its own session id; the store retains per-session records (history.Record.SessionID, Capture(session_id), List) and TestCaptureIdenticalSourceDistinctSessionsWritesBoth pins that distinct sessions yield distinct records. Concern (the spec's own declaration, judged sound): the criterion is met procedurally, not mechanically — the delivered range contains no reading run and no scribe run whose retention could be inspected (the ledger at a1d3ba9a has no readings/ or dispositions/ entries and the store holds no scribe session), and the 'no session holds both' half is not observable from the store because the host assembles context before anything is retained. Mechanical enforcement is deferred to the ingest verb.
  evidence: .abcd/development/brief/05-internals/01-agents.md:90 — "**The reading run and the scribe run are separate host sessions**, always. Each is retained under its own session id"
  evidence: agents/scribe.md:37 — "No session holds both — the reading session and this one are separate sessions, always."
  evidence: internal/core/history/history.go:50 — "SessionID    string    `json:\"session_id\"`"
  evidence: internal/core/history/history.go:97 — "func Capture(repoRoot, rootSHA, sessionID string, raw []byte, kind string)"
  evidence: internal/core/history/history.go:223 — "func List(rootSHA string) ([]Record, error)"
  evidence: internal/core/history/history_test.go:341 — "func TestCaptureIdenticalSourceDistinctSessionsWritesBoth(t *testing.T)"
  evidence: .abcd/development/specs/closed/spc-66-machine-assistance-in-maintaining-the-ledger-without-any-con.md:148 — "None — procedural (below)"
  evidence: .abcd/development/specs/closed/spc-66-machine-assistance-in-maintaining-the-ledger-without-any-con.md:155 — "The second criterion carries **no mechanical test**"

Gap audit:
- honoured:
  - The scribe definition with the inverse access rule: ledger content only, never the shipped repository as an object of judgement, not a transcript consumer
    evidence: agents/scribe.md:35 — "**The access rule is the inverse of the assembler's.**"
    evidence: agents/scribe.md:57 — "## Never in context"
    evidence: internal/core/lint/scribecontract_test.go:277 — "func TestScribeAccessCheckRefusesEveryBypass(t *testing.T)"
  - The fidelity-flag permission: may flag an internal inconsistency as a named field, never proposes a resolution
    evidence: agents/scribe.md:121 — "Emit each one as an entry in a named `fidelity_flags` list beside the transcribed material"
    evidence: agents/scribe.md:125 — "A flag is carried to the researcher **unresolved**. You never propose a resolution"
    evidence: agents/scribe/fixtures/injection-canary.json:94 — "\"fidelity_flags\": ["
  - The contribution stamp: anything beyond formatting opens with a stamped attribution; an unstamped contribution is never delivered
    evidence: agents/scribe.md:152 — "> SCRIBE CONTRIBUTION — composed by the scribe, not by the researcher."
    evidence: agents/scribe.md:156 — "An unstamped contribution is never delivered. This is a refusal, not a preference"
  - The protocol documented: entries transcribed when the reading returns, two host sessions, ordinary record path, flags carried unresolved
    evidence: .abcd/development/brief/05-internals/01-agents.md:81 — "## The scribe protocol"
    evidence: .abcd/development/brief/05-internals/01-agents.md:89 — "**Entries are transcribed when the reading returns**, not later."
    evidence: .abcd/development/brief/05-internals/01-agents.md:92 — "**A fidelity flag is carried to the researcher unresolved.**"
  - The definition satisfies the agent_contract trust shape: prompt_version, reads_untrusted_input, capability_scope, canary fixture, changelog entry
    evidence: agents/scribe.md:9 — "prompt_version: 0.1.0"
    evidence: agents/scribe.md:10 — "reads_untrusted_input: true"
    evidence: agents/scribe/fixtures/injection-canary.json:2 — "\"fixture\": \"injection-canary\""
    evidence: agents/CHANGELOG.md:17 — "### scribe 0.1.0"
    evidence: internal/core/lint/scribecontract_test.go:393 — "func TestScribeCanaryAssertsTheRefusals(t *testing.T)"
- diverged:
  - "session retention can show it" (press release) — retention shows two distinct sessions exist; it cannot show that no session holds both, and no reading/scribe session pair exists in the delivered range to inspect
    evidence: .abcd/development/brief/05-internals/01-agents.md:90 — "it cannot enforce that the practice held, because the separation happens in the host before anything is retained"
    evidence: .abcd/development/specs/closed/spc-66-machine-assistance-in-maintaining-the-ledger-without-any-con.md:155 — "The second criterion carries **no mechanical test**"
  - "when its context is assembled" — no assembly executes; the guard proves the definition names no shipped-tree directory or nested file, and a separator-free shipped filename is outside its reach
    evidence: internal/core/lint/scribecontract_test.go:10 — "they prove the prompt names the right paths, not that a host assembled the right context"
    evidence: .abcd/development/specs/closed/spc-66-machine-assistance-in-maintaining-the-ledger-without-any-con.md:150 — "No assembler runs for the scribe this cycle"
- missing:
  - The scribe's ingest path ("when the verb lands") — scoped out by the intent itself; the emitted shapes have no schema until spc-58's stores land, so the record gate refuses their directory as an undeclared bucket rather than judging a record
    evidence: agents/scribe.md:162 — "There is no ingest verb, so what you emit is committed through the ordinary record path"
    evidence: .abcd/development/brief/05-internals/01-agents.md:87 — "**There is no ingest verb.**"

## Grounds

- pursued: Pursued now because the ledger has to be maintained during this cycle and the obvious way to get help with it is the one thing the design cannot allow: a context holding both a reading and the ledger. Defining the scribe's inverse access rule before the first run is what keeps machine assistance from becoming the channel by which reading and ledger meet, and session retention is what lets that claim be checked rather than asserted.
