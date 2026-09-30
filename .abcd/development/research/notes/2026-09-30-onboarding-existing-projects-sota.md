# SOTA survey — how tools adopt an existing project, and where its non-standard files go

Dated 2026-09-30. The research ruling M35 (2026-09-23) put first: "research
how other tools onboard existing projects first (prefer-sota), then design
the placement interview, with iss-91 riding along as its narrower instance."
It serves iss-92. One host-run research pass over the tools' own
documentation (read on 2026-09-30), challenged for fit per
[`prefer-sota`](../../principles/prefer-sota.md).

**The question.** When abcd adopts a repository that already has its own
conventions — the case that raised iss-92 was a repository whose local tier
held a `WORKLOG.md`, a `DECISIONS.local.md` and a `SLICE_START` marker its own
runs depended on — how do comparable tools find the files that do not match
their canon, propose where each belongs, and let the person accept or
override?

## Seven adoption flows

| Tool | Entry | What it does with what is already there | Who decides |
| --- | --- | --- | --- |
| Renovate | an onboarding pull request, "Configure Renovate" | proposes a config tailored to what it detected, and previews the pull requests it would open; "will not make any changes to your repository or raise any further Pull Requests until after you merge" | the person, by editing the branch or merging; closing it stops Renovate, and the step is "reversible" |
| Copier | `copier copy` / `copier update` | a questionnaire whose previous answers become defaults, stored in `.copier-answers.yml`; conflicts with existing files land as inline markers or `.rej` files; `skip_if_exists` protects files the person owns | the person, question by question, and at the merge of each conflict |
| `nx init` | an interactive command | detects the workspace structure, writes one new config file and one dependency, and leaves scripts alone; its guide tells users not to pass `--interactive=false` because "the answers depend on how my repo is laid out" | the person, through prompts |
| Spec Kit | `specify init --here --force` | adds its own directory and integration files only; `--force` replaces files only "at conflicting managed paths", to be run "after establishing a reviewable baseline branch"; it "does not rewrite your application or infer specifications for existing behavior" | the team, afterwards, writing only rules "already true for the repository or that the team has explicitly agreed to adopt" |
| The first harness's `/init` | a command | reads other tools' instruction files (Cursor, Copilot, and with its new flow AGENTS.md, Windsurf, Cline) and folds the relevant parts into its own file; the new flow "explores your codebase with a subagent, fills in gaps via follow-up questions, and presents a reviewable proposal before writing any files"; an existing file is improved, not overwritten | the person, at the proposal |
| Biome | `biome migrate eslint` / `prettier` | reads the foreign config files and maps each setting; writes only with `--write`; declines to map "inspired" rules unless asked | the person, by running `--write` |
| Terraform | `import` blocks and `plan -generate-config-out` | produces "Terraform's best guess" as a template, recommended to be iterated — "removing some attributes, adjusting the value of others, and rearranging resource blocks into files and modules" — before apply | the person, by editing then applying |

## What the flows have in common

1. **Detect, then propose, then write.** Every flow separates a read-only
   detection from the write, and six of seven put a reviewable artefact (a
   pull request, a plan, a preview, a questionnaire with defaults) between
   them. The seventh (Spec Kit) avoids the question by writing only into its
   own namespace.
2. **A recommended default the person accepts or overrides.** Copier's
   questionnaire, Renovate's proposed config, Terraform's best guess and the
   new `/init`'s proposal are all defaults offered for override, not
   decisions taken.
3. **Leave what is not yours alone.** `nx init` leaves scripts; Spec Kit
   touches only managed paths; Copier's `skip_if_exists` never overwrites a
   file the person owns; Biome declines what it cannot map faithfully.
   No flow moves a foreign file into its own layout on its own authority.
4. **The answer is recorded so the next run does not re-ask.** Copier's
   answers file and Renovate's merged config are the durable record of the
   choice; a later update replays it.
5. **Reversible by construction.** A closed pull request, an unapplied plan,
   a preview without `--write`, a baseline branch.

**Where no established practice was found:** none of the seven proposes a
*placement* for an unrecognised file in the repository's own tree (move it,
fold it, keep it where it is, ignore it). The closest are Biome's report of
settings it would not map and `/init`'s fold of foreign instruction files
into its own. A per-file placement interview is not state of the art; it
would be a local composition of patterns 1, 2 and 4.

## Adversary filter

- **The user's directory is theirs**
  ([principle](../../principles/the-users-directory-is-theirs.md)). Pattern 3
  is the same stance; any placement interview whose default is "move" breaks
  it. The default for an unrecognised file must be "leave it where it is".
- **Verifier selects, gates decide**
  ([principle](../../principles/verifier-selects-gates-decide.md)). The
  proposal-then-accept shape matches: abcd recommends, the person's
  acceptance is the gate.
- **prepare-this-repo already has the skeleton.** Its Phase 2 is a read-only
  audit and its Phase 3 says "propose, then wait" for the legacy `.work/`
  migration, with "content-preserving (`git mv`, then fold)" moves. The gap
  is only that Phase 3 has no step for a member it does not recognise.
- **No new dependency, host-agnostic, script-first.** A questionnaire engine
  (Copier) or a pull-request bot (Renovate) is out of scope. The MVP is a
  documented step in the command page, whose answers a later verb can
  record.
- **Interview fatigue.** Copier and `nx init` ask per setting, not per file;
  a repository with dozens of foreign files would make a per-file interview
  long. Batching by kind (all unrecognised local-tier members at once, one
  default for the batch, per-file override) is the fit-preserving shape.

## Verdict (a proposal; the design is owed to a planning interview)

Adopt the common shape, not any one tool:

1. **Detect** every member of the adopted tiers that abcd's canon does not
   name (Phase 2, read-only), and classify each as *canon* (a known abcd
   file), *equivalent* (a file that plays a canon role under another name,
   such as a local decisions log beside `.abcd/work/DECISIONS.md`) or
   *foreign* (neither).
2. **Propose one disposition per file**, with the default **keep in place,
   untouched** for *foreign*, and **keep, and point to it from the canon
   file** for *equivalent*; *fold into the canon file* and *move* are offered
   only as overrides, content-preserving, and never as defaults. Present them
   in one batch the person accepts whole or edits line by line.
3. **Record the answers** in the adopted repository (the same role as
   Copier's answers file) so a re-run of adoption does not re-ask.
4. **Write nothing until the batch is accepted**, and make the write one
   reviewable change.

The three iss-91 members (`WORKLOG.md`, `DECISIONS.local.md`, `SLICE_START`)
would each come out *keep in place*, with `DECISIONS.local.md` offered the
*equivalent* pointer. What would show this wrong: an adoption where
keep-in-place leaves a file the repository's own runs can no longer find, or
a person who overrides the default on most files (the default is then the
wrong one).

## Review record

2026-09-30: authored in one pass by an implementer in autonomous run A (lane
drainResearch), with the fit-challenge run in-pass by the author. No
independent adversarial reviewer has read this note yet; per the
[research protocol](2026-08-22-sota-research-protocol.md) that review is owed
before the planning interview relies on it.

## Sources (all accessed 2026-09-30)

- [Renovate — Installing and onboarding](https://docs.renovatebot.com/getting-started/installing-onboarding/)
- [Copier — Updating a project](https://copier.readthedocs.io/en/stable/updating/)
- [Nx — Start with an existing project](https://nx.dev/docs/getting-started/start-with-existing-project)
- [Spec Kit — Existing projects guide](https://github.github.io/spec-kit/guides/existing-projects.html)
- [Claude Code — How Claude remembers your project (`/init`, migrating instructions from other tools)](https://code.claude.com/docs/en/memory)
- [Biome — Migrate from ESLint and Prettier](https://biomejs.dev/guides/migrate-eslint-prettier/)
- [Terraform — Generating configuration for imported resources](https://developer.hashicorp.com/terraform/language/import/generating-configuration)
