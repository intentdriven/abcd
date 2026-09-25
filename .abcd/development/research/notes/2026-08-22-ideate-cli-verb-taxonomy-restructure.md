# Ideate verdict — cli-verb-taxonomy-restructure

**Verdict: reframed.** Recorded on 2026-08-22 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

abcd's top-level verb list keeps growing and risks becoming unmanageable, so the verbs should be regrouped into a noun-verb hierarchy — for example `abcd banlist` becoming `abcd quality banlist`, and `memory` becoming a category owning adding, retrieving and local-sources verbs — with the CLI command tree decoupled from the flat plugin command surface so the CLI can nest while the plugin surface stays flat.

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation.

| Claim | Primary source | Finding |
|---|---|---|
| clig.dev, the Azure CLI command guidelines, the Heroku CLI style guide and the Nix CLI guideline state no numeric threshold for the number of top-level commands at which a CLI should restructure. | https://clig.dev/ | verified |
| Docker's 2017 management-command restructure was triggered by semantic ambiguity — commands such as `docker inspect` did not say whether they operated on an image or a container — rather than by the number of commands. | https://www.infoq.com/news/2017/01/docker-1.13 | verified |
| The GitHub CLI ships roughly 32 top-level commands in a flat list, with no grouping into management commands. | https://cli.github.com/manual/ | verified |
| Cobra provides first-class presentational command groups via AddGroup on the root and GroupID on each subcommand, so root help renders labelled sections without any invocation changing. | https://pkg.go.dev/github.com/spf13/cobra#Command | verified |
| Docker retained every pre-restructure spelling: `docker ps` remains co-equal with `docker container ls` nine years after the management commands landed. | https://docs.docker.com/reference/cli/docker/container/ls/ | verified |
| Terragrunt's CLI reorganisation required an RFC, a published old-to-new mapping table, deprecation warnings and opt-in strict controls. | https://docs.terragrunt.com/migrate/cli-redesign/ | verified |
| Azure CLI's published command guidelines warn against creating command subgroups that contain no commands of their own. | https://github.com/Azure/azure-cli/blob/dev/doc/command_guidelines.md | verified |
| The cobra version this repository pins (v1.10.2) exposes AddGroup and GroupID, so command groups require no dependency change and no new-dependency approval. | go.mod, this repository | verified |
| abcd already applies noun-verb nesting wherever the semantic-ambiguity trigger fires: the lint-shaped acts are disambiguated by their object as `abcd lint`, `abcd docs lint` and `abcd memory lint`, fifteen of the twenty-two top-level verbs already own sub-commands, and the committed command tree holds seventy-three command paths reaching depth three. | .abcd/development/release/surface.json and internal/surface/cli/cli.go, this repository | verified |
| A noun-verb command hierarchy measurably improves an AI agent's ability to explore a CLI's help output. | https://dev.to/uenyioha/writing-cli-tools-that-ai-agents-actually-want-to-use-39no | unverifiable |
| GitHub's Primer CLI design guide records the original rationale for the GitHub CLI's noun-verb surface design. | https://primer.style/cli — archived, not retrievable in this research pass | unverifiable |
| Twenty-two top-level commands sits below every observed restructuring point in the field. | research-pass coverage record: only Docker and gh had their surface history opened; gcloud, stripe, fly, deno, cargo and git were not examined, so the comparator set is two | unverifiable |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| iss-161 | contradicted | Nesting the markdown command surface is structurally refused: a harness maps each commands/ subdirectory to an extra namespace segment, so a nested verb registers as /abcd:abcd:< verb>. This constrains the plugin surface only, which is what the decoupling half of the idea rests on. 04-surfaces/README.md records the flat layout as load-bearing rather than tidiness, and realSurfaces in internal/core/lint/lint.go counts only files directly under commands/. The surfaces are already not one-to-one in both directions, so decoupling as such is precedented; what is new is two contradictory spellings of one act, against 109 fenced binary invocations and 18 argument-hint lines under commands/. |
| adr-40 | covered | Rules pre-1.0 renames as clean breaks with no aliases, which forecloses the aliased migration path both external precedents relied on. Decision 6's sub-verb tables already impose a naming test on assessment surfaces, and the ADR explicitly rejected the two nearest alternatives: one polymorphic verb over targets, and classifying acts rather than surfaces. Its bucket-to-role mapping is what a category noun spanning several buckets would blur. |
| iss-284 | superseded | The semantic-ambiguity event adr-40 recorded has already been answered, and answered without a restructure: the renames shipped in v0.6.0, taking four command paths off the surface (abcd audit to abcd lint, disembark oracle to disembark review, intent review and intent review ingest to intent audit and intent audit ingest), each successor refusing the old spelling by naming the new one. This falsifies the premise that a mandated breaking cut is still pending and available to bundle a wider restructure into. |
| adr-41 | contradicted | The repo-tier memory store and the user-tier sources corpus sit on opposite sides of a recorded trust boundary, so folding them under one memory category noun collides with it. |
| iss-246 | covered | The record once documented a surface substantially larger than the shipped one; the surface_coverage sub-verb pass (spc-27, blocker severity, internal/core/lint/subverbs.go) is the detector that closed it. That pass joins the brief surface file, commands/< verb>.md and the cobra snapshot on one shared top-level name, so a regroup either rewrites the gate or defeats it by declaring a user-facing category operator-internal. |
| itd-99 | covered | Commits abcd to never handing a product thinker a technical decision. A category noun filing a name store beside lint surfaces puts a product-thinker door and implementation-team doors behind one label, which is the guarantee adr-40 decision 6 made mechanically checkable. |
| iss-171 | covered | Establishes --impact breaking as live in release derivation, so a breaking cut is procedurally supported; the constraint on a restructure is not the release machinery but the no-alias rule and the unreachable installed artefacts. |
| itd-84 | covered | The decompose-before-filing discipline governs how this verdict's follow-on work splits across record homes; the routing hand-run is graded into the dated decomposition-calibration note. |
| itd-104 | covered | The idea-admission gauntlet this verdict is recorded through. Its leg-3 conduct requirements shaped the dispatch: two independent evaluators with disjoint targets, neither told the other existed, neither given authorship or the earlier legs' narrative. |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.

- **fatal** — Against the restructure: it has no trigger, and its whole benefit is already free. Docker's documented trigger was semantic ambiguity, not count, and abcd already applies noun-verb exactly where ambiguity fires — abcd lint, docs lint, memory lint, intent audit, site build, banlist add. No verb takes an argument whose type is ambiguous, so no docker inspect analogue remains. Cobra v1.10.2 is already pinned, so AddGroup and GroupID deliver the entire organisational benefit at zero breakage and zero record churn.
- **fatal** — Against the restructure: the repo's own rules make a safe migration structurally impossible. adr-40 forbids aliases pre-1.0, and .abcd/docs-lint.json makes present_tense/renamed-from, present_tense/previously and present_tense/formerly blocker-severity banned tokens, so the old-to-new mapping table the Terragrunt precedent depends on cannot be published. Worse, internal/core/ahoy/defaults/pre-commit is scaffolded by ahoy install into managed repos and hardcodes `abcd banlist add --private` and `abcd banlist list --private`; those files sit committed in third-party repos abcd cannot reach, and no migration verb exists that could.
- **partial** — Against the restructure: it breaks a blocker-severity gate that landed days ago. surface_coverage's sub-verb pass joins the brief surface file, commands/< verb>.md and the cobra snapshot on one shared top-level name. Moving banlist under quality fires both directions of the check, and the only escapes are to declare the user's front door operator-internal — a documented falsehood that also switches the check off for the whole subtree — or to mark a shipped group as a design target.
- **partial** — Against the restructure: half the proposition is already the shipped state, so it misdescribes its subject. memory ingest, memory ask and memory lint ship today; fifteen of twenty-two verbs own sub-commands and two paths already reach depth three. The proposal is therefore not to introduce nesting but to add a fourth level of invented meta-nouns, yielding `abcd quality banlist add --private < key> < pattern>`. No comparator in the evidence puts a leaf action four segments deep.
- **partial** — Against the restructure: `quality` is an unlicensed name in a closed, PR-to-extend vocabulary. 02-constraints/04-naming.md requires either a maritime cognate that adds meaning or a place on the explicit exemptions list with a stated reason. The proposition also invents a new class of name — the grouping noun — that no rule in the register covers. Its bare-command-as-render discipline compounds this: a category noun has no state, so `abcd quality` bare renders nothing, which is the Azure warning against empty subgroups restated as a local MUST.
- **partial** — Against the restructure: nouns cannot host abcd's top level, because it is a voyage sequence rather than a resource set. ahoy, disembark, embark, launch, dredge and loot are pinned to a maritime act sequence with no owning resource; `abcd lifeboat disembark pack < repo> < dest>` is five segments and reads worse than what ships. Docker and kubectl nouns work because their domains have resources; abcd's resources are records, and each already has its verb owning its sub-verbs.
- **partial** — Against the restructure: a quality grouping crosses adr-40's buckets. banlist's add, list and remove are recorded with bucket em-dash — it is a name store, not an assessment — so filing it beside lint surfaces mis-signals its bucket at the loudest point of the name and puts two different roles' doors behind one category.
- **partial** — Against the restructure: decoupling deletes the join key that makes the two surfaces auditable. The three-way name join is the whole mechanism; decouple, and commands/banlist.md documents /abcd:banlist while executing `abcd quality banlist`, permanently, with the docs forbidden from explaining the divergence.
- **partial** — Against the restructure: the blast radius is roughly 1,900 in-repo spellings, and the retire-the-name principle cannot be satisfied for them. That principle requires the retiring change to add each old spelling to banned_tokens, but the old spelling stays valid as /abcd:banlist under the proposition's own decoupling and persists legitimately in the CHANGELOG, the issue ledger and shipped ADRs, so the ban is either un-addable or drowns in allow_context escapes.
- **partial** — Against the restructure: renaming any hook-invoked verb path silently disables the guard. hooks.json runs `guard hook` and accepts exit 0/1/2 as a verdict; cobra returns exit 1 on an unknown command, which the guard contract maps to warn, non-blocking. A hooks.json updated against a binary predating the rename therefore yields a silent fake warn on every shell command, and --allow-stale-binary proves plugin/binary skew is a tolerated live state.
- **survived** — Against the restructure: the docs cost alone is prohibitive. Tried and failed — docs/reference/cli/commands.md is generated by cmd/abcd-gen-cli-ref with a drift test, and the surface snapshot by cmd/abcd-gen-surface; both regenerate from the tree, so the generated-docs cost is near zero.
- **survived** — Against the restructure: one-canonical-primitive forbids two spellings of one act. Tried and failed — the principle's own Bounds section restricts it to infrastructure primitives such as atomic write, frontmatter parse and path expansion, and explicitly says it covers infrastructure primitives rather than domain logic. It does not reach surface naming.
- **survived** — Against the restructure: the two surfaces can never be decoupled. Tried and failed — they already are, in both directions and by explicit config, with host_delegated and operator_internal lists. Decoupling as a principle is precedented; only contradictory spellings of the same act would be new.
- **survived** — Against the restructure: there is no semantic-ambiguity foothold at all. Tried and failed — three lint-shaped surfaces exist alongside a separate record-lint binary and make target, which is a genuine near-collision. But it is already resolved the correct way, by nesting each under its object, so the foothold exists and is already occupied.
- **fatal** — Against the defer-and-group alternative: hiding rules and spec contradicts the binary's own recorded definition of hidden, which turns on no user ever reading about the command. AGENTS.md and the ahoy-installed marker block both carry the line instructing the reader to inspect rules with `abcd rules`, and that block is embedded and written into every managed repo, so the binary would ship instructions naming a verb its own help denies exists.
- **fatal** — Against the defer-and-group alternative: hiding deletes published documentation and the release guardrail is built not to notice. The reference generator returns early on a hidden command, so regenerating the CLI reference deletes its rules and spec sections, while the surface break taxonomy states that a command becoming hidden is deliberately not a break — so the removal passes the ship guardrail silently, for a feature the terminology reference advertises as shipped.
- **fatal** — Against the defer-and-group alternative: spec is a documented workflow step, not plumbing. The process-coherence walkthrough instructs the reader to close the work with `abcd spec close spc-N`, and hiding also strips it from shell completion. The category error underneath is that operator_internal in record-lint.json means needs no surface chapter — a lint exemption, not a user-visibility verdict.
- **fatal** — Against the defer-and-group alternative: the deferral trigger it names has already fired on the record. adr-40 is exactly the semantic-ambiguity event, and the future collision is scheduled to sharpen, with /abcd:audit reserved for hash-chain verification carrying audit chain and audit lifeboat sub-verbs while intent audit ships. Deferring until confusable pairs appear is a condition satisfied today, on file, with an ADR number.
- **fatal** — Against the defer-and-group alternative: the deferral condition names no detector, which this repo treats as a defect class. fix-the-detector makes the detector the unit of fix, and enforcement-claims-are-facts holds that a phantom gate is worse than no gate because readers who believe a check exists stop compensating. A deferral with no detector defers indefinitely by construction.
- **fatal** — Against the defer-and-group alternative: its own parts contradict each other. Part one promises that every invocation stays exactly as it is; part three removes `abcd changelog`, which the surface diff taxonomy classes as command_removed, the most severe kind, forcing breaking version derivation and edits across the launch command page, the composer agent, the surface registry row and two plans. The package is sold as no-invocation-change housekeeping and is in fact a breaking release.
- **survived** — Against the defer-and-group alternative: since folding changelog is breaking anyway, the pre-1.0 clean-break budget would be better spent on a wider restructure bundled into the same cut, and that budget is a wasting asset. Tried and failed — the premise is falsified by iss-284: the adr-40 renames already shipped in v0.6.0, taking four command paths off the surface, so there is no pending mandated breaking cut to bundle a restructure into.
- **partial** — Against the defer-and-group alternative: folding changelog under launch violates the same standard the package applies to every other verb — no confusable pair drove it, only tidiness — and it hangs a sub-verb off a parent already out of compliance with the bare-renders-status discipline, since bare `abcd launch` only hints to pass --dry-run.
- **partial** — Against the defer-and-group alternative: command groups ship a convention with no detector, in a repo that gates everything else. The surface snapshot records path, hidden and flags but no group; the diff taxonomy has no group kind; the reference generator emits no groups, so the CLI reference stays a flat alphabetical list while help becomes sectioned — two user-facing surfaces disagreeing about the shape of the CLI, with a verb missing a GroupID falling silently into Additional Commands.
- **partial** — Against the defer-and-group alternative: deferring has a per-verb record cost, since every verb added meanwhile costs a surface chapter, a sub-verb table row, a snapshot entry, a command page and a reference section, with five reserved verbs queued. Lands, but not fatally — the per-verb cost is roughly layout-independent and moving table rows is mechanical.
- **partial** — Against the defer-and-group alternative: the agent-audience premise cuts against hiding. An agent handed the injected marker block naming `abcd rules` and a help listing without it has a contradiction it can only resolve by guessing, so hiding optimises a human's first-screen scan by degrading the reader the proposition itself calls dominant.
- **survived** — Against the defer-and-group alternative: the plugin surface should mirror a noun-verb CLI tree instead. Tried and failed — the markdown surface structurally cannot express a hierarchy, and the surfaces are already not one-to-one, so a CLI restructure would widen a divergence the harness forces regardless.
- **survived** — Against the defer-and-group alternative: command groups will break a gate or a golden test. Tried and failed — no test asserts root help text, the reference generator ignores groups so its drift test is unaffected, and the snapshot records no group so its drift test is unaffected. Groups are genuinely near-free to land.
- **survived** — Against the defer-and-group alternative: twenty-two top-level commands is over a threshold. Tried and failed — the help output is 22 commands, alphabetical, one screen, each with a distinct one-line summary; no published guide names a number, gh ships about 32 flat, kubectl is comparable. There is no demonstrable legibility failure to point at today.
- **survived** — Against the defer-and-group alternative: hiding breaks surface_coverage or another record-lint gate. Tried and failed — record-lint.json already exempts spec, rules, hook and completion, and the sub-verb loader already drops hidden subtrees, so hiding changes no lint outcome. That is precisely why the real damage falls on documentation, completion and the installed marker block, none of which any gate watches.

## Rejected alternatives

- **Regroup the top-level verbs into a noun-verb hierarchy now, with category nouns such as `quality` and a `memory` category, and the CLI tree decoupled from the plugin surface.** — Two independent fatal objections. The organisational benefit is already available free from cobra's AddGroup, and the semantic-ambiguity trigger that drove the comparable Docker restructure does not exist here, because abcd already nests under the object wherever ambiguity fires. Separately the migration is un-executable under abcd's own rules: adr-40 forbids aliases pre-1.0, the docs-lint blocker bans the change-narration tokens a mapping table needs, and the ahoy-scaffolded pre-commit file hardcodes `abcd banlist add` inside managed repos abcd cannot reach.
- **Hide the operator-internal verbs `rules` and `spec` from the default help listing.** — Fails three ways against the record. `rules` is named in the ahoy-installed AGENTS.md marker block, so help would deny a verb the install instructions tell the reader to type; hiding deletes both verbs' sections from the generated CLI reference while the surface break taxonomy deliberately does not class hiding as a break, so the deletion passes the release guardrail silently; and `spec close` is a documented process-board step whose shell completion would disappear. The operator_internal list in record-lint.json is a lint exemption meaning needs no surface chapter, not a user-visibility verdict.
- **Defer any restructure until confusable verb pairs appear in practice.** — The condition is already satisfied on the record — adr-40 is that event — and the deferral names no detector, which fix-the-detector and enforcement-claims-are-facts class as a phantom gate that defers indefinitely by construction. The reframing replaces it with a stated, checkable design-time test rather than an open-ended wait.
- **Fold `changelog` under `launch` as a standalone tidy-up.** — It is command_removed, the most severe kind in the surface break taxonomy, so it cannot ride inside a package presented as changing no invocations; and no confusable pair drove it, which is the standard the same package applies to every other verb. Its earlier justification — bundling it into adr-40's pending breaking cut — is falsified by iss-284, since that cut already shipped in v0.6.0.
- **Heroku-style colon topics, such as `abcd banlist:add`.** — An oclif-ecosystem convention, alien to a cobra binary and worse for shell completion; it carries the same vocabulary cost as nesting with none of the tooling support.
- **A gh-style extension or plugin verb system as the growth valve.** — That model exists to absorb third-party command authors. abcd has none, so it is machinery cost with no corresponding growth to absorb; it stays available if outside contributors later want verbs not wanted in-tree.

## What follows

The idea as posed does not survive, but the reframing recorded above
does. Any graduation to a draft intent carries the reframing, not the
original wording. It graduated as [itd-146](../../intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command-groups-and-the-group.md).

### The reframing, stated

The surface is organised by **presentation, not by hierarchy**. abcd's top
level is a sequence of acts rather than a set of resources, and the
noun-verb model is already applied wherever a resource genuinely owns
several verbs: sixteen of the twenty-one registered verbs carry
sub-commands across seventy-three command paths, and the tree reaches
depth three. What the growing list costs is
legibility in one screen of help output, and that is a rendering problem
with a rendering answer: labelled cobra command groups, every invocation
unchanged.

Three things the reframing rules out, each for a reason recorded above:
no category nouns (`quality` and its kind are unlicensed by the naming
register, carry no bare-invocation state, and cross adr-40's buckets); no
hiding of `rules` or `spec` (the installed marker block names `rules`, and
the release break taxonomy cannot see the documentation those verbs
lose); and no open-ended deferral (a condition with no detector is the
phantom gate `enforcement-claims-are-facts` forbids).

What replaces the deferral is a **design-time test at the point a verb is
proposed**: does an existing record-owning verb already own this act? The
test is answerable when a verb is designed, by the human designing it,
which is what a trigger conditioned on future confusion is not.

### The one gap the reframing inherits

Command groups are themselves a convention with no detector. The surface
snapshot records `path`, `hidden` and `flags` and no group; the break
taxonomy has no group kind; the reference generator emits no groups. Ship
groups alone and `--help` becomes sectioned while
`docs/reference/cli/commands.md` stays flat — two user-facing surfaces
disagreeing about the shape of the CLI, with a verb that omits its
`GroupID` falling silently into cobra's "Additional Commands". The
detector travels with the grouping or the grouping repeats the defect it
was chosen over.

### Coverage caveat

Three of leg 1's twelve claims are marked unverifiable, and they are
unverifiable for three different reasons: one names a source that could not
be retrieved, one rests on a single practitioner essay carrying no
measurement, and one records the limit of the comparator set. Only Docker
and gh had their surface history examined, so "below every observed
restructuring point" is an inference from two data points rather than a
survey. The load-bearing finding, that abcd already nests where ambiguity
fires, rests on this repository's own files and is independent of that gap.
The second load-bearing finding, that the migration is un-executable under
abcd's own rules, is weaker than leg 3 stated it: the change-narration bans
carry a documented escape and apply only to the `docs` and `README.md`
roots, so the true obstacle is the no-alias rule plus the scaffolded
artefacts abcd cannot reach, not the docs lint.

## Errata (2026-08-23)

An adversarial review of the records this verdict produced re-checked the
verdict itself and found six defects in the legs above. They are corrected here
rather than in place: `abcd ideate record` refuses a second record under the
same date and slug, and rewriting a leg would misrepresent what the gauntlet
actually produced. Each correction below is verified against this repository.

**The verdict is unchanged, and two corrections strengthen it.** The reframing
stands: no restructure, presentational grouping, a detector attached.

1. **Leg 1, the ninth claim, marked `verified`, is wrong in its denominator.**
   "Fifteen of the twenty-two top-level verbs already own sub-commands" counts
   sub-command owners excluding cobra's generated commands over a total that
   includes them. The tree holds twenty-one registered verbs, sixteen of which
   own sub-commands; `abcd --help` prints twenty-two lines because cobra adds
   `help` and `completion` and the hidden `hook` does not render. So the visible
   surface is twenty verbs. The claim's conclusion holds more strongly at the
   corrected ratio, which is the reason this went unnoticed. The wider lesson is
   the important one: `ideate record` proves every cited id resolves, and proves
   nothing about whether a claim is true. A `verified` mark is a claim about the
   author's diligence, not an assertion the binary checked.
2. **Leg 2's `iss-246` note repeats a framing that issue itself retracts.**
   "The record once documented a surface substantially larger than the shipped
   one" is the wording iss-246's closing paragraph supersedes: the detector
   already existed as `surface_coverage`, the brief is aspirational by design
   per adr-5, and the real defect was narrower, namely blindness inside a
   surface row. The corrected finding still supports the conclusion drawn from
   it, because a detector blind at the grain of a claim is blind at that grain.
3. **Leg 3's tenth kill attempt, on the guard hook, is false as stated.** An
   unrecognised sub-verb exits 2, not 1. Exit 1 on the hook plane is abcd's own
   deliberate fail-open, installed by `applyHookPlaneFailOpen` on seven named
   paths, with `guard check` excluded precisely to keep the opposite contract.
   The refusal is not silent: `hookPlaneSkewNote` prints a message naming plugin
   and binary skew and the command that repairs it. The attempt's supporting
   claim about `--allow-stale-binary` is also misread: that flag concerns a
   binary against its own source checkout under itd-111 staleness, and it
   refuses by default, so it demonstrates the opposite posture to the one
   claimed. The ground is already held by resolved `iss-267` and `iss-269`, the
   latter recording the top-level against sub-verb asymmetry as a known open
   question. The attempt's outcome tag of `partial` should be read as not
   landing at all.
4. **"The most severe break kind" is unsupportable.** The break taxonomy
   defines four kinds with no severity field and no ordering, and its only
   comparative remark points elsewhere. The phrase appears twice in leg 3 and
   once in the rejected alternatives; read it as "a removal, which the taxonomy
   does report" and nothing more.
5. **Leg 3 undercounts the depth-three paths.** Three reach depth three:
   `docs cite confirm`, `docs cite refresh` and `intent audit ingest`.
6. **Leg 3 overstates the docs-lint obstacle.** The banned change-narration
   tokens carry a documented `allow_context` escape and their roots are `docs`
   and `README.md` only, so a migration table is publishable. The un-executable
   half of the migration argument rests on the no-alias rule and on the
   `ahoy`-scaffolded artefacts inside managed repositories abcd cannot reach.

**What this errata changed downstream.** One intent drafted from this verdict
was withdrawn entirely, its central evidence resting on correction 3; a capture
of that same guard finding was withdrawn with it. The surviving intent, itd-146,
carries the corrected numbers and the corrected `iss-246` reading.
