---
id: itd-2609090559376002
slug: sub-agent-transcript-capture
spec_id: spc-2609090624222051
kind: standalone
suggested_kind: null
reclassification_history: []
related_adrs: [adr-29]
builds_on: []
severity: major
impact: additive
promoted_from: iss-2609081917287384
origin: extracted-from-record
production_mode: hand-written
---

# Every Sub-Agent a Session Spawns Leaves the Same Redacted Record Its Main Thread Does

## Press Release

> **abcd captures the reasoning of every sub-agent a session spawns, not just the session's main thread.** Today a session's own transcript is captured and redacted on write, but the sub-agents it spawns write their transcripts elsewhere and nothing collects them. What survives of a delegated task is the prompt that launched it and the summary it returned, while every command it ran, every file it edited and every judgement it made is dropped. Delegated work is the bulk of the work — better than three quarters of the recorded volume — so the store has been keeping the smaller half. This intent routes the sub-agent into the corpus that already exists, with the lineage that says which session and which agent produced it, at any nesting depth, under the same fail-closed redaction the session transcript already gets.
>
> "I delegate the reviews, the sweeps and most of the implementation to sub-agents, so that is where the reasoning I would want to audit actually lives," said Maya, an autonomous-development practitioner. "When I went back to reconstruct why a change was made, the store had the two lines where I handed the task off and the summary that came back, and nothing in between. The part I most needed to read was the part that was never kept."

## Why This Matters

abcd's transcript corpus exists so that work can be reconstructed after the fact. That promise is answered for the main thread and silently unanswered for everything delegated out of it:

- **The session transcript IS captured:** the session-end hook receives the path to it and stages it for redaction and storage.
- **Sub-agent transcripts are NOT:** they are written per sub-agent, and no hook abcd registers ever names them, so nothing enumerates or reads them.

The asymmetry is not marginal. On the measured corpus, sub-agent transcripts held roughly three quarters of all recorded bytes — 968 files and 673 MB against 68 session files and 206 MB — outweighing session transcripts by more than three to one. What the parent retains for each delegated task is the launch prompt and the returned report: in one measured case, two lines standing in for several hundred, discarding every tool call the agent made along the way.

This is a symmetry failure rather than a missing feature. The corpus already exists, already redacts on write, and already works for one caller class. The sub-agent caller was simply never routed into it. The harness offers a completion event for sub-agents whose payload names the finished sub-agent's own transcript, so the gap can be closed by feeding the existing store, with no new capture mechanism and no dependence on the harness's undocumented on-disk layout.

Two consequences make this urgent rather than tidy. Uncaptured transcripts age out under the harness's retention sweep, so the loss is permanent and ongoing. And the workaround already in use — inventing a composite identifier by hand — produces records that cannot be found from the identifier a reader actually holds: 176 of the store's 267 records carried one, and neither direction of the lookup worked.

Capturing the material is the load-bearing half. What is done with it afterwards — recovering the history that was never captured, and handing a whole session back as one readable artefact — rests on this and is separately delivered.

## Typed Links

- **refines `itd-59`** (autonomous-worker transcript capture): the adjacent half of the same corpus. itd-59 covers the worker on abcd's own run seam; this covers sub-agents spawned inside an interactive session.
- **refines `adr-29`** (native transcript corpus): feeds the store that ADR established rather than redesigning it.
- **corrects a premise in `itd-59`**, flagged for human confirmation and not auto-classified: itd-59's "What's Out of Scope" records interactive-session capture as already solved. That holds only of the main thread, and planning itd-59 on the unqualified claim would rebuild the same blind spot.
- **built on by `itd-2609091718566731`** (recovering transcripts already on disk) and **`itd-2609091718595846`** (reconstruction and telemetry): both consume the lineage this intent puts on a record, and neither is deliverable without it.

## What's In Scope

- **The capture gap:** every sub-agent transcript lands in the corpus under the repository it belongs to, with the same redaction and the same fail-closed refusal on a degraded scanner as a session transcript.
- **All sub-agent completions** at any nesting depth, including sub-agents spawned by other sub-agents.
- **Provenance enough to read the result back:** a stored sub-agent transcript can be traced from the identifier of the session that spawned it, and says what kind of agent produced it and which rung of the attribution ladder placed it.
- **Idempotence:** the same sub-agent transcript presented twice is one record, and two sub-agents whose transcripts happen to be byte-identical are still two.
- **A completion that yields nothing is reported**, on stderr and on disk, and never stalls or fails the session it fired inside.
- **Feed, don't fork:** sub-agent capture reuses the existing staging and redaction path rather than adding a second mechanism beside it.

## What's Out of Scope

- **Recovering history that already exists.** Transcripts on disk that were never captured, and records already filed under a composite identifier, are `itd-2609091718566731`.
- **Reconstruction and telemetry.** Emitting a session as one artefact with a machine-readable measurement file is `itd-2609091718595846`.
- **Redesigning the corpus.** The store's per-repo keying and its single-owner provisioning stay as they are.
- **The harness's own retention policy.** How long the harness keeps its transcripts is configuration outside abcd's control.
- **Structured extraction of findings.** Turning a review agent's transcript into structured findings is a separate concern from keeping the transcript.

## Mechanism

We expect routing sub-agent capture through the harness's sub-agent completion event to close the gap without new machinery, because that event's payload names the finished sub-agent's own transcript path directly, so the existing stage-then-redact path can consume it unchanged and nothing needs to read the harness's undocumented directory layout. We expect the lineage a reader needs to fit in explicit record fields rather than a composite identifier, because the payload carries the spawning session untruncated and the agent's own id separately, and a field cannot be lossy in the way a concatenation is. This is shown wrong if the completion event fires before the sub-agent transcript is flushed and readable, if a session spawning many sub-agents degrades under per-completion staging, or if the payload is absent on a harness version abcd claims to support.

## Scope Conditions

- Holds for harness versions whose sub-agent completion event carries the finished sub-agent's transcript path; a version without it falls back to no capture rather than to guessing at the on-disk layout, and records that it did so. <!-- cond: cond-2609090624228012 -->
- Holds at the observed working scale of a few hundred sub-agents per repository per month, with individual transcripts up to a few megabytes and sessions up to roughly a hundred sub-agents. <!-- cond: cond-2609090624226418 -->
- Assumes sub-agent transcripts share the line-delimited shape the session transcript already uses, so one reader serves both. <!-- cond: cond-2609090624223171 -->
- Assumes a transcript's owning repository is determined by the working directory the payload reports, or failing that by the store that has already seen the spawning session — never by decoding the harness's project directory name, which is not reversible. <!-- cond: cond-2609090624228630 -->
- Assumes a session run in a worktree belongs to the store of the repository that worktree derives from, and that the worktree may already be gone by the time the completion event fires. <!-- cond: cond-2609090624222424 -->
- **Whether the completion event fires before the harness has flushed the sub-agent's transcript is UNVERIFIED.** The mitigations shipped — a bounded stage-time wait on an incomplete final line, and a drain-time re-read that replaces the staged bytes only when the source strictly extends them — bound the damage and count their own effect, but they do not settle the question, and the residual rate has not yet been measured over a corpus. <!-- cond: cond-2609091722261577 -->

## Acceptance Criteria

- **Given** a session that spawns a sub-agent, **when** that sub-agent finishes, **then** its transcript is stored in the corpus for the session's repository, redacted on write, and listed by `abcd history`.
- **Given** a sub-agent that itself spawns a sub-agent, **when** both finish, **then** both transcripts are stored and each is attributable to the session that spawned it.
- **Given** a stored sub-agent transcript, **when** an operator looks it up from the identifier of the session that spawned it, **then** they can reach it and can tell what kind of agent produced it.
- **Given** a repository whose secret scanner is degraded, **when** a sub-agent transcript would be captured, **then** the capture refuses rather than storing under weakened redaction, matching the session-transcript path.
- **Given** the same sub-agent transcript presented twice, **when** capture runs again, **then** the second capture is a no-op rather than a duplicate record.
- **Given** a sub-agent completion the harness reports without a readable transcript, **when** capture runs, **then** the miss is reported rather than failing silently or aborting the session.

## Open Questions

- **The flush race is still open.** The mitigations are in and self-counting (`DrainResult.Extended` is the count of transcripts caught short and completed), but the rate over a real corpus has not been measured, so the scope condition above stands as written rather than resolved.
- **The decision record refining adr-29 is deferred and unminted.** The move from a composite identifier to explicit lineage fields deserves an ADR; until it is minted, `spc-2609090624222051` is the decision of record. Two committed surfaces disagree about how an ADR id is allocated (the root router says hand-numbered and coordinated; the ADR store's charter records the 2026-09-01 ruling that `abcd decide` mints through the same timestamp-numeric seam as every other family), and that disagreement is not this intent's to settle — but it blocks the minting.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-e6f812226e15 -->
Fidelity review — receipt rcp-e6f812226e15 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:df2d024151ac05432ce96f7cbdd6f41539b08054f359b00ccffefb12bbedd998
Input attestations: diff:feat/sub-agent-transcript-capture: 13eafece 4325d811 d751109c 8d5eadb9 (319da670 and 9af9a30a excluded by the host); digest over the concatenated per-commit patches@sha256:3e4e5886a2dd6c2e4896b8cdcfff552b07de6bdfcc241729d03f91dc0dd816ff;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The wired path exists end to end and is proven by test: SubagentStop is registered in the manifest, the hook stages the payload's transcript, a later drain runs the unchanged fail-closed Capture, and history.List (what `abcd history list` calls) returns the record. Concern: the record does not exist when the sub-agent finishes, only after a later drain (SessionStart, budget 32/4MB, or one entry per prompt) — until then the transcript is unredacted staged text, and a repository nobody reopens never stores it at all (age buys priority and reporting, never storage); the staged copy may also be short because the flush race is unverified.
  evidence: hooks/hooks.json:55 — ""SubagentStop": ["
  evidence: internal/surface/cli/cli.go:1463 — "hookCmd.AddCommand(newSubagentStopCommand())"
  evidence: internal/surface/cli/hook_subagent.go:133 — "res, err := history.Stage(rootSHA, meta, raw)"
  evidence: internal/core/history/staging.go:727 — "cr, err := Capture(repoRoot, rootSHA, body, s.captureMeta())"
  evidence: internal/surface/cli/history.go:119 — "records, err := history.List(rootSHA)"
  evidence: internal/surface/cli/hook_subagent_stop_test.go:345 — "func TestSubagentStopThenSessionStartStoresTheRecord(t *testing.T) {"
  evidence: internal/surface/cli/cli.go:1333 — "if dr, err := history.Drain(captureRoot(cwd), det.RootSHA, sessionStartDrainBudget); err == nil {"
  evidence: internal/surface/cli/cli.go:1534 — "dr, err := history.Drain(captureRoot(cwd), rootSHA, livePromptDrainBudget)"
- ac-2 — MET_WITH_CONCERNS: Both transcripts are stored — the hook fires and stages once per completion, keyed on (session, agent), and each record carries the FULL untruncated spawning session id, so attribution to the session holds by construction for every depth. Concern: attribution to the spawning AGENT (parent_agent_id, spawn_depth) rests entirely on rung 1, the harness's undocumented `<transcript>.meta.json` sidecar; rung 2 (the spawning transcript's tool result) is out of this delivered range, so with no sidecar a nested agent stores spawn_attribution=unattributed and its spawner is unrecoverable. Every sidecar fixture in the tests is hand-written, and no test stores two records at two depths end to end.
  evidence: internal/core/history/history.go:57 — "belongs to. On a sub-agent record it is the FULL, untruncated id of the"
  evidence: internal/surface/cli/hook_subagent.go:213 — "if side, ok := readHarnessAgentSidecar(in.AgentTranscriptPath); ok {"
  evidence: internal/surface/cli/hook_subagent.go:211 — "SpawnAttribution: "unattributed","
  evidence: internal/surface/cli/hook_subagent_stop_test.go:128 — "func TestHookSubagentStopReadsTheHarnessSidecar(t *testing.T) {"
  evidence: internal/surface/cli/hook_subagent_stop_test.go:134 — ""toolUseId":"toolu_01","spawnDepth":2,"parentAgentId":"a0","model":"opus""
  evidence: internal/core/history/lineage_test.go:562 — "func TestUnknownSpawnIsDistinguishableFromNoParent(t *testing.T) {"
  evidence: internal/core/history/store.go:122 — "func (m CaptureMeta) validateSpawnAttribution() error {"
  evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:213 — "reconstruction time, not in the hook — it is"
- ac-3 — MET_WITH_CONCERNS: In core the criterion is fully realised and tested: ListForSession returns the whole session from one untruncated session id, Read resolves filename then agent id then session, and every record carries agent_type; schema-1 records parse as main-thread records with empty lineage rather than as faults. Concern, and it is a `wired or it isn't done` concern: no operator surface in THIS delivered range consumes ListForSession — its only non-test caller is reconstruct.go, delivered by the excluded commit 9af9a30a — `history show <session-id>` deliberately returns the main-thread record ALONE, and the human render of `history list` and `history show` prints no agent_id and no agent_type. The operator's only route is `abcd history list --json` plus hand filtering on session_id, which the plugin page's documented field list does not mention. On legacy schema-1 sub-agent records the criterion does not hold at all: they were written under a composite session_id and carry no agent type, and their repair is explicitly out of this intent's scope.
  evidence: internal/core/history/history.go:490 — "func ListForSession(rootSHA, sessionID string) ([]Record, error) {"
  evidence: internal/core/history/lineage_test.go:320 — "func TestListForSessionReturnsMainThreadAndEverySubagent(t *testing.T) {"
  evidence: internal/core/history/lineage_test.go:286 — "func TestReadResolvesFilenameThenAgentThenSession(t *testing.T) {"
  evidence: internal/core/history/lineage_test.go:28 — "func TestSchemaOneRecordReadsAsMainThread(t *testing.T) {"
  evidence: internal/core/history/reconstruct.go:317 — "records, err := ListForSession(rootSHA, opts.SessionID)"
  evidence: internal/surface/cli/history.go:141 — "fmt.Fprintf(w, "%s %s %s redacted secrets=%d home=%d\n","
  evidence: internal/surface/cli/history.go:353 — "rec, body, err := history.Read(rootSHA, args[0])"
  evidence: commands/history.md:48 — "Summarise each record newest-first: `captured_at`, `session_id`, `source_kind`,"
  evidence: .abcd/development/intents/shipped/itd-2609090559376002-sub-agent-transcript-capture.md:58 — "records already filed under a composite identifier, are `itd-2609091718566731`"
- ac-4 — MET_WITH_CONCERNS: The refusal exists and the sub-agent path reaches it by construction: the drain calls the same Capture, which refuses on scanner.Unavailable before any redaction, and a residual survivor refuses the write; the lineage scalars are framed into the same two-stage pass so no externally supplied field bypasses the gate. Concern: NOTHING tests it. A grep for the refusal string or for Unavailable across internal/core/history's tests returns nothing — not on the sub-agent path and not on the session path — so the spec's claim that `the tests assert it on the sub-agent path specifically rather than inferring it` is false; the four tests it names cover lineage redaction, a blocking residual and a malformed scalar, none of which degrades the scanner. The guard itself also predates this delivery (88d22349), so the delivered work inherits it rather than establishing it.
  evidence: internal/core/history/history.go:221 — "return CaptureResult{}, fmt.Errorf("history: refusing to capture with a degraded scanner: %s", reason)"
  evidence: internal/core/history/history.go:231 — "text, err := frameLineage(meta, raw)"
  evidence: internal/core/history/lineage_test.go:186 — "func TestBlockingSpanInAgentTypeRefusesTheWrite(t *testing.T) {"
  evidence: internal/core/history/staging_lifetime.go:183 — "f.Permanent = true"
  evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:335 — "blocking span applies by construction; the tests assert it on the sub-agent"
  evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:390 — "**ac-4 (a degraded scanner refuses).** Unchanged `Capture`, asserted on the"
- ac-5 — MET: A repeat presentation of the same bytes is a no-op before anything is written: Capture short-circuits on (source_sha256, session_id, agent_id, kind) and returns the stored record with Wrote=false, and Stage does the same one directory up on (session, agent) with a content compare. The bounding case is held too — two sub-agents of one session with byte-identical transcripts still store two records. The divergent-body branch does NOT contradict this criterion: a body that is neither a prefix nor an extension of the stored one is a different transcript, not the same one presented twice, and it is deliberately kept side by side because the store holds the only copy of each; the prefix cases collapse (longer supersedes, shorter is a no-op).
  evidence: internal/core/history/history.go:205 — "if r.SourceSHA256 == sourceSHA && r.SessionID == sessionID &&"
  evidence: internal/core/history/history.go:207 — "return CaptureResult{Record: r, Wrote: false}, nil"
  evidence: internal/core/history/lineage_test.go:261 — "func TestSubagentCaptureIdempotentOnSourceSHA(t *testing.T) {"
  evidence: internal/core/history/lineage_test.go:227 — "func TestTwoSubagentsWithIdenticalBytesBothStore(t *testing.T) {"
  evidence: internal/core/history/history.go:399 — "case strings.HasPrefix(body, priorBody):"
  evidence: internal/core/history/lineage_test.go:534 — "func TestDivergentTranscriptsForOneAgentBothStore(t *testing.T) {"
  evidence: internal/core/history/staging_sidecar_test.go:114 — "func TestStageIdempotencyIsPerSessionAndAgent(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: The marker is real and operator-visible, not internal-only: an absent agent_transcript_path writes a stderr line AND a persistent per-repo marker via NoteSubagentGap, and `abcd history staged` renders a NOTE naming the event, the count and the first sighting — the test drives the actual CLI verb and asserts the rendered output, and every hook path returns nil so the sub-agent is never blocked. Concerns: (a) the on-disk half covers ONLY the absent-field class; the other four miss classes (absent/irregular/over-cap transcript, unusable agent_id, unresolvable repository) report on stderr and leave nothing on disk, against the intent's own `on stderr and on disk`; (b) the marker needs a resolvable rootSHA, so a completion that resolves no store records nothing at all; (c) the NOTE is rendered only in the human output — `history staged --json` deliberately omits it; (d) TestHookSubagentStopAlwaysExitsZero asserts the exit code only, not the `zero records and a non-empty stderr reason` the spec says it asserts.
  evidence: internal/surface/cli/hook_subagent.go:103 — "if err := history.NoteSubagentGap(rootSHA, in.Event); err != nil {"
  evidence: internal/core/history/locate.go:193 — "func NoteSubagentGap(rootSHA, event string) error {"
  evidence: internal/surface/cli/history.go:223 — "NOTE: this harness fired %s %d time(s) without an agent_transcript_path (first %s)."
  evidence: internal/surface/cli/hook_subagent_stop_test.go:230 — "stdout, _ := runHook(t, "", "history", "staged")"
  evidence: internal/surface/cli/hook_subagent.go:88 — "return nil // never an error: exit 2 is this event's BLOCKING code"
  evidence: internal/surface/cli/hook_subagent.go:124 — "return warn("%v; staging nothing", err)"
  evidence: internal/surface/cli/history.go:218 — "per-repo fact rather than a staged entry, so it is reported in"
  evidence: internal/surface/cli/hook_subagent_stop_test.go:263 — "_, stderr, failed := runHookAllowingFailure(tc.stdin, "hook", "subagent-stop")"
  evidence: .abcd/development/intents/shipped/itd-2609090559376002-sub-agent-transcript-capture.md:53 — "**A completion that yields nothing is reported**, on stderr and on disk"

Gap audit:
- honoured:
  - The capture gap is closed at its front door: SubagentStop is registered in the shipped hook manifest and the verb behind it exists and is wired, so the corpus starts accruing sub-agent transcripts
    evidence: hooks/hooks.json:55 — ""SubagentStop": ["
    evidence: internal/surface/cli/cli.go:1463 — "hookCmd.AddCommand(newSubagentStopCommand())"
    evidence: internal/surface/cli/cli.go:142 — "{"hook", "subagent-stop"},"
  - Lineage is carried in explicit record fields rather than a composite identifier, and the filename is decorative only — nothing decodes it back into fields
    evidence: internal/core/history/history.go:74 — "AgentID string `json:"agent_id,omitempty"`"
    evidence: internal/core/history/store.go:214 — "func recordFilename(capturedAt time.Time, sessionID, agentID string) string {"
    evidence: internal/core/history/store.go:211 — "string back into fields — listRecords parses frontmatter and never the"
  - Every externally supplied lineage scalar passes the SAME two-stage redaction gate as the body, with the frame refused rather than guessed at if it does not survive
    evidence: internal/core/history/history.go:231 — "text, err := frameLineage(meta, raw)"
    evidence: internal/core/history/history.go:284 — "scalars, body, err := unframeLineage(redacted)"
    evidence: internal/core/history/lineage_test.go:139 — "func TestLineageFieldsAreRedactedWithTheBody(t *testing.T) {"
  - Feed, don't fork: the hook stages and never captures, and the drain runs the pre-existing unchanged Capture, so there is one redaction mechanism and not two
    evidence: internal/surface/cli/hook_subagent.go:22 — "The hook STAGES; it does not capture."
    evidence: internal/core/history/staging.go:727 — "cr, err := Capture(repoRoot, rootSHA, body, s.captureMeta())"
    evidence: internal/surface/cli/hook_subagent_stop_test.go:108 — "func TestHookSubagentStopStagesRatherThanCaptures(t *testing.T) {"
  - Idempotence in both directions: the same transcript twice is one record, and two sub-agents whose transcripts are byte-identical are still two
    evidence: internal/core/history/history.go:205 — "if r.SourceSHA256 == sourceSHA && r.SessionID == sessionID &&"
    evidence: internal/core/history/lineage_test.go:227 — "func TestTwoSubagentsWithIdenticalBytesBothStore(t *testing.T) {"
  - A repository that cannot be resolved from a removed worktree cwd is still found through the spawning session's note, rather than guessed or dropped — the isolated implementation lanes are exactly the transcripts that would otherwise be lost
    evidence: internal/surface/cli/hook_subagent.go:168 — "func resolveSubagentStore(in hookInput) (rootSHA, via string) {"
    evidence: internal/surface/cli/hook_subagent_stop_test.go:171 — "func TestHookSubagentStopResolvesTheRepoThroughTheSession(t *testing.T) {"
    evidence: internal/core/history/locate.go:218 — "func SubagentGap(rootSHA string) (SubagentGapNote, bool, error) {"
  - Staged unredacted text is bounded and reported rather than silently accumulating: a live drain per prompt, an overdue TTL that buys priority not deletion, an all-repositories survey, and a quarantine for a deterministic refusal
    evidence: internal/surface/cli/cli.go:1534 — "dr, err := history.Drain(captureRoot(cwd), rootSHA, livePromptDrainBudget)"
    evidence: internal/core/history/staging_lifetime.go:177 — "func classifyDrainFailure(sdir, rootSHA string, s Staged, stagedBytes []byte, capErr error) DrainFailure {"
    evidence: internal/surface/cli/history.go:245 — "state = "OVERDUE (staged more than " + history.StagedTTL.String() + " ago), awaiting redaction""
- diverged:
  - "Provenance enough to read the result back" is delivered in the core library but NOT on an operator surface in this range: ListForSession's only non-test caller is reconstruct.go from the excluded intent, `history show <session-id>` returns the main-thread record alone, and the human render of list/show prints neither agent_id nor agent_type — the operator's only route is `history list --json` plus hand filtering, which the plugin page does not document
    evidence: internal/core/history/reconstruct.go:317 — "records, err := ListForSession(rootSHA, opts.SessionID)"
    evidence: internal/surface/cli/history.go:141 — "fmt.Fprintf(w, "%s %s %s redacted secrets=%d home=%d\n","
    evidence: internal/core/history/history.go:437 — "3. a session id, preferring the MAIN-THREAD record and newest first. A"
    evidence: commands/history.md:48 — "Summarise each record newest-first: `captured_at`, `session_id`, `source_kind`,"
  - Capture is deferred, not at completion: the press release says the sub-agent "leaves the same redacted record its main thread does", but the hook only stages, and the record exists only after a later drain — a repository nobody reopens keeps raw staged text and stores nothing, with age buying priority and reporting rather than storage
    evidence: internal/surface/cli/hook_subagent.go:133 — "res, err := history.Stage(rootSHA, meta, raw)"
    evidence: internal/surface/cli/cli.go:1333 — "if dr, err := history.Drain(captureRoot(cwd), det.RootSHA, sessionStartDrainBudget); err == nil {"
    evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:282 — "buys priority and volume and nothing else — an overdue transcript is never"
  - The spec's account of how ac-4 is satisfied is not the delivered reality: it says the degraded-scanner refusal is asserted on the sub-agent path by four named tests, and none of those tests degrades a scanner — no test in the repository covers the refusal at all
    evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:390 — "**ac-4 (a degraded scanner refuses).** Unchanged `Capture`, asserted on the"
    evidence: internal/core/history/history.go:221 — "return CaptureResult{}, fmt.Errorf("history: refusing to capture with a degraded scanner: %s", reason)"
    evidence: internal/core/history/lineage_test.go:186 — "func TestBlockingSpanInAgentTypeRefusesTheWrite(t *testing.T) {"
  - "Reported on stderr AND on disk" holds for one miss class only: the absent agent_transcript_path writes a marker, while an unreadable, irregular or over-cap transcript, an unusable agent_id and an unresolvable repository leave a stderr line and nothing durable
    evidence: internal/surface/cli/hook_subagent.go:101 — "if in.AgentTranscriptPath == "" {"
    evidence: internal/surface/cli/hook_subagent.go:124 — "return warn("%v; staging nothing", err)"
    evidence: .abcd/development/intents/shipped/itd-2609090559376002-sub-agent-transcript-capture.md:53 — "**A completion that yields nothing is reported**, on stderr and on disk"
  - The gap NOTE is a human-render-only fact: `abcd history staged --json` deliberately keeps the staged array unchanged, so a machine consumer of the staged surface cannot see that this harness delivers no sub-agent transcripts
    evidence: internal/surface/cli/history.go:218 — "per-repo fact rather than a staged entry, so it is reported in"
    evidence: internal/surface/cli/history.go:220 — "return render(cmd.OutOrStdout(), *asJSON, staged, func(w io.Writer) {"
  - TestHookSubagentStopAlwaysExitsZero asserts less than the spec credits it with: the table checks the exit code only, never the "zero records and a non-empty stderr reason" the spec claims each case asserts
    evidence: internal/surface/cli/hook_subagent_stop_test.go:263 — "_, stderr, failed := runHookAllowingFailure(tc.stdin, "hook", "subagent-stop")"
    evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:404 — "transcript path, each asserting exit 0, zero records and a non-empty stderr"
- missing:
  - The flush-race measurement — the intent's own falsifier — is not delivered: DrainResult.Extended is wired as the counter, but no instrumented run and no residual rate exists, so the mechanism's key uncertainty is still open by the record's own admission
    evidence: internal/core/history/staging.go:725 — "res.Extended++"
    evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:364 — "4. **The measurement**, still outstanding: an instrumented run over real sessions"
    evidence: .abcd/development/intents/shipped/itd-2609090559376002-sub-agent-transcript-capture.md:88 — "the rate over a real corpus has not been measured"
  - The decision record refining adr-29 is deferred and unminted, so the spec stands as the decision of record; the blocker named is an unresolved disagreement between two committed surfaces about how an ADR id is allocated
    evidence: .abcd/development/intents/shipped/itd-2609090559376002-sub-agent-transcript-capture.md:89 — "**The decision record refining adr-29 is deferred and unminted.**"
    evidence: .abcd/development/specs/closed/spc-2609090624222051-sub-agent-transcript-capture.md:142 — "The move from a composite identifier to explicit lineage fields refines"
  - No test anywhere exercises the degraded-scanner refusal that ac-4 rests on, on the sub-agent path or any other — the criterion's whole content is an untested guard inherited from an earlier change
    evidence: internal/core/history/history.go:220 — "if unavail, reason := sc.Unavailable(); unavail {"
    evidence: internal/core/history/lineage_test.go:139 — "func TestLineageFieldsAreRedactedWithTheBody(t *testing.T) {"
  - No verb takes a session id and returns that session's sub-agents within this delivered range; the session-keyed reader exists in core and its only consumer ships with a different intent, so ac-3's operator story is completed by work outside this record
    evidence: internal/core/history/history.go:490 — "func ListForSession(rootSHA, sessionID string) ([]Record, error) {"
    evidence: internal/core/history/reconstruct.go:317 — "records, err := ListForSession(rootSHA, opts.SessionID)"

Scope-condition dispositions:
- cond-2609090624228012 — survived: The delivered hook does exactly what the condition assumed: an absent agent_transcript_path stages nothing rather than guessing at the on-disk layout, and it records that it did so in a durable per-repo marker that `history staged` renders.
  evidence: internal/surface/cli/hook_subagent.go:101 — "if in.AgentTranscriptPath == "" {"
  evidence: internal/core/history/locate.go:193 — "func NoteSubagentGap(rootSHA, event string) error {"
  evidence: internal/surface/cli/history.go:223 — "NOTE: this harness fired %s %d time(s) without an agent_transcript_path (first %s)."
- cond-2609090624226418 — untested: Nothing in the delivery ran at the assumed working scale: the byte-and-count drain budget and a small concurrent-stage test bound the mechanism, but no run over a few hundred sub-agents per month, a multi-megabyte transcript, or a session of roughly a hundred sub-agents either exercised or contradicted the assumption.
- cond-2609090624223171 — survived: One reader does serve both: the sub-agent hook calls the same readTranscript the session-end hook calls, and the settle predicate treats the transcript as line-delimited JSON by validating the final non-blank line, with the drain's prefix comparison relying on the same line-local shape.
  evidence: internal/surface/cli/hook_subagent.go:278 — "raw, err = readTranscript(path)"
  evidence: internal/surface/cli/cli.go:1246 — "raw, err := readTranscript(in.TranscriptPath)"
  evidence: internal/core/history/staging.go:872 — "return json.Valid(bytes.TrimSpace(last))"
- cond-2609090624228630 — survived: Store resolution is the cwd first and the store that has already seen the spawning session second, with an ambiguous session refused rather than guessed; nothing in the delivered hook decodes the harness's project directory name.
  evidence: internal/surface/cli/hook_subagent.go:175 — "if det, err := ahoy.Detect(cwd); err == nil && det.RootSHA != "" {"
  evidence: internal/surface/cli/hook_subagent.go:183 — "if sha, err := history.SessionRepo(in.SessionID); err == nil {"
  evidence: internal/core/history/locate_test.go:39 — "func TestSessionRepoRefusesAnAmbiguousSession(t *testing.T) {"
- cond-2609090624222424 — survived: The already-gone worktree is the case the delivery was reshaped around and it is proven by test: a sub-agent whose worktree the harness removed still resolves its store through the spawning session's recorded tie, and the stderr line names the route taken.
  evidence: internal/surface/cli/hook_subagent.go:159 — "cwd, and the harness REMOVES the worktree when the agent stops — so the"
  evidence: internal/surface/cli/hook_subagent_stop_test.go:171 — "func TestHookSubagentStopResolvesTheRepoThroughTheSession(t *testing.T) {"
  evidence: internal/surface/cli/hook_subagent.go:178 — "_ = history.NoteSessionRepo(det.RootSHA, in.SessionID)"
- cond-2609091722261577 — survived: The condition asserts the race is UNVERIFIED with two self-counting mitigations, and the delivered reality matches it exactly: a bounded settle wait outside the staging lock, a drain-time re-read that replaces staged bytes only on a strict byte-prefix extension, an Extended counter, and no criterion or code path that assumes the question settled.
  evidence: internal/surface/cli/hook_subagent.go:274 — "func readSettledTranscript(path string) ([]byte, bool, error) {"
  evidence: internal/core/history/staging.go:725 — "res.Extended++"
  evidence: internal/core/history/staging.go:796 — "This does NOT settle whether SubagentStop fires before the flush. It bounds"
  evidence: .abcd/development/intents/shipped/itd-2609090559376002-sub-agent-transcript-capture.md:75 — "is UNVERIFIED"
## Grounds

- pursued: we expect the harness's sub-agent completion event to carry the finished sub-agent's own transcript path, so capture reuses the existing stage-then-redact path unchanged and never reads the harness's undocumented directory layout, and we expect reconstruction and telemetry to need no new instrumentation because the raw transcripts already carry per-message token counts, timestamps, models, agent attribution and tool calls; it is shown wrong if the event fires before the sub-agent transcript is readable, if per-completion staging degrades a session that spawns many sub-agents, if the payload is absent on a supported harness version, or if the telemetry fields vary enough across versions that derived measures cannot be compared
- pursued: narrowed to capture alone after the intent was split in three — we expect the harness's sub-agent completion event to carry the finished sub-agent's own transcript path, so capture reuses the existing stage-then-redact path unchanged and never reads the harness's undocumented directory layout, and we expect explicit lineage fields on the record to carry what a composite session id could not, because the payload delivers the spawning session untruncated and the agent id separately; it is shown wrong if the event fires before the sub-agent transcript is readable at a rate the two mitigations cannot absorb, if per-completion staging degrades a session that spawns many sub-agents, or if the payload is absent on a supported harness version
