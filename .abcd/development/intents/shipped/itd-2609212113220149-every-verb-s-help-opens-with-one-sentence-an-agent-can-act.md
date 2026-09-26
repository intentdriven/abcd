---
id: itd-2609212113220149
slug: every-verb-s-help-opens-with-one-sentence-an-agent-can-act
spec_id: spc-2609212139583822
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-146]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-172]
related_adrs: [adr-2609212115255771]
---

# Every verb's help opens with one sentence an agent can act on

## Press Release

> **Every verb's help opens with one sentence, does / writes / refuses, identical on the list, the verb's `--help` and its page.**
>
> "I read one line per verb before I decide whether to call it," said a technical facilitator watching an agent choose. "When that line says what the verb does, what it writes and when it refuses, the agent calls the right one. When it says 'manage things', it grep's the binary."

## Why This Matters

Fifty-three verbs and sub-verbs carry a short line today, of uneven shape, and nothing holds them to a form or keeps the three places they appear in agreement. The field's guidance for both people and agents is the same: one concise, actionable line per command, in one voice. Ruled 2026-09-21: does, writes, refuses, in that order, under the repository's writing-style guide, generated from one source.

## Mechanism

We expect an agent choosing a verb from its sentence to call the right one more often and to stop reading the binary for verbs, because the sentence answers the three questions an agent asks before a call; shown wrong if agent transcripts after it ships still show a verb called for what it does not do.

## Scope Conditions

None stated.

## What's In Scope

- **The form**: one sentence per verb and sub-verb naming what it does, what it writes (or "writes nothing"), and when it refuses, in that order, under `docs/reference/writing-style.md`.
- **One source**: the sentence lives in the surface manifest and is rendered onto the command list, the verb's own `--help` and its plugin page, byte-identical.
- **The test**: walks every visible verb; fails on a missing clause, a length over the declared cap, or a difference between the three places.
- **The agent block** (itd-146) shows the same sentences.
- **The lint**: the docs-lint writing-style checks run over the sentences.

## What's Out of Scope

- The long help body per verb.
- Examples per verb (the page's).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Does / writes / refuses, in that order, identical in three places, under the writing-style guide.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** any visible verb or sub-verb, **when** its sentence is read, **then** it names what the verb does, what it writes or that it writes nothing, and when it refuses, in that order, and obeys the writing-style guide.
- **Given** the command list, the verb's `--help` and its plugin page, **when** the sentence is compared across them, **then** it is byte-identical, rendered from the surface manifest.
- **Given** a verb whose sentence lacks a clause, exceeds the cap or differs between places, **when** the test runs, **then** it fails naming the verb and the defect.
- **Given** `--help --agent`, **when** the agent block renders, **then** each line is the verb's sentence.
- **Given** the docs lint, **when** it runs, **then** the sentences are checked as any page is.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-47bbe53f5074 -->
Fidelity review — receipt rcp-47bbe53f5074 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:7f6d17b45108836a6ea8570db0f0b0bd800cce168c60870c481353ee1b7a4b9b
Input attestations: diff:tree at 7c476185 (main lineage, itd-2609212113220149 shipped)@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: every sentence is declared once in the manifest and ParseSentence splits it on the declared separators into a doing clause, a Writes clause and a refuses/never refuses clause in that order, refusing a missing, doubled or out-of-order clause or a run over the 160-character cap; TestEverySentenceInTheManifestParses walks the manifest and TestEveryVisibleVerbCarriesItsSentenceEverywhere walks over a hundred visible commands, both green at BASE; writing-style adherence is the docs lint's (ac-5)
  evidence: internal/core/surface/sentences.go:23 — "var sentences = map[string]string{"
  evidence: internal/core/surface/sentence.go:54 — "func ParseSentence(s string) (Sentence, error) {"
  evidence: internal/core/surface/sentence.go:31 — "const SentenceCap = 160"
  evidence: internal/core/surface/sentence_test.go:99 — "func TestEverySentenceInTheManifestParses(t *testing.T) {"
  evidence: internal/surface/cli/sentences_test.go:160 — "func TestEveryVisibleVerbCarriesItsSentenceEverywhere(t *testing.T) {"
- ac-2 — MET: SentenceFor is the one source: the surface snapshot records it per command, the CLI's --help opens with it, and each commands/< verb>.md description is diffed against it by sentenceDefects; TestEveryVisibleVerbCarriesItsSentenceEverywhere fails on any byte difference between the list, the verb's help and the page, and TestVerbHelpOpensWithTheSentenceThenTheLongHelp pins the help's opening line
  evidence: internal/core/surface/sentences.go:269 — "func SentenceFor(path string) (string, bool) {"
  evidence: internal/surface/cli/surface.go:112 — "s, _ := surface.SentenceFor(cmd.CommandPath())"
  evidence: internal/surface/cli/sentences.go:86 — "sentence, ok := surface.SentenceFor(cmd.CommandPath())"
  evidence: internal/surface/cli/sentences_test.go:151 — "commands/%s.md's description differs from the manifest's sentence"
  evidence: internal/surface/cli/sentences_test.go:258 — "func TestVerbHelpOpensWithTheSentenceThenTheLongHelp(t *testing.T) {"
  evidence: commands/peers.md:3 — "description: "List the records sibling worktrees and local branches hold that this checkout does not: Writes nothing; refuses outside a git checkout.""
- ac-3 — MET: TestSentenceDefectsNamesTheVerbAndTheDefect is the negative control on a synthetic tree: a verb with no sentence, one over the cap, one missing a clause, one whose list text differs and one whose page differs are each named with the defect and the well-formed verb is not; TestParseSentenceNamesEachDefect covers the parser's messages; TestTheFrameworkCommandsAreTheOnlyExemption pins that only cobra's help and completion escape
  evidence: internal/surface/cli/sentences_test.go:203 — "func TestSentenceDefectsNamesTheVerbAndTheDefect(t *testing.T) {"
  evidence: internal/core/surface/sentence_test.go:45 — "func TestParseSentenceNamesEachDefect(t *testing.T) {"
  evidence: internal/surface/cli/sentences_test.go:174 — "func TestTheFrameworkCommandsAreTheOnlyExemption(t *testing.T) {"
- ac-4 — MET: with --help --agent each line of the agents-and-hosts block is the entry's name, its manifest sentence and its page, and nothing else; TestAgentBlockLinesAreTheSentences matches every line against SentenceFor and refuses a vacuous block
  evidence: internal/surface/cli/sentences_test.go:274 — "func TestAgentBlockLinesAreTheSentences(t *testing.T) {"
  evidence: internal/surface/cli/helpgroups.go:61 — "helpAgentsTitle = "For agents and hosts (each line names the page to read next):""
- ac-5 — MET: the generated CLI reference docs/reference/cli/commands.md renders every sentence as its own line under the docs root the committed docs-lint configuration declares, so the lint reads the sentences as any page; TestSentencesAreCheckedByTheDocsLint asserts the root covers the page, that every sentence is on it, that the banned-token rules find nothing, and that a sentence carrying a banned token is caught
  evidence: internal/surface/cli/sentences_test.go:309 — "func TestSentencesAreCheckedByTheDocsLint(t *testing.T) {"
  evidence: .abcd/docs-lint.json:3 — ""docs","
  evidence: internal/surface/cli/sentences_test.go:317 — "const reference = "docs/reference/cli/commands.md""

Gap audit:
- honoured:
  - one source: the sentence lives in the surface manifest and is rendered onto the list, the verb's --help and its plugin page byte-identically
    evidence: internal/core/surface/sentences.go:269 — "func SentenceFor(path string) (string, bool) {"
    evidence: internal/surface/cli/sentences_test.go:160 — "func TestEveryVisibleVerbCarriesItsSentenceEverywhere(t *testing.T) {"
  - the test walks every visible verb and fails on a missing clause, a length over the declared cap, or a difference between the three places
    evidence: internal/surface/cli/sentences_test.go:203 — "func TestSentenceDefectsNamesTheVerbAndTheDefect(t *testing.T) {"
  - the agent block (itd-146) shows the same sentences
    evidence: internal/surface/cli/sentences_test.go:274 — "func TestAgentBlockLinesAreTheSentences(t *testing.T) {"
  - a reworded sentence is visible in the committed surface snapshot and gated by its drift test
    evidence: internal/surface/cli/surface.go:112 — "s, _ := surface.SentenceFor(cmd.CommandPath())"
    evidence: internal/core/surface/sentence_test.go:261 — "func TestSentenceChangesNamesEachRewordedVerb(t *testing.T) {"
- diverged:
  - spec scope 5 says docs lint includes the manifest's sentences as a lint root; the delivery reaches the sentences through the generated CLI reference page under the existing docs root rather than a manifest root, which satisfies the criterion (checked as any page is) by another route
    evidence: internal/surface/cli/sentences_test.go:317 — "const reference = "docs/reference/cli/commands.md""
    evidence: .abcd/docs-lint.json:3 — ""docs","
- missing: (none)

## Grounds

- pursued: the run adds five verbs and the per-verb lines today are fifty-three sentences of uneven shape; we expect an agent to choose from the sentence and stop grepping the binary; shown wrong if transcripts still show a verb called for what it does not do
