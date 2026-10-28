# `/abcd:ahoy` — Install / Update

Get abcd working in a repository, and find out the truth about whether it is
working. One command installs, updates and repairs, and it is the same command
either way: there is no separate upgrade path to remember, and running it twice
costs nothing. Read-only forms answer the other half of the question — what is
installed here, what is missing, and what no future install will fix.

The property everything else rests on: **idempotency is a property of detection,
not of a version stamp.** Every check compares actual state — the ignore block
as it stands, the entry on `PATH` as it resolves, the marker block as it reads,
the registry entry as it is — and never a recorded `setup_version` alone. So a
marker block a user hand-deleted is reported missing and repaired, even on a
repo whose stamp says it is current.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `connect` | — | shipped |
| `credential` | — | shipped |
| `doctor` | — | shipped |
| `install` | — | shipped |
| `remote` | — | shipped |
| `remote apply` | gate | shipped |
| `uninstall` | — | shipped |


Bare `/abcd:ahoy` shows read-only status and mutates nothing. Four read-only
modes of the same act — the dry run, the identity check, the remote report and
the provider board — are flags on the bare verb rather than sub-verbs, one at a time, and the
appendix lists them. A sub-verb is a distinct action, a flag a mode of the same
one (itd-2609212130136102); the modes' former sub-verb spellings are unknown
commands. The slash command dispatches every sub-verb and
mode but the identity check, the write verbs included, and each announces that
it writes before it runs. The identity check is a plain command-line
entrypoint, because its exit code is the whole point of it and its home is a
pre-commit hook or CI rather than a conversation. `status` is a word neither
surface takes: the plugin page passes it to the binary, which refuses it like
any other word it does not register, so `abcd ahoy status` is an unknown
command. The table above is the sub-verb set, and the modes are the bare verb's
flags.

- **Install** installs or updates abcd in this repo, covering first install
  and upgrade alike. It runs the detection pass, then an apply pass over the
  resulting gaps. Through the plugin page it is also how a repository is
  prepared to abcd's working conventions, a workflow the host agent runs around
  that binary step (below).
  Its only-waiting mode is the setup after an update
  (spc-2610170635097509): it puts only the questions this repository never
  answered and the settings whose default changed in a release since its
  setup, never one answered or deferred with decide later, approves every kind
  of change up front, and records the running release. Run with the
  approve-everything flag it answers no question, so each still waits.
  The yes to the offer of update's news sub-verb runs it; that sub-verb is
  the update chapter's ([21-update.md](21-update.md#sub-verbs)).
- **Uninstall** is reversible removal: the marker block, abcd's README parts
  (and a README.md abcd created, once nothing but blank lines remains), abcd's
  own `PATH` entry where abcd owns it, and the provenance record that proves
  that ownership. It leaves `.abcd/` intact but for clearing `readme.created`
  from `.abcd/config.json` once the README.md that record vouches for is gone,
  never mutates the hook manifest, and a later install re-installs cleanly.
  It finds the entry by scanning `PATH`, so an entry that was installed into a
  directory `PATH` does not carry is removed by naming that directory again,
  the same bin directory the install was given.
- **The dry run** renders the detection envelope as JSON and mutates nothing.
- **The doctor** runs the full detection pass plus a read-only audit pass. Its
  distinct contribution is the audit, and its distinct value in the text render
  is that it names, one line each, every required gap that is **not** resolvable
  — the ones no later install will clear, such as a config file abcd refuses
  to touch until a human repairs it. A bare count of those would be a number the
  reader cannot act on. It is the check to reach for after a repo rename, a
  machine migration, or "why aren't my transcripts showing up".
- **The remote report** reports, read-only, the GitHub-native secret-scanning toggles on
  the repository this checkout's own origin names, and the changes an apply
  would make. A toggle it could not read reports `unknown`, never `disabled`.
  The same request also reads the repository's merge hygiene, and a second
  reads its fork pull-request approval policy (whose pull-request workflows
  wait for the technical facilitator's approval before they run, one of the
  controls that bear on which workflows report a required check's name). abcd
  mirrors these and never sets them: they encode the technical facilitator's
  workflow rather than a posture abcd holds the repository to, and each is
  reported only when the API answered for it, because `false` and "the API did not say" are
  different facts (iss-2608270512210664). A policy read that fails otherwise
  than GitHub's 404 is said in a note, and an apply keeps the policy the
  mirror already records.
- **The remote apply** is **the one abcd verb that mutates state outside this
  machine.** See below.
- **The identity check** exits non-zero when the author or the committer a
  commit would carry does not match the identity the person recorded on their
  own machine, or is a machine identity in either role, recorded or not. Read-only, CLI-only, for an
  operator or CI.
- **The provider board** explains the optional OpenAI-compatible
  provider adapter (itd-2609081951381895): what an aggregator is, that abcd
  would use one for decision models and cheap judgements pointed at it by name,
  and that everything works without one, because with no provider configured
  every delegated step runs on the host. It lists the providers configured on
  this machine, whether each one's key resolves (never the key), the
  `oracle.denylist` entries written (abcd bundles none), the roles and judgement types pointed at a provider, and
  where a key can live, the keychain recommended in the prose and never as a
  marked option. The bare board carries the same explanation as an optional,
  advisory gap while no provider is configured.
- **The provider setup** sets one provider up, and writes. See below.

**Not built yet:** `destroy`, a nuclear uninstall that would remove the `.abcd/`
namespace too (itd-10), as distinct from the uninstall's reversible behaviour.

### The remote apply, the one outward-visible write

It enables GitHub's native secret scanning and then push protection, in that
order, because GitHub refuses push protection on a repository whose secret
scanning is off. It then mirrors the desired state into the repo's committed
settings mirror: a managed block holding the two toggles abcd drives, and an
observed block holding the merge-hygiene settings and the fork pull-request
approval policy it only reads.

It answers to adr-44 and invariant 10 — no uninvited remote mutation, through a
verb the user invokes **and** confirms — with four gates that refuse rather than
guess. The folder must be a repo abcd manages. The repository must be the one
this checkout's origin names. The repo's own config must not have opted out. And
the caller must confirm the specific toggles named, an unanswered run declining
and a pre-given yes being the explicit advance answer.

Exactly two statuses exit non-zero: refused, a gate abcd itself closed, and
aborted, a confirmation the caller declined, which a non-interactive run without
a pre-given yes reaches by reading end-of-file. A run with nothing to change exits 0,
whether that is an idempotent re-run or a repo whose own config declined,
because leaving the repo alone is what the repo asked for. Every request pins
the API host explicitly, so an ambient host variable cannot send the write to an
endpoint the origin never named, and the call goes through the caller's own
authenticated identity: abcd never holds a token.

That identity is the GitHub CLI's, so a missing `gh` is met with the
explain-then-install mode (itd-63): after the first three gates and before the
read, the verb explains `gh` from the tool registry and offers to install it,
making the registry's checked download only on a yes typed at a terminal. The pre-given
yes answers the settings change and never the install of a program, and a
piped answer is not a person's answer, so both decline the offer; the verb then
refuses, its notes carrying the explanation and the by-hand download. A failed or
unverified install refuses the same way, before any request leaves the
machine. The read never offers the install, because looking is never acting;
it names the apply as the verb that does.

### The provider setup

The setup takes the provider's name, its base URL, its first allowlist (every
model it may serve) and where its key lives. It verifies the provider with one
call to the first model listed and, only when that call succeeds, keeps the key
in the home chosen through the credential store's walkthrough and then writes
the provider block (base URL, the key's name, the models) into
`~/.abcd.noindex/config.json`. A failed verification writes nothing. Nothing reaches the
repository or the harness's settings. Every fault the configuration read would
refuse (a model an `oracle.denylist` entry matches, a malformed model, a base URL that is plain HTTP to
another machine, a provider already configured, a key name already holding a
different value) is refused before the call, so a setup that cannot finish is
never billed.

The key arrives on stdin and nowhere else. A flag would leave it in the process
listing and the shell history, the install prompter echoes every answer into its
transcript, and a host's question tool would put it in an agent's context. Piped,
it is read whole; at a terminal it is read on hidden input, echo off, after one
line naming the provider, and an interrupt during the paste restores the
terminal and writes nothing. For the same reason the walkthrough is this
sub-verb, which the person runs, rather than a question the install pass asks:
declining is not running it, and changes nothing.

Run at a terminal with no model named, the setup lists the service's models
with the key it holds (one request to the service's model list, which follows
no redirect and gives up after ten seconds), offers the names abcd accepts as
model identifiers in the plain-Terminal list, typing part of a name to narrow
it, and verifies the model picked with one real completion before anything is
written; a list is never taken as the verification. Off a terminal, a setup
with no model named is refused.

### The guided setup

Inside a session, the guided form of the setup works its values out with the
person one question a turn and ends by printing the one command, with every
path it writes, for the person to paste into a terminal
(itd-2610030821294016). It never sets the connection up itself, for any home:
it writes nothing, reads no key home, never asks for the key, and refuses every
flag that would set the connection up. Each turn returns one question, the
question as the host's question tool takes it, and a resume object the next
turn reads back; the guide replays every answer from the first question each
turn, so an edited resume object cannot skip one. It asks for the address when
none is given; offers to look up the service's models, showing the scheme and
host first and sending one keyless request only on a yes; suggests the models
the person's other connections already use that the service lists, at most
three, and narrows the list by the typed part of a name with no second
request, by the same rule the plain-Terminal list narrows by; carries the
first listed ids that fit 32 KiB and the count it does not carry, asking for
the model's full name when a typed part matches none carried; falls back to a
typed model name, saying why, where the service publishes no list; asks
whether the service takes a key and where the key lives, offering the
credential store's three homes and never a key saved in the system keychain by
hand; and, for an environment variable, offers at most three variable names
ending in `_API_KEY`, never a value. A service that lists its models only for a
key is picked from in the terminal step, so its command names no model.

The key lives in one of the credential store's three homes (below), and a
fourth answer, no key, sets up a local server that takes none.
A delegating verb whose agent's `oracle.roles` entry points at a configured
provider sends the step there itself (spc-2609251028149555), and a provider
whose block names a key takes only the self-contained agents under ruling DR5
of 2026-09-29, with `oracle.bundled_context_providers` as the person's
machine-only override; both the board and the setup say so in their `dispatch`
line.

### The credential store and its walkthrough

Every external credential abcd holds goes through one store
(`internal/core/credential`, adr-2609221017021499): configuration names a
credential, and the value lives in the home the person chose for it, once, in
the credential walkthrough at `ahoy`. Without a name, the walkthrough lists
every credential an adapter reads (the site setup's hosting token, each
configured provider's key) with whether it is set and in which home, never the
value; with a name and no home, it explains what the credential unlocks and
what works without it, then the three homes, the keychain recommended in the
prose above them and never marked as an option. Given a home, it runs: the
reading adapter's own verification call (the provider's one short exchange,
the hosting provider's account read) with the value, and only when that
succeeds, the write. The provider setup runs the same walkthrough for a new
provider's key.

The three homes:

- `external` — a setup outside abcd: an environment variable, or a dotted
  field of a tool's JSON configuration file under the home directory. The
  store keeps only the pointer, in
  `~/.abcd.noindex/credential-homes.json`, and follows it on every read. A file
  pointer is refused, naming the link, when any directory between the home and
  the tool's file is a symlink, wherever the link leads; the
  environment-variable pointer stays open.
- `abcd` — the owner-only `~/.abcd.noindex/credentials.json`, which holds the value.
- `keychain` — the platform keychain under the service name `abcd` (the
  Keychain through `/usr/bin/security` on macOS, the secret service through
  `/usr/bin/secret-tool` on Linux), the value handed over on stdin, never in an
  argument; `credential-homes.json` records only that the name lives there. A
  platform with neither tool refuses this home and names the other two.

One reader, `credential.Store(home).Resolve(name)`, serves every adapter; a
name no home holds is a refusal naming the walkthrough, and the caller makes no
call. A test walks the production tree for any other read (a store file named,
a keychain command run, a secret-shaped environment variable read). One write,
`credential.Set`, is reached only through the walkthrough: it refuses the abcd
home when `~/.abcd.noindex` lies inside a git working tree, since that home alone keeps
a value there, a name another home already holds, and a different
value for a name already kept, and the secret scanner reads the index's bytes
before they are written, refusing any finding. A value is read from stdin only,
and never printed, logged or written to a record; a call's record names the
credential it used.

## Preparing a repository: the host-run half of the install

Take a repository you own and bring it up to abcd's working conventions in one
sitting: the install's plugin page audits first, shows you the gap report, and
only then adopts the three-tier `.abcd/` layout, a working-conventions section
in `AGENTS.md`, an identity block, and the commit gates. What you get back is a
repo a fresh agent session can build and test from `AGENTS.md` alone. What it
costs you is a sign-off at each phase, which is the point: nothing is adopted
before you have seen what it would change.

This half was its own plugin page until the product thinker's ruling of
2026-10-09 folded it into the install (itd-2610090831227812): the retired
page's name is gone with no alias, and typing it at a shell gets a note naming
the install instead. The fold is page-only. The binary's install is
unchanged and gains no refusal: it checks no ownership, so the workflow's first
phase does, as before, and a refusal in the binary would break installs that
work today. The detection pass decides where the page starts: a managed
repository needs only the binary's apply pass unless the person asks to prepare
it, and an unmanaged one starts at the workflow's first phase.

The workflow is an **interim bridge**. abcd cannot yet manage repositories
directly, so the page does by hand what the CLI will later take over, in a
shape the CLI can adopt without unpicking. It runs in the host agent from the
install section of [`commands/ahoy.md`](../../../../commands/ahoy.md), always
on the current repository.

### What the preparation does

- **Refuses on repos the user does not own.** The first phase checks the origin
  remote and stops entirely: no audit, no writes. Imposing these conventions on
  a third-party repo would interfere with its own development principles.
- **Audits before it touches anything.** It produces a gap report covering
  existing structure, documentation shape, decision and working-state hygiene,
  principles followed or violated, and privacy, and presents it before adopting
  anything. A tool's own conventions file it finds, one an agent tool reads in
  place of `AGENTS.md`, is named as setup names it, as holding the owner's words
  or as only repeating `AGENTS.md`, and the report points at setup's offer to
  retire the second kind rather than acting on it.
- **Adopts the conventions.** The three-tier layout, with the shared tier's
  decisions folder seeded (one file per decision, and a README so git keeps
  the folder) and any single decisions log the repository already keeps left
  untouched as frozen history; a merged, never
  overwritten, `AGENTS.md` carrying verified repo facts plus a marked
  working-conventions block; a registered identity block; and the commit gates.
  `AGENTS.md` is the one conventions file it writes: it makes no other tool's
  conventions file, as a link or as a copy
  ([adr-2610030814023326](../../decisions/adrs/2610030814023326-agents-md-is-the-one-conventions-file.md)).

### Where the binary does the work

The workflow is markdown, but three of its steps are the binary's, so the
result does not depend on an agent's memory of what a convention looks like.

The JSON form of `abcd lint` supplies the engine-backed conformance core,
read-only, which the workflow then supplements with the structural and
principles judgement the binary does not make. The identity verb's initialiser
records the identity block: the markdown itself, which stays the source of
truth, plus `.abcd/positioning.json`, the pointer recording where that block
lives and which surfaces render from it. The install's apply pass is the adopt
phase's workhorse, and it does more than write the commit gates: everything
this chapter describes, from the repo's settings file to the copy of the binary
on `PATH`, lands in that one run. It runs a second time, with the attribution
flag, where the user opts in: that run installs the committed
`prepare-commit-msg` prompt asking every commit to declare whether a tool
assisted it, and the choice is recorded, so a later install without the flag
keeps the hook. The outbound flag is the same kind of opt-in, and one switch
for two checks (iss-2610152024327989): it installs the committed `commit-msg`
gate that refuses a live agent-session URL or a tool attribution footer in a
commit message, and it records `outbound.forge: true` beside `outbound.hook:
true`, so the shell guard also refuses a posting `gh pr` or `gh issue` command
whose title, body or body file carries either shape (`forge-text-outbound`,
iss-2610142210511831). The gate runs only the abcd binary the clone names in
`git config --local abcd.binary` and lets the commit through with one warning
when there is none (iss-2609250834251447); the explicit flag writes that key,
clone-local, to the `PATH` copy whose build provenance the same run verified,
and writes nothing, saying why, when there is no such copy, when the entry is
the dev shim or when it lies inside the working tree (iss-2610152024267547).
The flag never sets `core.hooksPath`, which chooses every hook git runs in the
clone; setup's own commit-checks question is the one place that does. So that the check runs with no manual step, the explicit flag alone also
writes the same embedded template, marked as abcd's, as the clone's own
`commit-msg` in the common git directory's `hooks/` (shared by every worktree),
where no hooks path is set at any scope and nothing occupies that path; it
never runs or calls the committed copy. Where a hooks path is set (the empty
value included), or a `commit-msg` hook abcd did not write is there, it leaves
that hook alone, saves the same template as `hooks/abcd-commit-msg`, a name git
never runs, and one note says the check is not wired, why, and the line that
calls that copy from the person's own hook, so the chained check fails open as
the template does; a rerun leaves a current copy as it is, refreshes one only
when it is byte for byte a template an earlier abcd wrote, and leaves an edited
one, saying so. The binary is not
re-verified at commit time: a swap of the `PATH` copy by the same account is out
of scope. The two flags that each turned on one half are retired and refused
naming the switch, and a repository that recorded one half under them, leaving
the other key absent, raises `outbound.half_on`, which the next install
completes; a key the repository set to `false` is its choice and raises only
the advisory `outbound.half_chosen`. The identity verb's render is the follow-on surface and writes
nothing: it proposes a correction as a diff, and adopting it is always the
product thinker's move.

### The flow

Four phases, each gated on the one before.

0. **Refuse unless owned.** Origin-remote ownership check; stop if it fails.
1. **Orient.** Read the abcd record from the plugin root: the three-tier
   README, the brief, principles, ADRs, intents, the `docs/` Diátaxis rules, and
   the lint configs as patterns.
2. **Conformance lint.** Run `abcd lint` in its JSON form, supplement it, write
   the gap report to the target's `.abcd/.work.local/scratch/`, and present it
   before any change.
3. **Adopt.** Create the three tiers with a repo-specific `CONTEXT.md`; migrate
   a historical `.work/` layout at the repo root if one is found, proposing the
   mapping and waiting for sign-off, and never leaving a repo with both homes;
   merge into `AGENTS.md`; scaffold the commit gates through the apply pass;
   register the identity block, adopting one the repo already carries rather
   than re-interviewing; and, only where the user says the repo requires AI
   disclosure, install the attribution hook.

A committed hook is not a running hook until git is pointed at it, once per
clone, and the page keys that step on the hooks-path state detection reports:
it arms an unarmed clone, leaves a foreign hooks dispatcher alone, and does
nothing in an armed one. A declined config change is reported as
scaffolded-but-unarmed rather than passed over silently.

When abcd's own record has conflicting sources, the workflow trusts a fixed
authority order: `AGENTS.md`, then `work/CONTEXT.md`'s live-constraints section,
then ratified ADRs, then everything else read for understanding only.

### The preparation's boundaries

- **Nameless, self-contained output.** The working-conventions block written
  into `AGENTS.md` never mentions abcd, the page, or any private repository:
  the conventions read as the repo's own, between dated markers so later tooling
  can find and replace them. The paths it states into the `.abcd/` layout are
  the one trace of the tool, and the adopter accepts that namespace by adopting
  the layout. The apply pass leaves the conventions files nameless as well
  (below).
- **Never commit downstream assets.** Anything tooling will later provide
  (persona data, lint-config JSON copied in by hand, content copied from the
  abcd record) is applied, not copied. Only content about the target repository
  is committed. The config files the apply pass seeds, `.abcd/docs-lint.json`
  among them, are not downstream assets: each is the repository's own once it is
  written, and the gates read it on every commit, so it is committed.
- **Privacy.** A `private-names.txt`, if present, is read-only context for the
  audit and never reproduced in any committed or published artefact.
- **No secret-pattern hooks.** No hook the workflow scaffolds carries a
  secret-pattern set or an absolute-path check. Absolute-path detection is the
  binary's own `privacy-hygiene` lint rule, which the audit phase already runs.
- **Self-contained.** Every asset the adopt phase applies resolves from the
  abcd record or from the binary, never from a path only one machine has: a
  step reaching for one would silently do nothing on any other machine
  (itd-162).

## What abcd manages — repos and `~/.abcd.noindex/`

abcd manages exactly one kind of folder, a **repository**, and keeps one
user-scope directory for machine-local state. Every writer creates that
directory, and each folder in it, private to the account (`0700`), so the home
is the same whichever command creates it first, the hook's bootstrap included;
one that already exists keeps its mode. The records it writes there, the
path-entry record and the registry's index among them, are `0600`, and one an
earlier version wrote wider is narrowed when abcd next writes it
(iss-2610032205304585).

```
~/.abcd.noindex/                       USER SCOPE — one per machine (machine-local state only)
  .gitignore                     one * line, so a git repository in the home folder
                                 never stages this folder; laid once, owner-only, and
                                 never laid again once deleted (itd-2610140858048004);
                                 an ignore file already there is left as it is and
                                 only the marker is written
  gitignore-laid                 the marker recording that lay
  history/                       the REGISTRY only: identity and lineage keyed on the
                                 root-commit SHA. ahoy owns it; it holds no transcripts
  transcripts/<root-sha>/        the redacted transcript corpus, a SIBLING of the
                                 registry, creating itself on first use
                                 (adr-2610031751066232, superseding adr-2609091248201071)
  voyage/<root-sha>/             disembark/embark operations log, never committed
                                 (adr-35)
  lab/<root-sha>/<lab-id>/       one lab's evidence: snapshot, probe records, findings,
                                 harvest; index.jsonl beside the homes registers them,
                                 never committed (itd-2609212137128014)
  worktrees/<root-sha>/<name>/   session and agent worktrees, never beside the checkout;
                                 the build loop creates each lane's worktree here today
                                 and ahoy reports one its repository no longer links;
                                 the store's own verbs and its reclaim are NOT BUILT
                                 (itd-2609091014076309)
  notes/<root-sha>/              the store's notes archive: a reclaimed worktree's
                                 .abcd/.work.local/, redacted, beside a manifest
                                 (NOT BUILT — itd-2609091014076309)
  runs/<root-sha>/               an autonomous run's shared state: the run log, one
                                 claim per record, one record per joined session
                                 (itd-2609221656373558)
  inbox/                         reports managed repositories filed back to abcd,
                                 <received-stamp>-<sender-key>.md; promoted/ keeps
                                 the ones filed as captures (itd-2609221656361680)
  interviews/                    the setup answers that change the machine (the
                                 status line, the machine's routing table), one
                                 record per install run that asked them
  backups/                       copies of the harness's settings file, taken
                                 before the status line changes it; the newest 10
                                 are kept
  config.json                    the machine layer of the layered configuration,
                                 read-only except for the provider blocks
                                 (oracle.api.<provider>) the provider setup writes,
                                 and the only file a provider block may sit in;
                                 that write holds .config.json.lock beside it
  memory.json                    the machine layer of the memory notes: the person's
                                 own notes, which agents write through the memory
                                 verb and which it removes at the person's request;
                                 each is tied to the repository it was written in
                                 unless written for every project. The prompt router
                                 recalls them beside the project's .abcd/memory.json,
                                 a person's reviewed commit, whose note wins an id
                                 both hold
  memory/                        curated cross-project memory pages (a later phase;
                                 the prompt router's recall reads them, read only)
  sources/                       the local sources corpus the library verb maintains
                                 and /abcd:library reads. Created only by its explicit
                                 init: absent means every other corpus verb and the
                                 page say so and stop
  load-limits                    the load check's per-machine limits (stray-minutes,
                                 extreme-load, wait-load), read-only; abcd never creates it
                                 (itd-2609231434459890)
  credentials.json               the credential store's abcd home: external
                                 credentials by name (a hosting token, a provider's
                                 key), mode 0600; only the walkthrough writes it,
                                 one new name at a time, never replacing a stored
                                 value, holding .credentials.json.lock beside it
                                 across the read and the write (itd-2609221017023290)
  credential-homes.json          the credential store's index: which names live in
                                 the keychain, and the pointer for each in the
                                 external home; never a value, scanned before it is
                                 written, mode 0600, .credential-homes.json.lock
                                 beside it
  rules.json                     the machine's rule conventions, the user layer
                                 between the bundled domains and each repo's
                                 .abcd/rules.json, read-only; abcd never creates it
                                 (itd-117)
  oracle-routing.json            the machine's model-tier routing table, written
                                 owner-only on the install's answered consent and
                                 left by uninstall (itd-2609170822093401)
  statusline.json                the status line's settings: the elements switched
                                 on and the command the harness ran before abcd
                                 took the row; written on the status-line offer's
                                 consent, left by uninstall
  path-entry                     the abcd copy this machine owns, the one PATH binary a
                                 hook will run
  cache-attestation              the bootstrap's record that the plugin data
                                 directory's cached binary matched the published
                                 release manifest, written only after that check
                                 passed, naming the release; install promotes
                                 the cache to the owned PATH copy only when the
                                 record agrees and its release is no older than
                                 release-mark
  release-mark                   the newest release this machine has installed or
                                 verified, raised by the bootstrap and recorded by
                                 `abcd update`; lowered only by `abcd update <tag>`
                                 naming an older tag (iss-2609151847117977)
  trusted-roots                  foreign-uid configuration roots the caller vouches for
  local-transcript-roots         checkouts whose transcripts are pulled in to
                                 <repo>/.abcd/.work.local/transcripts/ instead

<anywhere>/<repo>/             REPO — a single repository (the only install target)
  .abcd/                         repo-scope record + config.json + rules.json
  AGENTS.md                      marker block (stands alone), only where the docs target
                                 is agents_md; the one conventions file abcd writes
```

The same inventory is stated in prose under *The two `.abcd/` scopes* in
[`05-internals/03-configuration.md`](../05-internals/03-configuration.md#the-two-abcd-scopes);
the two are one list and must agree. The three declaration files at the bottom
are caller-controlled and line-oriented. `trusted-roots` and
`local-transcript-roots` are the two that widen what a session will trust, so
each is honoured only when it is a regular file this uid owns that no one else
can write, and a file failing either test is ignored with one line saying which
test it failed. `path-entry`, `cache-attestation` and `release-mark` are held to the same test,
by the install verb and by the hook shims that consult `path-entry`, and a record
failing it vouches for nothing.
`load-limits` is a setting, not a declaration, but it is read through the same
guard as the two that widen trust, and a file failing it, or holding a line that
does not parse, is reported loudly and all three of its limits take their defaults.
`rules.json` is read through that guard too, because it injects text into every
session on the machine, but a file failing it — or failing to parse — fails the
rules load outright: nothing injects until it is fixed, and the file is named on
stderr. None of these files is honoured behind a `~/.abcd.noindex` that is itself a
symlink, and nothing ahoy or the bootstrap writes there goes through one: each
refuses the link and names it, as the rules loader does for `rules.json`, while a
symlinked `~/.abcd.noindex` holding none of them reads as absent (the rule is stated once,
under *The two `.abcd/` scopes* in
[`05-internals/03-configuration.md`](../05-internals/03-configuration.md#the-two-abcd-scopes)).

There is **no workspace, host, or development-environment layer.** A folder a
user keeps their repos in groups nothing, and abcd does not privilege it. abcd
lives in one repository (adr-28): the design record is repo-scoped and in-tree.
Everything genuinely machine-wide lives under `~/.abcd.noindex/`, which an install
bootstraps transparently before registering, so a user is never blocked by
missing user-scope state. Each repo's marker block stands alone: there is no
inheritance chain to resolve.

The detection pass classifies the working directory into one of three kinds, and
the install acts on the matching kind.

| Folder kind | Strong marker? | `.git/`? | What the install does |
|---|---|---|---|
| `managed-repo` | yes | not consulted | the repo install flow, as an idempotent update |
| `unmanaged-repo` | no | yes | the same flow, after the install adopts it |
| `unmanaged-folder` | no | no | nothing to act on: reports and stops |

Classification keys on a **signal hierarchy**, and this is the part worth
holding: abcd-owned markers decide managed against unmanaged, and they settle it
before `.git/` is looked at, so `.git/` only separates the two unmanaged kinds
from each other. A strong marker is a registry entry for this root-commit SHA or
an abcd marker block in the conventions file. An in-tree `.abcd/` directory is
recorded as a signal and reported, but it does not make a folder managed on its
own (iss-88): a directory holding nothing but `.abcd/` reports as
`unmanaged-folder`. A `.git/` directory means the folder is *a* repo, not that
it is *managed*, and a folder carrying a marker block is treated as a managed
repo whether or not it is a git checkout at all.

Bare `/abcd:ahoy` **reports the kind and stops.** It never adopts an unmanaged
repo; it names the install as the way to do that. The two unmanaged kinds need
distinct tokens precisely because the offer differs.

## Architecture: one detection pass, four consumers

The install, the dry run, the doctor and bare `/abcd:ahoy` all run the **same**
detection pass and differ only in what they do with its output: the bare form
renders a status board, the doctor adds an audit pass and renders gap counts,
the dry run renders the envelope, and the install runs the apply pass over the gaps.
Detection logic lives in exactly one place, so those four cannot drift apart.

The detection pass produces an in-memory state contract, and it is a **value
passed between passes, never a file**: nothing on disk holds it, and no state
file is written at either scope.

What it probes, in behaviour rather than in step order: the folder's kind and
the plugin root; which **opt-in** scanners are on `PATH` (the native secret and
PII scan needs no external tool, so this step only reports what a deeper scan
would find available); the repo skeleton; the repo's identity, both its
root-commit SHA against the registry and the git author and committer a commit
would carry, each resolved as git resolves it (the environment override, then
the role's own `author.*` or `committer.*` key, then `user.*`), against the
person's own identity record on their machine and the list of machine
identities, and whether the repository still stores the identity pin it once
committed; the registry's own wiring; the ignore
block against the visibility policy; marker-block drift against the current
template; the `PATH` entry; the hook manifest; the recorded setup version; and
the two-layer name-guard scaffolding.

Three of those carry decisions worth stating outright.

**There is no gap for an absent transcript corpus.** The corpus creates itself on
first use, so "absent" is the ordinary state of a repo nobody has captured yet.
A gap there would have the board assert that transcripts will not be captured,
which is false (iss-95).

**The `PATH` entry is classified, not assumed.** Detection scans `PATH`,
resolving symlinks, and classifies each hit as abcd's own entry, the dev shim,
or a foreign binary. An abcd-owned entry anywhere on `PATH` is the install; with
none, the default location answers the same question. A symlink whose target
has gone is abcd's own when it is the one a plugin update stranded or when the
home-scoped `path-entry` record names it, read exactly as the hook shims read
it; any other dangling link asserts no provenance (iss-2609100506263330).
Three states are named
rather than lumped together: an owned entry whose target has gone is dangling; an
install directory absent from `PATH` is required but not resolvable, carrying
the one-line export fix, beside which a person at a terminal is offered the
line added to their shell settings (below); and any
`abcd` that comes *before* abcd's own entry is shadowed, because an entry that is
correct and never reached is not an install (iss-171). A link whose target has
gone is the exception to "never reached" in wording, not in the gap: it runs
nothing, because the shell skips it, and what it still threatens is to answer
whatever reappears at its target, so neither the gap nor the note says it is what
runs. An owned one that is not the entry install acts on — typically a link a
plugin update stranded ahead of the one-liner's copy — is removed with its record
by an install that leaves a working entry of abcd's own behind it
(iss-2609280932480608), the same danglingness rule that clears one at the target
(iss-2609100506256636); an unowned one is named and left. Install carries the two
non-resolvable ones on its own result as notes, since a fresh user cannot run
the doctor by name on a machine where abcd is not yet on `PATH`. A foreign
regular file at an entry abcd would write, or ahead of its own, is described
rather than only named: its size, when it was last modified, and whether its
embedded Go build metadata identifies it as an abcd build and of which version,
read without running it, so the person deciding whether to clear it need not
inspect it by hand (iss-2609120447482255).

**Setup asks before it puts abcd's folder on the `PATH`.** When the folder the
entry is in is not on `PATH`, install asks a person at a terminal, once, whether
to add one line to the startup file their login shell reads, its one line
naming the line and the file (``Add `export PATH="$HOME/.local/bin:$PATH"` to
~/.zshrc?``), and `?` saying the rest, that the folder goes ahead of the rest
of the `PATH` among it (`path.shell_settings_offered`, adr-2610241022330872,
itd-2610240919030303). zsh gets `${ZDOTDIR:-$HOME}/.zshrc`; bash gets the login
file on macOS (the first of `.bash_profile`, `.bash_login` and `.profile`, else
a new `.bash_profile`) and `~/.bashrc` on Linux. On a yes abcd keeps a copy of
the file in `~/.abcd.noindex/backups/`, adds
`export PATH="$HOME/.local/bin:$PATH"` at the end with abcd's marker comment
after it, reads the file back, records the line in `~/.abcd.noindex/shell-path.json`, and
says to open a new terminal. It edits only a regular file it can read whole,
owned by the person and writable by no one else. Settings that already add the
folder are left alone, and a file that is a link kept elsewhere (shared
dotfiles) is never offered: its line goes on the checklist, as it does on a no,
under the approve-everything flag, and off a terminal. Uninstall removes the
line abcd added and nothing else in the file. The `path.bin_dir_not_on_path`
step says what is true of those settings, as the plan reads them: where a file
the shell reads already adds the folder (the person's line, or the one a yes
just added) it is to open a new terminal, because the shell reads the file
when it starts; where abcd will not edit the file it is the `export` line with
the reason; only where setup will ask does it name the question. Only abcd's
own marked line for this folder, compared whole, counts as abcd's: the marker
on a line that adds another folder adds nothing. An edit after a yes that
fails is a step on the report's checklist, the copy to put back by hand where
abcd could not restore the file.

**The name-guard scaffolding is reported at the granularity the technical
facilitator can act on.** Each absent artefact is a gap abcd will create; every other state is a
diagnostic, because abcd writes what is missing and never replaces what the
technical facilitator put there. A pre-commit guard present without abcd's own marker line is
foreign, and is reported rather than claimed as installed. A lint config with no
usable banned-names array, one that cannot be read, and one git ignores — so CI
never sees it, the state a public repo is in by default — are three distinct
diagnostics with three distinct remedies. The private stub's gap is resolvable
only when **git itself** reports the path as ignored, not when the ignore file's
text looks right: a stub git would track is the hazard the layer exists to
prevent, so a gap the apply would refuse to close is never advertised as
resolvable. The same pass always reports the private layer's **reach**, because
CI cannot enforce it and neither hook sees a fast-forward pull, a rebase, a
patch application, a revert, a cherry-pick, or a commit that skips hooks. See
[`20-banlist.md`](20-banlist.md).

### The hook manifest is verified, never written

Install verifies that the hook manifest is present in the plugin install and
carries the prompt-router entries it expects. Neither install nor uninstall ever
mutates it: the manifest is plugin-static. A missing or malformed manifest
surfaces as a non-resolvable diagnostic.

The shipped manifest wires ten event types, and every event command is a
resolving shim rather than a plain binary call. Six of them self-provision.
`UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `PermissionRequest` and
`PreCompact` each attempt
`hooks/bootstrap.sh` only when the plugin-root binary is missing, recording the
try in a `.bootstrap.attempt` marker that throttles the next one to a ten-minute
window. `UserPromptSubmit`, the hook that runs on every message, declares a
120-second `timeout`, the time a salvage on a slow link has before the host
cancels it; `PreToolUse`, `PostToolUse`, `PermissionRequest` and `PreCompact`
declare none and take the host's ten-minute default, so a slow first download finishes there. `SessionStart`
declares 240 seconds, the script's worst case with room to spare,
`SessionEnd`, `SubagentStop` and `StopFailure` declare none, and `Stop` declares
30 seconds, room for the away mode's stop check to make its one forge listing
under its own ten-second deadline and its git reads. Every event that runs the script,
`SessionStart` included, names a `statusMessage` the host shows as its spinner
text while the hook runs, so the wait is never a silent stall: the salvage
itself sends the script's output nowhere. The message states no duration,
because the limits of the events that show it differ. `SessionStart` runs the
script once at the top of every session instead,
whether or not the binary is already there, and relays whatever it says: with
the binary in place the script's own fast path costs a file test and does the
provisioning housekeeping that keeps the next plugin update served from the
local cache rather than the network, and it is the one place a binary that no
longer matches its provenance record is called out. It stamps the same marker,
so the five throttled events see a recent try, and reads no throttle of its
own. `SessionEnd` and `SubagentStop` are the deliberate exceptions and download
nothing: both fire where the host cancels a slow hook rather than wait — one as
the session is going away, the other inside a live session as a sub-agent
finishes — and a mid-flight fetch loses the very transcript the hook exists to
capture (iss-2608210934566223). Each resolves the plugin root, then `PATH`, then
says in one line that the transcript was not captured. `SubagentStop` never
returns a non-zero code of its own beyond that refusal, because exit 2 is the
host's BLOCKING status on that event and would stop the sub-agent finishing.
`Stop` and `StopFailure`, which carry the away mode's stop check
(spc-2610180914469489), download nothing either: both fire at a turn's end, where
a fetch would hold the session, and each resolves the plugin root, then `PATH`,
then says in one line what did not happen, with exit 1, never the `Stop`
event's blocking 2: `Stop` that the stop check did not run, `StopFailure` that
a turn ended on an API error was not recorded.

**The `PATH` rung is owned-only** (GHSA-gx3m-3224-qqcv, CWE-426). It accepts only
an absolute resolution out of a directory that is neither under the shim's
working directory nor world-writable, naming a binary that is not itself
world-writable once a symlink is followed to the file it names — the shapes the
documented install never produces (iss-2609012039117381, iss-2609020352438590)
— and only when the home-scoped `path-entry`
record names that exact path as this machine's installed binary. The record is a
string comparison and no hashing, because adr-46 keeps the fast path at one file
test. Both install routes write it, and the ahoy installer writes it for **every**
entry shape it leaves on `PATH`: the owned copy, the dev shim, and a working
pinned symlink into the plugin root that an earlier release wrote. The installer
never writes that symlink itself: with no verified artefact to copy from it
writes no entry, because a link into the plugin root dangles at the next plugin
update, and it names the install one-liner as the command to run first — the
one route that fetches and verifies the release binary on an explicit ask
(adr-38) — after which a re-run adopts the copy in place. A pin into the plugin
root is a required `symlink.legacy` gap whatever the cache holds (iss-2609100506263330). An entry
the record does not name is an install this rung refuses, and it is the one
state where a filesystem test alone would call the install healthy while every
hook quietly degrades, so the board raises it as a gap in its own right and
names the fix (iss-2609091126475539).

Recording the dev shim does not widen the rung. The record is home-scoped and
written only by an install the operator ran themselves, which is exactly the
distinction the rung draws: a checkout the session merely reads may not supply
the binary, a binary the operator installed may. What ownership *rests on*
differs by entry, and the copy predicate says so: the owned copy is the record
plus a byte-for-byte hash match, and explicitly not the shim, because `abcd
update` reads that predicate as permission to overwrite the file. An owned entry
the record does not name is its own gap, required and resolvable, because an
install with no actionable gap never builds an apply context and so could not
otherwise heal one; uninstall drops the record with the entry it names, and only
that one. Anything else is ignored with one line naming the binary and the
reason, and the shim degrades. For the pre-tool-use guard that degradation is an
unguarded line and exit 1, never the exit 0 the host reads as approval. Session
start carries no `PATH` rung at all and fails closed.

## The setup flow: basics, one plan, the person's own questions, three-part report

Setup is short, plain, and finishes (itd-2610240823055889,
[adr-2610241101463288](../../decisions/adrs/2610241101463288-setup-asks-short-questions-and-lists-its.md),
which amends brief invariant 1 for setup). It checks everything before it asks
anything, in this order:

1. **The basics.** No `git` command, a folder that is not the top of a git
   repository, or an account with no agent harness stops the run before its
   first question, with one message naming what is missing, writing nothing
   and exiting 2 (`setup_basics.go`). Every other missing basic, a folder not
   on `PATH`, a required tool, is a step on the report's checklist.
2. **The plan screen** (`setup_plan.go`). At a terminal, and through an answers
   file (`approve.plan`), setup shows one plan: every change grouped by where
   it lands, every default it takes with where to change it (visibility,
   docs target, deep scanning, the house-style rule, the artefact kind, the
   model routing tables, the drain rule, the README parts, the status line's
   parts, all on), and the person-only questions that follow. It asks once:
   proceed, customise or cancel. Proceed answers every approval of a kind of
   change and takes every listed default without asking; customise asks the
   approvals and the values in turn, as before the plan; cancel writes
   nothing. For a repository abcd does not manage, the plan is asked before the
   adoption and proceeding adopts it. A piped answer stream, the
   approve-everything flag, an explicit category set and the after-update
   run are not shown the plan.
3. **The person-only questions** (`person_only.go`): the abcd download, the
   shell settings, the commit checks (`commit_checks.go`), the status line,
   the commit identity and each tool abcd offers to install, each its own
   question naming the value and where it is written, whatever the answer to
   the plan, asked in that order and listed by the plan only where the run
   will put it: the shell settings only where the run leaves an entry, a tool
   only where abcd offers its own copy. The identity is never listed under,
   or declined by, the settings approval (iss-2610240824005655), and at a
   terminal no `dependency` approval is asked, so customising never drops a
   tool's question. At the stdin door's plan screen a bare Enter is no answer
   and asks again.
4. **The abcd download** (`abcd_download.go`). Where nothing is at the `PATH`
   target and the plugin's verified cache cannot supply a copy, setup asks to
   download the release this abcd was stamped as (in a source checkout of
   abcd, the newest release tag the checkout holds, read from disk), before
   the status line; on a yes it fetches the release binary through `abcd
   update`'s checked fetch (`update.FetchVerified`: `checksums.txt`, then the
   signed build provenance), writes it to the target and records it in
   `~/.abcd.noindex/path-entry`, so the status line is wired in the same run.
   The status-line offer is left out of the approvals, and never asked, when
   it cannot be wired (iss-2610240823574047), and its entry is a release copy,
   never the track-latest dev shim.
5. **One line each** (`setup_brief.go`). Every question is drawn on its one
   line with its answers' labels; `?` draws the whole question and asks again
   (`ask.ErrExplain`). The answers record keeps the whole question.
6. **The report** (`setup_report.go`): done, one line per change with where;
   needs you, a numbered checklist of exact steps, each with why; next, one
   step. A change the person approved that then failed (a tool or abcd
   download that did not verify, a shell-settings edit, the commit checks) is
   a needs-you step with its reason and the step by hand, not only a note. A
   repository already fully set up is one line, and a details flag adds the
   full record.

Run from a source checkout of abcd through `go run ./cmd/abcd`, setup is a
build of that checkout as it stands, so the stale-binary refusal treats it as
current (`gorun_vintage.go`, iss-2610240824047504). With no plugin root, the
copy `~/.abcd.noindex/path-entry` records still reports its install mode
(iss-2610240916382802).

## Gaps, and how the apply pass asks about them

Each detected discrepancy becomes a **gap** with a stable id, a category, a
scope, a title, detail and a fix hint. Under customise, and off a terminal,
the category is what the apply pass asks about, one question per category
present, never one per item. The question
shows what approving it would do (iss-2610071528375981): one line per change,
the gap's title, which names the file or the setting, with a value flag that
would change a saved setting listed under the settings approval; then the
question in plain words ("Create the files and folders listed above?", "Change
the settings listed above?"), never the category's internal name. Yes says how
many listed changes it writes, or, for a category whose changes are each asked
again (a tool, an offer, a conventions file), which questions it goes on to;
the settings approval's says it writes the changes that need no answer (the
`PATH` entry) and asks for each value not yet chosen, saving
none if one is left unanswered. Every change is listed on every route, never cut to
fit, so every door records the same question; a list taller than the host's
rows is shown with the question check's note rather than refused. The question check refuses an answer
whose meaning points at text above a question that has none, the shape the
approvals had before they listed anything.

| `category` | Examples | Apply behaviour |
|---|---|---|
| `safe-autocreate` | the repo skeleton, history-store directories, the name-guard artefacts | applied once the category is approved, no per-item prompt; create-if-absent, never overwriting |
| `config-change` | visibility, oracle adapter, the `PATH` entry, the artefact kind (the repository's own git identity carries this category but is its own question, never listed under or declined by the settings approval) | transparent confirm; skip-if-set with a "current value" notice |
| `machine-identity` | the person's own commit identity record on their machine; moving the identity pin a repository still stores to its owner's machine (itd-2610240918547728) | never a category approval: each is its own question, asked only at a terminal, written only on that yes; never under the approve-everything flag or a piped answer, and reported as optional work it skipped |
| `plugin-owned` | the marker block (itd-3); hook-manifest verification | silent overwrite on marker drift; a non-resolvable diagnostic for a malformed or missing manifest, for a conventions file whose block would land inside a fence or HTML comment nothing closes (`marker.unplaceable`), for one that exists but cannot be read whole, a folder, a pipe, a file past the size cap or one without read permission (`marker.unreadable`), and for one whose block is missing or out of date in a folder that does not let abcd create a file there, which the write needs for its lock and temporary file (`marker.unwritable`) |
| `dependency` | a tool a capability uses with no copy abcd can run: gitleaks, optional over the native secret scanner and required where the repository armed it in `.abcd/config/gitleaks.json`; missing, or installed some other way at a version outside the range abcd works with | at a terminal each tool is its own question, asked last of the person-only questions, and no category approval is asked (so customising never drops it); off one the category approval reaches the step; each tool is then explained from the tool registry (what it is, optional or required here, what works without it, the download abcd's release pins and how far it is checked, any copy already installed and which copy will run, what the install does) and its download is made only on a per-tool yes — typed at a terminal, or relayed by a host as a flag naming the tool — never under the approve-everything flag, a piped answer or CI; a compatible copy already installed is used and nothing installed; a no is reported as what the capability continues on |
| `status-line` | the offer of abcd's status line in the host harness | an advisory offer asked after its own question, written only on an answered consent; never under the approve-everything flag, and reported as optional work it skipped |
| `oracle-routing` | the offer of abcd's proposed model-tier routing table (itd-2609170822093401): the machine's `~/.abcd.noindex/oracle-routing.json`, then, as a separate question, the repository's `.abcd/config/oracle-routing.json` | the proposal said in counts (how many agents, how many at each tier, their fan-out bounds), naming no agent so the question fits, and each file written only on its own answered consent, the machine one owner-only; never under the approve-everything flag, and reported as optional work it skipped; a decline records nothing, so the next install offers again; uninstall leaves both files |
| `drain-rule` | the offer of the repository's drain eligibility record (ruling BX2, itd-82): abcd's strict baseline as an accepted decision record carrying the four `drain_` fields, minted through the decision store's seam | the rule stated in one question and the record written only on a consent answered at a terminal; never under the approve-everything flag and never off a terminal, where neither its category nor the offer is asked (so a piped answer stream keeps its order), and reported as optional work it skipped; a decline records nothing, so the next install offers again; raised only while no accepted record states the rule, so a record stating it badly is never offered a second; only ever the baseline, never a loosened rule |
| `conventions-file` | an agent tool's own conventions file at the repository root, read in place of `AGENTS.md` (itd-2610030814013772): `CLAUDE.md` and `.claude/CLAUDE.md` for Claude Code, `GEMINI.md` for Gemini CLI at its default settings, `.rules`, `.cursorrules` and `.github/copilot-instructions.md` for Zed, each classified with one guarded read inside the project and never through a link, and named as it is spelt on disk | a file holding the owner's words (or one that cannot be read whole) is never edited, moved, merged or removed and raises the warning `conventions.owner_file`, which install reports in `warnings`, printed first; a file that only repeats `AGENTS.md` (a link the system resolves to the root `AGENTS.md`, or, before there is one, a link whose target names it; that link's target saved as text, an exact copy, a lone `@AGENTS.md` line, or one blank once abcd's block is stripped; a dangling link's target, a saved target or an import counts only in its plain spelling, `AGENTS.md` or `../AGENTS.md`, never one stepping into a folder and back) raises the optional `conventions.retire_offered`, asked after the drain rule as one question per file (`retire`, `keep`, `later`, the default) and only at a terminal, as the drain rule is; on `retire` the file is classified again and removed from the working tree only if it still repeats, else left and named, and the removal takes the entry it checked, in its folder held open, so a folder swapped for a link cannot carry it elsewhere; keep and later record nothing, so the next install asks again |
| `hooks-path` | pointing git at the committed hooks in this clone (`banlist.hooks_unarmed`, iss-2610241134315056): abcd's guard is committed and no hooks path is set at any scope | its own question at a terminal, never an approval: a yes sets `core.hooksPath` to `.githooks` in the clone's `.git/config` and reads it back; off a terminal, under the approve-everything flag or on a no, the report lists the command, and the install is never reported clean while the hooks it committed do not run; a hooks path set outside the clone, or a local one naming another directory, is never replaced |
| `readme` | the README question (itd-2610031348087517), raised as the optional `readme.offered` while `readme.parts` is absent from `.abcd/config.json` | never asked as a category: the question pair is the consent, asked after `stepMarker` and before the status-line offer, only of a person at a terminal, in a run without the approve-everything flag, and only once `.abcd/config.json` exists; every other run reports the gap under optional work it skipped and names the install flag that answers the pair, which it does in any run; the chosen parts are written into `README.md` on every setup run (below) |
| `user-state` | the registry entry, re-founding, stale or duplicate entries | guided; never auto-edit user-scope state, report extras read-only |

**A docs-lint root that does not resolve is a report-only gap**
(iss-2610100649479892). Each `roots` entry of `.abcd/docs-lint.json` that names
nothing on disk, judged as the documentation check judges it, raises one
non-required, non-resolvable `config-change` gap, `docs_lint.root_missing`,
whose fix hint names the file to edit and the entry to take out (or create).
The check refuses to run while one is listed, so every rule it carries, the
banned names included, checks nothing. The usual cause is a `CLAUDE.md` retired
after it was listed, by the conventions-file offer or by hand: the offer
removes the file and nothing else, and `.abcd/docs-lint.json` is the person's
file, which setup reads but never edits once it exists. A missing or unloadable
config raises nothing here, and an entry that is not a plain path inside the
repository is not looked up: the check's own refusal names both.

**The README question pair** (itd-2610031348087517, spc-2610031704338417).
Setup asks whether abcd keeps parts of the project's `README.md` current and,
on a yes, which. The first question, keyed `readme`, stands alone with `yes`,
`no` and `later`, the default. `yes` asks one question per offered part, keyed
`readme.<part>` in the fixed order `release-badge`, `build-badge`,
`abcd-badge`, `licence`, `links`, `contents`, each answered `on`, `off` or
`later`; the two forge badges are offered only where `repo.visibility` is
`public`, the owner's declaration, and a line above the parts says so where it
is not. The defaults are on for the licence, the links and the offered forge
badges, and off for the table of contents and the managed-with-abcd badge
(decision 2). Each part's question carries, from core's help (`readmeHelp`,
reached through `HelpIn`), what the part is, its current setting, the
change-later line naming the install flag that answers the pair, and the exact
lines it would write in this project or the reason it writes none, which the
terminal prints through `HelpedPrompter` and the plugin page relays from the
`readme` view the dry run carries while the gap is open, its `readme_tool`
calls holding the first question alone and then the parts as tabs of at most
four. The answer is `readme.parts`: a list, which every later setup run derives
and rewrites; `[]` for `no`, never asked again, and abcd's parts already in the
file removed where their text is still abcd's; nothing for `later`, at either
question, so the next run asks again. That flag takes a comma-separated list of
part names or `none`, refuses an unknown or empty name, or `none` beside a
name, with exit 2 before anything is written, and answers the pair in any run,
the approve-everything flag included. An answer that changes the stored set is
echoed as a change (`readme.parts: licence -> licence,links`), and
`readmeWouldChange` keeps an otherwise current repository from returning early
while a part would be written, rewritten or removed. Without
`.abcd/config.json` (the starter files declined) the answer has nowhere to be
stored, so the pair is not asked, the flag's answer is refused into the notes,
and the summary names the starter files first (iss-2610120505551170).

**The artefact kind is a gap until it is declared** (itd-2609150819432059). A
managed repository with no `.abcd/config/artefact.json` raises a required,
resolvable `artefact.missing` gap, because the launch verbs choose what to
preview, check and scaffold by the kind declared there and refuse to guess it.
The apply pass writes the file once config changes are approved: a repository
carrying `.claude-plugin/plugin.json` takes `kind: plugin` without a question,
so the shipped shape adopts silently; any other is asked its kind, last of all
the install's questions. An unanswered prompt takes `application`, the kind that
assumes least about the build. An unattended install is not asked, and an
answer naming none of `plugin`, `binary` and `application` is not refused: both
declare `application` with a note saying what was heard, as the house-style
question does, because withholding the declaration would leave every launch verb
refusing the repository. The file is validated by
the one reader the launch verbs share, before it is written and whenever it is
read, so a declaration that is present and refused raises a non-resolvable
`artefact.invalid` diagnostic instead: it is the user's file, and the install
never overwrites it.

**The questions come in a fixed order**, and the order is a contract rather than
a presentation choice: answers are positional, so without it the Nth piped
answer approves a different category on every run — a wrong answer that exits 0
and reads as a clean install. One line answers one question, so a caller must
supply one per category present, which is why `yes` piped in is the reliable
form: it never runs out.

**Answers arrive from stdin whether or not stdin is a terminal**
(iss-167). The prompts are the same prompts; only the reader differs. At a
terminal a human types them; off one, a caller pipes them, which is how a host
agent drives the approvals no flag covers. Off a
terminal each answer is echoed to the diagnostic stream, so a piped run leaves a
transcript rather than a column of questions with no visible reply.

**At a terminal the questions are drawn** (spc-2610030911534855,
itd-2610030810370060). When stdin, stdout and stderr are all terminals the
install asks through the drawn door instead of the line reader: each value
question and each approval is built as the shared question type from core's
own words (`ahoy.SetupValueQuestion`, `ahoy.SetupConfirmQuestion`), chipped
"Setup Q<n>" in the order setup asks it (no total, since the gaps decide how
many are asked), and put through the answer loop on stderr: arrow keys first,
a number, or decide later, which answers nothing, so a config value stays
unset and its gap listed while a question with a default takes it, as an
unanswered question does. Ctrl-C ends the run with exit 130 and keeps the
answers given before it. Off a terminal the line reader stays, unchanged, and
an answers file named on the command line answers the questions wherever the
install runs: each question is written as plain text (the drawing in Mono at
80 columns, no escape byte), and one the file does not answer stops the run
with exit 2, naming the question's id, the flag that answers it and the file
line that would, with no answers record written. The file is checked before
the first write: an entry whose question has a fixed set of answers must
give one of them, and every config value the run would ask must be
answered, by its flag or by the file, walked in the order and as far as the
install asks them (`ahoy.WalkConfigValueQuestions`), so either refusal says
that nothing was written. The adoption and the approvals are asked before the
first write too. A stop at a question the run itself decides to ask (the
status line's elements, the offers, the artefact kind) leaves the steps before
it done, and says so. A file never answers the questions put only to a person
at a terminal (the git identity, the drain rule, installing a tool).

**Every answer is recorded with where it was given.** The drawn door and the
answers file write one answers record through `interview.Write`: per question,
the sanitised question as asked, the value, the note and `answered_in`,
`Terminal` or `Claude Code` and nothing else. The drawn door stamps
`Terminal`; an answers-file entry carries its own, or takes the place the
run names for its file (`Terminal` unless it names another), which the plugin
page names as `Claude Code` on the host path. The repository's answers go to
`.abcd/.work.local/interviews/setup-<stamp>.json`, never creating the local
tier; the machine-wide ones (the status line, the machine's routing table)
to `~/.abcd.noindex/interviews/`, made through the guarded home-scope maker. The stamp
is in the name only, so two runs given the same answers write the same bytes
and the same configuration, whichever door asked. The line reader writes no
record, and neither does an install that ends aborted (the adoption declined)
or refused, which changed nothing.

**Every value question carries its own explanation** (iss-163). A question that
picks one of several values (the repo visibility, the docs target, the
deep-scan toggle, the house-style question and each status-line
element) is rendered with core's canonical help above it: what is being
decided, then what each answer means, including what it asks of the person in
keys, tools or cost. The oracle backend is not asked while host-delegated is
the only answer with an adapter, because a question with one defensible answer
is not put to a person: the install records host-delegated and says in one note
that other reviewers arrive later, naming the install flag that chooses one then.
The question returns on its own once a second answer has an adapter, and its
help, kept for that day, defines an oracle and says plainly that every answer but
host-delegated is recorded without changing how reviews run
(iss-2610031236155833). The words live in core, so every
front door shows the same explanation and none invents its own; the question
line itself is unchanged, so a piped answer stream lines up with it. The four
config values' help also carries the install flag that answers the question
without asking it, and both the question and the missing-value gap's fix hint
name it, because a flag is the reliable answer in a piped run
(iss-2609120447486547). A question no flag answers, such as the artefact kind,
says instead where its answer is changed later, as its change-later line, so its
explanation need not repeat it (iss-2610031236155833). The explanation is the one
for the repository the install runs in: core gives the front door the help for
that repository, and the public visibility answer's caveat, that git cannot hide
records it already tracks, is part of it only where `.abcd/` holds tracked
files, the evidence on which the install narrows the public ignore block. Approving every kind of
change up front chooses no value, so a run approved that way that still has a
value to ask says so once, in core's words, above the first value question.

**The result explains itself to the person who ran it** (iss-164). Beside the
exact record (every write, change, note, declined category, outstanding step and
optional step left undone), the install returns a one-sentence headline for its
status and a plain-language summary: one item per kind of write, per declined
category, for the required work still outstanding, and per optional step left
undone, each saying what it is, why it matters and what, if anything, to do, and
naming the paths or identifiers it explains. The words are core's, written for
the product thinker and the technical facilitator rather than abcd's
implementers, with no raw environment names; the text render leads with them and
prints the exact record after as detail. Ahead of the headline come the
`warnings`: one line each for what keeps abcd's rules from an agent tool in this
project and only the person can end, such as a tool's own conventions file that
holds their words (the `conventions-file` row above). The text render prints
them before anything else, and every outcome past detection carries them, the
early returns included.

Install, and install alone, adds the host-reach warnings after them
(`conventions.host_reach` and `conventions.host_version`, never resolvable and
never a refusal): neither `Detect`, which the board and the hooks call, nor
`Managed`, which the status line calls on every refresh, makes them, because
the check walks every folder above the project and starts the agent tool,
which a call made on every prompt or refresh must not. That is why the
owner-file warnings, which ride detection's gaps, also appear in the dry run
and the doctor report, and the host-reach ones do not. Install makes them only
once the run is past its refusals and its adoption question, so a run that
refuses or is declined starts nothing. A `CLAUDE.local.md` at the project root
is named and never read, classified or offered for removal. From the root's
parent up to the file-system root, each folder is asked with `lstat` alone for
`CLAUDE.md`, `.claude/CLAUDE.md` and `CLAUDE.local.md`, skipping the user-level
`.claude/CLAUDE.md` in the home folder but naming a `CLAUDE.md` directly in it.
A `.claude` folder that cannot be searched hides only its own `CLAUDE.md`, and
the walk goes on past it; a folder whose own entries cannot be looked for ends
the walk quietly. Each path is shown through `fsutil.RedactHome`, and each
warning carries one fixed sentence: "abcd reads no settings from
folders above this project. This check only asks whether a file of this name
exists there; it reads nothing in it and changes nothing abcd does, because the
agent tool itself reads that folder." The presence check is not a read of
configuration from above the working tree, which the loader's rule forbids: it
takes nothing, so the most a hostile file above the tree can cause is a
warning. The version warning asks the `claude` command on `PATH` once for its
version (`host_version.go`), through the runner's one launch primitive for a
vendor binary (`runner.Exec`): the command is refused when it resolves to a
relative path or inside the project, or when group or other can write it or a
folder it is reached through, and it runs with the git-scrubbed environment,
each stream bounded, and a three-second limit that kills the process group it
leads. The warning reads the first `major.minor.patch` the command prints on
stdout, and warns, naming no version, when it is below the one floor constant,
cited beside it from the 2026-10-03 research note. No command, no answer, an
error exit or no parsable version raises nothing.

Answers that run out read as end-of-file, and end-of-file declines every confirm
and takes the default for every prompt, so an unattended run adopts nothing it
was not told to adopt. The cost is that a stdin held open and silent makes a
prompt wait rather than decline, which is the contract every prompting CLI has.
A run that must neither block nor prompt closes stdin and pre-answers with
flags.

**A piped answer never installs a tool**, just as it never answers the identity,
shell-settings or commit-checks questions. Installing a tool runs
a program on the machine, so it is asked only of a person at a terminal, after
the tool registry's explanation is shown, and its default is no. Off a terminal
the answer is a flag naming the tool, which is how a host relays the answer its
own question tool got; the approve-everything flag never installs a tool, and a
CI runner never installs one and is not asked: the canonical CI detector
(`internal/cienv`, `GITHUB_ACTIONS=true` or a truthy `CI`) refuses, and so does
`CI` set to any value, `false` and `0` included. What is fetched is fixed in
abcd's own release (`internal/core/tools/release.go`, [adr-2610241057124151](../../decisions/adrs/2610241057124151-abcd-installs-the-tools-it-needs-from.md)):
one GitHub release download per tool and platform, named by its full URL with
its sha256 pinned beside it, so no version is ever looked up (invariant 7) and a
tool updates only when abcd does. The download is bounded in size and time,
https only, and reaches GitHub's release hosts alone; its sha256 must equal the
pin, and for a tool whose makers sign their builds (gh) its signed build record
must verify in process against the pinned signing workflow, both before
anything is written. A tool that publishes no signed build record (gitleaks) is
checked by checksum only, and the question and the result say so. Only then is
the program read out of the archive by its exact name and written atomically
into the person's own bin folder (`$XDG_BIN_HOME`, else `$XDG_DATA_HOME/../bin`,
else `~/.local/bin`), never one inside the repository and never with admin
rights; no step abcd shows or runs needs them. Its verify must report the pinned
version, or the copy is removed again. The copy is recorded, path and digest, in
`~/.abcd.noindex/tools.json`: abcd runs that copy whatever comes first on
`PATH` while it is still the bytes abcd wrote, and uninstall removes it on the
same condition. A copy installed some other way is used when its version lies in
the tool's compatible range, and is never changed: when the bin folder already
holds a program of that name abcd did not install, it installs nothing and says
why.

The non-interactive flags pre-answer the prompts: approve every resolvable
category, decide the adoption question either way, set the marker target, the
oracle backend, the deep-scan toggle and the repo visibility, select track-latest
dogfood mode, proceed despite a stale running binary (the default refuses before
the adoption question and before any write, the writability probe of a named
`PATH` directory included, and names the rebuild fix), name the directory for the `PATH` entry,
opt the repo into the attribution prompt hook and into both outbound checks,
answer the README question with the parts abcd keeps current (or none), answer
yes to installing a named missing tool, and answer the setup questions from an
answers file, naming where an entry that names no place of its own was
answered. A value flag (marker
target, oracle backend, deep-scan toggle, visibility) is held to the
`config-change` approval like any other settings change: one that would change
a saved setting puts that approval even with no `config-change` gap, and a
decline drops it, saves nothing and names it (the product thinker's ruling on
iss-2610071538032843, "do what the user tells you in their response, no
surprises").

**The seeded roots are what exists** (iss-2610040758095861). The seeded
docs-lint config lists `docs` in `roots` only when that folder exists and
`README.md` only when that path exists, each looked up inside the project at
install time. A seed naming a root that is not there armed a check that refused
from the moment it was written, and with `docs_lint.root_missing` and the
bare lint reporting that refusal, every such fresh install would open on an
error. With neither, `roots` is `[]`: the check runs, reads no
document and warns that nothing was checked, at exit 0, the honest report for a
repository with no documentation yet, which a refusal would overstate and a
silent pass would hide. Only the seed of a new config is shaped this way; an
existing config is never changed.

**The house-style question.** When the install seeds the docs-lint config, it
asks one more question: whether the em-dash-in-list-item rule, abcd's own house
style rather than a currency rule, blocks or warns in this repository (the
product thinker's ruling of 2026-09-23 in the decision log). The chosen severity is written into the seeded config, which is
where the choice is recorded and where the repository changes it later. Blanket
approval does not ask and seeds a warning, and says so in the result. A bare
Enter or end-of-file takes the displayed default, also a warning. An answer that
names neither choice (the `y` a piped `yes` sends) is not guessed into a gate: it
seeds the warning and the result says what was heard. The question is asked
only when the seed is written: a repository that already carries a docs-lint
config keeps its own severity and is not asked.

Blanket approval does **not** adopt an unmanaged repo or record who a person
commits as: those still need their own answer. The identity-record exclusion is
stated rather than assumed — the flag's own help names it, the install envelope
carries it as skipped-and-optional, and the completion output prints it with the
way to apply it (iss-166).

**Each person commits as themselves** (itd-2610240918547728, which reverses
the committed pin of itd-131; adr-2610241022541063). Nothing in a repository
names who may commit to it. Each person's identity is their own record, on
their own machine: `~/.abcd.noindex/identity/<root-sha>.json` for one
repository, keyed on its root commit, else `~/.abcd.noindex/identity.json` for
every other repository on the machine, read under the guards the rules loader
applies (a regular file this account owns that no other account can write,
size-capped, refused behind a symlinked `~/.abcd.noindex`); a record that is
there and fails a guard fails the check closed. The committed shell guard in
abcd's own `.githooks/pre-commit` reads the same two files in the same order,
with no binary. With no record (`git_identity.unrecorded`, advisory) the
install proposes the person's usual git identity at a terminal, as its own
question, and records it only on a yes. A repository that still stores the
pin (`git_identity.repo_pin`) has it moved on its owner's machine (the person
whose git identity is the stored one), on a yes at a terminal: recorded as
that person's record for the repository, its `production_mode` kept in
`.abcd/config/provenance.json`, and `identity.json` removed from the working
tree. On anyone else's machine it is not copied, not removed, and not named.

**Who commits is proposed, never assumed** (itd-131). Detection raises
`git_identity.mismatch` for an author that is not the person's own record and
`git_identity.unset` for an author with none (required once a record applies),
`git_identity.committer` for a committer that diverges on its own (required
where a record applies, advisory where none does), and
`git_identity.tool` wherever the author or the committer is a machine identity
(the harness's own default, a `[bot]` account, a vendor's address, a configured
automation's name such as `semantic-release-bot`), recorded or not, because the
human is the author of record either way. The machine
identities are one list, `internal/core/identity/tool-identities.txt`, which the
CI attribution gate reads too, role asymmetry included: a `noreply@` mailbox is
a machine as the author, and the forge's own committer stamp passes. The
install proposes the human identity (the person's own record, else their global
git identity, read from disk, never from the repository, and never looked up
over the network, per adr-38, and never a machine's) as its own question, behind
no category approval, and sets `user.name` and `user.email` in the repository's
own `.git/config` only when the person confirms at a terminal. Otherwise it writes nothing and its notes say why: under the
approve-everything flag; with no terminal, where it asks nothing at all, so a
piped or routine run can neither write unasked nor wait on an answer no one can
give; when an environment override or an `author.*` or `committer.*` key
outranks `user.*`, so the write would change nothing; and when there is no human
identity to propose. An autonomous routine is therefore detected here and
established by whatever launches it, before its first commit
(iss-2608210932052003); on a fresh machine with no record, the commit check lets
the human identity the launcher supplied through and refuses a machine's. A
record is never written from a machine identity.

### What the apply pass writes

The marker block and the guard hooks come from canonical files under
`internal/core/ahoy/defaults/`, never from inline prose in this chapter, which
is what makes drift detection meaningful: the block has one canonical source. If
a template is stale, the template file is what to edit. The block names abcd and
documents its rule loader, so the docs target defaults to `skip`: a default
install writes it into none of the repository's committed conventions files
(iss-2609110944498549), and a project that wants it names `agents_md`, which
is the approval to plant it into AGENTS.md, the one conventions file abcd writes
(adr-2610030814023326). `claude_md` and `both` are read, never written: a
project that saved one still classifies as managed on its CLAUDE.md block, and
uninstall still strips the block from both files, but detection raises the one
required, non-resolvable gap `config.docs_target_retired`, setup is refused
before its first write, and the install flag that names a docs target refuses
both values, each with the one explanation `RetiredDocsTarget` holds, naming
`docs.target` and the command that changes it. An install that changes the
setting to `agents_md` or `skip` runs as any target change does and takes the
block out of CLAUDE.md. The name-guard
hooks and the ignore fence are the one sanctioned mention of abcd outside
`.abcd/` (ruled 2026-09-23; see the preparation's boundaries above). Every name-guard
write is create-if-absent **and** contained: paths resolve through an `os.Root` opened
at the repo, so a symlink committed at the hooks directory or at the local tier
cannot land an artefact outside it. The private stub is written only where git
reports the path as ignored. A clone arms the hooks once by pointing git at the
hooks directory. Setup asks that as its own question at a terminal and sets
`core.hooksPath` to `.githooks` in the clone's `.git/config` only on a yes,
reading the value back; it never sets it under the approve-everything flag or
off a terminal, and never over a hooks path set elsewhere. Until the hooks run,
the install is reported partial, never clean, and no surface reports a committed
hook as a running one.

**The README parts** (spc-2610031704338417) are a named sibling of the
conventions block: one marked place per part, opened by
`<!-- BEGIN ABCD:<part> sha256=<hash> -->` and closed by
`<!-- END ABCD:<part> -->`, the hash taken over the body abcd wrote
(`readme_marker.go`). A named fence never matches the
conventions block's pattern, and the conventions fences never match the named
ones. Each part is derived from the project on this computer
(`readme_derive.go`): the badges from the origin remote, github.com only, the
build badge naming the one workflow file, or `ci.yml` or `ci.yaml` among
several; the licence named only from an `SPDX-License-Identifier:` line in the
first twenty lines or a forge-standard first line, else linked unnamed; the
links from four guide files at the root, in `.github/` or in `docs/`; the
contents from the README's live level-two and level-three headings outside
every part. Each body then passes three checks in order (`readme_check.go`):
the badge-address check (https, the one allowed host per badge and role,
`img.shields.io` for the release and abcd badges' images, and a query only on
the build badge's image with the forge's `branch` and `event` keys), the
outbound scrub (`scanner.ScrubOutbound`), and the project's own docs-lint bans
where it keeps `.abcd/docs-lint.json`; a part failing any one, or one that
cannot be derived, is left out with the reason in the notes, and the derivation
imports no network package. The write keeps a current part, rewrites a stale
one where it stands, places an absent one beside the nearest present part, else
after a leading frontmatter block, else after the first live level-one heading,
else at the top, and removes one no longer chosen, or no longer derivable,
whose body still matches its hash. A part whose body does not match its hash
was edited by hand and is named with its line and left; a duplicated or
unclosed fence refuses the whole write, naming the line. A run in which every
part is current writes nothing, so the file's bytes and modification time are
unchanged. With no `README.md` and a part to write, setup creates the file
holding only the chosen parts, and records `readme.created: true` so uninstall
may delete it once nothing but blank lines remains; the record is cleared once
that file is gone, so it never vouches for a `README.md` the owner writes
later. The fences name abcd, a mention outside `.abcd/` written only on the
owner's yes (decision 2 extends ruling F by that consent). Uninstall
(`uninstallReadme`) removes every part whose body matches its hash, whatever
`readme.parts` says, leaves and names the others, and reports them in the
receipt's `readme` field, which the text render prints as its `readme:` line
("links removed; licence edited by hand, left").

Two writes deserve their own note. The visibility step rewrites the ignore block
under the config-change approval already given, with no confirmation of its own;
its receipt adds a post-hoc note when a public fence had to be narrowed,
because an ignore rule cannot untrack committed records, so the reader learns
from the receipt that the committed record tiers stay published (iss-255). Like
every install write, a block it could not write (a symlinked `.gitignore`, say)
is a note naming the file and the reason, never a silent omission. And a
remote URL recorded in the registry carries no credential: it is scrubbed where
the identity is derived, scrubbed again as the index is *loaded* so every
rewrite drops a credential from every entry rather than only the one being
registered, and a per-repo file that is otherwise written once is rewritten in
place when it holds one. A store that already holds a credential raises its own
gap, so an otherwise up-to-date repo does not short-circuit past the heal.

**Same-version re-install:** when detection reports zero actionable gaps — gaps
both required and resolvable — the install prints that it is already up to date
and exits without writing. That falls out of detection; it is not a
version-stamp short-circuit.

## Re-founding (the `supersedes` flow)

When a repo is re-created with clean history, typically to strip in-repo
transcripts before sharing, it is genuinely a new repo with a new root SHA.
Detection flags this when the current root SHA is absent from the registry
**and** a sibling entry has a matching name under a different SHA.

ahoy never auto-decides it. It surfaces the candidate predecessor and asks. On
confirmation the apply pass registers the new SHA, sets the new entry's
`supersedes` and the old entry's `superseded_by`, and leaves the old repo's
corpus in place under its own key for lifeboat review: nothing is moved or
deleted. If the user declines, ahoy registers the new SHA with no lineage link
and notes the orphaned-predecessor possibility in the summary.

## What the read-only renders carry

**Bare `abcd ahoy`** prints the status board: the folder kind, plugin-root
status, root SHA, install mode where one resolves, vintage and staleness, the
superseded-root note when the answering binary sits in a plugin root other than
the one this session resolves, the citation baseline's coverage and age on a
repo that has armed the citation gate, the gap count with each gap's title
beneath it, and — on a repo — guard
health and the banlist block with its reach, closing on a next-step line for the
unmanaged kinds. In JSON form the same pass renders the detection envelope plus
vintage and staleness, and `superseded_root` when the note applies; the plugin
command reads those from exactly this render, so they are a contract with the
plugin surface rather than a convenience. The note is the one the version flag
carries, under the same conditions
([`08-abcd.md`](08-abcd.md#a-superseded-plugin-root-names-itself)).

**The dry run** renders the detection envelope as JSON and nothing else, so the
plugin command can summarise state off the folder kind and the gaps and name
the install for anything actionable. Two of the envelope's keys are pointers
omitted entirely on an unmanaged folder: guard health and the banlist block
report definite booleans and named states, so a never-computed zero value would
serialise facts that read as a broken guard to a consumer that never asked about
a repo.

**The doctor** adds the read-only audit pass, and its JSON carries full per-gap
detail on both halves. Detection covers user-scope state (the store exists and is
writable, the registry entry matches this root SHA, the `PATH` entry and hook
manifest are intact); the audit reconciles the registered path against the
registry. It never mutates, and never auto-fixes user-scope state.

**Uninstall** removes the marker block, abcd's own `PATH` entry where abcd owns
it, and the provenance record by which that ownership is proven. Ownership is
the same three-shape predicate detection classifies with, and only one of the
three is a pointer at all: the dev shim; the owned copy the `path-entry` record
names and whose bytes still hash to the recorded value, which is the default
install and a regular file pointing at nothing; and lastly a legacy symlink
whose target is this plugin's binary, or whose target has gone when it is the
link a plugin update stranded or the one the `path-entry` record names. The
recorded dangling link needs no plugin root to be recognised, so uninstall finds
it on `PATH` and removes it with its record on a machine where abcd itself is
gone. Anything else is foreign and is left where it stands. It leaves the entire `.abcd/` namespace and the history store intact.

**Uninstall then install is a tested round-trip invariant**: afterwards the
detection pass must report zero actionable gaps, and the resulting state must be
byte-identical to a fresh install save for the setup date.

## Acceptance

- **Given** any abcd-aware terminal, **when** the user runs bare `/abcd:ahoy`,
  **then** the detection pass runs and the status board is shown, with a
  next-step line on the unmanaged kinds and none on a managed repo, and nothing
  is mutated.
- **Given** a git repository with no abcd markers and no registry entry for its
  root-commit SHA, **when** the user runs bare `/abcd:ahoy`, **then** it reports
  `unmanaged-repo`, names the install as the way to adopt it, and mutates nothing:
  bare invocation never adopts. **Given** a folder that is not a git repository,
  it reports `unmanaged-folder` and that there is nothing to act on.
- **Given** a fresh repo with no `.abcd/` directory, **when** the install runs to
  completion, **then** the repo carve-out is written, nothing in the repository
  names who commits to it, the visibility-driven ignore
  entries are present, the registry entry exists, the marker block from the
  canonical template is installed in the files a chosen docs target names and in
  none at the default, and the hook-manifest check runs verify-only
  with a missing or malformed manifest surfacing as a non-resolvable diagnostic.
- **Given** a repo with the install already run and no state changes, **when**
  the install runs again, **then** detection reports zero actionable gaps, the
  message reads that it is already up to date, and nothing is written.
- **Given** a repo where the marker block was hand-deleted but the setup version
  is current, **when** the install runs, **then** detection reports the marker
  missing and the apply pass restores it: idempotency keys off state, not the
  version stamp.
- **Given** a repo with the install run at an older setup version, **when**
  the install runs, **then** the version is updated, the marker block refreshed,
  and existing config keys preserved.
- **Given** a tool a capability uses has no copy abcd can run, **when** the
  tool's own question is reached (at a terminal, whatever the approvals; off
  one, once the dependency category is approved), **then** the person is shown the tool
  registry's explanation and asked per tool; the pinned, checked download is
  made only on a yes typed at a terminal or relayed by a host naming the tool,
  nothing is written unless it passes its check, the result reports what was
  installed and how far it was checked (checksum only for a tool that signs
  nothing), and a no reports what the capability continues on (itd-63,
  itd-2610240918594138).
- **Given** no oracle adapter is wired, **when** detection resolves the oracle,
  **then** it stays host-delegated: abcd needs no API keys or model config,
  because it emits prompts the host runs (adr-25), and an adapter can be
  configured later. A first install records host-delegated without asking and
  says so in one note naming the install flag that chooses another reviewer;
  that flag still sets any of the five values.
- **Given** a repo whose root SHA is absent from the registry while a sibling
  entry matches its name, **when** the install runs, **then** detection flags a
  re-founding candidate, ahoy asks before linking, and on confirmation records
  the lineage both ways and leaves both corpora in place.
- **Given** the user runs the uninstall then the install, **when** both complete,
  **then** detection reports zero actionable gaps and the resulting state is
  byte-identical to a fresh install save for the setup date.
- **Given** the user runs the dry run, **when** it completes, **then** the
  detection pass runs, the canonical envelope is printed to stdout, and no files
  are modified.
- **Given** the user runs the doctor on an installed repo whose registered path no
  longer matches the registry, **then** an audit gap citing both paths appears
  in the JSON envelope, reported read-only, and no files are modified.
- **Given** a fresh machine with no `~/.abcd.noindex/`, **when** the install runs in a
  repo, **then** the user-scope directory is bootstrapped before the repo is
  registered, so the user is not blocked by missing user-scope state.
- **Given** a registered repo that has been moved on disk, **when** the install or
  the doctor runs, **then** detection notices the stale registered path and
  the install refreshes it: the root SHA is unchanged, so the entry is updated
  rather than duplicated.
- **Given** a person whose repo-local `user.name` and `user.email` differ from
  their own record, **when** the install runs at a terminal, **then** it
  proposes the recorded identity and writes repo-local config only after the
  person confirms; declined, git config is unchanged.
- **Given** a person with no record on this machine, **when** the install runs
  at a terminal, **then** it proposes their usual git identity as its own
  question and records it on their machine on a yes; off a terminal it records
  nothing and says so.
- **Given** a contributor in a repository that still stores another person's
  identity, **when** they set up and commit, **then** nothing names that
  person and their commits are checked against their own record; on the
  stored identity's owner's machine the install moves it to their record, removes it from the repository, keeps its `production_mode`,
  and says so.
- **Given** a committer that differs from the author or the record, or an author or
  committer that is a machine identity, **when** detection runs, **then** the
  divergence is reported as its own gap, and with no terminal the install asks
  nothing, writes nothing and says so.

- **Given** a repo the user does not own, **when** they ask the plugin page
  to install, **then** the preparation stops at the ownership phase with no
  audit and no writes.
- **Given** an owned repo, **when** the preparation runs, **then** a gap report
  exists under `.abcd/.work.local/scratch/` and was presented before anything
  was adopted.
- **Given** sign-off, **when** it adopts, **then** the three-tier layout exists
  with a repo-specific `CONTEXT.md`, `AGENTS.md` carries verified repo facts and
  the marked nameless working-conventions section, the done-test passes (a fresh
  agent can build and test from `AGENTS.md` alone), and any historical `.work/`
  layout is fully migrated or fully left alone.
- **Given** sign-off, **when** it adopts, **then** one identity block is
  recorded and registered, adopted where the repo already had one and
  interviewed only where it did not, and the lint's identity report shows every
  rendered surface against it.
- **Given** the adoption completes, **then** nothing from `private-names.txt`
  and no abcd-internal content appears in any committed artefact, with one
  sanctioned exception (ruled 2026-09-23): the name-guard hooks
  (`.githooks/pre-commit`, `.githooks/pre-merge-commit`) and the `.gitignore`
  fence name abcd, and cite none of its record ids.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd ahoy`

Sub-verbs: `abcd ahoy connect`, `abcd ahoy credential`, `abcd ahoy doctor`, `abcd ahoy install`, `abcd ahoy remote`, `abcd ahoy uninstall`.

| Flag | Type |
|---|---|
| `--dry-run` | bool |
| `--identity` | bool |
| `--providers` | bool |
| `--remote` | bool |

### `abcd ahoy connect`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--answer` | string |
| `--base-url` | string |
| `--env` | string |
| `--field` | string |
| `--file` | string |
| `--guide` | bool |
| `--home` | string |
| `--key` | string |
| `--model` | stringArray |
| `--resume` | string |
| `--stream-usage` | bool |

### `abcd ahoy credential`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--env` | string |
| `--field` | string |
| `--file` | string |
| `--home` | string |

### `abcd ahoy doctor`

Sub-verbs: none.

Flags: none.

### `abcd ahoy install`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--adopt` | bool |
| `--allow-stale-binary` | bool |
| `--answered-in` | string |
| `--answers` | string |
| `--attribution` | bool |
| `--bin-dir` | string |
| `--details` | bool |
| `--dev` | bool |
| `--docs-target` | string |
| `--install-tool` | stringSlice |
| `--live-sessions` | string |
| `--only-waiting` | bool |
| `--oracle-backend` | string |
| `--outbound` | bool |
| `--readme-parts` | string |
| `--refuse-adopt` | bool |
| `--scan-deep` | string |
| `--visibility` | string |
| `--yes` | bool |

### `abcd ahoy remote`

Sub-verbs: `abcd ahoy remote apply`.

Flags: none.

### `abcd ahoy remote apply`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--yes` | bool |

### `abcd ahoy uninstall`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--bin-dir` | string |

<!-- surface-appendix:end -->
