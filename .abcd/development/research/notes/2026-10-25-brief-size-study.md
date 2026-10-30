# Is a brief this size required? A single-source-of-truth study of `.abcd/development/brief/`

Date: 2026-10-25. Tree: origin/main `2e03c61e4`. Asked for by the product thinker during the
low-CPU run of 2026-10-25. The readers' raw outputs (the brief they were given, the section
groups, one classification file per reader and the shingle comparison) were kept on the machine
that ran the study, not in the record; this note is their synthesis. It is the evidence for
adr-2610290238168123.

**Outcome.** The product thinker ruled all three on 2026-10-27: R1 answer A, the command page is the
home of behaviour (dcn-2610271112339516); R2 and R3 as recommended, under the standing rule that
everything has one home (dcn-2610271112331508).

## The answer

**Not at this size.** The brief holds about 247,000 words over 116 pages. Thirteen
read-only readers classified every section against the files that state the same thing.
They put about **170,000 words (69%) as removable**: restated elsewhere (and replaceable by
a pointer), history that ADRs and git already hold, or stale text that is wrong today.
Only about 31,000 words (13%) are the brief's own: rationale held nowhere else (13,600) and
current behaviour no other prose file states (17,600).

The size is not a style problem. It is a second copy problem. **Verb behaviour has two
hand-maintained homes:**

- the brief's surface chapters (`04-surfaces/`, 159,000 words, 76% removable);
- the command pages (`commands/*.md`, 132,000 words).

No rule says which one is the source of truth. They have drifted apart in both directions.

## Two tests

1. **Verbatim (cheap, mechanical).** Ten-word shingles of every brief page were compared
   with `commands/`, `docs/`, the ADRs, `agents/`, the principles, the intents and the root
   files. Only **15%** of the brief repeats another file word for word (4% another brief
   page, 2% a command page), so the duplication is not copy-paste.
2. **Paraphrase (13 readers, one or more chapters each).** Each section was classified as
   restates, conflicts, unique-rationale, unique-behaviour, history or stale, with its
   counterpart cited as `path:line`. This found the duplication the shingles cannot:

| Class | Words | Share |
|---|---|---|
| restates another file | 165,742 | 69% |
| unique behaviour | 17,558 | 7% |
| unique rationale | 13,604 | 6% |
| conflicts with a counterpart or the code | 10,649 | 4% |
| stale (the code moved) | 8,356 | 3% |
| history (belongs to git or an ADR, per adr-5) | 7,359 | 3% |
| not assessed (security) | 18,018 | 7% |

By folder:

| Folder | Pages | Words | Removable |
|---|---|---|---|
| 04-surfaces | 35 | 159,219 | 121,603 (76%) |
| 05-internals | 11 | 35,020 | 19,743 (56%) |
| glossary | 47 | 19,690 | 6,483 (33%) |
| 02-constraints | 5 | 12,367 | 7,417 (60%) |
| 06-delivery | 4 | 10,163 | 8,825 (87%) |
| 01-product | 7 | 6,637 | 4,535 (68%) |
| 03-evidence | 5 | 2,332 | 1,116 (48%) |
| README and 00-meta | 2 | 1,884 | 622 (33%) |

Largest single cuts: `05-intent.md` 14,030 of 17,296; `01-ahoy.md` 12,150 of 14,737;
`34-build.md` 8,866 of 10,269; `04-launch.md` 8,838 of 10,510; `03-configuration.md`
7,000 of 13,860; `35-drain.md` 6,774 of 7,973; `17-guard.md` 6,677 of 10,018;
`06-capture.md` 6,460 of 7,045; `08-abcd.md` 6,260 of 6,728; `27-implement.md` 5,765 of
6,641; `06-delivery/02-verification-matrix.md` 4,532 of 5,295.

## The structural finding

Every surface reader reached the same conclusion independently. These rules make the
chapter the home for behaviour:

- "a chapter is where a claim about behaviour belongs" (`04-surfaces/README.md:3-7`);
- adr-2609231028044006:102: behavioural claims stay hand-maintained in the chapter;
- invariant 18 (`02-constraints/03-invariants.md:173`);
- the generated appendix's "behaviour is the prose's to state" (`internal/core/surface/appendix.go:54`).

**None of them says anything about the command pages**, which state the same behaviour
in full, for the agents that actually run the verbs.

The two homes are not equal:

- **The brief chapter is the gated copy.** The doc-fidelity gate (`04-surfaces/10-docs.md:66-82`,
  `commands/docs.md:92`) checks it against the code, and a false sentence there holds a
  spec close or a release cut.
- **The command page is the operative copy, and nothing checks it.** Agents load and
  follow it. Several readers found it wrong where the brief was right (for example
  `commands/intent.md:232`, `commands/launch.md:533`).
- **Equally often the brief is the stale one** (the ahoy uninstall and re-founding passages,
  `34-build.md` on hooks and validators, `17-guard.md`'s carried-value example, the
  superseded itd-20, itd-27, itd-33 and itd-72 designs still staged).

Two copies maintained by hand, with a gate on only one, drift by construction.

## Rulings needed (recommended answers first among the options, as reasons)

### R1. Which file is the single home of a verb's behaviour?

The thing being decided: today `04-surfaces/<nn>-<verb>.md` and `commands/<verb>.md`
both state each verb's behaviour (refusals, exit codes, outputs, sequences) in full prose,
and the fidelity gate checks only the first.

- **A. The command page is the home.** The chapter keeps why the surface exists, the
  trade-offs, what is refused and why, its `## Sub-verbs` table and its generated appendix,
  and points to the command page for behaviour. The doc-fidelity gate moves to the command
  pages (and `docs/`).
- **B. The chapter is the home.** The command page keeps only the agent's operating steps
  and points to the chapter for behaviour.
- **C. Keep both, and add a gate** that reads the two against each other.
- **Decide later.**

Recommended: **A.** The command page is the one agents execute, so making it the gated
copy puts the check where a falsehood does harm. It is already the fuller copy for most
verbs (`commands/reading.md`, `commands/capture.md`, `commands/implement.md` cover more
than their chapters). And B would strip the page an agent loads of what it needs to act.
C doubles the maintenance the readers measured instead of removing it. A reverses the
behaviour half of adr-2609231028044006 and invariant 18, so it needs a superseding ADR,
not a silent edit.

### R2. What happens to the other brief-wide copies?

These need no new rule beyond adr-5 ("history lives in git; inflection rationale in
ADRs"), so the recommended answers follow from it:

- **Invariants** (`02-constraints/03-invariants.md`): 21 of 24 restate their ADR at
  paragraph length. Reduce each to one line naming the rule and its ADR.
- **The verification matrix** (`06-delivery/02-verification-matrix.md`, 4,532 of 5,295
  removable): its rows paraphrase command pages and name no test. Generate it from the
  tests, or remove it.
- **Out of scope** (`06-delivery/03-out-of-scope.md`): mostly a generated copy of the
  draft intents' titles (the shingle pass found 38% verbatim). Generate it, or point to the
  drafts shelf.
- **The machine-folder inventory**, kept twice (`01-ahoy.md:419-522` and
  `03-configuration.md:621-692`, each saying the two are one list): keep one copy.
- **Superseded designs still staged** (itd-20, itd-27, itd-33, itd-72, the grill and PRD
  material, the retired phase vocabulary): delete them, since git and the ADRs keep the
  history.

### R3. Does the glossary stay as it is?

The glossary is the least duplicated folder (33%). It is the canonical terminology store,
and most of its 47 pages are their own home. Recommended: keep it, and fix the 17 entries
S11 found out of date or contradictory (spec, record, voyage, loop, transport,
reading-position, and the retired phase, roadmap and milestone bodies).

## What a smaller brief would hold

On the readers' figures, the brief after R1 and R2 is about 70,000 to 80,000 words.

- **Kept:** the product chapters, the constraints as one-line rules with their ADRs, the
  glossary, and each surface chapter's purpose, trade-offs and refusal reasons, with its
  `## Sub-verbs` table and generated appendix.
- **Reached by a pointer:** everything else, now held in the command pages, the ADRs, the
  code's doc comments and git.

That fits one reader's context window, where today's corpus needs 66 chunks for a single
consistency pass (the 2026-10-26 pass over the brief and every intent). It also cuts the second copy that causes most of the
`inconsistency` findings the ledger keeps receiving.

## Findings

The readers recorded **125 findings** (117 minor, 8 nitpick). These are conflicts and stale
passages in the brief, in command pages and in a few code comments. About 15 are already
held by open records, and the readers named those. The rest were filed with `abcd capture` on
2026-10-25, deduplicated against the ledger. Security sections (122) were counted and not
assessed.

## Limits of the measure

- "Removable" is each reader's estimate, section by section. It counts conflict and stale
  text, which should be fixed at its single home rather than simply deleted.
- It does not count the generated appendices, which adr-2609231028044006 requires.
- A section the reader called "restates" was checked against a named counterpart, but
  nobody re-read the counterpart to confirm that every clause survives. R1's change should
  move the few unique sentences (each reader named them) before it deletes anything.
