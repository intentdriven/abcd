---
id: itd-2609212137128014
slug: abcd-lab-mechanises-the-lab-conventions-three-hand-run
spec_id: spc-2609212141418943
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-22]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-75, itd-59]
related_adrs: [adr-2609212115255771]
---

# abcd lab mechanises the lab conventions three hand-run experiments proved

## Press Release

> **`abcd lab` mints, preflights, records, sweeps and harvests a lab, and the procedure the labs converged on becomes a discipline record.**
>
> "Three labs, and by the third the procedure returned SHIP with zero findings, because every amendment had been written down and the scaffolding had been rebuilt by hand each time," said a technical facilitator reading the capstone. "Now the verb builds the scaffolding, the procedure is a record, and no rule can be forgotten under pressure."

## Why This Matters

Between 31 August and 1 September eight labs ran under `~/.abcd/lab/`, three of them the core series; the capstone's series review shows review churn falling from four rounds with three application failures to one round to SHIP as the procedure accumulated amendments, and a draft intent for the verb family was written as evidence and never filed. The recording model (evidence at operator level, knowledge through ceremony, pointers in the local tier) held across the runs. Ruled 2026-09-21: file it from the capstone's text; the four product findings the series left unfiled are captured on the same branch.

## Mechanism

We expect a mechanised lab to reproduce the third lab's SHIP on its first run, because the variable that moved across the series was the procedure and the procedure is what the verb encodes; shown wrong if a lab run under the verb needs more review rounds than lab 3 did.

## Scope Conditions

- Holds on a machine whose lab store is `~/.abcd/lab/` keyed as the other machine-scoped stores are; the repository never holds lab evidence. <!-- cond: cond-2609212141418357 -->
- Holds while a real-session smoke stage is available: offline suites passed while the plugin was unloadable by the host. <!-- cond: cond-2609212141414079 -->

## What's In Scope

- **The verb family** `abcd lab mint | preflight | record | sweep | harvest`: the home minted with its registry entry and snapshot pin; the preflight artefact (harness isolation, dual-binary vintage); probe-record scaffolding; the retraction sweep (grep the pattern, not the instance); harvest assembly against the lifeboat's section shape.
- **The procedure as a discipline record**: the amendments across the three chains, present tense, host-agnostic.
- **The recording model** as the recorded convention: evidence at `~/.abcd/lab/`, knowledge through ceremony, pointers in the local tier; the never-in-repo rules.
- **Halt-and-record on a gate refusal** as a lab rule the verb enforces.
- **No cost claim**: the series could not measure token cost; the verb records what the runner reports and claims nothing more.

## What's Out of Scope

- Auto-merge for lab filings.
- Labs as a grouping of the record.
- A next-labs runner; the menu stays a note.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Filed from the capstone's draft with its ten evidence items (ruled 2026-09-21).
2. Auto-merge for lab-derived work stays out; a lab's findings are captured and drained like any other.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `abcd lab mint <question>`, **when** it runs, **then** a lab home exists under the machine-scoped lab store with a registry entry, the snapshot pin and the lifecycle's sections scaffolded, and nothing is written into the repository.
- **Given** `abcd lab preflight`, **when** it runs, **then** the harness-isolation and dual-binary checks are written as an artefact, and a failed check halts the lab naming it.
- **Given** a lab's corrections, **when** `abcd lab sweep` runs, **then** every instance of a retracted pattern is listed and an unapplied correction fails the sweep.
- **Given** a finished lab, **when** `abcd lab harvest` runs, **then** the harvest is assembled in the lifeboat's section shape with the probe records cited, and its product findings are listed as capture candidates.
- **Given** a gate refusal during a lab, **when** it occurs, **then** the lab halts and records it as a finding rather than adapting around it.
- **Given** the discipline record, **when** it is read, **then** it carries the procedure's amendments in present tense, host-agnostic.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-2e848c471a09 -->
Fidelity review — receipt rcp-2e848c471a09 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:db1fb13e6920b3bc6ba25cd8ce987662648bdc1d6cf5ce1ea5f9c67bc4bbbd82
Input attestations: diff:internal/core/lab, internal/surface/cli/lab.go, commands/lab.md and intents/disciplines/itd-2609251624540864 at chore/audit-run-a-1 5b4a43b6 (git ls-tree -r; spc-2609212141418943 closed, itd-2609212137128014 shipped)@sha256:fba4544c04fa39af51fde0aabe9789398732117e51dee5531efbca65bc59499d;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Mint lays the lab home under ~/.abcd/lab/< root-sha>/ with one registry line, a detached snapshot at the pin, INTENTION.md carrying snapshot_pin and the lifecycle sections; the test asserts the registry line, the snapshot HEAD equals the pin, no remote, and git status --porcelain --ignored of the repository is empty afterwards
  evidence: internal/core/lab/lab.go:9 — "// ~/.abcd/lab/< root-sha>/index.jsonl one registry line per lab"
  evidence: internal/core/lab/mint.go:35 — "func Mint(repoRoot, question, pin string) (Minted, error) {"
  evidence: internal/core/lab/mint.go:107 — "if _, err := gitutil.Run(snap, "checkout", "--quiet", "--detach", e.Pin); err != nil {"
  evidence: internal/core/lab/lab_test.go:106 — "func TestMintLaysDownALabAndWritesNothingInTheRepo(t *testing.T) {"
  evidence: internal/core/lab/lab_test.go:147 — "if got := r.Git("status", "--porcelain", "--ignored"); got != "" {"
- ac-2 — MET: Preflight runs the harness-isolation checks (home, snapshot, remotes, hooks) and the dual-binary checks (work, pinned, test), writes state/preflight.md and .json, and on any failure halts with ErrHalted naming the failed check ids and recording a gate finding; the test asserts the halt names binary.work, the artefact records both groups, and Record refuses while halted
  evidence: internal/core/lab/preflight.go:31 — "GroupIsolation = "harness-isolation""
  evidence: internal/core/lab/preflight.go:68 — "func Preflight(repoRoot, id string) (Preflighted, error) {"
  evidence: internal/core/lab/preflight.go:128 — "return res, fmt.Errorf("%w: the preflight refused %s; recorded as %s in %s", ErrHalted, strings.Join(failed, ", "), fid, l.display(findingsName))"
  evidence: internal/core/lab/lab_test.go:234 — "func TestPreflightHaltsNamingTheFailedCheckAndRecordsIt(t *testing.T) {"
- ac-3 — MET: Sweep searches every retracted literal across the lab's own documents, lists every instance by file and line, and an unapplied or unreadable correction fails the sweep with a halt; the test plants two retractions and asserts the instances in findings.md and review-1-triage.md are listed while probe stdout and the snapshot are not, and the sweep halts
  evidence: internal/core/lab/sweep.go:109 — "// Sweep verifies every correction the lab recorded is applied: each retracted"
  evidence: internal/core/lab/sweep.go:118 — "func Sweep(repoRoot, id string) (Swept, error) {"
  evidence: internal/core/lab/lab_test.go:505 — "func TestSweepListsEveryInstanceAndFailsOnAnUnappliedCorrection(t *testing.T) {"
- ac-4 — MET: Harvest assembles the lifeboat's sections from the intention, the findings and the probe records, refuses and writes nothing on a finding no record backs, and lists each product finding as a capture candidate carrying the abcd capture command; the tests cover the assembled shape with probe citations and the refusal
  evidence: internal/core/lab/harvest.go:65 — "func Harvest(repoRoot, id string) (Harvested, error) {"
  evidence: internal/core/lab/harvest.go:127 — "Command: "abcd capture " + shellQuote(f.Title) + " --found-during " + shellQuote(found),"
  evidence: internal/core/lab/harvest.go:135 — "return res, fmt.Errorf("%w: %d finding citation(s) cannot be verified from the lab's records; nothing was written", ErrHalted, len(res.Gaps))"
  evidence: internal/core/lab/lab_test.go:661 — "func TestHarvestAssemblesTheLifeboatShapeCitingProbes(t *testing.T) {"
  evidence: internal/core/lab/lab_test.go:733 — "func TestHarvestRefusesAFindingItsRecordsCannotBackAndWritesNothing(t *testing.T) {"
- ac-5 — MET_WITH_CONCERNS: haltAndRecord writes a gate finding and marks the lab halted so no probe is recorded until the gate passes, and a repeated refusal reuses its finding; the concern is reach: the verb enforces this for its own two gates (preflight, sweep) and the harvest's citation gaps only, while a refusal by the world's own gates during the mutate stage or a STOP condition is the discipline record's rule with nothing mechanical behind it, since the verb runs nothing inside the snapshot
  evidence: internal/core/lab/findings.go:145 — "// haltAndRecord is the lab rule a gate refusal enforces: the refusal is written"
  evidence: internal/core/lab/findings.go:137 — "for _, g := range []string{"preflight", "sweep"} {"
  evidence: internal/core/lab/lab_test.go:272 — "// Halted, the lab records no probe: the refusal is not adapted around."
  evidence: .abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention-to-discard-and-every.md:172 — "- **Given** a gate refusal or a STOP condition during a lab, **when** it occurs,"
  evidence: commands/lab.md:60 — "The verb runs nothing:"
- ac-6 — MET: the discipline record itd-2609251624540864 states the six-stage procedure in present tense as the rule every lab inherits, with the amendments the hand-run labs earned folded in and a BDD gate section; no agent-harness name appears in it
  evidence: .abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention-to-discard-and-every.md:5 — "kind: discipline"
  evidence: .abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention-to-discard-and-every.md:20 — "A lab is a throwaway world pinned at one commit, run to answer one question. It"
  evidence: .abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention-to-discard-and-every.md:164 — "## The gate"

Gap audit:
- honoured:
  - the five-verb family is wired on the CLI and the plugin page
    evidence: internal/surface/cli/lab.go:60 — "Use: "mint < question>","
    evidence: internal/surface/cli/lab.go:200 — "Use: "harvest < lab-id>","
    evidence: commands/lab.md:28 — ""${CLAUDE_PLUGIN_ROOT}/abcd" lab mint --json "< question>""
  - no lab verb writes into the repository; evidence stays at operator level
    evidence: internal/core/lab/lab_test.go:147 — "if got := r.Git("status", "--porcelain", "--ignored"); got != "" {"
    evidence: commands/lab.md:11 — "**no lab verb writes into"
  - the procedure is a discipline record
    evidence: .abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention-to-discard-and-every.md:16 — "# A lab runs one procedure from intention to discard"
- diverged:
  - halt-and-record on a gate refusal as a lab rule the verb enforces
    evidence: internal/core/lab/findings.go:137 — "for _, g := range []string{"preflight", "sweep"} {"
    evidence: commands/lab.md:60 — "The verb runs nothing:"
- missing: (none)

Scope-condition dispositions:
- cond-2609212141418357 — survived: the store is ~/.abcd/lab/< root-sha>/ keyed on the root commit like the other machine-scoped stores, and the mint test proves the repository top level and status are untouched
  evidence: internal/core/lab/lab.go:6 — "// The store is machine-scoped and keyed on the repository's root commit, the way"
  evidence: internal/core/lab/lab.go:49 — "const storeRelPath = ".abcd/lab""
  evidence: internal/core/lab/lab_test.go:152 — "t.Errorf("repository top level changed: %d entries before, %d after", len(before), len(after))"
- cond-2609212141414079 — untested: nothing in the lab package runs or checks a real-session smoke stage; the only mention is a fixture finding title in the harvest test

## Grounds

- pursued: three hand-run labs proved the procedure and paid for its scaffolding three times; we expect the verb to reproduce lab 3's SHIP on the first mechanised run; shown wrong if it needs more review rounds than lab 3 did
