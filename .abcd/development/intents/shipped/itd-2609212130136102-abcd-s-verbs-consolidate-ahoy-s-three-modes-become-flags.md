---
id: itd-2609212130136102
slug: abcd-s-verbs-consolidate-ahoy-s-three-modes-become-flags
spec_id: spc-2609212139587510
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-146]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-122, itd-123, itd-124, itd-125]
related_adrs: [adr-2609212115255771]
---

# abcd's verbs consolidate: modes become flags and five checks become one lint

## Press Release

> **abcd's command list loses its modes-as-verbs and its five spellings of "check this repository", so the person's list holds about a dozen verbs.**
>
> "Twenty-four verbs, and three of them were the same check wearing different hats," said a product thinker reading `abcd --help`. "Now `lint` is the check, `ahoy` has flags instead of sub-verbs for its modes, and `--version` is where every tool keeps it. I can hold the list."

## Why This Matters

On 2026-09-21 the product thinker asked whether fifty-three verbs all make sense. The command-line guidelines the field converges on say: a sub-verb for a distinct action, a flag for a mode of the same action, and a top-level list a newcomer can hold. Three of `ahoy`'s sub-verbs are modes; `version` is a flag everywhere else; `intent new` is a dead alias; and `lint`, `docs lint`, `lint outbound`, `site check` and `identity render` are five spellings of one act. Together with the agent block (itd-146) the person's list falls to about fourteen. The cut is breaking, in the same major as the four rename intents already in the run.

## Mechanism

We expect a person to hold a list of about a dozen verbs and stop asking which of five checks to run, because the checks become one verb with targets and the modes stop looking like actions; shown wrong if the same questions recur after it ships.

## Scope Conditions

None stated.

## What's In Scope

- **Modes to flags**: `ahoy dry-run`, `ahoy identity-check`, `ahoy remote` become `ahoy --dry-run`, `--identity`, `--remote`; the old spellings answer with the new one and exit non-zero for one release.
- **`--version`** replaces `abcd version`; `version --check` becomes `update --check`; `intent new` is removed.
- **One lint with targets**: `abcd lint` (all), `lint docs`, `lint outbound`, `lint site`, `lint identity`; `docs cite` and `site build` stay, because they write.
- **The record of the move**: every moved spelling in the surface snapshot with its successor; the command pages and the brief's surface chapters say the new forms only; the release derives as breaking.
- **The count**: the person's default list (itd-146) is at most fourteen verbs after this ships.

## What's Out of Scope

- The people/agent split (itd-146).
- Renaming any record family verb (`capture`, `intent`, `spec`).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Modes to flags, `--version`, the dead alias removed, and one lint with targets, in one breaking change (ruled 2026-09-21).

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `ahoy dry-run`, `ahoy identity-check` or `ahoy remote`, **when** run after this ships, **then** each answers naming its flag form and exits non-zero, and the flag form does what the sub-verb did.
- **Given** `abcd --version`, **when** run, **then** it prints what `abcd version` printed; `abcd version` answers naming the flag; `update --check` does what `version --check` did; `intent new` is unknown.
- **Given** `abcd lint docs`, `lint outbound`, `lint site` and `lint identity`, **when** run, **then** each does what its old spelling did, `abcd lint` runs them all, and `docs cite` and `site build` are unchanged.
- **Given** the surface snapshot, **when** regenerated, **then** every moved spelling is recorded with its successor, the pages and the brief say the new forms only, and the release derives as breaking.
- **Given** `abcd --help`, **when** it renders after itd-146 and this ship, **then** the person's list counts at most fourteen verbs.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-dd80f3fcae80 -->
Fidelity review — receipt rcp-dd80f3fcae80 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:a7dec6f142b4d0effcdc25320638cce616b7883884e5365c9861437d3b0294be
Input attestations: diff:tree at 7c476185 (main lineage, itd-2609212130136102 shipped)@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: ahoy dry-run and ahoy identity-check are movedStub commands and bare ahoy remote is markMoved with its apply sub-verb live; each names its flag form on stderr and exits 2 (TestMovedSpellingsAnswerWithTheirSuccessor), the three flags are declared mutually exclusive on ahoy, and TestAhoyDryRunFlagPrintsTheDetectionEnvelope, TestAhoyIdentityFlagHoldsTheCommitIdentityToThePin and TestAhoyRemoteFlagReportsTheRemoteSettings show the flag doing what the sub-verb did; all green at BASE
  evidence: internal/surface/cli/cli.go:3093 — "ahoyCmd.AddCommand(movedStub("dry-run", "abcd ahoy --dry-run"))"
  evidence: internal/surface/cli/cli.go:3153 — "markMoved(remoteCmd, "abcd ahoy --remote")"
  evidence: internal/surface/cli/cli.go:2950 — "ahoyCmd.MarkFlagsMutuallyExclusive("dry-run", "identity", "remote")"
  evidence: internal/surface/cli/moved.go:58 — "func movedStub(use, successor string) *cobra.Command {"
  evidence: internal/surface/cli/consolidate_test.go:54 — "func TestMovedSpellingsAnswerWithTheirSuccessor(t *testing.T) {"
  evidence: internal/surface/cli/consolidate_test.go:141 — "func TestAhoyDryRunFlagPrintsTheDetectionEnvelope(t *testing.T) {"
- ac-2 — MET: version is a movedStub naming abcd --version and its --check flag names update --check; update carries --check with the old report; TestRootVersionFlagPrintsWhatVersionPrinted and TestIntentNewIsUnknown (which also proves no draft titled new is filed) are green
  evidence: internal/surface/cli/version.go:71 — "cmd := movedStub("version", "abcd --version")"
  evidence: internal/surface/cli/version.go:79 — "cmd.Flags().BoolVar(&check, "check", false, "moved to: abcd update --check")"
  evidence: internal/surface/cli/update.go:106 — "cmd.Flags().BoolVar(&check, "check", false, "fetch the latest release once and compare it with this binary"
  evidence: internal/surface/cli/consolidate_test.go:276 — "func TestRootVersionFlagPrintsWhatVersionPrinted(t *testing.T) {"
  evidence: internal/surface/cli/consolidate_test.go:295 — "func TestIntentNewIsUnknown(t *testing.T) {"
- ac-3 — MET: lint registers outbound, docs, site and identity sub-verbs; docs lint and site check are stubs naming lint docs and lint site and bare identity names lint identity; TestLintTargetsAreRegistered, TestLintIdentityRendersTheIdentityReport, TestLintSiteRefusesWithoutAComposition and TestBareLintRunsEveryTarget are green, and docs cite and site build stay live; identity render stays as a distinct act by the 2026-09-25 ruling, so the fifth spelling is read as the bare identity report
  evidence: internal/surface/cli/lint.go:90 — "cmd.AddCommand(newLintDocsCommand(asJSON))"
  evidence: internal/surface/cli/cli.go:604 — "docsCmd.AddCommand(movedStub("lint", "abcd lint docs"))"
  evidence: internal/surface/cli/site.go:83 — "siteCmd.AddCommand(movedStub("check", "abcd lint site"))"
  evidence: internal/surface/cli/identity.go:35 — "markMoved(identityCmd, "abcd lint identity")"
  evidence: internal/surface/cli/consolidate_test.go:311 — "func TestLintTargetsAreRegistered(t *testing.T) {"
  evidence: internal/surface/cli/consolidate_test.go:359 — "func TestBareLintRunsEveryTarget(t *testing.T) {"
  evidence: .abcd/work/DECISIONS.md:2556 — "`identity render` proposes a correction, a distinct act, and stays"
- ac-4 — MET: the snapshot records moved_to per stub (TestMovedSpellingsAreRecordedWithTheirSuccessor), the brief appendix omits a whole-moved stub and prints the successor where a bare form moved, the generated reference and the pages carry no section for an old spelling (TestReferenceNamesTheNewFormsOnly, TestReferenceUsageNeverOffersAMovedBareForm), and the intent declares impact: breaking, which DeriveNext maps to the breaking bump
  evidence: internal/surface/cli/consolidate_test.go:92 — "func TestMovedSpellingsAreRecordedWithTheirSuccessor(t *testing.T) {"
  evidence: internal/core/surface/appendix.go:120 — "fmt.Fprintf(&b, "### `%s`\n\nIt moved to `%s`.\n\n", p, c.MovedTo)"
  evidence: .abcd/development/brief/04-surfaces/12-version.md:100 — "It moved to `abcd --version`."
  evidence: internal/surface/cli/consolidate_test.go:451 — "func TestReferenceNamesTheNewFormsOnly(t *testing.T) {"
  evidence: .abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three-modes-become-flags.md:10 — "impact: breaking"
  evidence: internal/core/changelog/version.go:38 — "func DeriveNext(prev launch.Semver, bump Impact) (launch.Semver, bool) {"
- ac-5 — MET: TestPersonsListHoldsAtMostFourteenVerbs counts the verbs the default help lists and TestPersonVerbsCountsEveryListedVerb is its negative control on a fifteen-verb tree; both green at BASE
  evidence: internal/surface/cli/consolidate_test.go:420 — "func TestPersonsListHoldsAtMostFourteenVerbs(t *testing.T) {"
  evidence: internal/surface/cli/consolidate_test.go:434 — "func TestPersonVerbsCountsEveryListedVerb(t *testing.T) {"

Gap audit:
- honoured:
  - modes to flags: the old spellings answer with the new one and exit non-zero for one release
    evidence: internal/surface/cli/consolidate_test.go:54 — "func TestMovedSpellingsAnswerWithTheirSuccessor(t *testing.T) {"
  - a --json caller of a stub reads exactly one refusal on stdout
    evidence: internal/surface/cli/moved.go:66 — "cmd.SetOut(stderrOf{cmd})"
  - one lint with targets; docs cite and site build stay because they write
    evidence: internal/surface/cli/lint.go:92 — "cmd.AddCommand(newLintIdentityCommand(asJSON))"
    evidence: internal/surface/cli/consolidate_test.go:114 — "for _, live := range []string{"abcd lint docs", "abcd ahoy remote apply", "abcd identity init"} {"
  - the person's default list is at most fourteen verbs
    evidence: internal/surface/cli/consolidate_test.go:420 — "func TestPersonsListHoldsAtMostFourteenVerbs(t *testing.T) {"
- diverged:
  - the intent's Why names `identity render` among the five spellings of one check folded into `lint`; the delivery moves the bare `identity` report to `lint identity` and keeps `identity render` (a proposal, a distinct act) as a live sub-verb documented in the brief, by the 2026-09-25 ruling
    evidence: internal/surface/cli/identity.go:38 — "Use: "render","
    evidence: .abcd/development/brief/04-surfaces/19-identity.md:160 — "### `abcd identity render`"
    evidence: .abcd/work/DECISIONS.md:2556 — "`identity render` proposes a correction, a distinct act, and stays"
  - spec scope 3 says the old lint verbs become stubs for one release and the Approach names a follow-on remainder spec for their removal; the ruling records that no remainder spec is minted and the removal is captured as an issue instead
    evidence: .abcd/work/DECISIONS.md:2556 — "The spec's remainder spec for removing the stubs is not minted"
- missing: (none)

## Grounds

- pursued: batch 3 already carries four breaking renames, so this lands in the same major cut at no extra cost to adopters; we expect the person's list to be held and the which-check question to stop; shown wrong if the same questions recur
