---
id: itd-2609212130146198
slug: the-status-line-badge-is-true-at-every
spec_id: spc-2609212139593041
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-200]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609212137129937, itd-201]
related_adrs: [adr-2609212115255771]
---

# The status-line badge is true at every stop

## Press Release

> **The status-line badge always reads one of three states, a question to the human is refused until the mode is set, and the answer resets it.**
>
> "The badge was set by whichever agent remembered," said a product thinker who had watched it read managed while an agent waited on them. "Now an agent cannot ask me anything until it has said which of us it is asking, and the moment I answer the badge goes back. It is the one signal I have that something is waiting; now it is true."

## Why This Matters

itd-200 shipped the badge and the `mode` verb; the state is whatever the agent last set, so it lags, sticks or never changes, and two captures already sit on it (a default that reads bare "abcd"; a colour that bleeds past the badge). Ruled 2026-09-21: the verb stays the setter, because the agent knows whom it addresses; the guard enforces it; abcd resets it on the next human message.

## Mechanism

We expect a question refused until the mode is set, and a reset on the answer, to make the badge true at every stop, because the two moments the badge must change are the two moments the harness already sees; shown wrong if a stop is still observed with the badge reading managed.

## Scope Conditions

None stated.

## What's In Scope

- **Three states, never bare**: `abcd-managed`, `waiting on the product thinker`, `waiting on the technical facilitator`; an unmanaged repository shows nothing.
- **The guard**: a question to the human through the host's question tool is refused while the mode reads managed; the refusal names the two settings and the verb.
- **The setter**: `abcd mode product-thinker|facilitator` sets the state; the line changes on its next render.
- **The reset**: the prompt hook resets the mode to managed when the next human message arrives after a question, and says so on stderr once.
- **The paint**: the line's colour ends at the badge (closes iss-2609170709035405); the default state closes iss-2609170627427239.

## What's Out of Scope

- Deriving the addressee from the question's text (the agent names it).
- Any state beyond the three.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. The verb sets the state, enforced by the guard on the question tool; the agent is reminded to choose the product thinker or the technical facilitator (ruled 2026-09-21).
2. abcd resets the mode to managed on the next human message.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** an abcd-managed repository, **when** the status line renders, **then** it reads exactly one of `abcd-managed`, `waiting on the product thinker`, `waiting on the technical facilitator`, never a bare `abcd`; an unmanaged repository shows nothing.
- **Given** the mode reads managed, **when** an agent calls the host's question tool, **then** the guard refuses the call naming the two settings and `abcd mode`.
- **Given** `abcd mode product-thinker` or `facilitator`, **when** the line next renders, **then** it shows the corresponding state.
- **Given** a question was asked and the next human message arrives, **when** the prompt hook runs, **then** the mode is reset to managed and one line on stderr says so.
- **Given** the line, **when** it renders, **then** its colour ends at the badge.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-68f6cce6a701 -->
Fidelity review — receipt rcp-68f6cce6a701 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:dc8f6a33e6a1e0d65927ac2c34bc3c74f642e9c3849476ecf308bb9ad2e7eb11
Input attestations: commit:c1c05edeb (spc-2609212139593041 close, itd-2609212130146198 ships); tree audited at 52c2236a55830421c2af5fa58f5a196c4eb7fdd5@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: badgeWord maps the three states to `abcd-managed` and mode's two WaitingOn phrases, an unknown or empty state falls back to the managed word (never bare), TestBadgeNeverReadsABareTag holds all five inputs to the three labels; the CLI renders abcd's row only where ahoy.Managed, otherwise the previous command or nothing (TestStatuslineUnmanagedRunsThePreviousCommand, TestStatuslineWithNoPreviousCommandPrintsNothing); all pass at BASE
  evidence: internal/core/statusline/badge.go:62 — "var badgeWord = map[State]string{ StateManaged: "abcd-managed","
  evidence: internal/core/statusline/badge.go:121 — "s = StateManaged word = badgeWord[StateManaged]"
  evidence: internal/core/statusline/render_test.go:253 — "func TestBadgeNeverReadsABareTag"
  evidence: internal/surface/cli/statusline.go:106 — "managed := rootErr == nil && ahoy.Managed(root) if !managed || set.Disabled {"
  evidence: internal/surface/cli/statusline_test.go:116 — "func TestStatuslineWithNoPreviousCommandPrintsNothing"
- ac-2 — MET_WITH_CONCERNS: the hook manifest matches the question tool beside the shell tool, questionGate exits 2 on a managed mode with one stderr line naming `abcd mode product-thinker`, `abcd mode facilitator`, the product thinker and the technical facilitator (TestGuardRefusesAQuestionWhileManaged). Concern: the refusal is issued only where the badge shows AND the verb could set the state — no local tier, or a tier the verb cannot write, lets the question run ungated with a loud line (the guard's fail-open-loud contract, iss-2609260100382261), so a managed repository without its local tier is not refused
  evidence: hooks/hooks.json:26 — ""matcher": "Bash|AskUserQuestion""
  evidence: internal/surface/cli/guard_question.go:45 — "const questionRefusal = "Blocked by the abcd guard (question tool): the mode reads managed"
  evidence: internal/surface/cli/guard_question.go:80 — "fmt.Fprintln(stderr, questionRefusal) return &exitError{Code: 2}"
  evidence: internal/surface/cli/guard_question.go:66 — "if err != nil || !ahoy.Managed(root) || !mode.HasTier(root) { return nil"
  evidence: internal/surface/cli/guard_question.go:77 — "if err := mode.CanSet(root); err != nil { return questionFailOpen("
  evidence: internal/surface/cli/guard_question_test.go:44 — "func TestGuardRefusesAQuestionWhileManaged"
- ac-3 — MET: `abcd mode product-thinker|facilitator|managed` writes the store and the next `abcd statusline` render carries the matching label; TestModeVerbSetsTheLine drives all three through the CLI
  evidence: internal/surface/cli/mode.go:35 — "`facilitator` (the loop is parked on the facilitator"
  evidence: internal/surface/cli/guard_question_test.go:256 — "func TestModeVerbSetsTheLine"
  evidence: internal/core/mode/mode.go:150 — "func (s State) WaitingOn() string"
- ac-4 — MET: an admitted question writes the question_open marker; the UserPromptSubmit hook calls resetModeOnAnswer before the rules work, ResetOnAnswer writes Managed then clears the marker and the front door prints exactly one stderr line; a hand-set mode with no marker is left alone (TestPromptHookResetsTheModeAfterAQuestion)
  evidence: internal/core/mode/question.go:112 — "func ResetOnAnswer(repoRoot string) (bool, error)"
  evidence: internal/surface/cli/cli.go:1421 — "resetModeOnAnswer(cmd.ErrOrStderr(), cwd)"
  evidence: internal/surface/cli/guard_question.go:113 — "abcd mode: the question was answered, so the mode is reset to managed"
  evidence: internal/surface/cli/guard_question_test.go:174 — "func TestPromptHookResetsTheModeAfterAQuestion"
- ac-5 — MET: the badge closes with SGR 39;49 on the badge itself, not at the row's end and not a full reset, and no later element carries an escape; TestBadgeColourEndsAtTheBadge asserts all three, and iss-2609170709035405 is resolved on it
  evidence: internal/core/statusline/badge.go:104 — "const reset = "\x1b[39;49m""
  evidence: internal/core/statusline/badge.go:146 — "return "\x1b[" + fg.sgr(38) + ";" + bg.sgr(48) + "m" + text + reset"
  evidence: internal/core/statusline/render_test.go:283 — "func TestBadgeColourEndsAtTheBadge"

Gap audit:
- honoured:
  - the two captures the intent names are closed by the delivery: the bare `abcd` default and the colour bleeding past the badge
    evidence: .abcd/work/issues/resolved/iss-2609170627427239-the-status-line-badge-s-default-state.md:12 — "no state renders the bare tool name"
    evidence: .abcd/work/issues/resolved/iss-2609170709035405-the-status-line-paints-everything-after.md:12 — "the badge's colour ends at the badge"
  - the reset writes the state before it clears the marker, so a failure between the two retries on the next message rather than parking the badge
    evidence: internal/core/mode/question.go:101 — "The state is written before the marker is removed"
  - a mode the human set by hand is not clobbered by their next message: the reset acts only on an open question marker
    evidence: internal/core/mode/question.go:10 — "Without the marker the prompt hook // changes nothing, so a state the human set by hand"
    evidence: internal/surface/cli/guard_question_test.go:198 — "A human who sets the hat by hand keeps it across their next message."
  - the badge's word carries the meaning and the role labels are mode's own vocabulary, so badge and notice name the same person
    evidence: internal/core/statusline/render_test.go:644 — "func TestRoleBadgeWordsAreTheModeVocabulary"
    evidence: internal/surface/cli/guard_question_test.go:281 — "func TestBadgeAndNoticeNameTheSamePerson"
- diverged:
  - the press release says an agent cannot ask the human anything until it has said whom it is asking; the delivery gates only the host's question tool, so a stop that asks in prose at the end of a turn is ungated and the badge can still read managed at that stop — the Mechanism's own falsifier — while the intent's scope confines the guard to the question tool
    evidence: .abcd/development/intents/shipped/itd-2609212130146198-the-status-line-badge-is-true-at-every.md:23 — "Now an agent cannot ask me anything until it has said which of us it is asking"
    evidence: internal/surface/cli/guard_question.go:30 — "var questionTools = []string{"AskUserQuestion"}"
    evidence: .abcd/development/intents/shipped/itd-2609212130146198-the-status-line-badge-is-true-at-every.md:40 — "a question to the human through the host's question tool is refused while the mode reads managed"
  - the guard refuses only where the verb it names could run: a managed repository without its local tier, or with one the verb cannot write, lets the question through loudly
    evidence: internal/surface/cli/guard_question.go:59 — "the guard's fail-open-loud contract"
- missing: (none)
<!-- abcd-review-end receipt=rcp-68f6cce6a701 -->

## Grounds

- pursued: the badge is how the product thinker tells an agent is waiting on them, and today it is set by memory; we expect a guarded verb and an automatic reset to make it true at every stop; shown wrong if a stop is still seen with the badge reading managed
