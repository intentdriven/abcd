---
id: itd-2610301428390406
slug: before-an-agent-submits-a-filled-in-form
spec_id: spc-2610301431131636
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# An agent can practise a filled-in form before it counts

## Press Release

> Before an agent submits a filled-in form to abcd, it can do a practice run that checks the form and shows what would be written, writing nothing.

## Why This Matters

Split out of itd-2610301346085246 on 2026-10-30 at the product thinker's ruling after the planning interview, keeping every decision and criterion the interview confirmed; this record depends on no other: the practice run goes through the same validator each writing command already has.

Today the only way an agent learns whether a filled-in input is right is to
submit it: a wrong one is refused and nothing is written, but the agent finds
out only by trying, as the 2026-10-30 `ideate` refusal showed. The nearest
measured evidence is SWE-agent's edit-time lint guardrail, which resolved
18.0% of SWE-bench Lite against 15.0% without it in the paper's ablations
(GPT-4 Turbo; Yang et al. 2024, https://arxiv.org/abs/2405.15793): checking an
action before it takes effect helped the agent.

## Mechanism

- We expect agents to submit a wrong input less often, because a practice run
  shows the refusal before the input counts. Shown wrong if refused writes per
  session do not fall where a practice run is available.

## Scope Conditions

- Agents that run abcd's commands and read their output as text. <!-- cond: cond-2610301431133725 -->
- Any host that can run the abcd program, with the plugin or without it. <!-- cond: cond-2610301431137567 -->
- The agent asks the running program before it fills in an input, as each <!-- cond: cond-2610301431135398 -->
  command page's first line tells it to.

## Acceptance Criteria

Each confirmed by the product thinker on 2026-10-30, one at a time.

- **Given** a filled-in input, **when** an agent runs the writing command as a
  practice run, **then** the binary checks it through the same validator the
  real write uses, reports what it would write or why it would refuse, and
  writes nothing: the working tree and every abcd store are byte-identical
  afterwards.

## Decisions

Ruled by the product thinker on 2026-10-30, in the planning interview of
itd-2610301346085246 (its decision 8), and carried here when it was split the
same day:

1. **A practice run is wanted**: a command that writes a record from a
   filled-in input can check it and show what it would write, writing
   nothing, through the same validator the real write uses. Chosen over not
   at all.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-997517c0f4ba -->
Fidelity review — receipt rcp-997517c0f4ba (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:aff70dd48e911ba0ec81a665621e54c5d151a5bd58538dc6619a3674cff07e55
Input attestations: diff:c6e6bfb50e55580f38495c81b3dbd74a0a783095..8984ae4a5dc1b4e187d0f701f365fc4d28026035 (lane-1, build/run-2610312053380475-lane-1, PR #1313)@sha256:f9413e7eebfd9176bd3e3d93d5d88bf3b0e7dd4e83e05e4231eec4cfa1fcf7cf; diff:0b024dd0ce6ed45ab6d26ef453e7752f71b81013..66d21b2d5e4a91509c6db08cecc3aaea74e83932 (lane-2, build/run-2610312053380475-lane-2)@sha256:718ad9603f8e1cf8d96a58adc096820381e5f6383aa71e25fafaf2576fda86c1; log:.abcd/.work.local/run/run-2610312053380475/lane-2/dod.log (make preflight at 66d21b2d5e4a)@-;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: All fourteen commands that read a -json input take --dry-run. Each runs its owning core's Plan stage, and the real write is that same Plan followed by Apply. A tree-wide test hashes the repository (its .git included) and abcd's home before and after each practice run and refusal. It asserts the practice run's refusal equals the real run's and that the real run writes exactly the planned paths, and it passed in the lane's preflight. Concern 1: for reading ingest and scribe ingest, the capture verbs' own checks (ordering gate, record schema validation, size cap, redaction) are left out of the practice run, so a practice run can pass and the real write can still be refused. That validator is narrower than the real write's. Concern 2: past the point where reading ingest records a refusal, and where launch ship renders a payload, the refusal is no longer word-for-word the real run's. Concern 3: launch ship's archive/payload dry-run path is not covered by the tree-wide fixture. No follow-up record for any of these was captured in the delivered range.
  evidence: internal/surface/cli/dryrun_test.go:39 — "var dryRunPending = map[string]bool{}"
  evidence: internal/surface/cli/dryrun_test.go:77 — "t.Errorf("%s reads a filled-in input and takes no --dry-run", path)"
  evidence: internal/surface/cli/dryrun_test.go:423 — "if d := hashDiff(before, treeHashes(t, roots...)); len(d) != 0 {"
  evidence: internal/surface/cli/dryrun_test.go:441 — "if dcode == 0 || dcode != rcode || derr != rerr || dout != rout {"
  evidence: internal/surface/cli/dryrun_test.go:482 — "t.Errorf("the real run wrote %s, which the practice run did not plan (planned %+v)", p, got.WouldWrite)"
  evidence: .abcd/.work.local/run/run-2610312053380475/lane-2/dod.log:1572 — "ok github.com/intentdriven/abcd/internal/surface/cli 327.658s"
  evidence: internal/core/scribe/ingest.go:165 — "out, m, err := checkIngest(req)"
  evidence: internal/core/scribe/plan.go:35 — "const ledgerPending = "the capture verb mints its id and applies its own redaction and refusals when it writes""
  evidence: internal/core/reading/ingest.go:1127 — "return fmt.Errorf("reading: %s; the real run would record the refusal at %s", reason, rel)"
  evidence: internal/core/ideate/plan_test.go:23 — "func TestPlanRecordRefusesAnUnresolvableCitationBeforeAnyWrite(t *testing.T) {"

Gap audit:
- honoured:
  - Every writing command with a filled-in (-json) input takes --dry-run, and the tree-wide walk fails any future one that lacks it
    evidence: internal/surface/cli/dryrun_test.go:43 — "func filledInInputCommands() map[string]*cobra.Command {"
    evidence: internal/surface/cli/reading.go:347 — "ingestCmd.Flags().BoolVar(&readingDryRun, "dry-run", false, dryRunFlagUsage)"
    evidence: internal/surface/cli/cli.go:1196 — "graveyardCmd.Flags().BoolVar(&graveyardDryRun, "dry-run", false, dryRunFlagUsage)"
  - The practice run shares the real write's checks by construction (a plan/apply split in each owning core)
    evidence: internal/core/ideate/record.go:189 — "func PlanRecord(repoRoot, slug string, store Store, raw []byte, at time.Time) (RecordPlan, error) {"
    evidence: internal/core/release/plan.go:36 — "func PlanIngest(root string, current surface.Snapshot, raw []byte, at time.Time) (IngestPlan, error) {"
    evidence: internal/core/lifeboat/synthesis_review.go:119 — "func PlanReview(lifeboatDir, sourceRepo string, raw []byte, scan SecretScan) (ReviewPlan, error) {"
    evidence: internal/core/memory/plan.go:238 — "func PlanIngest(req IngestRequest) (IngestPlan, error) {"
  - The practice run reports what it would write (would_write) or do (would_do: tag, push, release), in text and in --json
    evidence: internal/surface/cli/dryrun.go:41 — "fmt.Fprintf(w, "%s — practice run: the input is checked and nothing is written\n", verb)"
    evidence: internal/core/release/plan.go:91 — "{Effect: "push", Target: "refs/tags/" + nextTag + " to origin, by the release workflow (this command pushes nothing)"},"
  - The working tree, git directory and abcd's home are byte-identical after a practice run and after a refused one
    evidence: internal/surface/cli/dryrun_test.go:403 — "roots := []string{fx.repo, fx.home}"
    evidence: internal/surface/cli/dryrun_test.go:437 — "if d := hashDiff(before, treeHashes(t, roots...)); len(d) != 0 {"
  - A practice run that would write to the network or fetch is refused rather than performed (launch ship --fetch-baseline)
    evidence: internal/surface/cli/ship.go:409 — "if dryRun && fetchBaseline {"
    evidence: internal/surface/cli/dryrun_test.go:543 — ""ship with a fetch": {[]string{"launch", "ship", "--changelog-json", payload, "--dry-run", "--fetch-baseline"}, "--fetch-baseline"},"
- diverged:
  - Same validator as the real write: for reading ingest and scribe ingest, the capture verbs' ordering gate, schema validation, size cap and redaction are left out of the practice run, so a practice run that passes can still be refused by the real write
    evidence: internal/core/scribe/plan.go:35 — "const ledgerPending = "the capture verb mints its id and applies its own redaction and refusals when it writes""
    evidence: commands/scribe.md:101 — "when they write, so a practice run that passes can still be stopped there."
  - The refusal is the real run's in the same words: past reading ingest's identity point, the practice run says where the refusal would be recorded instead of recording it, so the message differs from the real run's
    evidence: internal/core/reading/ingest.go:1127 — "return fmt.Errorf("reading: %s; the real run would record the refusal at %s", reason, rel)"
    evidence: internal/surface/cli/ship.go:625 — "preflightNote = "; the real run would record the pre-flight report in the local tier""
- missing:
  - Tree-wide proof of launch ship's archive/payload practice-run path (the fixture declares no plugin archive, so the pre-flight, archive-pin and staged-payload branch under --dry-run is not exercised)
    evidence: internal/surface/cli/dryrun_test.go:231 — ""abcd launch ship": func(t *testing.T) dryRunFixture {"
    evidence: internal/surface/cli/ship.go:590 — "scratch, err := os.MkdirTemp("", "abcd-ship-")"
  - A recorded follow-up for the capture-check gap in the scribe and reading practice runs: neither lane's diff adds an issue under .abcd/work/issues/
    evidence: internal/core/reading/plan.go:94 — "Pending: "the item's id is minted, and its record composed and redacted, by the ledger writer when it is recorded"})"

Scope-condition dispositions:
- cond-2610301431133725 — survived: The practice run prints its report as text (a 'practice run' heading and 'would ...' lines) next to the --json form, and the tree-wide test checks that text output for every command.
  evidence: internal/surface/cli/dryrun.go:41 — "fmt.Fprintf(w, "%s — practice run: the input is checked and nothing is written\n", verb)"
  evidence: internal/surface/cli/dryrun_test.go:427 — "!strings.Contains(text, "practice run") || !strings.Contains(text, "would ") {"
- cond-2610301431137567 — survived: --dry-run is a flag in the binary's own command tree, so a host can reach it with or without the plugin. The plugin's command pages list it in argument-hint as well.
  evidence: internal/surface/cli/dryrun.go:21 — "const dryRunFlagUsage = "check the input and report what would be written; write nothing""
  evidence: commands/reading.md:4 — "ingest --reading-json < path> [--dry-run]"
- cond-2610301431135398 — untested: Nothing in either lane exercises an agent asking the running program before it fills in an input. The edited command pages add a 'Practise before it counts' paragraph and no first-line instruction of that kind, which belongs to the sibling intent's page block.
<!-- abcd-review-end receipt=rcp-997517c0f4ba -->

## Grounds

- pursued: agents are abcd's main users, and on 2026-10-30 one built an input from an out-of-date page and was refused; we expect fewer such refusals once agents ask the running program itself, and we are wrong if agents still build inputs from pages.
