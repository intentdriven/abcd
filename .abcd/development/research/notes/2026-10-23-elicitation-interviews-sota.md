# Eliciting a product thinker's intent: state of the art

Dated 2026-10-23. A state-of-the-art pass over the open intents that decide
how abcd asks a product thinker what they want and how it addresses them:
[itd-142](../../intents/drafts/itd-142-the-brief-creation-interview-abcd.md)
(the brief-creation interview),
[itd-143](../../intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under.md)
(the framing chapter),
[itd-90](../../intents/drafts/itd-90-brief-interview-for-the-blanks.md)
(the blanks interview),
[itd-2609292108373494](../../intents/drafts/itd-2609292108373494-when-the-person-has-installed-an.md)
(hand-over to an installed interviewing skill),
[itd-2610030810370060](../../intents/planned/itd-2610030810370060-a-person-can-run-abcd-s-interviews-in-a.md)
(interviews in a plain Terminal),
[itd-168](../../intents/drafts/itd-168-the-product-thinker-sets-how-the-system.md)
(the register and the role badges),
[itd-30](../../intents/drafts/itd-30-design-fictions-as-intent-format.md)
(design fictions as a capture format),
[itd-172](../../intents/drafts/itd-172-every-record-has-a-short-title-and-a-one.md)
(short title and one-line summary),
[itd-99](../../intents/drafts/itd-99-a-team-of-product-thinkers-decides-as.md)
(a team of product thinkers),
[itd-97](../../intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md)
(the facilitator as a mode) and
[itd-174](../../intents/drafts/itd-174-each-repository-configures-how-far-its.md)
(how far the facilitator is consulted, and when it escalates).

Every source linked below was opened in this pass (by the researcher or by
one of four research agents it ran, each told to cite only what it opened);
a source believed to exist but not opened is named in words under "Not
verified" and carries no link. Fetched pages were read as data. The pass
reuses three earlier notes rather than redoing their ground: the
[host question layout calibration](2026-10-03-host-question-layout-calibration.md)
(the measured 80-by-24 figures), the
[brief-to-record traceability pass](2026-10-04-brief-to-record-traceability-sota.md)
(present-tense brief, status in links) and the
[autonomous loops pass](2026-10-23-autonomous-loops-sota.md) (the stop
record shaped on MCP elicitation, under its itd-169 section). The asking
rules abcd already enforces are the GRILL domain, generated from
`internal/core/question/asking.go`: one thing at a time, the material quoted
in full, two to four options plus decide-later, nothing recommended, a
register per role, deferral recorded as an answer.

Tiers: [EVIDENCE] an eval, experiment, dataset or primary document;
[CONSENSUS] several independent practitioners or standards agree;
[CONTESTED] credible disagreement, both sides given; [ANECDOTE/MARKETING] a
single report or a vendor's claim with no methodology behind its numbers.
Each recommendation declares a path: Path 1 adopts the outside thing as is;
Path 2 builds natively to a named shape with a seam, so a later move
replaces one part; Path 3 diverges deliberately, naming the local preference
that forces it. The researcher is the proposer here; the independent
fit-challenge is still owed at plan time.

## itd-142: the brief-creation interview

### Findings

- **Open narrative first, options later.** In a split-ballot experiment,
  "fully 43% of those asked the open-ended version provided a response not
  listed in the closed-ended version"; in self-administered forms people
  "tend to choose items at the top of the list (a 'primacy' effect)".
  [EVIDENCE] [Pew Research Center, Writing Survey Questions](https://www.pewresearch.org/our-methods/u-s-surveys/writing-survey-questions/)
- **An interview yields six to eight times what a form does per person, and
  the best condition introduced structure late.** Distinct needs per
  participant: interviews 10.67 (no taxonomy), 11.67 (taxonomy up front),
  14.78 (taxonomy introduced later); surveys 1.74 to 1.94. One industrial
  case, descriptive. [EVIDENCE] [Obaidi et al., arXiv 2505.23684, June 2025](https://arxiv.org/html/2505.23684v2)
- **Requirements are co-created in the interview.** Only 30 to 38 per cent
  of post-interview requirements traced fully to the customer's initial
  ideas (30 student analysts, one fictional customer); the interviewer's own
  suggestions enter the record, which is what confirmation and provenance
  are for. [EVIDENCE] [Ferrari, Spoletini and Debnath, arXiv 2208.00825](https://arxiv.org/abs/2208.00825v1)
- **Machine interviewers under-probe and bundle questions.** A multimodal
  model interviewer was "acknowledgment-heavy but probe-light" (deepening
  probes 4.9 per cent of 428 turns) and packed several questions into 28.7
  per cent of question-bearing turns "despite an explicit instruction to
  ask one question at a time"; participants felt heard only on
  "content-grounded paraphrase, not generic social filler" (15
  participants). [EVIDENCE] [Zhang et al., arXiv 2608.10412, HCOMP 2026](https://arxiv.org/abs/2608.10412)
  A randomised student study found similar error counts for a model and a
  human interviewer but different kinds: the model judged answers and failed
  to follow up; humans led respondents (11 interviews). [EVIDENCE]
  [Wuttke et al., arXiv 2410.01824](https://arxiv.org/abs/2410.01824)
- **A model's follow-up questions are no worse than a human's, and better
  when told which mistakes to avoid.** GPT-4o follow-ups were "no worse than
  the human-authored questions" and "outperform human-authored questions
  when guided by common mistakes types"; the 14 mistake types (fails to
  elicit tacit assumptions, asks for solutions, jargon, too long, vague,
  mixes requirement kinds, and so on) are a ready-made question lint.
  [EVIDENCE] [Shen, Singhal and Breaux, arXiv 2507.02858, RE 2025](https://arxiv.org/abs/2507.02858)
  An end-to-end model interviewer elicited "up to 73.7 %" of planted
  requirements in 33 simulated interviews, with its own suggestions
  steering the conversation (full text; the abstract says only "a large
  portion"). [EVIDENCE] [Korn, Gorsch and Vogelsang, arXiv 2507.02564](https://arxiv.org/html/2507.02564)
- **The measured failure is under-asking, not over-asking.** Across 6,120
  dialogues and ten models there is "a consistent under-clarification
  bias: LLMs tend to answer prematurely". [EVIDENCE]
  [Luo et al., ClarifyMT-Bench, arXiv 2512.21120](https://arxiv.org/abs/2512.21120)
  Models "often identify ambiguity when explicitly asked to judge it, yet
  in the QA setting they overwhelmingly default to direct answers".
  [EVIDENCE] [Su and Cardie, arXiv 2605.25284](https://arxiv.org/abs/2605.25284)
  So "ask only what the record cannot settle" must be derived from the
  record's state, never left to the model's inclination.
- **Defaults and recommendations decide for people.** A meta-analysis of 58
  studies (n = 73,675) puts the default effect at d = 0.68, stronger when
  the default is read as "conveying what the choice architect thinks the
  decision-maker should do". [EVIDENCE]
  [Jachimowicz et al., Behavioural Public Policy 2019](https://collaborate.princeton.edu/en/publications/when-and-why-defaults-influence-decisions-a-meta-analysis-of-defa/)
- **"Don't know" is a state, not a middle.** Offering it moved 7 to 17
  points out of the middle category and 6 to 10 points out of non-response,
  and "respondents who choose a DK-O may not have a neutral opinion"
  (n = 1,282). [EVIDENCE] [Wetzelhütter, Survey Practice 2020](https://www.surveypractice.org/article/14484)
- **The nearest tool convention recommends; abcd's rule reverses it.** Spec
  Kit's clarify command asks "EXACTLY ONE question at a time", at most five
  per session, each with two to five options, a `**Recommended:**` option
  with reasoning, and appends `- Q: … → A: …` to a Clarifications section
  after each answer. [EVIDENCE] [github/spec-kit, templates/commands/clarify.md](https://github.com/github/spec-kit/blob/main/templates/commands/clarify.md)
  A widely installed brainstorming skill takes the other side: "Each
  message ends with one question", "Open questions come first", "Offer a
  short menu only when they're stuck", "Options steer." [CONSENSUS on one
  question per turn; CONTESTED on options]
  [obra/superpowers, brainstorming SKILL.md](https://github.com/obra/superpowers/blob/main/skills/brainstorming/SKILL.md)
- **The practitioner methods agree on order and on not leading.** Working
  Backwards: the press release first, then the FAQ "as the map to that
  destination", reviews of silent reading then discussion, "truth-seeking
  vs. selling". [CONSENSUS] [workingbackwards.com, the PR/FAQ process](https://workingbackwards.com/concepts/working-backwards-pr-faq-process/)
  The switch interview: anchor on "When did you first start thinking about
  this?", "Move forward in time through concrete events, not opinions",
  "Never feed the customer your words", and "connect the wrong dots" so the
  person corrects you, a non-leading confirmation. [CONSENSUS]
  [jobstobedone.org, How to Run a Switch Interview](https://jobstobedone.org/switch-interview/)
  The Mom Test's author is confirmed; its rule text was not on any page of
  his that opened (see Not verified).
- **Confirmation is a provenance control, not a validity proof.** Member
  checking "is often mentioned as one in a list of validation techniques";
  its validating power is argued over. [CONTESTED]
  [Birt et al., Qualitative Health Research 2016](https://ueaeprints.uea.ac.uk/id/eprint/59602/)
- **Fewer questions can be better.** A value-of-information policy asked
  five times fewer questions than a fine-tuned clarification policy while
  improving preference satisfaction by 13 to 15 per cent; a cheap way to
  amend later lowers the value of asking now. [EVIDENCE, workshop paper]
  [Nguyen-Hien et al., NeurIPS 2026 workshop](https://neurips.cc/virtual/2026/165420)
- **Free chat as elicitation omits implicit requirements; a structured
  space of dimensions helped.** [EVIDENCE, authors' own metrics]
  [Jin et al., arXiv 2605.05828, RE 2026](https://arxiv.org/abs/2605.05828)

### Options weighed

1. Adopt Spec Kit's clarify shape as is: capped questions, options plus a
   recommended one, a Clarifications log. Its recommendation breaks
   `widen-options-never-recommend`, and its five-question cap is a count,
   not a record-state gate.
2. A native staged interview shaped to the BDD discovery pattern (story,
   then rules, then concrete examples) and to the measured mistake list,
   with two seams: the question itself is the stop record the autonomous
   loops pass already recommends (an MCP-elicitation envelope, so another
   renderer or interviewer can ask it), and the question lint is a list a
   later study can replace.
3. A one-page form or free-text dump interrogated afterwards. The evidence
   favours the dump first and the interrogation second, not either alone.

### Recommendation

**Path 2.** Narrative first (the dump), then frontier rounds gated on the
record's unresolved fields rather than a count or the model's inclination;
one question per turn enforced structurally; the confirmation step
paraphrases the person's own words (the "wrong dots" move is a cheap
check); the null answer is a named option in a fixed, labelled position
while the up-to-three construals rotate between sessions; every committed
product carries authored-by provenance because co-creation is measured, not
hypothetical; "can't articulate yet" is recorded as its own state, never
inferred from. Draft follow-ups through the question-drafter with the
14 mistake types as its refusal list. The async variant is the same record
answered later (see itd-90). No Go dependency. The product thinker's
rulings on the escalation rule, the one-intent final round and the held
working principle are untouched by anything here.

### Local fit risks

- `widen-options-never-recommend` against the field's habit (Spec Kit's
  recommended option, the grill-me skill's recommended answer): the
  divergence is deliberate and the defaults evidence backs it, but a
  challenger should check that rotating option order does not itself read
  as a hidden preference.
- `one-canonical-primitive`: the interview must ask through the stop record
  and the GRILL layout, not a second question schema of its own.
- Host-delegated LLM work: the mistake-type lint runs on the host's agent;
  the binary checks only what it can check mechanically (one question per
  call, option count, null answer present).
- `the-record-lands-with-the-act`: the two-output rule (adr-50) discards
  the transcript; the evidence that confirmation is a provenance control
  rather than proof argues for keeping the ledger's grounds rich.

## itd-143: the framing chapter

### Findings

- **Design-document convention keeps the frame out of "context".** Context
  "isn't a requirements doc" and must be "entirely focused on objective
  background facts"; non-goals are "things that could reasonably be goals,
  but are explicitly chosen not to be goals"; post-launch changes go into
  separate documents, like constitutional amendments. [CONSENSUS]
  [Ubl, Design Docs at Google, 2020](https://www.industrialempathy.com/posts/design-docs-at-google/)
- **The problem is a single story, not an abstraction.** "The best problem
  definition consists of a single specific story that shows why the status
  quo doesn't work." [CONSENSUS] [Shape Up, chapter 6](https://basecamp.com/shapeup/1.5-chapter-06)
- **The one artefact with a structural line between exploring and
  committed is the opportunity solution tree**: the outcome at the root
  "sets the scope for discovery"; "Whenever we put a solution in our
  delivery backlog, we are making a bet." [CONSENSUS]
  [Torres, Product Talk, 2023](https://www.producttalk.org/opportunity-solution-tree/)
- **Built against intended, in living specs.** OpenSpec: specs "describe how
  your system currently behaves", changes "live in separate folders until
  you're ready to merge them", archiving merges the deltas. [EVIDENCE,
  documented tool behaviour] [OpenSpec, docs/concepts.md](https://github.com/Fission-AI/OpenSpec/blob/main/docs/concepts.md)
  Spec Kit goes the other way, banning speculation from a spec ("No
  speculative or 'might need' features") and confining open points to
  `[NEEDS CLARIFICATION: specific question]`. [EVIDENCE]
  [github/spec-kit, spec-driven.md](https://github.com/github/spec-kit/blob/main/spec-driven.md)
- **RFC 2119 words cannot carry "intended, not built".** They "must be used
  with care and sparingly" and "only … where it is actually required for
  interoperation"; OpenSpec uses them on current and proposed requirements
  alike. [EVIDENCE] [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119)
- **Status in links, not prose** is the earlier pass's fourth
  recommendation; the brief stays present tense and a stable section
  identifier with a `realises` link carries what is built. [CONSENSUS]
  [brief-to-record traceability pass](2026-10-04-brief-to-record-traceability-sota.md)
- **A frame is a lens, not a solution** (Dorst's frame creation "focuses not
  on the generation of solutions but on the ability to create new
  approaches"), from a publisher's description only; the primary texts were
  paywalled. [EVIDENCE, bibliographic]
  [Frame Innovation, publisher description](https://www.bookshare.org/browse/book/1017415)

### Options weighed

1. Adopt OpenSpec's split as is: a current-truth brief plus change folders.
   It would replace the brief's single present-tense document and the
   ratified passage-level marker.
2. A native section shaped to the design-doc convention: the construal
   statement sits with goals and non-goals, never in context; the ratified
   `**Status: NOT YET REAL.**` marker stays; the section carries the stable
   identifier the traceability pass asks for, so status can later move from
   the marker to links without rewriting the section.
3. No framing section; a plain problem statement in scope. Shape Up and the
   PR/FAQ already do this, which is the honest contrarian case.

### Recommendation

**Path 2.** The construal is interpretive, so by the convention readers
know it belongs beside goals rather than in the context chapter; the
one-or-two-sentence form and the marker form are ratified and stay. Give
the section a stable identifier at birth. Where the research disagrees
with the ruling it is on one point only: the field keeps status out of
prose and in links, while the marker puts status in prose. The marker is
recorded as the first instance of that status, to be superseded by links
when the traceability work lands, not abandoned now. No Go dependency.

### Local fit risks

- adr-5 (the brief holds the present) against the marker's status-in-prose:
  the challenger should decide whether a passage-level status token is a
  present-tense statement about the project or a status field in disguise.
- `less-but-better`: option 3 is a real contender; the construal earns its
  section only as a revisable, goal-level statement with its history on the
  local ledger side (adr-55), which the intent already specifies.

## itd-90: the blanks interview

### Findings

- **Asynchronous written interviews work and are slow.** Email interviews
  "garnered rich, insightful data that augmented the phone interviews",
  with email participants' share of cited quotes (20 to 37.5 per cent)
  comparable to their share of the sample (31 per cent chose email, in a
  sensitive-topic health study); the paper does not judge the two equally
  rich. They took "27 to 28 days" on average and ran "4 to 5 cycles" of
  exchange against an hour by phone. [EVIDENCE]
  [Hershberger and Kavanaugh, Applied Nursing Research 2017](https://pmc.ncbi.nlm.nih.gov/articles/PMC5658038)
- **A static form is not the async interview.** Per-person yield from a
  survey is a sixth to an eighth of an interview's (above, Obaidi et al.);
  self-administered model interviews were acceptable (90.9 per cent
  positive) but criticised for "generic questions" and "insufficient
  depth", and the report "do[es] not establish … equivalence to
  human-conducted interviews". [EVIDENCE, descriptive]
  [Gheyi et al., arXiv 2607.14452](https://arxiv.org/abs/2607.14452)
- **"Defer" must be a distinct recorded state** (Wetzelhütter, above): it is
  not a neutral answer and not non-response.
- **The stop record already has the shape this needs.** MCP elicitation's
  envelope (a message, a flat schema, `enum` with `enumNames`, and
  accept / decline / cancel) with the autonomous loops pass's additions
  (addressee role, judgement-or-fact kind, non-recommendation flag, the
  record concerned, a version), rendered by the host's question tool today
  and by a web page or a terminal later; a2a and the agent frameworks make
  the waiting state durable data that resumes on the same id. [EVIDENCE]
  [autonomous loops pass, itd-169 section](2026-10-23-autonomous-loops-sota.md)
- **Provenance vocabulary exists.** PROV's `Agent` "bears some form of
  responsibility" and `actedOnBehalfOf` covers a facilitator relaying a
  product thinker's answer (same pass). [EVIDENCE]
- **Reviewers of a hand-off read the frame they are given.** Co-creation
  (Ferrari et al., above) means a pre-populated or assistant-helped answer
  needs the `agent-assisted` flag the intent already specifies.

### Options weighed

1. Live only: the facilitator runs a session with the product thinker
   present. Highest depth, lowest reach; the intent's own reason for
   existing is that the thinker is not present.
2. One record, two renderings: the coverage blank becomes a stop record;
   answered live through the host's question tool or later from an
   exported file; defer is a named option; the answer appends to the record
   with authored-by and `actedOnBehalfOf`; a re-import runs the adaptive
   follow-up the async evidence says a form lacks.
3. A questionnaire export with no follow-up round. Measured yield is a
   fraction of an interview's.

### Recommendation

**Path 2**, option 2. Both synchronous and asynchronous are supported
because they are the same record; the canonical thing is the record, not
the session. Budget the async path for several cycles and weeks, not one
pass. The in-progress answers live keyed on the source (the voyage log is
the natural home) because the lifeboat is regenerable; the challenger
should test whether that duplicates the hold register of itd-142 (it
should be the same store with a different entry point). Partials are
confirmable, by analogy with Obaidi et al.'s delayed-taxonomy condition (a
classification taxonomy introduced after free elicitation yielded the most
needs), which is not a direct test of showing what was found and asking for
the top-up. No Go dependency.

### Local fit risks

- Host-delegated: the "minimum viable assistant" is the host's agent
  drafting follow-ups; the deterministic export and import must stand
  without it, as the intent says.
- `durable-state-lives-where-the-platform-says-it-survives`: an export
  answered on a train is a file; the import must validate it as untrusted
  input (the agent frameworks warn that a resumed snapshot "does not
  authenticate").
- Host-agnostic prose: the record's rendering names no host; the
  question tool is one renderer.

## itd-2609292108373494: hand-over to an installed interviewing skill

### Findings

- **A skill has a small, stable on-disk shape.** `SKILL.md` with required
  `name` (which "Must match the parent directory name") and `description`
  (1 to 1024 characters, "what the skill does and when to use it"), an
  optional `compatibility` field, and three-stage progressive disclosure:
  name and description at startup, the body on activation. [EVIDENCE]
  [Agent Skills specification](https://agentskills.io/specification)
- **Where a host looks differs per host.** Claude Code: personal
  `~/.claude/skills/<name>/SKILL.md`, project `.claude/skills/<name>/`,
  nested, additional directories, plugin `<plugin>/skills/<name>/`;
  `user-invocable: false` hides a skill from the person;
  `disable-model-invocation: true` stops the agent loading it on its own;
  through the SDK or `claude -p "/skill-name args"` only the first skill
  loads. [EVIDENCE] [Claude Code, Skills](https://code.claude.com/docs/en/skills)
  Other hosts use other directories (`.agents/skills/`, `.github/skills/`,
  `.gemini/skills/`, `.cursor/skills/`, `.windsurf/skills/` per one 2026
  survey). [ANECDOTE]
  [Vaughan, cross-agent skills, May 2026](https://codex.danielvaughan.com/2026/05/09/cross-agent-skills-ecosystem-maturity-npm-analogy-codex-cli-portability/)
- **abcd can start a skill only through the host, and a headless start
  cannot hold an interview.** A program reaches a skill through the host's
  CLI or SDK (`claude -p "/skill-name args"` loads the first skill named);
  in such a headless run the host's question tool auto-resolves with an
  empty answer (issue opened 2026-04-19, closed "not planned"). A skill with
  `disable-model-invocation: true` cannot be invoked by the agent "on its
  own" and its description is not in context, so detection must read that
  field too. [EVIDENCE]
  [anthropics/claude-code issue 50728](https://github.com/anthropics/claude-code/issues/50728)
  Even interactively the question tool has an open rendering bug: it
  "immediately returns … and an empty/blank answer" in some sessions
  (opened February 2026, still open). [ANECDOTE, many reporters]
  [anthropics/claude-code issue 29530](https://github.com/anthropics/claude-code/issues/29530)
- **Public interviewing skills disagree on abcd's core rule.** The
  brainstorming skill forbids option menus ("Options steer"); a grill-me
  skill asks one question at a time and gives "your recommended answer"
  with each. [CONTESTED]
  [obra/superpowers brainstorming](https://github.com/obra/superpowers/blob/main/skills/brainstorming/SKILL.md),
  [grill-me on skills.sh](https://www.skills.sh/petekp/claude-skills/grill-me)
- **The skill ecosystem is not vetted.** A vendor scan of 3,984 skills found
  "13.4% of all skills, or 534 in total, all contain at least one
  critical-level security issue"; the vendor sells the remedy.
  [ANECDOTE/MARKETING for the figure; CONSENSUS on fragmentation]
  [Snyk, ToxicSkills, February 2026](https://snyk.io/blog/toxicskills-malicious-ai-agent-skills-clawhub)

### Options weighed

1. Detect any skill whose description mentions interviewing and hand over.
   Rejected: unvetted ecosystem, divergent disciplines, and a description
   match is the host's heuristic, not a contract.
2. The person names one skill (user-level setting, overridable per
   repository); abcd checks the host-specific path for a `SKILL.md` whose
   `name` matches its directory; abcd's command page tells the agent to
   invoke that skill and prints the asking discipline it expects (one
   question at a time, options that widen, decide-later present, the record
   abcd will write); with no such skill, or outside a live interactive
   session, abcd's own interview runs and says so.
3. Vendor or ship an interviewing skill. Out of scope by the intent and by
   "abcd ships zero skills".

### Recommendation

**Path 2**, option 2. The Agent Skills layout is the seam: detection reads
the spec's two required fields plus `disable-model-invocation` (a skill that
sets it is detected but cannot be handed over, so abcd tells the person to
type its command or runs its own interview), and the host path table is the
one part a new host replaces. The hand-over covers both the grill and
the planning interview but only in a live session; a headless run never
hands over. abcd's asking rules travel as a brief to the external
interviewer, not as a demand, and the record shape is unaffected because
abcd writes the record from the answers it receives, whoever asked. The
resumption criterion (abcd's own interview continues from the last
recorded answer) follows from the record being abcd's. No Go dependency.

### Local fit risks

- `the-users-directory-is-theirs`: detection reads under the person's home
  (`~/.claude/skills/`); it must be a read of one named path, never a scan,
  and never a write.
- Host-agnostic prose: the user-facing page names "the interviewing skill
  you installed", not a host or a marketplace; the host path table is
  internal.
- `widen-options-never-recommend`: a skill that recommends will recommend;
  abcd can state its rules and record which interviewer asked, and the
  challenger should decide whether a recommending interviewer's answers
  need a marker in the record.
- adr-22 (native default, external as opt-in adapter) is the shape; the
  intent already cites it.

## itd-2610030810370060: interviews in a plain Terminal

The product thinker decided on 2026-10-03 (decisions 1 to 5) and the intent
is planned; this pass checks those decisions against what is public on
2026-10-23 and adds what the local-tier report could not be cited for.

### Findings

- **The line-mode fallback has a shipped convention.** huh's accessible
  mode prints options as `"%d. %s\n"`, prompts `"Enter a number between %d
  and %d: "`, and re-asks on invalid input; its TUI select filters on `/`.
  [EVIDENCE] [charmbracelet/huh, field_select.go](https://github.com/charmbracelet/huh/blob/main/field_select.go)
  huh's README: accessible forms "drop TUIs in favor of standard prompts",
  enabled by the application. [EVIDENCE] [charmbracelet/huh](https://github.com/charmbracelet/huh)
- **GitHub's CLI exposes three independent switches, not one mode.**
  `GH_ACCESSIBLE_PROMPTER` ("prompts that are more compatible with speech
  synthesis and braille screen readers"), `GH_SPINNER_DISABLED`,
  `GH_FORCE_TTY` (also a column count), `NO_COLOR`, `CLICOLOR=0`.
  [EVIDENCE] [gh help environment](https://cli.github.com/manual/gh_help_environment)
  Its accessible prompter is huh in accessible mode with the Base16 theme.
  [EVIDENCE] [cli/cli, prompter.go](https://github.com/cli/cli/blob/trunk/internal/prompter/prompter.go)
  What breaks speech synthesis is "Non-alphanumeric visual cues and uses of
  constant screen redraws"; the braille spinner became static text.
  [CONSENSUS] [GitHub blog, May 2025](https://github.blog/engineering/user-experience/building-a-more-accessible-github-cli/)
- **Screen-reader users want a substring filter in the numbered mode**:
  "the accessible prompter doesn't allow you to filter choices by a
  substring … I could, of course, memorize the numbers, but then someone
  might add or remove a workflow or label." [ANECDOTE, direct report]
  [GitHub community discussion 158037](https://github.com/orgs/community/discussions/158037)
- **The host's own screen-reader mode is the fullest current line-mode
  specification**: arrow-key menus "become numbered lists", "No color-only
  cues", "No redraws of content that hasn't changed", yes/no as typed
  `y`/`n`, a bell when input is needed, a first line confirming how the
  mode was enabled, and it "doesn't turn on automatically when a screen
  reader is running"; its question timeout never runs in that mode.
  [EVIDENCE] [Claude Code, Use Claude Code with a screen reader](https://code.claude.com/docs/en/accessibility),
  [Tools reference, AskUserQuestion](https://code.claude.com/docs/en/tools-reference)
- **Practitioner consensus in 2026 still prefers linear output**: "a dumb,
  linear CLI stream is infinitely superior" to a redrawing TUI; avoid
  spinners that move the cursor; keep one column of focus. [CONSENSUS]
  [The Inclusive Lens, January 2026](https://xogium.me/the-text-mode-lie-why-modern-tuis-are-a-nightmare-for-accessibility)
  The one controlled study (12 developers) names unstructured output,
  missing status, ASCII decoration and jargon errors. [EVIDENCE]
  [Sampath, Merrick and Macvean, CHI 2021](https://storage.googleapis.com/gweb-research2023-media/pubtools/6065.pdf)
- **Interactivity and interrupt rules**: prompt only when stdin is a TTY,
  exit promptly on Ctrl-C, honour `NO_COLOR`, `TERM=dumb` and `--no-color`,
  check stdout and stderr separately. [CONSENSUS]
  [Command Line Interface Guidelines](https://clig.dev/)
  `NO_COLOR` is "present and not an empty string (regardless of its
  value)", and per-run flags override it. [EVIDENCE] [no-color.org](https://no-color.org/)
- **The signed-off dependency does what is needed and no more.**
  `golang.org/x/term` (signed off at v0.46.0 in decision 5; v0.47.0 was
  published 2026-10-09 and the package page lists a newer release, so the
  version is pinned at build time, not here): `MakeRaw` "returns
  the previous state of the terminal so that it can be restored",
  `Restore`, `GetSize`, `IsTerminal`. [EVIDENCE] [pkg.go.dev, golang.org/x/term](https://pkg.go.dev/golang.org/x/term)
  The former Go default, survey, is archived and "reading from piped stdin
  or writing to piped stdout is not supported". [EVIDENCE]
  [AlecAivazis/survey](https://github.com/AlecAivazis/survey)
  Inquirer: "In non-interactive shell, Inquirer cannot run"; its select
  has type-to-search on by default. [CONSENSUS]
  [Inquirer.js](https://github.com/SBoudrias/Inquirer.js)
- **Sixteen-colour slots have no values to measure.** "there's no
  standard, terminal emulators just choose colours"; bright-black is
  unreadable in Solarized Dark, white in light themes, and bold may promote
  a colour to bright. [CONSENSUS]
  [Evans, Terminal colours are tricky, 2024](https://jvns.ca/blog/2024/10/01/terminal-colours/),
  [Razzaghipour, How to Choose Colors for Your CLI Applications, 2023](https://blog.xoria.org/terminal-colors/)

### Options weighed

1. Adopt huh (and so Bubble Tea) for the renderer. It is the convention the
   GitHub CLI chose, but it is a framework, not a function, and the
   signed-off dependency already covers raw mode and size.
2. The native renderer on x/term as decided, shaped to huh's accessible
   contract for the numbered mode (same strings, same re-prompt) and to the
   host's screen-reader rules (no redraw of unchanged content, no
   colour-only cue, a bell on a question), with the numbered mode as the one
   code path every fallback shares and the arrow-key mode strictly additive.
3. Numbered mode only. Rejected by decision 3; the evidence still favours it
   as the default, and the ruling stands with the trade-off stated.

### Recommendation

**Path 2**, option 2, which is the decided design with four additions the
research supplies: the numbered mode accepts a unique substring of a label
as well as a number; the mode is a setting plus a flag, with `TERM=dumb`
and a non-TTY honoured independently and never auto-detected from a screen
reader (so decision 3's screen-reader case is met through the setting and
the flag, since no terminal can detect a screen reader; this reading is
recorded for the product thinker to confirm); the renderer appends lines and repaints only the option list, only
in arrow-key mode, only on a real TTY; and the badge contrast is certified
against a named reference palette (and the role word carries the meaning
with no colour at all) because no sixteen-colour value can be certified for
an unknown terminal. Where the research disagrees with decision 3
(arrow-key default) it is on accessibility grounds the product thinker
already weighed; it is recorded, not re-opened.

### Local fit risks

- Minimal dependencies: option 1 would add a framework for what one
  package covers; the challenger should confirm that the bespoke key reader
  stays at arrow keys, Enter, Escape, Ctrl-C and type-to-filter.
- Invariant 13 (sanitise before drawing) and B7 are already in the intent;
  the CHI study's "non-ASCII characters" finding adds glyphs (`✓`, braille
  spinners) to what the fallback never draws.
- Host-agnostic prose: the host's screen-reader mode is cited here as
  evidence; the user-facing page describes abcd's own behaviour.

## itd-168: the register and the role badges

### Findings

- **Plain-language rules that a linter can apply exist in two public
  standards.** GOV.UK: "Plain English is mandatory", "Try to split up
  sentences that are over 25 words long", no more than five sentences a
  paragraph, specialist terms explained on first use. [EVIDENCE]
  [GOV.UK, Use clear language](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/writing-guidelines/clear-language/)
  US federal guidance: write "for your specific audience", present tense
  "the simplest and strongest form of a verb", no hidden verbs ("We analyze
  data", not "We conduct an analysis of the data"). [EVIDENCE]
  [Digital.gov, Principles of plain language](https://digital.gov/guides/plain-language/principles),
  [Writing for understanding](https://digital.gov/guides/plain-language/writing)
  ISO 24495-1:2023 "establishes governing principles and guidelines for
  developing plain language documents" and applies to text "intended for
  the general public" and to technical writing; its four principle names
  were not readable on any page that opened. [EVIDENCE for scope]
  [EVS catalogue entry, ISO 24495-1:2023](https://www.evs.ee/en/iso-24495-1-2023)
- **Readability formulas are not quality gates.** Four formulas rated the
  flawed survey question easier than its improved version often enough that
  "the success rate … was below 50 percent". [EVIDENCE]
  [Lenzner, Sociological Methods and Research 2014](https://ideas.repec.org/a/sae/somere/v43y2014i4p677-698.html)
  Of eight metrics "most correlate poorly with human judgments, including
  the most popular metric, FKGL"; the best model judge reaches Pearson 0.56.
  [EVIDENCE] [Cachola, Khashabi and Dredze, arXiv 2508.19221](https://arxiv.org/abs/2508.19221)
- **Contrast thresholds**: text needs "at least 4.5:1" (large text 3:1),
  computed values "should not be rounded (e.g., 4.499:1 would not meet the
  4.5:1 threshold)"; "Color is not used as the only visual means of
  conveying information". [EVIDENCE]
  [W3C, Understanding SC 1.4.3](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html),
  [WCAG 2.2](https://www.w3.org/TR/WCAG22/#contrast-minimum)
- **Colour-vision guidance from the palette authors.** Okabe and Ito: "Do
  not convey information in color only", "Show difference BOTH in color and
  shape", avoid red with green. [EVIDENCE]
  [Okabe and Ito, Color Universal Design](https://jfly.uni-koeln.de/color/)
  Tol disagrees on the pair: "Contrary to popular belief, pure red and green
  can be distinguished. Instead, yellow/green, blue/violet and … cyan/pink/
  grey combinations can be problematic"; his high-contrast scheme
  (#004488, #DDAA33, #BB5566) is "colour-blind safe and optimized for
  contrast" and survives monochrome. [CONTESTED on red-green; EVIDENCE on
  the scheme] [Tol, colour schemes, 2026](https://sronpersonalpages.nl/~pault/)
  The intent's dark blue and amber pair is close to Tol's high-contrast
  blue and yellow, and its polarity difference is his lightness-separation
  principle.
- **Sixteen-colour slots cannot be certified** (Evans; Razzaghipour, above):
  the badge's 6.45:1 and 11.38:1 hold where abcd owns the pixels, and on a
  terminal only against a named reference palette.
- **Polarity is a weak redundant cue**: the evidence on light-on-dark
  against dark-on-light is about reading performance, not category
  recognition. [EVIDENCE, indirect]
  [Nielsen Norman Group, Dark Mode vs. Light Mode, 2020](https://www.nngroup.com/articles/dark-mode/)
- **Stating that nothing is recommended is justified by the mechanism of
  the default effect**: defaults work partly by "conveying what the choice
  architect thinks" (Jachimowicz et al., above), which is exactly what an
  unstated absence of preference leaves a reader to infer.
- **The host's own accessible theme and screen-reader rules** ("No
  color-only cues") are the nearest shipped precedent for a role label
  that survives without colour. [EVIDENCE]
  [Claude Code, Use Claude Code with a screen reader](https://code.claude.com/docs/en/accessibility)

### Options weighed

1. Adopt a readability score as the register check. Rejected on the two
   studies above.
2. A configured profile whose defaults are the GOV.UK and Digital.gov
   surface rules (sentence length, an example present, no record ids or
   `file:line` pointers in the product thinker's layer, present tense,
   bounded vocabulary) checked mechanically, the badges as specified by
   value where abcd draws, by role word plus polarity everywhere, with the
   non-recommendation statement at conjectural questions only.
3. No profile: one register for everyone. Rejected by the 2026-08-29 ruling.

### Recommendation

**Path 2**, option 2. The seam is the profile: the check reads it, and a
later standard (ISO 24495-1's guidelines once readable, or a validated
model judge) replaces the rule list without touching the surface. Keep the
badge specification by value and measured ratio as written; add that on a
sixteen-colour terminal the ratio is certified against a named palette and
the role word is the primary cue. Note Tol's dissent on red-green in the
spec rather than re-litigating the pair, which is blue and amber anyway. No
Go dependency.

### Local fit risks

- `facilitator-default-thinker-optional`: the product thinker's register
  is the optional layer; the profile must not make the facilitator's
  default vaguer.
- Host-agnostic prose: the badge is abcd's; where the host draws the
  question, abcd supplies the label text only.
- `guidance-carries-its-evidence-and-its-purpose`: the mechanical rules
  should cite which standard each came from, so a disputed rule can be
  traced.

## itd-30: design fictions as a capture format

### Findings

- **The originators define design fiction as making things, not writing
  prose.** Sterling: "the deliberate use of diegetic prototypes to suspend
  disbelief about change … It's not a kind of fiction. It's a kind of
  design." [EVIDENCE, primary] [Slate, March 2012](https://www.slate.com/blogs/future_tense/2012/03/02/bruce_sterling_on_design_fictions_.html)
  Bleecker: "Design Fiction is making things that tell stories."
  [EVIDENCE, primary] [Near Future Laboratory, 2009](https://nearfuturelaboratory.com/2009/03/17/design-fiction-a-short-essay-on-design-science-fact-and-fiction/)
- **Design fiction in requirements engineering has an entry point and thin
  evidence.** The RE 2018 paper used it "to explore how plausible new
  technologies might impact on dementia care"; no comparison with
  conventional elicitation. [EVIDENCE, abstract]
  [Darby, Tsekleves and Sawyer, RE 2018](https://research.aston.ac.uk/en/publications/speculative-requirements-design-fiction-and-re/)
  Recent HCI uses run with nine or ten participants and do not evaluate the
  method. [EVIDENCE, small] [Penney et al., arXiv 2311.06887](https://arxiv.org/abs/2311.06887),
  [Wadinambiarachchi et al., arXiv 2509.20731](https://arxiv.org/abs/2509.20731)
  A 2025 game-RE study measured acceptance, not requirement quality.
  [ANECDOTE-adjacent] [Leifheit et al., SBGames 2025](https://sol.sbc.org.br/index.php/sbgames/article/view/37350)
- **A fiction and a scenario have different plots.** "Traditional HCI
  scenarios employ the plot of 'Overcoming the Monster' where the monster
  is some problem to be solved"; "Design Fiction more frequently takes the
  form of a 'Voyage and Return' or a 'Quest'." Given/When/Then presumes the
  first; extracting a "Then" from the second invents it (our inference from
  Blythe's plots, not his claim; the abstract supports the plot distinction
  only). [EVIDENCE]
  [Blythe, CHI 2017](https://researchportal.northumbria.ac.uk/en/publications/research-fiction-storytelling-plot-and-design)
- **The field's own reviewers lacked guidelines** for design-fiction
  submissions as of 2020, and its critics say corporate use "declaw[s]"
  it into "a PR exercise". [CONSENSUS]
  [Muller et al., CHI EA 2020](https://researchportal.northumbria.ac.uk/en/publications/understanding-the-past-present-and-future-of-design-fictions/),
  [MIT Technology Review, February 2020](https://www.technologyreview.com/s/615190/)
  Sterling in 2025: "We don't much need another sci-fi prop."
  [ANECDOTE] [Near Future Laboratory, SXSW 2025](https://nearfuturelaboratory.com/blog/2025/03/bruces-sxsw-2025-design-fiction)
- **BDD already runs story to rules to examples**, and the examples come
  from conversation first: rules are "the acceptance criteria, or rules
  that we already know"; "Concrete examples are a tremendous way to help us
  explore the problem domain"; a story maps in "about 25 minutes".
  [CONSENSUS] [Wynne, Introducing Example Mapping, 2015](https://cucumber.io/blog/bdd/example-mapping-introduction/)
  Scenarios "should describe the intended behaviour of the system, not the
  implementation"; UI-mechanics steps "often require more work to
  maintain". [CONSENSUS] [Cucumber, Writing better Gherkin](https://cucumber.io/docs/bdd/better-gherkin/)
- **Extraction by a model works on clear input and echoes otherwise.**
  Industrial case: generated acceptance scenarios "helpful 95% of the
  time"; downstream tests "60% were usable as generated, 8% required minor
  fixes, and 24% needed to be regenerated with additional inputs".
  [EVIDENCE] [Ferreira et al., AST 2025, arXiv 2504.07244](https://arxiv.org/abs/2504.07244)
  A 2026 comparison found the zero-shot baseline echoes requirement
  wording (full text), lexical metrics "may miss coverage when scenarios
  paraphrase requirements at a higher level of abstraction", and expert
  raters preferred the abstracted output (completeness 4.31 against 3.50).
  [EVIDENCE, one study] [Siddeeq et al., arXiv 2607.01980](https://arxiv.org/abs/2607.01980)
- **The baseline format is also defective in the wild**: 56 per cent of
  1,023 user stories from 18 companies had at least one detected defect.
  [EVIDENCE] [Lucassen et al., Requirements Engineering 2016](https://link.springer.com/article/10.1007/s00766-016-0250-x)

### Options weighed

1. A free-standing fiction record type with its own review path. No
   retrievable study stores the fiction as the specification; the
   community has no reviewer rubric.
2. The fiction as the discovery input to the same contract: the interview
   collects the vignette, then runs the BDD step (rules, then concrete
   examples) to extract criteria, judged by blind human rating rather than
   word overlap; the fiction is kept in the record as the `## Design
   Fiction` section the intent specifies, but the criteria are what the
   gate and the auditor read first.
3. Drop the second format. The contrarian case: it splits the corpus and
   the press release's problem paragraph is already a story.

### Recommendation

**Path 2**, option 2. This keeps the intent's acceptance criteria intact
(same gate, same `capture_format` field, same auditor), narrows one of its
open questions and answers another from evidence: the nearest evidence
(requirements- and story-to-Gherkin generation; neither study extracts
criteria from a fiction) suggests acceptance extraction echoes its input
unless the prompt carries contrasting examples and the check is not
lexical, so that question narrows but stays open; and the artefact-type set
should drop or caveat the screenshot caption, which anchors to a screen
before one exists. The
revisit triggers stand. No Go dependency.

### Local fit risks

- `one-canonical-primitive`: the fiction interview must extend the grill
  machinery, as the intent's implementation note already requires.
- `evaluator-outside-the-loop`: the extracted criteria are judged by a
  reader who did not write the fiction.
- `less-but-better`: option 3 is live until a revisit trigger fires; the
  evidence for the format's benefit in RE is acceptance, not quality.

## itd-172: short title and one-line summary

### Findings

- **GOV.UK's title and summary rules are mechanical and public.** Titles:
  "65 characters or less (including spaces)", "unique and informative",
  content type not in the title. Summaries: "Keep all summaries to 160
  characters, including spaces", "Use as few words from the title as
  possible", end "with a full stop". [EVIDENCE]
  [GOV.UK, Write clear titles](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/writing-guidelines/clear-titles/),
  [Write clear summaries](https://guidance.publishing.service.gov.uk/writing-to-gov-uk-standards/writing-guidelines/summaries/)
- **Register rules** (sentence length, present tense, no hidden verbs) are
  the itd-168 findings above; readability formulas are not gates (Lenzner;
  Cachola et al.).
- **Software conventions for the handle.** git: "a single short (no more
  than 50 characters) line summarizing the change". [EVIDENCE]
  [git-commit documentation](https://git-scm.com/docs/git-commit)
  Conventional Commits: the description is "a short summary of the code
  changes", no limit stated. [EVIDENCE] [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)
  MADR: "{short title, representative of solved problem and found
  solution}", files `NNNN-title-with-dashes.md`. [EVIDENCE] [MADR](https://adr.github.io/madr/)
  RFC abstracts "should be complete in itself" with no citations.
  [EVIDENCE] [RFC 7322](https://www.rfc-editor.org/rfc/rfc7322.html)
- **Two fields, two jobs, is an existing split**: Hugo's `description`
  (metadata) against `summary` ("summarizes the content or serves as a
  teaser"). [CONSENSUS] [Hugo front matter](https://gohugo.io/content-management/front-matter/)
- **A plain-language summary field with mechanical rules has a precedent.**
  Cochrane's standards: "no more than 700 words", "Limit sentences to one
  key point", "Avoid more than two hard words in a sentence unless you
  explain them", the title to "Avoid declarative statements and
  recommendations"; a fixed hedging vocabulary ("will / probably / may").
  [EVIDENCE] Cochrane PLEACS v3.0 (2013) and the 2017 how-to, both opened
  as PDFs from Cochrane group sites (see Not verified for the current
  template).
- **The Agent Skills spec's `description` field** ("Max 1024 characters …
  Describes what the skill does and when to use it", with an example of a
  poor one, "Helps with PDFs.") is the same two-field shape applied to
  agent-read records. [EVIDENCE] [Agent Skills specification](https://agentskills.io/specification)

### Options weighed

1. Adopt the GOV.UK limits as abcd's rule set: title at most 65 characters,
   summary at most 160 ending in a full stop, few shared words with the
   title, a verb, no record-type word in the title; present tense and the
   itd-168 register rules for the summary's words.
2. A native rule set with other numbers. The evidence offers no other
   numbers; the 50-character git subject is a different artefact. GOV.UK's
   own reason for 65 is search-result truncation ("Google cuts off the rest
   of the title at around 65 characters") and the limit is soft ("You can
   use more than 65 characters if it's essential"), so abcd borrows the
   numbers without their reason; its own constraint is a filename and a
   terminal listing.
3. A readability score. Rejected above.

### Recommendation

**Path 1**, option 1, with the warn-then-promote migration the intent
already names, and the numbers adopted as a convenient bound rather than as
GOV.UK's reason: the title cap is checked against the slug length and an
80-column listing, and warns rather than refuses above 65, as the source's
own limit is soft. The summary is the product thinker's field (itd-168's
register applies to it, not to the body). No Go dependency.

### Local fit risks

- `retire-the-name`: a title cap changes slugs and so filenames; existing
  records keep their ids, and the migration must not rename files whose
  slugs other records link to without updating the links.
- `enforcement-claims-are-facts`: the rule is advisory until the corpus is
  clean, and the record must say so.

## itd-99: a team of product thinkers

### Findings

- **Independence before aggregation is measured, not doctrinal.** Social
  influence "diminishes the diversity of the crowd without improvements of
  its collective error" and "boosts individuals' confidence after
  convergence of their estimates despite lack of improved accuracy"
  (144 subjects, estimation tasks). [EVIDENCE]
  [Lorenz et al., PNAS 2011](https://pmc.ncbi.nlm.nih.gov/articles/PMC3107299)
- **The one large trial of machine collation used this exact protocol.**
  Private opinions, a machine draft, private critiques, a revised draft:
  5,734 participants preferred the machine statement to the human
  mediator's 56 per cent of the time ("More than half (56%) of the time, the
  participants chose the AI statement", each round showing one of each); the
  report describes the successful statements as carrying the dissenting
  views while keeping the majority position, and says the model "doesn't
  have the mediation-relevant capacities of fact-checking". [EVIDENCE; the
  journal page was not reachable, so the figures are the magazine report's
  and no wording is quoted from the paper]
  [MIT Technology Review, October 2024](https://www.technologyreview.com/2024/10/17/1105810/ai-could-help-people-find-common-ground-during-deliberations/)
- **Off-the-shelf summarisers under-represent minorities and a model judge
  does not notice.** LLMs "risk underrepresenting minority perspectives and
  exhibiting bias with respect to the input order"; "LLMs as judges … show
  weak alignment with human judgments" (3,000 opinion contributors, 4,500
  judgement annotators). [EVIDENCE]
  [Zhu et al., arXiv 2510.05154](https://arxiv.org/abs/2510.05154)
- **Standing bodies decide by addressed objections, not counts.** IETF:
  "Lack of disagreement is more important than agreement", "the existence
  of the unaddressed open issue, not the number of people, is
  determinative", "Hums should not be used as votes"; a dissenter whose
  objection was considered and answered is "in the rough". [CONSENSUS]
  [RFC 7282](https://www.rfc-editor.org/rfc/rfc7282)
  Sociocracy 3.0: an objection "reveals consequences or risks that are
  preferably avoided", a concern is "an assumption that cannot … be backed
  up by reasoning or enough evidence", proposals pass when "good enough for
  now and safe enough to try". [CONSENSUS]
  [Sociocracy 3.0, Consent Decision-Making](https://patterns.sociocracy30.org/consent-decision-making.html)
- **Dissent is recorded structurally where it is recorded at all.** Rust:
  "@rfcbot concern NAME_OF_CONCERN", "only the original author can mark
  their concern as resolved", "only one concern per comment". [CONSENSUS]
  [rust-lang/rfcbot-rs README](https://github.com/rust-lang/rfcbot-rs/blob/master/README.md)
  Python: "Those rejected ideas should be recorded along with the reasoning
  as to why they were rejected." [CONSENSUS] [PEP 1](https://peps.python.org/pep-0001/)
  The single-approver frameworks have no dissent slot: RAPID ("ideally,
  there should only be one decider"), DACI ("The one person (yes: one!)
  who makes the decision"), MADR (decision-makers, consulted, informed;
  "Good/Bad, because"). [ANECDOTE/MARKETING for the frameworks; EVIDENCE
  for the template]
  [Bain, RAPID](https://www.bain.com/insights/rapid-decision-making/),
  [Atlassian, DACI](https://www.atlassian.com/team-playbook/plays/daci),
  [MADR](https://adr.github.io/madr/)
- **"Disagree and commit"**: "Leaders are obligated to respectfully
  challenge decisions when they disagree … Once a decision is determined,
  they commit wholly." [ANECDOTE]
  [Amazon leadership principles](https://www.aboutamazon.com/about-us/leadership-principles)
- **Breadth across groups without counting heads** is Polis's
  group-informed consensus, the product of per-group agreement "to counter
  tyranny of the majority". [CONSENSUS]
  [Computational Democracy Project](https://compdemocracy.org/Group-Informed-Consensus/)

### Options weighed

1. Adopt a decision framework (RAPID, DACI) as the record. None carries
   dissent; all end in one approver, which is not what the intent promises.
2. A native protocol shaped to the trial above and to rfcbot: each thinker
   answers the itd-90 interview privately; the collator (the press-release
   composer's cite-or-be-dropped rule) drafts an attributed proposal;
   each thinker privately confirms that their own position is rendered
   (the step the summariser evidence makes necessary, since a model judge
   cannot); disagreements are named, author-owned concerns that only their
   author clears; the closing test is RFC 7282's "addressed, but not
   necessarily accommodated"; the decision lands with adopters, date and
   each concern's verbatim text and state.
3. Polis-style scoring of statements by breadth. Useful later for larger
   teams; counting is out of scope for v1.

### Recommendation

**Path 2**, option 2, extending the press-release composer rather than
adding a collator (`one-canonical-primitive`). The privacy collision
resolves the way the stop record does: participants are roles or handles
in the record, mapped to people in an uncommitted local file, with PROV's
`actedOnBehalfOf` for a facilitator relaying an answer. Dissent does not
block planning by itself; an unresolved concern blocks because it is
unaddressed, and the author's clearing it is what resolves it. One point
of tension with the seeded press release: it has the joint proposal name
"its recommendation", while `widen-options-never-recommend` binds any
conjectural question put to a product thinker; a technical choice priced
as options is the facilitator's domain and may carry one, a framing
choice may not. Recorded for the product thinker, who owns the press
release. No Go dependency.

### Local fit risks

- PII discipline against named adopters: the handle indirection must be
  proven against `abcd lint` before any schema exists, as the intent says.
- Host-delegated: the collator is an agent; the binary checks citations,
  attribution and that every named participant's session is present.
- `evaluator-outside-the-loop`: the per-thinker confirmation step is the
  evaluator; no model judges whether a minority view survived.

## itd-97: the facilitator as a mode

### Findings

- **Autonomy is a design choice on a scale, and the interesting boundary
  is between approval and notification.** Sheridan and Verplank's level 5
  "executes that suggestion if the human approves", 6 "allows the human a
  restricted time to veto before automatic execution", 7 "executes
  automatically, then necessarily informs humans". [CONSENSUS]
  [MIT OCW 16.422 lecture notes](https://ocw.iti.hr/courses/aeronautics-and-astronautics/16-422-human-supervisory-control-of-automated-systems-spring-2004/lecture-notes/020504hmancenter.pdf)
  A 2025 framework names five levels by the user's role (operator,
  collaborator, consultant, approver, observer), autonomy "treated as a
  deliberate design decision, separate from its capability". [CONSENSUS]
  [Feng, McDonald and Zhang, arXiv 2506.12469](https://arxiv.org/abs/2506.12469)
  Human mode is level 5 (approver); auto-merge is level 7 (observer); a
  merge queue with a veto window would be level 6.
- **The coding hosts have converged on classifier-gated auto with hard
  residual refusals.** Claude Code's `auto` runs "Everything, with
  background safety checks"; it never auto-approves the question tool or
  writes to protected paths; it pauses after 3 consecutive or 20 total
  blocks; and its classifier blocks by default "Merging a pull request no
  human has approved, approving Claude's own pull request, or disabling CI
  checks"; `auto` and `bypassPermissions` set in a project's settings
  "doesn't take effect". [EVIDENCE] [Claude Code, Permission modes](https://code.claude.com/docs/en/permission-modes)
  Cursor: "Ask Every Time was deprecated"; file deletion, browser and
  outside-workspace files still prompt in every mode; "The classifier can
  make mistakes." [EVIDENCE] [Cursor, Run Modes](https://cursor.com/docs/agent/run-modes)
  Gemini CLI: "YOLO mode (auto-approve all actions) can only be enabled via
  command line". [EVIDENCE] [Gemini CLI configuration](https://geminicli.com/docs/reference/configuration/)
- **Auto-merge is only a rule if something is required.** "Auto-merge
  merges a pull request automatically after all required reviews and
  status checks pass"; the option "is shown only on pull requests that
  cannot be merged immediately". [EVIDENCE]
  [GitHub Docs, Automatically merging a pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request)
  A merge queue ensures the changes "pass all required status checks when
  applied to the latest version of the target branch". [EVIDENCE]
  [GitHub Docs, Managing a merge queue](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue)
- **The bots already live under the human-review rule.** Renovate: "enable
  automerge for any dependency update where you would select 'merge'
  anyway"; it "can't automerge its own PR until such a review has
  happened". [CONSENSUS] [Renovate, Automerge](https://docs.renovatebot.com/key-concepts/automerge/)
  Kodiak: "GitHub considers a 'neutral' check run conclusion as passing";
  so "checks pass" must mean success, not absence of failure. [EVIDENCE]
  [Kodiak configuration reference](https://kodiakhq.com/docs/config-reference)
  GitHub's own coding agent: the requester's approval "won't count",
  "Another reviewer must approve", workflows wait for "Approve and run
  workflows"; its reviewer "will not approve pull requests" by default, "An
  approval assessment alone does not count toward merge requirements", and
  "When enabled, Copilot can submit an approval that counts toward the
  repository's required-approvals rule", off by default and switched on by an
  administrator at enterprise, organisation or repository level, so the forge
  now lets a model approval satisfy a required review where that is enabled.
  [EVIDENCE] [GitHub Docs, Reviewing a pull request created by Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/review-copilot-prs),
  [GitHub Changelog, September 2026](https://github.blog/changelog/2026-09-01-copilot-code-review-can-now-approve-pull-requests)
- **A repository's policy can forbid the whole mode.** QEMU will "DECLINE
  any contributions which are believed to include or derive from AI
  generated content" on sign-off grounds. [EVIDENCE]
  [QEMU, Code provenance](https://www.qemu.org/docs/master/devel/code-provenance.html)
- **The human gate decays.** Automation bias "occurs in both naive and
  expert participants, cannot be prevented by training or instructions".
  [EVIDENCE] [Parasuraman and Manzey, Human Factors 2010](https://depositonce.tu-berlin.de/items/0c787d91-6021-480b-a333-009e0d1d0092/full)
  When a decision aid is wrong, users were more likely to err, pooled risk
  ratio 1.26 (95% CI 1.11 to 1.44); 6 to 11 per cent of correct answers
  were flipped. [EVIDENCE] [Goddard, Roudsari and Wyatt, JAMIA 2011](https://pmc.ncbi.nlm.nih.gov/articles/PMC3240751)
- **Vendor guidance**: agents "pause for human feedback at checkpoints or
  when encountering blockers"; "find the simplest solution possible".
  [CONSENSUS] [Anthropic, Building effective agents, 2024](https://www.anthropic.com/research/building-effective-agents)

### Options weighed

1. Adopt a host's permission mode as the facilitator mode. It is the
   host's setting, not the repository's, and it governs tool calls, not
   who merges.
2. A native two-point mode shaped to the levels scale and to the hosts'
   convergence: human is the approver level, auto the observer level; both
   share the residual hard list (questions to a person, protected
   configuration paths, deletion, the product thinker's decisions); the
   merge gate reads the mode; "checks pass" is defined as success on the
   merge-group ref with a human approval the requester cannot supply in
   human mode; repository configuration can forbid auto but should not
   alone arm it, since a committed `.abcd/` file is writable by any agent in
   the run and by any pull request (the guard's protected-path list,
   `internal/core/guard/approvepaths.go`, does not include it) and the hosts
   refuse to take their own auto mode from project or local settings; so
   either auto is armed from a user-level setting and the repository may
   only lower it, or the key is a protected path the guard refuses to write
   and only the person's own act changes; an audit of auto-merged pull requests
   (random re-review, rejection rate) is part of the mode because the gate
   evidence says the human one will decay too.
3. A third mode (unattended, no gates). The hosts refuse to persist their
   equivalent; if ever added it is command-line only.

### Recommendation

**Path 2**, option 2. On the open questions: "Where does the mode live?"
stays open. Repository configuration can forbid auto but should not alone
arm it (the hosts refuse to take their auto mode from project or local
settings, and a committed file is writable by any agent in the run); the
planning interview rules between a user-level setting the repository may
only lower and a protected path only the person's own act changes. The
itd-19 stage axis names an intent, not an operator; solo v1 can run
in-session, host-delegated, per `script-first-mvp`; `pause` is a temporary
drop to human; the hard rule is enforced by the guard reading the setting,
as the intent already suggests. On "is merging irreversible": no opened
source states a position; that Renovate and the GitHub agent allow a merge
under required reviews and checks is consistent with treating a reviewed,
queued merge as a two-way door (revertable, re-validated by the queue), but
that reading is this pass's inference, and the question stays with the
product thinker, whose 2026-10-15 refinement reads the same way. No Go
dependency.

### Local fit risks

- `verifier-selects-gates-decide` and `evaluator-outside-the-loop`: auto
  mode's review pair stands in for the human approver; the challenger
  should decide whether a model's approval may satisfy a forge's required
  review, given GitHub's own reviewer does not count by default.
- Host-agnostic prose: the user-facing page describes "human" and "auto",
  never a host's mode names.
- `guards-prove-themselves`: the human-mode refusal needs a test that an
  agent cannot talk past it.

## itd-174: how far the facilitator is consulted, and when it escalates

### Findings

- **Both ends of the dial fail measurably.** In HiL-Bench's own words,
  "75–89% pass rates with complete information collapse to 4–24% when
  agents must decide whether to ask", and "no frontier model recovers more
  than a fraction of its full-information performance when deciding whether
  to ask"; Ask-F1 "captures the tension between over-asking and silent
  guessing". [EVIDENCE, vendor benchmark]
  [Trinh et al., HiL-Bench, arXiv 2604.09408](https://arxiv.org/html/2604.09408)
  In Edwards and Schuster, an uncertainty-aware multi-agent scaffold
  (UA-Multi) reached 69.40 per cent on underspecified tasks, against 70.80
  with the fully specified issue for the same model (p = 0.458, so not
  distinguishable) and 70.40 for that model's Interactive Baseline, a prompt
  that makes it "compulsory to query the user before proceeding with any
  execution"; the 47.20 figure is a second model's Interactive Baseline,
  which the paper attributes to "instruction-following failures, where the
  model erroneously used the task completion tool to ask questions". Neither
  baseline is an ask-at-every-step policy, so the source does not show that
  asking at every step costs accuracy. [EVIDENCE, preprint]
  [Edwards and Schuster, Ask or Assume?, arXiv 2603.26233](https://arxiv.org/html/2603.26233)
  ClarEval proposes "average turns to clarify" and "key question coverage"
  as the metrics. [EVIDENCE, abstract]
  [Li, Wu and Chang, arXiv 2603.00187](https://arxiv.org/abs/2603.00187)
- **The model's stated confidence is not the threshold.** Verbalised
  confidence has "a high calibration error" and is "overconfident most of
  the time"; its reliability "strongly depends on how the model is asked",
  and calibrated scores are possible "with certain prompt methods".
  [EVIDENCE; CONTESTED on usability]
  [Groot and Valdenegro-Toro, arXiv 2405.02917](https://arxiv.org/abs/2405.02917),
  [Yang, Tsai and Yamada, arXiv 2412.14737](https://arxiv.org/abs/2412.14737)
- **Practitioner guidance keys escalation on action class and failure
  count.** "Two primary triggers typically warrant human intervention":
  exceeding failure thresholds, and "Actions that are sensitive,
  irreversible, or have high stakes". [CONSENSUS]
  [OpenAI, A practical guide to building agents (PDF)](https://cdn.openai.com/business-guides-and-resources/a-practical-guide-to-building-agents.pdf)
  "Pause before side effects", "fail closed if review times out or becomes
  unavailable", serialise state and resume the same run. [CONSENSUS]
  [OpenAI, Guardrails and human review](https://developers.openai.com/api/docs/guides/agents/guardrails-approvals)
  LangGraph "waits indefinitely until you resume". [CONSENSUS]
  [LangGraph, Human-in-the-loop](https://docs.langchain.com/oss/python/langgraph/human-in-the-loop)
  Microsoft's standing approvals "persist in `AgentSession` state" and
  "aren't bound to the agent or tool instance that created them", a warning
  against a one-off answer becoming a standing rule unnoticed. [CONSENSUS]
  [Microsoft Agent Framework, tool approval](https://learn.microsoft.com/en-us/agent-framework/agents/tools/tool-approval)
- **Addressing by role, with timeouts, is incident practice**: an
  escalation policy is "an ordered set of rules", default timeout 30
  minutes, "acknowledgment does not resolve the incident" (from the
  autonomous loops pass). [EVIDENCE]
  [autonomous loops pass, itd-169 section](2026-10-23-autonomous-loops-sota.md)
- **Defaults are what ships.** "Less than 5% of the users we surveyed had
  changed any settings at all" (a consultant's own survey of several
  hundred word-processor users, 2011); "91% of participant routers run with
  default settings" (392 participants, CHI 2025). [ANECDOTE; EVIDENCE]
  [Spool, Do users change their settings?](https://archive.uie.com/brainsparks/2011/09/14/do-users-change-their-settings/),
  [Ye et al., CHI 2025](https://research.vu.nl/en/publications/understanding-home-router-configuration-habits-amp-attitudes/)
- **The approver decays** (Parasuraman and Manzey; Goddard et al., above);
  the EU AI Act's Article 14 makes awareness of "the possible tendency of
  automatically relying or over-relying on the output" a design duty for
  high-risk systems. [EVIDENCE, regulatory text; abcd is unlikely to be in
  scope] [Article 14, Human Oversight](https://artificialintelligenceact.eu/article/14/)
- **Auto-asking cannot be delegated to the model's inclination** (the
  under-clarification bias under itd-142).

### Options weighed

1. A confidence threshold ("ask when below 0.7"). The number is not
   portable across prompts or models.
2. A native policy shaped to the two-trigger guidance with a seam: the
   escalation rule is a small ordered list per repository (action class
   first: irreversible, outside the footprint, a new dependency, a
   product-thinker decision; then a stuck count; then, optionally, a
   calibrated uncertainty signal named per prompt template), the stop is
   the stop record, the product thinker answers in their own medium through
   the itd-90 path, the facilitator's answer is never silently promoted to a
   standing rule, and each repository logs ask precision and recall so the
   dial is tuned on data; the defaults are "consult at the gates, ask when
   stuck, escalate on action class", because defaults are what most
   repositories will run.
3. A fixed framework policy. The intent's 2026-08-29 ruling already asks
   for per-repository settings.

### Recommendation

**Path 2**, option 2. The four settings the ruling names map onto it:
machine or human is itd-97's mode; consult at each step or not is the
consult list; ask when unsure or not is the stuck threshold; how unsure is
the optional calibrated signal. Shipping that signal off by default demotes
a setting the ruling names, so it is a proposal for the product thinker to
confirm, not a given. No Go dependency.

### Local fit risks

- `one-canonical-primitive`: the escalation stop is the stop record, not a
  new interrupt type.
- `loud-staging`: an escalation that times out fails closed and says so;
  silence is never consent, as the GRILL rules already hold.
- Host-delegated: uncertainty signals come from the host's agent; the
  binary only records and routes.
- `facilitator-default-thinker-optional`: the product thinker is reached
  only through the facilitator's threshold, and in their register.

## Not worth adopting

- A recommended, starred or pre-selected option at a conjectural question
  (Spec Kit's `**Recommended:**`, grill-me's recommended answer); the
  default effect is large and works through endorsement.
- A question count as the gate (Spec Kit's five); gate on the record's
  unresolved fields.
- Letting the model decide whether to ask; measured under-asking.
- A readability formula (Flesch, FKGL, SMOG) as a merge or planning gate.
- RFC 2119 words as the "intended, not built" marker.
- Design fiction as a free-standing record type with its own review path,
  and the screenshot-caption sub-format.
- Lexical overlap as the consistency check between two capture formats.
- Voting arithmetic or Delphi-style percentage thresholds for a team of
  two to four thinkers; a single-approver framework as the whole record.
- A general model judge to confirm that each thinker's position survived
  collation.
- The model's self-reported confidence as the escalation threshold on its
  own; "ask at every step" as a quality lever.
- A third, unattended facilitator mode persistable in repository
  configuration.
- A prompt framework (huh, Bubble Tea) where one package covers raw mode;
  an `ACCESSIBLE` environment variable as the only switch; Unicode or
  braille spinners and glyphs in the fallback renderer; certifying badge
  contrast "for sixteen colours" without naming a palette.
- Scanning for "any interviewing skill"; treating skill marketplaces as
  vetted.

## Not verified

Believed to exist, not opened in this pass; no link is given for any of
them.

- ISO 24495-1:2023's four governing principles by name; the ISO catalogue
  and two resellers returned 403, and one federation page that should
  carry them had been replaced by unrelated content.
- Rob Fitzpatrick's own wording of the Mom Test rules; his sites confirm
  authorship and carry no rule text.
- The Working Backwards "five customer questions"; absent from both
  authors' pages opened.
- The GOV.UK "reading age of nine" figure; not on any current GDS page
  opened.
- Cochrane's 2022 plain-language-summary template; only the 2013 and 2017
  documents were readable.
- The Science article page for the deliberation trial (403); figures come
  from the magazine report and a PDF copy.
- A quantitative Delphi methodology review (round counts, consensus
  thresholds); Sociocracy For All's "paramount objection" page (403).
- Krosnick's no-opinion-option work and the 2010 choice-overload
  meta-analysis with a near-zero mean; both cited in summaries only.
- The Spoletini and Ferrari interview-review protocol result (ambiguities
  found in review against during the interview), and the Bano et al. full
  list of 34 interviewer mistakes.
- Blythe's CHI 2014 "imaginary abstracts" paper, Dunne and Raby's critical
  design FAQ, Pichler's product vision board, Dorst's primary articles,
  Schön's problem-setting passage.
- The two LLM-Gherkin studies (Karpurapu et al.; Rathnayake et al.) known
  only as cited inside the 2026 comparison.
- Codex CLI's approval-mode documentation; SAE J3016's level text; the
  2015 Amazon shareholder letter; Parasuraman, Sheridan and Wickens (2000)
  itself; the "learning to defer" and interruption-cost literature; any
  incident report of an auto-merged agent-authored pull request.
- The huh accessible-mode behaviour under `TERM=dumb` and its
  `ErrUserAborted` handling, seen only in a third-party snippet.
- The Okabe-Ito hex table; the authors' page gives one value.
