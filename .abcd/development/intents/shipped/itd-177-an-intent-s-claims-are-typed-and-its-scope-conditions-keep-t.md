---
id: itd-177
slug: an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t
spec_id: spc-55
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: major
impact: additive
---

# An intent's claims are typed and its scope conditions keep their identity across edits — the readiness gate refuses an intent that leaves a context claim unrecorded

Typed links: consumes [adr-51](../../decisions/adrs/0051-intents-declare-mechanism-and-scope-conditions.md)
(the sections exist as a format; this intent is the anticipated enforcement);
gated by the `claim-recording-gradient` discipline; identities minted on the
adr-45 mint.

## Press Release

> **An intent now says what kind of claim each of its sections carries, and
> its scope conditions survive their own rewording.** Criteria were always
> mandatory; the mechanism claim is prompted and may be declined with a
> recorded nullity; scope conditions are required — or their absence is
> declared, explicitly, as "none stated". Each scope condition carries a
> persistent identity, so when its text is edited the condition is still the
> same condition, and the disposition that later attaches to it (survived,
> narrowed, falsified, untested) attaches to the claim rather than to
> wording that an edit replaces.

## What's In Scope

- Schema first: the three claim types and the per-condition identity, in the
  intent record shape, before any command enforces them.
- Command second: the readiness-gate refusals per the gradient.
- The nullity forms: one token, `None stated.`, alone on its line under the
  section heading — the same grammar for scope conditions and for a
  declined mechanism. An absent section, an empty section, and a recorded
  nullity are three distinct byte states with three semantics: a claim not
  carried; a gate fault; a claim considered and declined.
- Condition identity: each scope-condition bullet closes with a stamped
  identity marker minted on the adr-45 mint, written by `intent plan` (the
  lifecycle's write-capable verb) — never hand-typed. `abcd intent --json`
  renders conditions with their identities, which is the observable surface
  the identity criteria assert against; the gate refuses a duplicated or
  missing marker by name.
- Identity lifecycle: an edit keeps the marker; a split keeps the marker on
  the first part and mints for the second; a merge keeps the surviving
  marker and retires the other (surfaced as `narrowed` at the next fidelity
  verdict); a deletion retires the marker, reported at the next gate pass.
- "Prompted" has a named surface: the create-path scaffold comment and the
  planning interview prompt for the mechanism claim; the readiness gate
  only checks and reports.

## What's Out of Scope

- The gradient's rationale and staging — the discipline record owns it.
- Scope-condition dispositions — a separate intent (scope-condition disposition); this one only makes
  them attachable.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** an intent without context conditions or an explicit nullity,
  **when** the readiness gate runs, **then** the gate exits non-zero and
  names the missing field.
- **Given** a scope condition whose text is edited, **when** the intent is
  re-read, **then** the condition's identity is unchanged.
- **Given** an intent whose `## Mechanism` section carries the nullity
  token, **when** the readiness gate runs, **then** the gate passes and
  reports the recorded nullity.
- **Given** an intent whose `## Mechanism` section is present but empty,
  **when** the readiness gate runs, **then** the gate exits non-zero and
  names the section: write the claim or the nullity token.
- **Given** a scope condition with a duplicated or missing identity marker,
  **when** the readiness gate runs, **then** the gate exits non-zero and
  names the fault.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-8ca51595aa96 -->
Fidelity review — receipt rcp-8ca51595aa96 (verifier intent-auditor claude-fable-5).

Provenance: intent-auditor@claude-fable-5 · rubric_hash sha256:f3bba86a84b329fcbfbd3df64ec5d77d382dda9b4b1a61baf887e316bc41f38e · prompt_hash sha256:aa98d86544f4ca3ea46a014c1647d25b21242034538493d57685528eb2f7917f
Input attestations: diff:349b6886^1...349b6886^2 (merge of build/itd-177 into experiment/cold-reading, merge commit 349b6886)@sha256:0b497f943432d6f8aeb07179b398af423404a7fc42141fdc677c9e7217eafe26;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: scopeConditionsCheck fails ClaimAbsent (and the unanswered scaffold prompt) with a detail naming '## Scope Conditions', the CLI maps any !Ready to exit 1, and TestReadyScopeConditionsAbsent pins both the failure and the section name; go test passes on both packages
  evidence: internal/core/intent/ready.go:237 — "case ClaimAbsent: c.OK = false; c.Detail = \"no '## Scope Conditions' section — the context claim is unrecorded\""
  evidence: internal/core/intent/ready.go:222 — "if claims.ConditionsPrompt { c.OK = false; c.Detail = \"the '## Scope Conditions' prompt is unanswered — the context claim is unrecorded\""
  evidence: internal/surface/cli/cli.go:1630 — "if !res.Ready { return &exitError{Code: 1} }"
  evidence: internal/core/intent/ready_test.go:327 — "func TestReadyScopeConditionsAbsent(t *testing.T)"
- ac-2 — MET: the identity is a byte marker read anywhere inside the bullet (condMarkerRe, not end-anchored), so an edited or reflowed bullet re-reads to the same cond- id; TestConditionIdentitySurvivesEdit and TestConditionMarkerSurvivesAReflow pin it and the stamp never re-mints a marked bullet
  evidence: internal/core/intent/claims.go:66 — "condMarkerRe = regexp.MustCompile(`< !-- cond: (cond-[0-9]{16}) -- >`)"
  evidence: internal/core/intent/claims_test.go:120 — "func TestConditionIdentitySurvivesEdit(t *testing.T)"
  evidence: internal/core/intent/claims_test.go:404 — "func TestConditionMarkerSurvivesAReflow(t *testing.T)"
  evidence: internal/core/intent/claims.go:551 — "if len(ids) > 0 || malformed { continue }"
- ac-3 — MET: mechanismCheck returns OK with detail 'mechanism claim declined (nullity recorded)' on ClaimNullity, which claimState assigns only to the exact token alone on its line; TestReadyMechanismNullityPasses asserts OK, the detail, and res.Ready
  evidence: internal/core/intent/ready.go:174 — "case ClaimNullity: c.Detail = \"mechanism claim declined (nullity recorded)\""
  evidence: internal/core/intent/claims.go:291 — "case len(nonBlank) == 1 && nonBlank[0] == NullityToken: return ClaimNullity"
  evidence: internal/core/intent/ready_test.go:363 — "func TestReadyMechanismNullityPasses(t *testing.T)"
- ac-4 — MET: ClaimEmpty is a distinct state (heading with only blank lines) and mechanismCheck fails it naming '## Mechanism' with a remedy offering the claim or the nullity token; TestReadyMechanismEmptyFails asserts !Ready, the section name and the token in the remedy
  evidence: internal/core/intent/ready.go:178 — "case ClaimEmpty: c.OK = false; c.Detail = \"'## Mechanism' is present but empty — neither a claim nor a recorded decline\""
  evidence: internal/core/intent/ready.go:181 — "c.Remedy = \"write the falsifiable claim (\\\"we expect X because Y\\\") under '## Mechanism', or record the exact token `\" + NullityToken + \"` alone on its line to decline it\""
  evidence: internal/core/intent/claims.go:289 — "case len(nonBlank) == 0: return ClaimEmpty"
  evidence: internal/core/intent/ready_test.go:376 — "func TestReadyMechanismEmptyFails(t *testing.T)"
- ac-5 — MET: scopeConditionsCheck fails an unmarked bullet by ordinal (remedy naming `abcd intent plan <itd-N>`), a duplicated identity by the repeated cond- id, and additionally a bullet carrying two markers or a malformed one; TestReadyConditionMarkerMissing and TestReadyConditionMarkerDuplicated pin both halves and TestIntentPlanStampsAPlannedRecord proves exit 1 at the CLI for the missing half
  evidence: internal/core/intent/ready.go:261 — "if unmarked := UnmarkedConditionOrdinals(claims.Conditions); len(unmarked) > 0 { c.OK = false; c.Detail = fmt.Sprintf(\"condition(s) %s carry no identity marker\", joinInts(unmarked))"
  evidence: internal/core/intent/ready.go:267 — "if dupes := DuplicateConditionIDs(claims.Conditions); len(dupes) > 0 { c.OK = false; c.Detail = fmt.Sprintf(\"identity %s is carried by more than one condition\", strings.Join(dupes, \", \"))"
  evidence: internal/core/intent/ready_test.go:400 — "func TestReadyConditionMarkerMissing(t *testing.T)"
  evidence: internal/core/intent/ready_test.go:417 — "func TestReadyConditionMarkerDuplicated(t *testing.T)"
  evidence: internal/surface/cli/intent_cli_test.go:479 — "if exitCodeOf(err) != 1 || !strings.Contains(report, \"abcd intent plan itd-10\")"

Gap audit:
- honoured:
  - Three claim types typed in the record shape, with three distinct byte states (absent / empty / nullity) and one exact nullity token, before any command enforces them
    evidence: internal/core/intent/claims.go:28 — "const NullityToken = \"None stated.\""
    evidence: internal/core/intent/claims.go:277 — "func claimState(lines []string, headRe *regexp.Regexp) ClaimState"
    evidence: internal/core/intent/claims_test.go:63 — "func TestNullityTokenIsExact(t *testing.T)"
  - Readiness-gate refusals per the gradient: mechanism prompted-and-nullable, scope conditions mandatory-or-nullity, two checks added in fixed order after acceptance_criteria
    evidence: internal/core/intent/ready.go:91 — "res.Checks = append(res.Checks, mechanismCheck(it, claims)); res.Checks = append(res.Checks, scopeConditionsCheck(it, claims))"
    evidence: internal/core/intent/ready_test.go:292 — "func TestReadyChecksOrderAndCount(t *testing.T)"
  - Condition identities minted on the adr-45 mint by `intent plan` (the write-capable verb), never hand-typed; the gate refuses, it does not repair
    evidence: internal/core/intent/claims.go:586 — "id, err := minter.Mint(condFamily)"
    evidence: internal/core/intent/lifecycle.go:195 — "stampedContent, conditionsStamped, err := stampScopeConditions(content, recordid.Minter{})"
    evidence: internal/core/intent/ready.go:264 — "run `abcd intent plan %s` — the write-capable verb stamps every unmarked condition; markers are never hand-typed"
  - Identity lifecycle for edit and split: an edit keeps its marker, a split keeps the marker on the first part and mints for the unmarked second (idempotent stamp)
    evidence: internal/core/intent/claims_test.go:148 — "func TestStampScopeConditionsMarksOnlyUnmarkedBullets(t *testing.T)"
    evidence: internal/core/intent/claims_test.go:177 — "func TestStampScopeConditionsIsIdempotent(t *testing.T)"
  - 'Prompted' has a named surface: the create-path scaffold comment for both sections and the planning-interview steps for the mechanism claim and scope-condition elicitation
    evidence: internal/core/intent/create.go:288 — "b.WriteString(\"## Mechanism\\n\\n\"); b.WriteString(MechanismPrompt + \"\\n\\n\"); b.WriteString(\"## Scope Conditions\\n\\n\")"
    evidence: commands/intent.md:167 — "5. **Mechanism claim (prompted, nullable):** ask why the authors expect this to work"
    evidence: internal/core/intent/claims_test.go:294 — "func TestSeedDraftCarriesClaimSections(t *testing.T)"
  - Discipline records are exempt from both claim checks, and shipped/superseded records are never backfilled (forward-only population)
    evidence: internal/core/intent/ready.go:284 — "func claimCheckExemption(it Intent) (string, bool)"
    evidence: internal/core/intent/ready.go:288 — "case BucketShipped, BucketSuperseded: return \"not applicable — a \" + it.Bucket + \" record's claims are never backfilled\", true"
  - Corpus pass and loud two-step staging: every planned/ record received `## Scope Conditions` + `None stated.` (commit d95f085d) before the refusal was promoted (commit 698e779b); the live gate on itd-20 reports the nullity as OK
    evidence: .abcd/development/intents/planned/itd-20-top-level-abcd-dispatcher.md:55 — "## Scope Conditions  None stated."
    evidence: .abcd/development/specs/closed/spc-55-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:116 — "**Staging is loud and two-step.**"
  - Identities rendered with their conditions on a per-intent --json surface, plus the text report
    evidence: internal/core/intent/ready.go:45 — "Conditions []ScopeCondition `json:\"conditions\"`"
    evidence: internal/surface/cli/intent_cli_test.go:425 — "func TestIntentReadyJSONRendersConditionIdentities(t *testing.T)"
- diverged:
  - The observable surface for condition identities is `abcd intent --json` (intent) — delivered as `abcd intent ready --json` (spec decision: bare `abcd intent` is a corpus-wide status with no per-record body)
    evidence: .abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:44 — "`abcd intent --json` renders conditions with their identities"
    evidence: .abcd/development/specs/closed/spc-55-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:83 — "**Decision — the observable surface is `abcd intent ready --json`, not `abcd intent --json`.**"
    evidence: commands/intent.md:132 — "identities are rendered by `abcd intent ready <itd-N> --json` under `conditions`"
  - Each bullet 'closes with' a stamped marker (intent) — delivered as written at the end of the first line but READ anywhere in the bullet, a deliberate widening so an editor reflow cannot orphan the identity (iss-2608300235377731)
    evidence: .abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:42 — "each scope-condition bullet closes with a stamped identity marker"
    evidence: internal/core/intent/claims.go:58 — "condMarkerRe matches the identity marker ANYWHERE inside a condition bullet — first physical line or a folded continuation line"
  - `intent plan` is the drafts→planned verb (intent/spec scope) — delivered additionally as a stamp-only step on an already-planned record, so the gate's remedy is a command that runs (iss-2608300210588874); a run with nothing unmarked refuses
    evidence: internal/core/intent/lifecycle.go:311 — "func stampPlanned(repoRoot string, it Intent) (PlanResult, error)"
    evidence: internal/core/intent/lifecycle.go:329 — "return fmt.Errorf(\"intent: %s is already planned and carries no unmarked scope condition; nothing to stamp\", it.ID)"
    evidence: internal/surface/cli/intent_cli_test.go:468 — "func TestIntentPlanStampsAPlannedRecord(t *testing.T)"
  - A merge 'retires the other marker (surfaced as `narrowed` at the next fidelity verdict)' — delivered as: the retired marker simply stops appearing, and surfacing it is deferred to spc-59's verdict ingest; nothing in this change computes or reports a retirement
    evidence: .abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:50 — "marker and retires the other (surfaced as `narrowed` at the next fidelity verdict)"
    evidence: internal/core/intent/claims.go:514 — "a merge keeps the surviving marker and the retired one simply stops appearing."
    evidence: .abcd/development/specs/closed/spc-55-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:106 — "and nothing here reads the record's history to notice — the gate judges the record in front of it"
  - The spec's acceptance mapping names a CLI test `TestIntentReadyMissingConditionsExit1` for ac-1 — no function of that name exists in the delivered tree; the CLI exit-1 contract for the absent-conditions case rests on the generic !Ready → exit 1 mapping and the core test, and the only condition-specific CLI exit-1 assertion is the missing-marker case
    evidence: .abcd/development/specs/closed/spc-55-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:127 — "`TestReadyScopeConditionsAbsent` (core), `TestIntentReadyMissingConditionsExit1` (CLI)"
    evidence: internal/surface/cli/intent_cli_test.go:332 — "func TestIntentReadyNotReadyExit1(t *testing.T)"
    evidence: internal/surface/cli/cli.go:1630 — "if !res.Ready { return &exitError{Code: 1} }"
- missing:
  - 'a deletion retires the marker, reported at the next gate pass' — the gate reports nothing about a deleted condition: it judges only the record in front of it and reads no history, so a deleted bullet's identity vanishes silently until spc-59 accounts for it
    evidence: .abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:51 — "a deletion retires the marker, reported at the next gate pass."
    evidence: .abcd/development/specs/closed/spc-55-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md:106 — "nothing here reads the record's history to notice — the gate judges the record in front of it, and a retired identity is spc-59's to account for"
    evidence: internal/core/intent/ready.go:193 — "func scopeConditionsCheck(it Intent, claims Claims) ReadyCheck"

## Grounds

- pursued: Pursued now because adr-51 shipped the sections as a format and explicitly deferred their enforcement to "its own record", and because spc-59 cannot key a disposition to anything until a condition has an identity that outlives its wording. Enforcement without identity would record dispositions against sentences, which is the failure the gradient exists to prevent.
