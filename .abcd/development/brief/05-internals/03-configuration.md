# Configuration Model

Almost nothing here is a decision anyone has to make. Installing abcd asks four
questions, records the answers, and gets on with it; four further keys can be
hand-set and are read but never written. That is the whole of the configuration
surface the shipped binary consults. Everything else on this page is either a
store abcd lays out for itself, a policy that follows from one of those four
answers, or an axis the design commits to and has not built.

**Read the schema as two lists.** The keys the binary reads come first and are
shipped behaviour. Everything after them is **staged**, per the truth rule in
[`../00-meta.md`](../00-meta.md#the-truth-rule).

## `.abcd/config.json`

### The keys the binary reads

Nine keys plus a `meta` block. Install asks about visibility, the docs target,
the oracle backend, and the deep scan (the last only when visibility is private
and the deep scanner is on `PATH`), and writes exactly those four values back.
`attribution.hook` is written by `ahoy install --attribution`. The remaining four —
whether abcd should enable the forge's own secret scanning, the rules loader's
refresh backstop, and the filing-time match's threshold and compared fields —
are hand-set: the binary reads them and never writes them.

```json
{
  "meta": {                             // setup block — stamped and read by ahoy
    "schema_version": 1,
    "setup_version": "0.1.0",
    "setup_date": "2026-05-04",
    "project_name": "abcd-cli"
  },
  "repo": {
    "visibility": "private"             // "private" | "public" — set by ahoy each run, no silent default
  },
  "docs": {
    "target": "agents_md"               // "agents_md" | "skip" — whether AGENTS.md carries the marker
                                        //   block; "skip" is the default, so a default install names abcd
                                        //   in no conventions file. "claude_md" and "both" are read for
                                        //   detection and uninstall, and refused at setup with the
                                        //   explanation naming this setting (itd-2610030814013772)
  },
  "oracle": {
    "backend": "host-delegated"         // "host-delegated" (default: abcd emits a prompt, the host runs it —
                                        //   no API keys, no model config) | "native" | "cli" | "api" | "mcp"
  },
  "scan": {
    "deep": false,                      // the native secret/PII scan is the default; deep adds an opt-in
                                        //   external backend, asked only when it is available
    "native_secret_scanning": true      // an explicit false opts the repo out of abcd enabling the forge's
                                        //   own secret scanning through `ahoy remote apply`
  },
  "attribution": {
    "hook": true                        // the committed prepare-commit-msg prompt asking each commit to
                                        //   declare whether a tool assisted it
  },
  "rules": {
    "force_refresh_every_n": 15         // prompt-router refresh backstop (itd-3). The primary refresh is
                                        //   event-driven, on SessionStart and PreCompact, so this large
                                        //   counter only re-injects always-relevant domains
  },
  "match": {                            // the filing-time match of capture and the intent create
                                        //   (itd-2609212137116617), a lexical heuristic
    "threshold": 0.6,                   // share of the new text's terms a record must hold, in (0, 1],
                                        //   for a duplicates/refines link to be written
    "fields": ["issue.body", "intent.title", "intent.press_release"]
                                        // what each candidate offers; one or more of the three
  }
}
```

The `match` keys are read through the layered configuration reader
(`internal/core/layered`), as the provider adapter's `oracle` keys below are:
`.abcd/config.json` wins, then
`~/.abcd/config.json`, then the bundled default above, and each value is
reported with the file it came from. The reader claims the `match` namespace,
so a misspelt key, a threshold outside its range and a field outside the set
are refused naming the file, never passed over for the default. A refusal
there never refuses a filing: the record is written unlinked and the verb says
which key was refused.

There is no separate `.abcd/meta.json` at repo scope: setup metadata is the `meta`
block. This repository's own config carries four of these blocks — `docs`, `meta`,
`oracle` and `repo` — which is what an unremarkable managed repo looks like.

### The provider adapter's keys

The OpenAI-compatible API adapter (itd-2609081951381895, adr-2609221009491186)
reads four keys under `oracle` through the layered resolver, every one validated
when the configuration is read and refused loudly, naming the file and the key,
rather than skipped:

```json
{
  "oracle": {
    "api": {                             // MACHINE LAYER ONLY: ~/.abcd/config.json
      "openrouter": {
        "base_url": "https://openrouter.ai/api/v1",   // https, or http to this machine
        "key": "openrouter",             // a credential NAME, resolved through the credential
                                         //   source; omitted for a server that takes no key
        "models": ["typesafe/jev-1.13"]  // the allowlist: the only models it may serve
      }
    },
    "denylist": ["openai/*"],            // optional, none bundled; repo or machine
    "roles": { "scribe": "openrouter/typesafe/jev-1.13" },            // an agent in the roster; a keyed
    "judgements": { "duplicate-match": "openrouter/typesafe/jev-1.13" } //   provider's routes: machine only
  }
}
```

- **A provider block sits on the machine alone.** It names the address a key is
  sent to, so a repository's `.abcd/config.json` declaring `oracle.api` is
  refused: a checkout must never be able to aim the person's key at a server of
  its choosing. `abcd ahoy connect` writes the block, after one verification
  call, and it is the one write abcd makes to `~/.abcd/config.json`.
- **The allowlist alone decides; the denylist is optional and a union.** abcd
  bundles no vendor denylist (adr-2609300107513982), so a model a provider lists
  is served whichever vendor made it. `oracle.denylist` is the configuration's
  own: each layer's entries apply, an entry is a vendor prefix (`vendor/*`) or
  one model, and matching ignores case, OpenRouter's `~` alias prefix and a
  `:variant` suffix. No layer removes another's entry, and a block listing a
  model an entry matches is refused, naming the entry, whatever else it lists.
- **A route is `<provider>/<model>`.** A role or a judgement type the
  machine's file points at a model its provider does not list is refused naming
  the list. The same route in a repository's `.abcd/config.json`, to a provider
  that holds no key, is skipped with one diagnostic naming the file, the route
  and the list, and the machine's own route to that name, if it has one,
  applies in its place (the technical facilitator's ruling CD3 of 2026-10-02);
  a model the denylist matches is refused from either file. A route pointed at
  a provider this machine has not configured is a diagnostic, not a refusal: a
  repository's such route is skipped where `~/.abcd/config.json` routes the
  same name, so the owner's setting applies (ruling CD4 of 2026-10-02), and
  otherwise the step stays on the host, as it would with nothing configured
  (adr-25, amended 2026-10-02). A role outside the
  roster is named and skipped, like an orphan routing row. A route's name is a
  plain lower-case name. A repository route that is not `<provider>/<model>`,
  or whose name only the repository spells otherwise (a lookalike letter from
  another script, a space, a control character), is skipped with one
  diagnostic naming the repository's file and the offending text, sanitised
  and with any non-ASCII letter spelled as an escape; the rest of the
  configuration loads, and the machine's own route to that name, if it has
  one, applies in its place (ruling CD2 of 2026-09-29). The same fault in
  `~/.abcd/config.json` is refused, because that file is the person's own and a
  route they set is never dropped silently.
- **A route to a provider that holds a key sits on the machine alone.** Only a
  route the person set up on their own machine may spend their paid key (the
  product thinker's ruling AA(b) of 2026-09-29), so a repository's
  `.abcd/config.json` pointing a role or a judgement type at such a provider is
  skipped, with one diagnostic on stderr naming the route, `~/.abcd/config.json`
  as where to set it, and the repository's file as where to remove it (the
  technical facilitator's ruling CD2 of 2026-09-29). The rest of the
  configuration loads, so every other route and every command that reads it
  keeps working, and the machine's own route to that name, if it has one,
  applies in its place. Every front door that reads the configuration says
  each skipped route: `abcd ahoy connect` and `abcd ahoy credential` on
  stderr, the `abcd ahoy --providers` board among its lines, and the bare
  `abcd ahoy` board as the optional gap `oracle_api.route_skipped`. A route the denylist matches is refused whichever
  provider it names. A
  provider holds a key when its block names `key`, judged from the block and
  never by reading the credential store. A repository's route to a provider
  whose block names no key (a local server) is admitted and wins over the
  machine's per name, and a `--route` the person types is unaffected.
- **A call that spends a key takes no repository settings unless the person typed its route.** A
  provider leg sends the connection's defaults, then the winning routing row's
  settings, then the `--route`'s. On a leg to a provider that holds a key and
  that the person did not type with `--route`, a row from the repository's
  `.abcd/config/oracle-routing.json` that names
  settings (`max_tokens`, `temperature`) is refused before the step runs, never
  dropped, naming each setting, the repository's file, and
  `agents.<agent>.settings` in `~/.abcd/oracle-routing.json` as where to move
  them, because the settings size and shape a call the person pays for. A
  keyed leg the person typed with `--route` is theirs, so there the repository
  row's settings merge within the provider's accepted set. A repository row
  without settings, the machine's own row and a keyless leg keep the merge.
- **Which agents a paid provider takes is the person's to widen, on their
  machine alone.** By default a provider that holds a key takes only the
  self-contained agents, the four cold-reading positions (ruling DR5 of
  2026-09-29, adapters chapter). `oracle.bundled_context_providers` in
  `~/.abcd/config.json` is the person's override: a list of provider names that
  may take bundled-context requests for file-reading agents. It is read from the
  machine layer alone, and a repository's `.abcd/config.json` declaring it is
  refused, as a repository's provider block is; a name this machine has not
  configured is a diagnostic that admits nothing. A named provider takes a
  file-reading agent only once abcd builds that agent's bundle, and none is
  built, so every file-reading agent stays refused with a reason saying so. The
  list inherits the machine layer's trust in `$HOME`, pending
  iss-2609300012273350.
- **The model a provider reports is held to the denylist too.** An aggregator
  that answers with a model an `oracle.denylist` entry matches has substituted
  a model the configuration refuses; the answer is discarded and the refusal
  names what it reported. Every call records the provider, the model asked for
  and the model reported, so any other substitution is visible in the record,
  and an answer that reports no model is refused rather than recorded with an
  empty one.

Unconfigured, nothing changes: no provider block means no connection, and every
delegated step runs on the host. A role pointed at a configured provider takes its
agent's steps there whatever tier the routing tables name, and only a `--route`
overrides it for one run. The delegating verb sends such a step through the
adapter itself and ingests the answer (spc-2609251028149555), and its receipt
names the provider as the connection used.

### The interview key

How a plain-Terminal interview takes a choice from a list
(itd-2610030810370060, spc-2610030911534855) is one key, read through the
layered resolver from the machine's file alone:

```json
{
  "interview": {
    "list": "numbered"                  // MACHINE LAYER ONLY: ~/.abcd/config.json
                                        //   "arrows" (the bundled default): the arrow-key
                                        //   list, typing to narrow it | "numbered": whole
                                        //   lines, a number or part of a name, the
                                        //   terminal's modes never touched
  }
}
```

- **It is the person's own.** How a list is read is the person's to say, so a
  repository's `.abcd/config.json` that sets `interview.list` is refused naming
  `~/.abcd/config.json` as where it belongs; the reader claims the `interview`
  namespace, so a misspelt key and a value outside the two are refused naming
  the file, never passed over for the default.
- **A refusal never fails an interview, and never lets a repository choose.**
  The answer loop says the refusal in one line on stderr and goes on: a fault
  a repository's file holds (the key set there, a misspelt key under
  `interview`, a malformed file) is passed over for the machine's own setting,
  read from `~/.abcd/config.json` alone; a fault in the machine's own file
  gives `numbered`.
- **Two variables choose `numbered` for one session.** `ABCD_ACCESSIBLE`
  non-empty, else `ACCESSIBLE` non-empty, selects it whatever the file says,
  so a screen-reader user who cannot set a file first gets it; when both are
  set, `ABCD_ACCESSIBLE` is the one named. Neither can select `arrows`.
- **Where no keyboard mode is possible, the list is numbered.** `TERM=dumb`
  cannot move its cursor, and a terminal that refuses raw mode cannot take the
  arrow keys; each falls to `numbered` with one line on stderr saying so, and
  the interview goes on.

### Staged config keys

No shipped code reads any of the keys below. None appears in any repository's
config today, and writing one has no effect. They are the axes the design commits
to as the seams and the sync namespace land:

- **`ai_transparency.level`**: a second axis, independent of visibility, deciding
  what is captured at all rather than what is committed. Described below.
- **`spec.backend`, `run.backend`, `history.backend`**: the selectors for three
  seams. The native spec store and the native history store both ship; what does
  not ship is anything that would choose between backends.
- **`disembark.maxAgentTokens`**: a per-agent context budget, over which an input
  would stream and summarise.
- **`memory.harvest`**: vendor memory harvest as an opt-in read-only source.
- **`reviews.capture`** and the **`dev_sync.*`** per-source enable flags: the
  sync machinery of § 2, staged in full.
- **`intent.auto_link` / `intent.auto_ship`**: staged as keys only. The behaviour
  ships unconditionally: `abcd intent plan` mints the spec and writes both sides of
  the link, and the spec-close hook moves the intent. Neither is switchable, so
  neither key is read.
- **`capture.default_severity`**: today the severity rides on the invocation.

Two axes are absent from both lists on purpose. There is no `embark.scan` key,
because there is no `embark scan` sub-verb for it to switch. There is no
`adapters` registry block, because the wired-adapter registry it indexed has no
home: no registry package exists and no code refers to one.

**Owed-review draining (staged)** is receipt gating in the `run` seam
([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)). No `run`
seam ships, so nothing drains today. When it does, owed intent audits drain at
the seam's iteration boundary: each iteration gates on a receipt and applies the
safety guard, report-not-block, whichever adapter provides the loop. There is
deliberately no autodrain config knob — receipt gating is part of the seam
contract, inherited by every adapter loop rather than re-implemented per loop.

**Audit-loop mode and budget (staged: itd-50)** are elected **per intent**, in the
intent's own frontmatter rather than in `config.json`, so the choice is portable
with the intent and one intent can loop while another stays record-only. No intent
carries these keys today and no validator reads them. The shape:
`audit_mode` is `record-only` by default, meaning a failed criterion is recorded
and nothing re-opens; `loop-to-acceptance` re-opens the linked work and iterates
until the criterion is met or `audit_budget` (default 3) bounds it. Budget
validation is **fail-closed**: a malformed, zero, or negative budget on a looping
intent never starts the loop and never silently coerces to the default. The
review-queue entry snapshots the effective mode and budget at enqueue time, so a
mid-loop edit cannot change an in-flight loop. The loop terminal and the gate
belong in a drainer layer that does not exist.

Schema versioning and cross-version migration come in a later phase: a
configuration record carries `schema_version: 1`, and migrators are added if a
later phase changes the shape (itd-9). The stamp is a convention rather than a
held rule, and the tree is not uniform — eight of the thirteen committed records
carry it, and five do not, the record-lint and docs-lint configuration among
them. Nothing refuses an unstamped record, so a migration that arrives before the
stamps do has no version to read on those five.

## The history store

The history store is a **user-scope** artefact, shared across every abcd-managed
repo on the machine, living at `~/.abcd/history/`. ahoy bootstraps it once, on the
first install on a fresh machine. It holds a registry keyed on each repo's
root-commit SHA, and one identity-and-lineage file per repo under that key.

The root commit is the only immutable identity a repo has: it survives a rename, a
remote move, and a change of forge handle. Everything else the registry records —
the repo's name, its remote URL, its path on this machine — is a mutable label
ahoy refreshes on every run rather than treating as write-once. A repo entry also
carries its lifecycle status and, where a repo was re-founded, the cross-references
to and from the entry it supersedes.

Re-founding a repository (a clean-history rebuild) produces a *new* root SHA. ahoy
registers the new entry, marks the old one superseded, and leaves the old corpus
in place under its own key for lifeboat review. A per-repo entry can also carry
prior names, free-text provenance about why the repo was re-founded, and a pointer
to where this repo's captured evidence lives.

This registry is the **sole user-scope registry**. abcd manages one repository per
tree, so there is no workspace-to-repo grouping to register and no second registry
file.

### The transcript corpus

The transcript corpus is a **sibling** user-scope store rather than a sub-tree of
the registry, at `~/.abcd/transcripts/<root-sha>/`, holding redacted records and a
staging area for raw transcripts awaiting redaction
([adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)).
Every machine-scoped store keyed on the root commit takes the full object name
as its `<root-sha>`: Forty hex digits under SHA-1, sixty-four under SHA-256, the
form `gitutil.RootCommit` returns and `gitutil.IsFullSHA` admits as a path
segment. An abbreviated key is a different directory, and no verb reads it.
One package owns its layout: `internal/core/history` declares both the user-scope
default and the opt-in per-repo location, and every resolver goes through it. The
rule is a convention with nothing behind it, and it already has one exception —
the cold-reading assembler repeats the per-repo path as a literal in its
exclusion rows rather than reading the constant, so moving that store would leave
the exclusion pointing at the old place.

It is **self-creating**: the store bootstraps on first use, so no install step
stands between a wired hook and a stored transcript (iss-95). Every level is
created individually and re-verified as a real directory on every resolve, so the
store never creates or writes through a symlink.

A repo may **pull its transcripts in**, by declaring the checkout in the caller's
own home — one absolute path per line in `~/.abcd/local-transcript-roots`, the
same home-scoped, abcd-owned, line-oriented idiom as the other declarations, and
honoured only when the file is a regular file this uid owns that no one else can
write. A declared checkout keeps its store in the gitignored per-worktree local
tier. A corpus left at an earlier path is moved into the store on first resolve,
reported, and tombstoned where it was.

## The voyage store

`voyage/` is **operations** — the verb, what we did — as distinct from the
lifeboat, which is the **artefact**, the noun that gets carried. It is user-scope
and keyed on the source repo's root-commit SHA exactly as the history store is, and
it is **never committed**
([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md),
superseding adr-4's in-tree home). What ships is an append-only manifest log of
every disembark run; nothing else is written anywhere under the store, and an
embark side recording what a write did is **staged**.

Two properties follow from keeping it out of the tree, and both are load-bearing:

- **`disembark` never writes to the source repository.** It reads the source and
  writes the lifeboat to an operator-chosen destination, and the operations log
  lands at the operator level. Mining a dead or archived project must not require
  installing abcd into a repo we only want to read.
- **Voyage records absolute source paths, so it must not be committed anywhere.**
  abcd's own privacy rule flags a home path in a committed file, so an in-tree
  voyage namespace would have made abcd fail its own audit. Keeping it at the
  operator level dissolves the collision rather than exempting it, which is why no
  gitignore rule for voyage appears in § 1: there is nothing in-tree to ignore.

Because the lifeboat lands out of tree, `embark` reads it from wherever disembark
wrote it. The destination is protected by a **safety gate** rather than an
overwrite-with-backup model: abcd refuses unless the destination is absent, an
empty directory, or one carrying a provenance file abcd itself wrote. See
[`../04-surfaces/03-embark.md`](../04-surfaces/03-embark.md).

## The lab store

The lab store holds the evidence of labs: throwaway worlds pinned at one commit
of a repository, run to answer one question
([`../04-surfaces/31-lab.md`](../04-surfaces/31-lab.md), itd-2609212137128014).
It is user-scope and keyed on the repository's root-commit SHA exactly as the
transcript store is, and it is **never committed**:

```
~/.abcd/lab/
  <root-sha>/
    index.jsonl               one registry line per lab: id, pin, question
    <lab-id>/                 one lab home: the intention, the snapshot, the lab's
                              own HOME and binaries, probe records, findings,
                              corrections, the preflight and sweep artefacts,
                              the harvest
```

A lab id is `lab-<yymmddHHMMSS>-<pin7>`, the UTC mint time and the pin's first
seven digits, and the exclusive creation of its home is what keeps two ids
apart. Three properties are load-bearing:

- **Evidence stays at the operator level, knowledge moves by ceremony.** A lab
  touches private-tier material and must outlive any checkout it cites, so its
  evidence never enters a repository; what a lab learns enters one only through
  capture or a record that cites the lab, and the local ephemeral tier holds
  pointers at most.
- **It creates itself through one seam, never through a symlink**, on the
  transcript store's discipline, private to the account, and every write inside a
  lab home goes through a containment root opened on that home.
- **Labs that predate the keyed layout are left alone.** Hand-run labs sit at the
  top of the store with their own index; the verb neither reads nor writes them.

## The worktree store

**Design target (itd-2609091014076309, `intents/planned/`; unbuilt).** No
`worktree` verb exists in the shipped binary, and nothing in it creates or reads
this store. What follows is the layout the intent commits to, on the rule
[adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md)
records: a tool never creates directories in user-owned project space, so a
session's or an agent's worktree is machine-scoped rather than a sibling of the
checkout. It would be user-scope and keyed on the repository's root-commit SHA,
exactly as the history, transcript and voyage stores are, and never committed:

```
~/.abcd/worktrees/
  <root-sha>/
    <name>/                   one git worktree of that repository, on its own branch
```

Three properties are load-bearing, and each is the intent's to deliver:

- **Git is the registry.** `git worktree list --porcelain` already names every
  worktree wherever it sits, so the store keeps no index of its own. The verb reads
  git's answer and says which entries are in the lane, which are outside it, and
  whether each is clean and merged. The worktree's own `.git` file, not a mutable
  path label, says which checkout a lane belongs to.
- **It creates itself through one seam, never through a symlink**, on the
  transcript store's discipline: each level made individually and re-verified as a
  real directory.
- **Reclaim is explicit and provable.** `prune` removes a worktree only when its
  branch is merged into the default branch and its tree is clean, names everything
  it declined and why, and never deletes a directory git does not recognise as a
  worktree of the repository.

Worktrees already sitting beside a checkout are outside the store by definition:
listed as such, never moved, and retired by the user's own `git worktree remove`.

**One primitive owns the store and its notes archive.** One package under `ahoy`
derives the lane from the root commit, makes every level of
`~/.abcd/worktrees/<root-sha>/` and of the notes archive
`~/.abcd/notes/<root-sha>/` one at a time as a real directory that is the
caller's alone, and adds, lists and removes worktrees through git. The build
loop's lanes are made through it, so a worktree enters the store one way. Before
`prune` removes a worktree it moves the worktree's git-ignored
`.abcd/.work.local/` into `~/.abcd/notes/<root-sha>/<UTC timestamp>-<name>/`
beside a manifest, redacted on write by the transcript store's pass; nothing
reclaims the archive.

**Invariant: the store deletes only what passes its proof of belonging.** A
directory leaves the lane only when git lists it as a worktree of this
repository, its real path lies inside this repository's lane, and its own common
directory is this checkout's; anything else is reported and left in place, and no
removal is forced. It is the reclaim half of
[adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md):
a tool that must not create in the user's space must not delete there either.

## The two `.abcd/` scopes

`.abcd/` is **one namespace pattern instantiated at two scopes**. abcd lives in
one repository ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)):
its design record is repo-scoped and in-tree, and the user scope holds only state
that is genuinely machine-wide. `/abcd:ahoy` classifies the folder it runs in and
acts on the scope that applies.

**User scope, `~/.abcd/`** — one per machine, machine-local shared state only: the
history registry, the transcript corpus, the voyage operations namespace, the
lab store, the staged worktree store, the run state an autonomous run's sessions share
([`../04-surfaces/27-implement.md`](../04-surfaces/27-implement.md)), the inbox of reports managed repositories file back to abcd
([`../04-surfaces/29-report.md`](../04-surfaces/29-report.md)), the machine layer of the layered configuration in `config.json`
(its readers are the provider adapter's and the filing-time match's `match`
keys, each read beneath the repo-scope `.abcd/config.json`, and its one write
the provider block `ahoy connect` adds; every other config read resolves the
repo-scope `.abcd/config.json` alone), the load check's two limits in
`load-limits` (read-only and never created, itd-2609231434459890), the external
credentials adapters resolve by name through the credential store, whose abcd
home is `credentials.json` and whose index is `credential-homes.json` (each
refused unless it is a regular file this uid owns at mode 0600 that names each
credential once, a repeated key or a case twin included; the walkthrough adds
one name at a time and never replaces a stored value, holding the file's lock
across the read and the write as the provider block's write holds
`config.json`'s, so concurrent setups lose nothing), the
machine's rule conventions in `rules.json` (the user layer of the rules loader,
read-only and never created, itd-117 — see
[the rules layers](#the-rules-layers--bundled-user-repo) below), user-scope memory for personal cross-project knowledge (a later
phase too: the shipped memory store is repo-scope), and the `sources/` corpus `/abcd:ingest` and
`/abcd:consult` read (abcd never creates that one, and both verbs say so and stop
when it is absent). It also holds the caller-controlled declarations: the owned
PATH entry, the trusted configuration roots, and the checkouts whose transcripts
are pulled in. **Never the design record.** The same inventory is drawn as a tree
in [`../04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md#what-abcd-manages--repos-and-abcd);
the two are one list and must agree.

**A symlinked `~/.abcd` hosts nothing abcd trusts.** Every file in the user
scope whose contents abcd acts on — `rules.json`, `trusted-roots`,
`local-transcript-roots`, `path-entry`, `cache-attestation`, `config.json`,
`oracle-routing.json`, `statusline.json`, `load-limits`, `credentials.json` and
`credential-homes.json` — is refused when `~/.abcd`, or a directory below it on the way to the file, is a
symlink: the rule the rules loader states for `rules.json`, applied by one check
(`fsutil.HomeScopeLink`, read through `fsutil.ReadHomeDeclaration`) so it cannot
drift per file. The rule holds against a race as well as a layout: a reader or
writer opens `~/.abcd` and each level below it relative to the descriptor of the
level above (`fsutil.OpenHomeScope`, or `fsutil.EnsureHomeScope` to create the
missing levels), confirms each descriptor is the real directory its judgement
saw, and reaches the file only through that descriptor, so a process swapping
`~/.abcd` for a link between the check and the use is refused rather than
followed (iss-2609281310017733). The file's own guards are judged on that
descriptor too: the credential store's mode 0600 is judged on the fstat of the
file that is opened (`fsutil.ReadHomeDeclarationDenying`), never on its path,
so a store swapped for a group-readable file after any check is refused. A symlinked `~/.abcd` holding no such file reads as absent and
costs nothing. A file that is there behind the link is refused the way its reader
refuses any declaration that is not the caller's word: the rules load fails, a
declaration is ignored with a note, the path entry and the cache attestation
vouch for nothing, the credential store refuses loudly. Every write abcd makes
into those files — the credential and the provider block `ahoy connect` adds, the
path entry, the routing table and the status-line setting `ahoy install` writes,
and the path entry and cache attestation `hooks/bootstrap.sh` writes — refuses
the link rather than writing through it, naming it and the repair: replace the
link with a real directory. The path entry's removal on uninstall goes through
the same descriptor, so it removes nothing behind the link. A credential
setup refuses a symlinked `~/.abcd` in every home, the keychain and external
homes included, before it creates anything: the index, the value and both
locks are reached through the one walk that created and judged `~/.abcd`,
with the index's lock taken there and the abcd home's lock nested inside it.
An external home's pointer at a file under a symlinked directory (a
`~/.config` linked into a dotfiles repository) is refused the same way,
naming the link, and the file is read through the descriptor walk. The hook shims refuse a `path-entry` behind the link
too, before they read it. The home directory itself may be a link; only
`~/.abcd` and what lies under it are judged. The stores are not declarations:
`transcripts/`, `voyage/`, `lab/`, `inbox/` and `runs/` refuse a symlinked
level through their own create-then-prove seam (`fsutil.EnsureRealDir`), and the
`sources/` corpus is the caller's to place. The `history/` registry applies
both: it is neither read nor written behind a symlinked `~/.abcd` or
`~/.abcd/history`, and it is created, locked, read and written through
`fsutil.EnsureHomeScope`'s descriptor (iss-2609281129171021). `ahoy install` skips the registration with a note naming
the link and the repair, and the detector reports it as a diagnostic rather
than a gap install would try and fail to close.

**Repo scope, in-tree `.abcd/`** — this repository's record and working files: the
three-tier layout below, the config file with its `meta` block, the rules
overrides, the per-surface machine records under `config/`, the lint and site
configuration records, the identity and positioning registry, the citation
baseline, and the native spec store. A `memory/` namespace is written
where memory is curated. **The home for project work.** There is no in-tree
lifeboat directory: the lifeboat is out-of-tree output at an operator-chosen
destination. Two namespaces are not part of it either: `logbook/`, a retired name,
and `rp/`, staged with its adapter.

**The repo-scope three-tier working layout** (matching
[`../02-constraints/01-platform.md`](../02-constraints/01-platform.md)):

| Tier | Path | Committed? | Holds |
|---|---|---|---|
| **record** | `.abcd/development/` | committed — in every repository checkout, never in the released binaries | the durable design record: brief, roadmap, intents, ADRs, research |
| **shared work** | `.abcd/work/` | committed | current orientation, the append-only decision log, the external-contribution runbook, the issue ledger, the reviews charter, and the branch-ruleset mirror |
| **local ephemeral** | `.abcd/.work.local/` | gitignored | machine-local scratch: handover notes, scratch, logs, intent-audit receipts, the per-machine banlist layer. Per-worktree, so it never merge-conflicts |

**The record is repo-scoped and in-tree.** No workspace layer holds it and there is
no workspace registry: abcd is one repository, so the record lives in that
repository's tree, and there is no dev-to-public mirror. The user scope survives
only for state that cannot live in any one repo's tree. The repo keeps its config
and rules files in-tree too, so the hooks and the marker-block installer read them
from the repo directory deterministically.

**The vendor harness directory is not abcd's.** abcd writes nothing into it beyond
the plugin install itself; everything else routes to the scope-appropriate
`.abcd/`. The one interaction the design gives abcd with it is read-only: the
staged memory harvest of § 2 would read from it as a source.

## The rules layers — bundled, user, repo

The rules loader composes three layers, each overriding the one before it per
field: the domains bundled in the binary, then the user scope's
`~/.abcd/rules.json`, then the resolved repo root's `.abcd/rules.json`. Both
files take one schema — `schema_version`, the `disabled` kill switch, and a
`domains` map whose entries override a bundled domain's `state`, `recall`,
`aliases` or `rules` or declare a custom domain outright. A field one layer sets
replaces the field below it wholesale; a field it leaves out is inherited. So
the bundled domains are the floor, the machine's house conventions refine them
once for every repo abcd manages there, and a repo that genuinely differs still
overrides locally (itd-117, spc-23).

| Layer | File | What it is for |
|---|---|---|
| **bundled** | none — embedded in the binary | abcd's own opinions, the same on every machine |
| **user** | `~/.abcd/rules.json` | the delta between those opinions and one machine's house style: a definition-of-done wording, an attribution example, a custom domain wanted across that person's own projects |
| **repo** | `<repo>/.abcd/rules.json` | what one repository needs that differs from both |

**Suppression is sticky downward.** A repo's `dormant` state beats a user layer
declaring the domain active, because the repo is applied last. The kill switch is
sticky in both directions — either file's `"disabled": true` silences the set —
so a user-scope kill switch silences every repo on the machine and no repo file
re-enables it. That is the fail-safe direction: a kill switch a lower layer
could override is not a kill switch.

**Absence costs nothing.** A machine with no `~/.abcd/rules.json` loads exactly
the set it would without the layer, and abcd never creates the file or its
directory: it is hand-edited, and `abcd rules` is its read-only render. A `HOME`
or `~/.abcd` this account cannot search reads as absent too, as it does for the
other home-scoped declarations, so a sandboxed or foreign `HOME` never warns on
every prompt about a file nobody can see; a `rules.json` that is there and
cannot be read is refused.

**The user file is read as the caller's word.** It injects text into every
session on the machine, so it is read through the same guard as the home-scoped
declarations: a regular file — never a symlink, FIFO or device — of at most
256 KiB, owned by this account and writable by no one else, with `~/.abcd`
itself refused as a symlink once a `rules.json` sits behind it. A
dotfiles-symlinked `~/.abcd` can therefore never host a `rules.json`: the file
is refused behind a symlinked `~/.abcd`, and only a symlinked `~/.abcd` with no
`rules.json` in it is spared, so that a machine whose `~/.abcd` lives in a
dotfiles checkout keeps injecting exactly what it did. The repo layer's `.abcd`
has no such exemption and is refused as a symlink unconditionally. A file failing
any of those, or failing to parse or validate, fails the whole load: the hook
injects nothing and names the file on stderr, and `abcd rules` exits non-zero.
It never degrades to a partial set built from the layers that did load
([`../../principles/loud-staging.md`](../../principles/loud-staging.md)). A
domain left with no rules after all three layers is skipped with a note naming
the file that last named it, the same treatment a repo file's ruleless domain
gets.

**Provenance names the layer.** Each domain carries the layer that last named
it: `## NAME (user override)` or `## NAME (repo override)` in the injected block,
in `abcd rules` and in the hook's diagnostic, and `"source": "user"`, `"repo"`
or `"bundled"` in `abcd rules --json`. A disabled set names every file whose kill
switch is set.

**Known limitation.** Replacement is per field, so a user or repo layer cannot
add one rule to a bundled domain's list: it restates the list. A third layer
makes that grain more visible; finer-grained merging, detecting a repo file that
duplicates the user layer, and moving conventions out of per-project harness
memory are all recorded in itd-117 as follow-up questions.

**A withheld guardrail is named.** Because a list replaces the bundled list, an
override written before a release added an entry keeps withholding that entry.
For the four guardrail domains — `PII`, `COMMITTING`, `LOAD` and `SHELL` — the
load compares every recall, alias and rule list an override set against the list
the load built before any `rules.json` layer: the running binary's bundled list,
and for `SHELL` the lessons of the repository's own `.abcd/guard.json` entries
too. It names each entry left out, and the file whose list is in force, on
stderr from `abcd rules` and from the hook on every prompt. The effective set is
unchanged. Restating the entry keeps it; leaving the field out inherits the
list. The other bundled domains are conventions a
repository restates in its own words, so a replacement there is not reported.

**Two bundled domains are generated.** `SHELL` is the teaching plane of the
shell-hazard guard (itd-103, spc-16 "Two planes, one registry"): its rules and
recall keywords are built from the same hazard registry `abcd guard` enforces,
never written in the bundled `rules.json`. The bundled set carries it built from
the bundled registry; every load rebuilds it, by the same generator, from the
registry the guard enforces in the repository, which is the bundled entries
merged with the repository's own `.abcd/guard.json` (ruling CK1). Each registry
entry becomes one rule — whether the guard refuses or warns, the entry id, the
command it matches, the plain-language why, and the safe successor — in entry-id
order. The recall keywords are the command heads the registry matches (`rm`,
`git push`, `gh repo delete`, `pkill`, …), which carry their subcommands so
the bare words "push" or "reset" never recall the domain, plus a short fixed
list for shell work in general (`shell`, `bash`, `zsh`, `command line`,
`force push`). An entry added to or removed from the registry changes the
domain with no second edit, and a test fails the build if the domain and the
registry ever part. To every other contract it is an ordinary bundled domain:
a user or repo layer overrides it per field, `dormant` silences it, `*SHELL`
activates it, the kill switch suppresses it, and dedup and provenance treat it
like any other. Its injected block costs about 2k tokens for the bundled
registry, one rule per registry entry, and each entry a repository adds or
rewords in its `.abcd/guard.json` adds its own rule. An entry's why and its
successor are each capped at 1,024 bytes, over three times the longest bundled
one, so one rule is a few hundred tokens at most; a guard file carrying a longer
one is refused like any invalid entry. When the matched rules still overflow
the 64 KiB injection budget, the truncation notice names the file whose words
filled it, `.abcd/guard.json` for these lessons and the layer's `rules.json`
for a list an override set. The block is paid once per session per
signature, so dedup never injects it again while its rules are unchanged, and
an edit to the guard file re-injects it once. A rule whose words are the
repository's — an entry the file adds, or a bundled entry whose tier, pattern,
why or successor it changes — carries `(repo)` after its entry id, so whose
words an agent is taught is never invisible; a fixture-only change teaches
the bundled words and is not marked. Under a committed `"disabled": true`
the guard refuses nothing, so every rule opens `Hazard (guard off)` in place of
`Refused by the guard` or `Warned by the guard`: the hazard is still taught, and
the sentence stays true. A `.abcd/guard.json` the guard refuses
(unreadable, invalid, or an uncommitted edit that weakens it) is refused here
too and never skipped in silence: `SHELL` teaches the registry the guard falls
back to, none of the refused entries, and the load names the file and the
reason on stderr, from `abcd rules` and from the hook on every prompt, while
every other domain loads as usual. The switches stay independent: the guard
file decides what is refused, and `rules.json` overrides, silences or kills the
teaching of it.

`GRILL` is the second (itd-201, spc-2610030944505997 "GRILL generated from one
Go source"): the asking rules every abcd interview follows. Its rules are
written once in `internal/core/question`, beside the question's field limits,
and every limit they state (the header chip's width, the options per question,
the words per label, the rows at eighty columns) is filled from the one value
the question check in `abcd guard hook` enforces, so the rules and the check
cannot state a limit differently. Its recall keywords are words of asking and
choosing (`which`, `choose`, `decide`, `options`, `interview`, …), so it lands
in most sessions, at about 1,100 tokens. A bundled `rules.json` that declares
it by hand panics at load, as one declaring `SHELL` does. It reaches every
managed repository through the binary, abcd's own repository included, which
declares no override of it; to every loader contract it is an ordinary bundled
domain, and a repository silences it with `{"GRILL": {"state": "dormant"}}`.

## The prompt router's output

`abcd hook prompt-router` is the `UserPromptSubmit` entrypoint the hook manifest
wires. It reads the host's hook payload on stdin, recall-matches the prompt
against the loaded set, and exits 0 on every path, so it can never wedge a
session. It writes to two streams:

| Stream | What it carries | Who reads it |
|---|---|---|
| **stdout** | the rendered block of the domains new this turn, and nothing else. A prompt that matches no domain, or matches only domains already injected unchanged this session, writes zero bytes | the host, which adds it to the model's context |
| **stderr** | one diagnostic line per prompt — the turn, the labels of the injected domains, the byte count — plus the load's notes and refusals | the operator, out of band |

**The machine reader's envelope.** With `--json`, the flag every verb takes for
a machine reader, stdout carries one JSON document in place of the bare block.
It is for a client that snapshots injected rules rather than appending them to
a transcript — a host adaptor that stages them into a system prompt, or a later
MCP consumer — and it is the protocol's removal signal (ruling J15,
iss-2608261550580260):

```json
{
  "text": "# abcd rules — 1 domain(s) active\n## WIDGETS (repo override)\n- Widgets are counted twice.\n",
  "injected": ["WIDGETS"],
  "active": ["COMMITTING", "DOCUMENTATION", "GRILL", "INTENTS", "ISSUES",
             "LIFEBOAT", "LOAD", "OPINIONS", "PII", "ROADMAP", "SHELL", "WIDGETS"]
}
```

That is the bundled set with one repo domain, `WIDGETS`, declared in
`.abcd/rules.json` and matched by the prompt: the text carries one domain and
the set names all twelve.

| Field | Meaning |
|---|---|
| `text` | byte for byte what the plain form writes to stdout: empty on a turn with nothing new |
| `injected` | the domains whose text `text` carries; an empty list when it carries none |
| `active` | the FULL set of domain names in force this turn, sorted, on every evaluated prompt: every domain that is not dormant, plus a dormant one this prompt activated with `*NAME`. An empty list when nothing is in force, as under the kill switch |
| `error` | present only when the router could not evaluate the prompt — an unreadable payload, or a `rules.json` that will not load |

A client keeps a snapshotted domain while its name is in `active` and prunes it
the first turn the name is absent: absence is the stop, whether the domain was
deleted, renamed or made dormant. A renamed domain arrives under its new name
the next time a prompt matches it. An envelope with `error` carries no `active`
field at all, which means the set is unknown and the client changes nothing; an
empty list is a set, and a missing one is not, so a typo in `rules.json` never
reads as every domain stopping.

**The set never enters the model's context.** The hook manifest invokes the
plain form, so what the host injects is exactly the block above and the
zero-token promise holds unchanged: a turn with nothing new adds nothing, and
the active set is written only to a reader that asked for the envelope.

**A domain that stops is forgotten.** The per-session ledger drops a domain the
turn it leaves the active set, so if it comes back its text is injected again
the next time a prompt matches it, even when its rules are unchanged. A client
that pruned it gets it back, and a host that appends to a transcript pays one
re-render of that domain for the round trip; a domain that stays in force is
still never re-injected unchanged within a session. The plain path meets this
in two cases:

- a domain whose `rules.json` entry is deleted, renamed or made dormant and
  later restored is rendered again the next time a prompt matches it;
- a dormant domain activated with `*NAME` is in force for that prompt alone, so
  a prompt without the prefix drops it from the set, and the next `*NAME`
  renders it again, with no edit to `rules.json`.

## The rules root — which `.abcd/` governs a session

The rules, the hazard registry and the per-repo config are read from ONE resolved
root, so a session's injected rules and its hazard registry can never come from two
different directories. The resolution is bounded at the git working tree
(GHSA-vvqc-3mv2-5p49): the toplevel git reports for the working directory is
resolved first, the walk runs upward from the working directory and stops at that
toplevel inclusive, so the nearest `.abcd/` inside the tree wins and one above the
tree is never reached.

**When git will not answer.** abcd runs every git command under an isolated
environment so an inherited `GIT_DIR` or an injected config cannot redirect it.
That isolation also discards the operator's safe-directory exceptions, so the
toplevel query fails inside a checkout owned by another uid, and it fails outright
when the host launches a hook with git off its `PATH`. "Not a repository" and "a
repository git will not answer for" are different outcomes: only the first resolves
to the working directory with no walk, because collapsing the second onto it
dropped the repository's own configuration layer with no error anywhere — the
guard's hazards became allows, and the rules kill switch stopped applying. The
second falls back to the `.git` marker, under two bounds:

| Bound | What it refuses | What it costs a real checkout |
|---|---|---|
| **shape** | a `.git`-named entry that is not a repository: an empty file, a HEAD-less directory, a dangling symlink. The marker walk is a name check that runs to the filesystem root, so without this an unprivileged empty `.git` beside a planted `.abcd` in a world-writable directory governs every session under it | nothing: the two shapes git itself reads both pass |
| **ownership** | a marker root whose owner is not the caller. Shape alone is not a trust boundary: `git init` in a shared world-writable directory produces a genuine repository, and git's refusal on ownership is the same signal in that attack as in the legitimate foreign-uid case (iss-2609020259564193) | a declaration, once, per foreign-uid checkout |

A refused root is refused **loudly and fail-closed**: the session resolves to its
own working directory with no walk, and every front door prints one line naming
the refused directory, the two uids, what the session reads instead, and the
exact command that re-admits it
([`../../principles/loud-staging.md`](../../principles/loud-staging.md)). The
refusal bounds the walk, not the working directory. From a directory with no
`.abcd/` of its own, the bundled rule defaults (under the user layer, which is
the caller's own) and the bundled hazard registry stand in for the repository's.
A `.abcd/` at the working directory is still read, so a session started at the
refused root reads that root's configuration, and the line says so rather than
promising the defaults. The
ownership bound applies only to the git-refused fallback: where git answers, the
toplevel it named stands whoever owns it, because that is a repository git itself
vouched for.

**The opt-in.** A foreign-uid checkout, a container bind mount and a shared CI
checkout are all legitimate, so the refusal has a supported route back: one
absolute path per line in `~/.abcd/trusted-roots`, matched exactly both as written
and symlink-resolved, with `#` starting a comment. The file is read through the
guarded bounded read, and only when it is a regular file this uid owns that no one
else can write — a declaration anyone could have written is not the caller's word.

**Why the home scope, and not an environment variable.** The bar the opt-in must
clear is that the untrusted tree cannot assert it: a file inside the foreign root
declaring itself trusted would be circular. A home-scoped file clears that
structurally, because writing it needs write access to the caller's own home, the
same authority the caller already holds over everything abcd trusts. An
environment variable clears the letter of the bar and not its spirit: a repository
can ship the shell or task-runner configuration that sets it, and an operator whose
shell auto-loads that has the tree declaring itself trusted one indirection out.
abcd has ruled on this shape once already, in refusing an env-supplied data
directory, and
[adr-46](../../decisions/adrs/0046-persistence-never-weakens-the-verification-posture.md)
treats home write as the ownership root.

**The home directory is never a repo root.** Its `.abcd/` is the user layer, and
a home that is itself a git working tree (dotfiles in the home) is not thereby a
project. The walk passes over the home, and a toplevel that is the home resolves
like a directory outside any repository — the working directory, no walk — when
nothing below the home carries a `.abcd/`. So a session beneath such a home reads
`~/.abcd/rules.json` once, as the user layer, and never the home's `guard.json`
or `config.json` as a repository's own. A toplevel that contains the home — a
test harness that points `HOME` inside its checkout — stays the root, because it
is a repository git vouched for, and its own `.abcd/` stays its own.

## 1. Visibility-driven gitignore policy

One question at install decides what a repo commits. Set by ahoy:

| Directory | Public default | Private default |
|---|---|---|
| `.abcd/` | gitignored² | **committed** (the entire namespace: the record, the working tier, the machine records, and `memory/` where it exists — visibility is the single switch, no per-subdirectory exceptions) |
| `memory/` (legacy snapshot) | gitignored | **committed** if present¹ |
| `.abcd/.work.local/` | gitignored | gitignored (local-only scratch) |

Two artefacts are absent from this table by construction rather than by exception:

- The native transcript store is **always** gitignored. By default it is
  user-scope, so it is not a repo directory at all; where a checkout has pulled it
  in, it sits under the local tier, which the row above already gitignores.
- **The voyage store and the lifeboat are not repo directories either.** Voyage is
  user-scope and the lifeboat is out-of-tree output, so no gitignore rule applies
  to either under any visibility.

¹ New projects use `.abcd/memory/`, the substrate `abcd memory` curates. The
root-level `memory/` is the legacy snapshot pattern some existing projects
maintain by hand; abcd respects it if present and never writes to it.

² On a repo whose `.abcd/` already holds **tracked** files, install narrows the
`.abcd/` entry to the local tier and keeps every other entry (iss-255). An ignore
rule cannot untrack committed records: the wholesale fence on such a repo only
hides every new record file from `git status` and makes `git add` refuse them,
while the committed tiers stay published regardless. Narrowing needs positive
evidence of tracked files, so a repo whose git cannot be asked keeps the declared
fence, and the install receipt states the narrowing out loud.

**No exceptions to the visibility rule.** Locked decision: visibility is one
switch, with no always-gitignored carve-out for any namespace on sensitivity
grounds. If sensitivity is a concern, set visibility to public, which gitignores
all of `.abcd/`. Per-subdirectory exceptions create maintenance burden and
contradict the transparent-prompts principle. The tracked-tier narrowing above is
not a carve-out of that decision: it is the mechanical boundary of what an ignore
rule can do at all.

**The launch payload is unconditional regardless.** Whatever the visibility, the
release artefact excludes `.abcd/` entirely, so a private repo that commits its
whole record locally still leaks none of it on launch. In a private repo the whole
namespace is reproducible from a fresh clone; the lifeboat is not part of what a
clone carries, because it is out-of-tree output.

**Memory locations to keep straight.** Curated memory exists at both scopes, and
there is one non-abcd location alongside them: the repo-scope `.abcd/memory/` is
the **primary** store, holding the curated summaries `abcd memory ingest` writes
and the canonical input for principle distillation; the user-scope `~/.abcd/memory/` is a
**later phase**, which will hold personal preferences and cross-project
principles with no single repo home (nothing in the binary resolves it today);
and a root-level `memory/` is the legacy snapshot abcd respects and never writes
to. Which scope a curated page lands in is a routing decision — see
[`07-memory.md`](07-memory.md). Retrieval across the two is not a flat union, which
would overflow context, but keyword recall with a budget bracket (itd-39). When
the brief says "memory" unqualified, it means the repo-scope store.

**AI transparency level (staged).** A second axis, independent of visibility. It
has no backing intent or ADR, no shipped code reads it, and ahoy neither prompts
for it nor writes it. The shape the design commits to is three levels: `full`
captures conversations, plans, tasks and metadata; `metadata` drops the
conversation transcripts; `none` captures nothing. Visibility decides what is
*committed*; transparency would decide what is *captured at all*, which is why the
two are separate — a private repo might want metadata only to keep storage tight,
and a public project might want full capture for credibility. Any combination is
valid. Launch's pre-flight scrub strips absolute paths and PII whatever the level:
the level would determine what exists, the pre-flight decides what ships.

## 2. `.abcd/work/` namespace and `dev-sync`

**This whole section is staged.** No `dev-sync` verb is registered on the binary,
no Go source names one, and none of the adapters below is built. `.abcd/work/`
itself is real and committed; what is staged is the machinery that would populate
it by sweeping volatile sources.

The one thing that already works this way got there without a sync: `abcd capture`
writes the issue ledger directly, so the migration this design was drawn around has
nothing left to migrate.

The idea is a curated-from-volatile-sources namespace. Noisy inputs stay
gitignored or external; the lessons drawn from them get tracked; and abcd does not
have to re-read volatile sources every time. Four source-to-target pairs are
drawn: an opt-in agent-memory harvest into `.abcd/memory/`; ad-hoc oracle reviews
not tied to a spec into `.abcd/work/reviews/`, where the reviews charter already
governs the shape any sweep would have to meet; the local tier's notes into a
curated notes target; and an external tool's workspace state into its own
namespace (itd-7). Only the memory package exists today, and its shipped front
door takes a URL rather than a harvested directory.

Two triggers are drawn: an implicit one at disembark's first phase, and a manual
CLI refresh. Per-source enable flags would default to on for a private repo and off
for a public one. Scheduled sync comes in a later phase (itd-13).

> **Open question (adr-35):** the implicit trigger is stated against the old model,
> in which disembark packed *this* repo. adr-35 makes disembark read-only against
> the source, while a sweep **writes** into the source repo. The two cannot both
> hold. What must be decided: whether the sweep is dropped from disembark entirely
> and becomes a separate operator-run step, with disembark reading whatever curated
> artefacts happen to be present and none at all in a repo abcd never touched — the
> primary case — or whether the implicit trigger survives only where the source repo
> is abcd-managed and the operator opted in. adr-35 does not settle it. The question
> is not urgent while neither half ships, and it is the first thing to settle when
> either does.

Two curation rules are worth stating now, because they bound what a sweep may do.
Output is **not verbatim**: raw memories grow unbounded and carry personal
phrasing, so what lands is distilled and grouped by domain. And files in the local
tier are never moved or deleted: the sweep reads and curates, and the source stays
put.

**Reviews are a first-class pitfall source.** Plan, implementation and completion
reviews are exceptionally good at spotting issues, and the lesson survives even
where the original issue was fixed. A collator would extract every such finding as
a candidate pitfall for the distiller to dedupe against its other sources.

**Distinct from the lifeboat's own synthesis:** the curated tiers are persistent
rolling artefacts in the source repo, used as ongoing input to future agents. The
lifeboat's principles file is per-disembark synthesis written out of tree at the
operator-chosen destination, never back into the source. The lifeboat consumes
`.abcd/work/`; `.abcd/work/` is not the lifeboat.

## 3. Plugin shape — directory layout

A Go binary plus the markdown plugin surface that shells to it:

```
abcd/
├── .claude-plugin/                     # plugin.json + marketplace.json
├── cmd/                                # the shipped entrypoint plus six build-time binaries
│   ├── abcd/main.go                    #   entrypoint — wires the CLI front door to the core
│   ├── record-lint/                    #   the record gate `make preflight` runs (06-lint.md)
│   ├── scaffold-sync/                  #   keeps the scaffolded release workflows in step
│   ├── scaffold-render/                #   writes every scaffolded workflow profile for CI's workflow audit
│   ├── asking-sync/                    #   writes the asking rules into commands/intent.md's generated block
│   ├── abcd-gen-surface/               #   writes the command-surface snapshot and the surface chapters' appendices
│   └── abcd-gen-cli-ref/               #   writes the generated CLI reference page
│                                       #   The six are developer tooling, not user surface: they run
│                                       #   from the Makefile, `go generate` or CI, and ship in no release
├── internal/
│   ├── core/                           # transport-agnostic core, one package per capability
│   │                                   #   (adr-23); each returns structured results
│   ├── adapter/                        # the seam packages (adr-22): interface + native default
│   │   ├── scanner/                    #   + optional plug-in. The scanner is the one seam with a
│   │   └── gitleaks/                   #   package today; 02-adapters.md is the catalogue
│   └── surface/cli/                    # the only front door that ships (an mcp/ door is adr-23's third)
├── commands/<verb>.md                  # markdown command surfaces, flat; the gated list lives in
│                                       #   ../04-surfaces/README.md (abcd.md is the bare /abcd board)
├── agents/<name>.md                    # host-delegated agent prompts, plus per-agent fixtures/,
│                                       #   README.md and CHANGELOG.md — see 01-agents.md
└── hooks/                              # host event hooks; every event command runs through a
    ├── bootstrap.sh                    #   resolving shim. bootstrap.sh PROVISIONS the plugin-root
    └── hooks.json                      #   binary and never builds one
```

The seams that are planned rather than present are gated rather than trusted: the
`index_drift` record-lint rule holds every path in the planned-seams region of
[`internal/README.md`](../../../../internal/README.md) to being absent from the
tree, so a seam that ships cannot go on being described as planned.

**The hook wiring, and why it is shaped this way.** `bootstrap.sh` either copies a
re-verified artefact out of the persistent download cache, or resolves the latest
release tag and downloads the pinned asset, verified against that release's own
checksums; its build strings are printed instructions to the operator for the cases
it cannot provision. `SessionStart` runs one chained command rather than siblings,
because siblings would run in parallel and share one stdin.

`SessionStart`, `UserPromptSubmit`, `PreToolUse` and `PreCompact` are the events
that reach `bootstrap.sh` at all. The last three self-provision only when the
plugin-root binary is missing, throttled by a `.bootstrap.attempt` marker within a
ten-minute window, and then fall back to a PATH-resolved abcd that must be
absolute, outside the working directory, in a directory and a file that are not
world-writable, and recorded as this
machine's own, before failing loudly. `UserPromptSubmit`, which runs on every
message, declares a 120-second `timeout`; `PreToolUse` and `PreCompact` declare
none and take the host's ten-minute default; `SessionStart` declares 240 seconds,
and the transcript hooks declare none. Every event that runs `bootstrap.sh`
names a `statusMessage`, the host's spinner text while the hook runs, because the
salvage sends the script's output nowhere; the text states no duration, since
those limits differ. A test pins every event's timeout and the message
(`internal/surface/cli/hooks_timeout_test.go`). `SessionEnd` and
`SubagentStop` are the two
exceptions and download nothing at all: each fires where the harness cancels a slow
hook rather than wait — one as the session exits, the other inside a live session as
a sub-agent finishes — so a fetch there races that cancellation and loses the
transcript the hook exists to capture (iss-2608210934566223). Both resolve the
plugin root, then PATH, then say the transcript was not captured. `SubagentStop`
additionally never exits non-zero beyond that refusal, because exit 2 is the host's
BLOCKING code on that event.

`SessionEnd` carries `hook session-end`, which stages the session's own transcript;
`SubagentStop` carries `hook subagent-stop`, which stages a finished sub-agent's
transcript beside it with the lineage the harness payload and its per-agent sidecar
supply. `SessionStart` drains both through the fail-closed redaction path.

**The plugin-internal development namespace** (committed in private repos,
gitignored in public) holds, at its root, the config file and the per-surface
machine records under `config/`, the rules overrides, the lint and site
configuration records, the identity and positioning registry the surfaces are
held to, and the citation baseline `docs cite` maintains. Under `development/`
sit the durable record's families, flat by artefact type (adr-30): the chaptered
brief with its glossary, the intents
store with directory-as-status, the principles, the decisions, the roadmap with its
phases and RFCs, dated plans, the native spec store, the cold-reading ledger, the
release surface declaration, the release gate's manifest, research, and the
persona roster a press-release quote must attribute to. Under
`work/` sit the shared working files. The local tier is the third and is
gitignored under every visibility.

Three namespaces are deliberately absent from every tree: `voyage/`, which is
user-scope and never committed; `logbook/`, retired in favour of the local tier's
logs; and `lifeboat/`, because the lifeboat is out-of-tree output at an
operator-chosen destination.

**User-facing docs** live under `docs/` and are packaged into the release
artefact, split by Diátaxis type with a shared assets directory and the pinned
site-build toolchain. The native spec store, memory and config live under
`.abcd/`; plugin-internal design docs under `.abcd/development/`. The split is a
shape rather than any one tool's location convention.
