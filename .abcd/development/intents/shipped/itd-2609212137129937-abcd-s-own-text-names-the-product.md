---
id: itd-2609212137129937
slug: abcd-s-own-text-names-the-product
spec_id: spc-2609212141412864
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609212130146198, itd-201]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-97, itd-174, itd-2609211913453478]
related_adrs: [adr-2609212115255771]
---

# abcd's text names the product thinker or the technical facilitator, never the maintainer

## Press Release

> **Every place abcd writes names which of the two people it means, and the word maintainer leaves the vocabulary.**
>
> "Agents kept asking 'the maintainer' and I never knew if they meant me deciding what to build or me running the gates," said a product thinker. "They learned the word from abcd's own pages. Now the pages say which of us, the badge says which of us, and the word cannot come back."

## Why This Matters

On 2026-09-21 the product thinker noted that agents blur the two roles under one word. The word is abcd's: twelve occurrences in the command pages agents read, twenty-two in the brief, five in the principles, one in the bundled rules and one in this repository's rule overrides ("a planned intent's adoption is a maintainer decision"), and a persona hint. Ruled: everywhere abcd writes, each use rewritten to the role it meant, and a lint from then on. The sweep is the run's, not this session's.

## Mechanism

We expect agents to stop saying maintainer once no abcd text says it, because agents say what they read; shown wrong if transcripts after the sweep still use the word for either role.

## Scope Conditions

None stated.

## What's In Scope

- **The sweep**: every occurrence in the command pages, the bundled and repository rules, the brief, the principles, the docs, the personas registry and abcd's rendered text rewritten to the product thinker (what to build, adoption, rulings) or the technical facilitator (how, gates, mechanics); one change, each rewrite reviewed.
- **The lint**: the docs-lint banned-token list gains the word for every lint root including the command pages and the rules; the acknowledgements and a persona's outside job title are the only escapes, marked.
- **The questions**: every question or stop abcd's agents put to a human names which role it asks (the GRILL rule made mechanical where the page templates carry the question).
- **The glossary**: the two roles defined beside the record-families page, citing itd-97's stance that the facilitator is a mode.
- **The test**: walks the rendered help and the plugin pages for the word.

## What's Out of Scope

- Defining a third role.
- Changing what either role decides (itd-174, itd-97).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Everywhere abcd writes, each use names the role it meant; a lint bans the word after (ruled 2026-09-21).
2. Captured for the run to build; no sweep in the session that ruled it.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** the command pages, the rules, the brief, the principles, the docs and the personas, **when** the sweep lands, **then** no occurrence of the word remains outside the marked escapes, and each rewrite names the product thinker or the technical facilitator.
- **Given** a new page or rule carrying the word, **when** the docs lint runs, **then** it is refused as a banned token on every lint root.
- **Given** a question or stop an agent puts to a human through the page templates, **when** it renders, **then** it names which of the two roles it asks.
- **Given** the glossary, **when** the two roles are looked up, **then** each has an entry beside the record-families page citing itd-97.
- **Given** the rendered help and the plugin pages, **when** the test runs, **then** it fails on the word.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-12ccf7627aac -->
Fidelity review — receipt rcp-12ccf7627aac (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:0d2779a6a8aa938ebbc363b05eb85aad759fc0c1ca7f93e139b6f26dcdb0ddf1
Input attestations: diff:commits 124b24510 b4df4f25d 993dc7b61 ee9e01dd6 88830a301 83596e7b1 (feat/roles-not-maintainer via integ/land-12), judged at main 7fb52a6b5@sha256:9abdab7a96537431406b403981c9e83be49a034304d905ea0323638f1fcb23ae;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: A case-insensitive grep at BASE over the command pages, .abcd/rules.json, the brief, the principles, docs/, README.md, AGENTS.md, ACKNOWLEDGEMENTS.md, the bundled rules source and the personas registry finds zero occurrences of the word, so no marked escape was even needed; the sweep landed as one 41-file change, the persona hint reads 'open-source project lead', and the rewrites name a role (the ready verb's remedies name the product thinker). The word survives only outside the listed surfaces: six .github workflow YAML comments, Go source comments, and the agents' injection-canary test fixtures.
  evidence: .abcd/development/personas.json:16 — "{ "name": "Kira", "role_hints": ["open-source project lead", "DX engineer", "framework author"] }"
  evidence: internal/core/intent/ready.go:163 — "confirm the Acceptance Criteria with the product thinker, then run `abcd intent plan %s`"
  evidence: internal/core/rules/defaults/rules.json:1 — "(the bundled rules source: zero occurrences at BASE)"
  evidence: .abcd/development/brief/glossary/core/product-thinker.md:4 — "The person who decides what is built and why — who rules on intents, signs off acceptance criteria, adopts or declines a proposal"
- ac-2 — MET_WITH_CONCERNS: The repository's docs-lint carries `roles/retired-role-word` as a blocker with the marked `< !-- docs-lint: allow -- >` escape, reaching the docs roots plus the extra roots `commands`, `.abcd/rules.json` and `internal/core/rules/defaults`, and the test runs the entry over a tree shaped like this one and asserts the refusal on each of the five roots and the escape's suppression. Concern: 'every lint root' is every root the lint defines, and the brief, the principles and the personas registry — three of the six surfaces ac-1 swept, the brief holding the largest share — are in no root the token reads, so the word can return there with nothing refusing it.
  evidence: .abcd/docs-lint.json:180-193 — ""id": "roles/retired-role-word", "pattern": "(?i)\\bmaintainer", "severity": "blocker", ... "extra_roots": ["commands", ".abcd/rules.json", "internal/core/rules/defaults"]"
  evidence: .abcd/docs-lint.json:2-5 — ""roots": ["docs", "README.md"]"
  evidence: internal/core/lint/roleban_test.go:17 — "func TestRepoRoleWordIsRefusedOnEveryLintRoot"
  evidence: internal/core/lint/lint.go:3263-3272 — "lintTokenExtraRoots runs each banned token that declares extra_roots over"
  evidence: internal/core/lint/tokenroots_test.go:24 — "func TestTokenExtraRootsCarryThatTokenOnly"
- ac-3 — MET_WITH_CONCERNS: The test walks every command page and agent prompt, finds each paragraph that puts a question to a human (an imperative ask of a person, a relayed question, a pre-answering `--yes`, an act on someone's word), requires the paragraph to name one of the two roles the mode's Addressee() seam defines, and requires the section to set `abcd mode <state>` first; the follow-up commit made every stop on the ahoy, intent and site pages comply. Concern: the detector is a regex grammar over paragraph shapes, so a question phrased outside it is not held to the rule, and the floor is ten blocks found.
  evidence: internal/surface/cli/roles_vocabulary_test.go:105 — "func TestPluginQuestionBlocksNameTheAddressee"
  evidence: internal/surface/cli/roles_vocabulary_test.go:98 — "var questionBlock = regexp.MustCompile("
  evidence: internal/surface/cli/roles_vocabulary_test.go:137-139 — "asks the %s but its section never sets `abcd %s` first"
  evidence: internal/core/mode/mode.go:136-141 — "func (s State) Addressee() string { ... return "technical facilitator" ... return "product thinker""
- ac-4 — MET: `product-thinker.md` and `technical-facilitator.md` sit in the core glossary beside `record-families.md`, each introduced_in this intent, each naming the other and record-families under not_to_be_confused_with, and each citing itd-97's stance that the facilitator is a mode, not a person.
  evidence: .abcd/development/brief/glossary/core/product-thinker.md:25 — "The role does not change with who runs the machinery. itd-97 (a draft) holds that the"
  evidence: .abcd/development/brief/glossary/core/technical-facilitator.md:24 — "itd-97 (a draft) holds that **the facilitator is a mode, not a person**."
  evidence: .abcd/development/brief/glossary/core/record-families.md:1 — "(the record-families page, in the same directory)"
- ac-5 — MET: The test renders every command's `--help` through the cobra tree (at least twenty pages) and reads every command page and agent prompt (at least twenty, the agents' CHANGELOG aside), failing on any line matching the retired word, spelled apart in the test so the file never trips its own ban; it passes at BASE.
  evidence: internal/surface/cli/roles_vocabulary_test.go:51 — "func TestRenderedHelpAndPluginPagesNameTheRoles"
  evidence: internal/surface/cli/roles_vocabulary_test.go:23 — "var retiredRoleWord = regexp.MustCompile(`(?i)\b` + "main" + "tainer")"
  evidence: internal/surface/cli/roles_vocabulary_test.go:28-47 — "for _, glob := range []string{"commands/*.md", "agents/*.md"}"

Gap audit:
- honoured:
  - Lint before sweep, so the sweep was watched red then green (spec Approach)
    evidence: internal/core/lint/roleban_test.go:17 — "func TestRepoRoleWordIsRefusedOnEveryLintRoot"
    evidence: internal/core/lint/tokenroots_test.go:74 — "func TestTokenExtraRootsLoadFromConfig"
  - The sweep is one reviewed change across the command pages, the rules, the brief, the principles, the docs and the personas, with the persona hint rewritten rather than escaped
    evidence: .abcd/development/personas.json:16 — ""open-source project lead""
    evidence: internal/core/rules/defaults/rules.json:1 — "(bundled rules: zero occurrences at BASE)"
  - The addressee comes from the mode, not from a restated list: the test reads mode.States() and their Addressee()
    evidence: internal/surface/cli/roles_vocabulary_test.go:108-115 — "for _, s := range mode.States() { if who := s.Addressee(); who != "" {"
  - The two glossary entries cite itd-97 and sit beside the record-families page
    evidence: .abcd/development/brief/glossary/core/technical-facilitator.md:4 — "itd-97 holds that the role is a mode, not a person."
  - The escape is the marked allow comment, the only way past the blocker
    evidence: .abcd/docs-lint.json:184-186 — ""allow_context": ["(?i)< !--\\s*docs-lint:\\s*allow\\b"]"
- diverged:
  - The press release says the word cannot come back; the lint guards the docs, the command pages and the rules only, so the brief, the principles and the personas registry — swept by ac-1 — are guarded by nothing
    evidence: .abcd/docs-lint.json:187-191 — ""extra_roots": ["commands", ".abcd/rules.json", "internal/core/rules/defaults"]"
    evidence: internal/core/lint/roleban_test.go:48-50 — "for _, want := range []string{"commands", ".abcd/rules.json", "internal/core/rules/defaults"} {"
  - The word survives outside abcd's rendered text: six .github workflow YAML comments, Go source comments (internal/core/lint among them) and the agents' injection-canary fixtures; none is a surface the intent lists, so none is captured
    evidence: .github/workflows/ci.yml:475 — "# github-actions ecosystem, so bumps are a deliberate maintainer act"
    evidence: internal/core/lint/collect.go:32 — "// manual queue can tell the maintainer where to look."
  - The question-block rule is mechanical only as far as the test's regex grammar reaches
    evidence: internal/surface/cli/roles_vocabulary_test.go:89-97 — "questionBlock finds a paragraph in which a page tells the agent to put a question to a human: an imperative ask"
- missing: (none)
<!-- abcd-review-end receipt=rcp-12ccf7627aac -->

## Grounds

- pursued: the badge intent and every interview today turn on which of two people is being asked, and the pages that brief agents blur them; we expect agents to stop saying maintainer once no abcd text says it; shown wrong if transcripts after the sweep still use the word
