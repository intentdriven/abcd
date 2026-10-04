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
pre-commit hook or CI rather than a conversation. `status` is a plugin-page
alias for the bare form and has no CLI sub-command behind it: `abcd ahoy
status` is refused as an unknown command. Every other word ships on the CLI:
the table above is the sub-verb set, and the modes are the bare verb's flags.

- **Install** installs or updates abcd in this repo, covering first install
  and upgrade alike. It runs the detection pass, then an apply pass over the
  resulting gaps.
- **Uninstall** is reversible removal: the marker block, abcd's own `PATH`
  entry where abcd owns it, and the provenance record that proves that
  ownership. It leaves `.abcd/` entirely intact, never mutates the hook
  manifest, and a later install re-installs cleanly. It finds the entry by
  scanning `PATH`, so an entry that was installed into a directory `PATH` does
  not carry is removed by naming that directory again, the same bin directory
  the install was given.
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
  The same request also reads the repository's merge hygiene, which abcd mirrors
  and never sets: those settings encode the technical facilitator's workflow rather than a
  security posture, and each is reported only when the API answered for it,
  because `false` and "the API did not say" are different facts
  (iss-2608270512210664).
- **The remote apply** is **the one abcd verb that mutates state outside this
  machine.** See below.
- **The identity check** exits non-zero when the author or the committer a
  commit would carry does not match the repo's identity pin, and names a machine
  identity in either role whether or not it fails. Read-only, CLI-only, for an
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
observed block holding the merge-hygiene settings it only reads.

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
running the registry's step only on a yes typed at a terminal. The pre-given
yes answers the settings change and never the install of a program, and a
piped answer is not a person's answer, so both decline the offer; the verb then
refuses, its notes carrying the explanation and the command. A failed or
unverified install refuses the same way, before any request leaves the
machine. The read never offers the install, because looking is never acting;
it names the apply as the verb that does.

### The provider setup

The setup takes the provider's name, its base URL, its first allowlist (every
model it may serve) and where its key lives. It verifies the provider with one
call to the first model listed and, only when that call succeeds, keeps the key
in the home chosen through the credential store's walkthrough and then writes
the provider block (base URL, the key's name, the models) into
`~/.abcd/config.json`. A failed verification writes nothing. Nothing reaches the
repository or the harness's settings. Every fault the configuration read would
refuse (a model an `oracle.denylist` entry matches, a malformed model, a base URL that is plain HTTP to
another machine, a provider already configured, a key name already holding a
different value) is refused before the call, so a setup that cannot finish is
never billed.

The key arrives on stdin and nowhere else. A flag would leave it in the process
listing and the shell history, the install prompter echoes every answer into its
transcript, a host's question tool would put it in an agent's context, and a
terminal would echo it as it is typed, so stdin from a terminal is refused. For
the same reason the walkthrough is this sub-verb, which the person runs with the
key piped in, rather than a question the install pass asks: declining is not
running it, and changes nothing.

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
  `~/.abcd/credential-homes.json`, and follows it on every read. A file
  pointer is refused, naming the link, when any directory between the home and
  the tool's file is a symlink, wherever the link leads; the
  environment-variable pointer stays open.
- `abcd` — the owner-only `~/.abcd/credentials.json`, which holds the value.
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
home when `~/.abcd` lies inside a git working tree, since that home alone keeps
a value there, a name another home already holds, and a different
value for a name already kept, and the secret scanner reads the index's bytes
before they are written, refusing any finding. A value is read from stdin only,
and never printed, logged or written to a record; a call's record names the
credential it used.

## What abcd manages — repos and `~/.abcd/`

abcd manages exactly one kind of folder, a **repository**, and keeps one
user-scope directory for machine-local state.

```
~/.abcd/                       USER SCOPE — one per machine (machine-local state only)
  history/                       the REGISTRY only: identity and lineage keyed on the
                                 root-commit SHA. ahoy owns it; it holds no transcripts
  transcripts/<root-sha>/        the redacted transcript corpus, a SIBLING of the
                                 registry, creating itself on first use
                                 (adr-2609091248201071, superseding adr-2609090717039680)
  voyage/<root-sha>/             disembark/embark operations log, never committed
                                 (adr-35)
  lab/<root-sha>/<lab-id>/       one lab's evidence: snapshot, probe records, findings,
                                 harvest; index.jsonl beside the homes registers them,
                                 never committed (itd-2609212137128014)
  worktrees/<root-sha>/<name>/   session and agent worktrees, never beside the checkout
                                 (NOT BUILT — itd-2609091014076309)
  runs/<root-sha>/               an autonomous run's shared state: the run log, one
                                 claim per record, one record per joined session
                                 (itd-2609221656373558)
  inbox/                         reports managed repositories filed back to abcd,
                                 <received-stamp>-<sender-key>.md; promoted/ keeps
                                 the ones filed as captures (itd-2609221656361680)
  config.json                    the machine layer of the layered configuration,
                                 read-only except for the provider blocks
                                 (oracle.api.<provider>) the provider setup writes,
                                 and the only file a provider block may sit in;
                                 that write holds .config.json.lock beside it
  memory/                        user-scope memory (personal, cross-project — a later
                                 phase; the shipped store is repo-scope .abcd/memory/)
  sources/                       the local sources corpus the source verb maintains and
                                 /abcd:ingest and /abcd:consult read. Created only by
                                 its explicit init: absent means every other verb and
                                 both commands say so and stop
  load-limits                    the load check's per-machine limits (stray-minutes,
                                 extreme-load), read-only; abcd never creates it
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
  path-entry                     the abcd copy this machine owns, the one PATH binary a
                                 hook will run
  trusted-roots                  foreign-uid configuration roots the caller vouches for
  local-transcript-roots         checkouts whose transcripts are pulled in to
                                 <repo>/.abcd/.work.local/transcripts/ instead

<anywhere>/<repo>/             REPO — a single repository (the only install target)
  .abcd/                         repo-scope record + config.json + rules.json
  AGENTS.md                      marker block (stands alone), only where the docs target
                                 is agents_md; the one conventions file abcd writes
```

The same inventory is stated as a table under *The two `.abcd/` scopes* in
[`05-internals/03-configuration.md`](../05-internals/03-configuration.md#the-two-abcd-scopes);
the two are one list and must agree. The three declaration files at the bottom
are caller-controlled and line-oriented. `trusted-roots` and
`local-transcript-roots` are the two that widen what a session will trust, so
each is honoured only when it is a regular file this uid owns that no one else
can write, and a file failing either test is ignored with one line saying which
test it failed. `path-entry` and `cache-attestation` are held to the same test,
by the install verb and by the hook shims that consult `path-entry`, and a record
failing it vouches for nothing.
`load-limits` is a setting, not a declaration, but it is read through the same
guard as the two that widen trust, and a file failing it, or holding a line that
does not parse, is reported loudly and both of its limits take their defaults.
`rules.json` is read through that guard too, because it injects text into every
session on the machine, but a file failing it — or failing to parse — fails the
rules load outright: nothing injects until it is fixed, and the file is named on
stderr. None of these files is honoured behind a `~/.abcd` that is itself a
symlink, and nothing ahoy or the bootstrap writes there goes through one: each
refuses the link and names it, as the rules loader does for `rules.json`, while a
symlinked `~/.abcd` holding none of them reads as absent (the rule is stated once,
under *The two `.abcd/` scopes* in
[`05-internals/03-configuration.md`](../05-internals/03-configuration.md#the-two-abcd-scopes)).

There is **no workspace, host, or development-environment layer.** A folder a
user keeps their repos in groups nothing, and abcd does not privilege it. abcd
lives in one repository (adr-28): the design record is repo-scoped and in-tree.
Everything genuinely machine-wide lives under `~/.abcd/`, which an install
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
committed identity pin and the list of machine identities; the registry's own wiring; the ignore
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
install directory absent from `PATH` is required but not resolvable, for which
abcd prints a one-line export fix and never edits a shell profile; and any
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

The shipped manifest wires six event types, and every event command is a
resolving shim rather than a plain binary call. Four of them self-provision.
`UserPromptSubmit`, `PreToolUse` and `PreCompact` each attempt
`hooks/bootstrap.sh` only when the plugin-root binary is missing, recording the
try in a `.bootstrap.attempt` marker that throttles the next one to a ten-minute
window. `UserPromptSubmit`, the hook that runs on every message, declares a
120-second `timeout`, the time a salvage on a slow link has before the host
cancels it; `PreToolUse` and `PreCompact` declare none and take the host's
ten-minute default, so a slow first download finishes there. `SessionStart`
declares 240 seconds, the script's worst case with room to spare, and
`SessionEnd` and `SubagentStop` declare none. Every event that runs the script,
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
so the three throttled events see a recent try, and reads no throttle of its
own. `SessionEnd` and `SubagentStop` are the deliberate exceptions and download
nothing: both fire where the host cancels a slow hook rather than wait — one as
the session is going away, the other inside a live session as a sub-agent
finishes — and a mid-flight fetch loses the very transcript the hook exists to
capture (iss-2608210934566223). Each resolves the plugin root, then `PATH`, then
says in one line that the transcript was not captured. `SubagentStop` never
returns a non-zero code of its own beyond that refusal, because exit 2 is the
host's BLOCKING status on that event and would stop the sub-agent finishing.

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

## Gaps, and how the apply pass asks about them

Each detected discrepancy becomes a **gap** with a stable id, a category, a
scope, a title, detail and a fix hint. The category is what the apply pass asks
about, one question per category present, never one per item.

| `category` | Examples | Apply behaviour |
|---|---|---|
| `safe-autocreate` | the repo skeleton, history-store directories, the name-guard artefacts | applied once the category is approved, no per-item prompt; create-if-absent, never overwriting |
| `config-change` | visibility, oracle adapter, the `PATH` entry, the git-identity pin, the repository's own git identity, the artefact kind | transparent confirm; skip-if-set with a "current value" notice |
| `plugin-owned` | the marker block (itd-3); hook-manifest verification | silent overwrite on marker drift; a non-resolvable diagnostic for a malformed or missing manifest, and for a conventions file whose block would land inside a fence or HTML comment nothing closes (`marker.unplaceable`) |
| `dependency` | a tool a capability uses and cannot find: gitleaks, optional over the native secret scanner and required where the repository armed it in `.abcd/config/gitleaks.json` | the category approval reaches the step; each tool is then explained from the tool registry (what it is, optional or required here, what works without it, the exact install step, what the install does) and its install step runs only on a per-tool yes — typed at a terminal, or relayed by a host as a flag naming the tool — never under the approve-everything flag, a piped answer or CI; a no is reported as what the capability continues on |
| `status-line` | the offer of abcd's status line in the host harness | an advisory offer asked after its own question, written only on an answered consent; never under the approve-everything flag, and reported as optional work it skipped |
| `oracle-routing` | the offer of abcd's proposed model-tier routing table (itd-2609170822093401): the machine's `~/.abcd/oracle-routing.json`, then, as a separate question, the repository's `.abcd/config/oracle-routing.json` | the proposal said in counts (how many agents, how many at each tier, their fan-out bounds), naming no agent so the question fits, and each file written only on its own answered consent, the machine one owner-only; never under the approve-everything flag, and reported as optional work it skipped; a decline records nothing, so the next install offers again; uninstall leaves both files |
| `drain-rule` | the offer of the repository's drain eligibility record (ruling BX2, itd-82): abcd's strict baseline as an accepted decision record carrying the four `drain_` fields, minted through the decision store's seam | the rule stated in one question and the record written only on a consent answered at a terminal; never under the approve-everything flag and never off a terminal, where neither its category nor the offer is asked (so a piped answer stream keeps its order), and reported as optional work it skipped; a decline records nothing, so the next install offers again; raised only while no accepted record states the rule, so a record stating it badly is never offered a second; only ever the baseline, never a loosened rule |
| `user-state` | the registry entry, re-founding, stale or duplicate entries | guided; never auto-edit user-scope state, report extras read-only |

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
agent drives the git-identity pin, the one approval no flag covers. Off a
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
line that would, with no answers record written. The approvals are asked
before the first write; a later question's stop leaves the steps before it
done. A file never answers the questions put only to a person at a terminal
(the git identity, the drain rule, installing a tool).

**Every answer is recorded with where it was given.** The drawn door and the
answers file write one answers record through `interview.Write`: per question,
the sanitised question as asked, the value, the note and `answered_in`,
`Terminal` or `Claude Code` and nothing else. The drawn door stamps
`Terminal`; an answers-file entry carries its own, or takes the place the
run names for its file (`Terminal` unless it names another), which the plugin
page names as `Claude Code` on the host path. The repository's answers go to
`.abcd/.work.local/interviews/setup-<stamp>.json`, never creating the local
tier; the machine-wide ones (the status line, the machine's routing table)
to `~/.abcd/interviews/`, made through the guarded home-scope maker. The stamp
is in the name only, so two runs given the same answers write the same bytes
and the same configuration, whichever door asked. The line reader writes no
record.

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
prints the exact record after as detail.

Answers that run out read as end-of-file, and end-of-file declines every confirm
and takes the default for every prompt, so an unattended run adopts nothing it
was not told to adopt. The cost is that a stdin held open and silent makes a
prompt wait rather than decline, which is the contract every prompting CLI has.
A run that must neither block nor prompt closes stdin and pre-answers with
flags.

**Installing a tool is the one question a piped answer never answers.** It runs
a program on the machine, so it is asked only of a person at a terminal, after
the tool registry's explanation is shown, and its default is no. Off a terminal
the answer is a flag naming the tool, which is how a host relays the answer its
own question tool got; the approve-everything flag never installs a tool, and a
CI runner never installs one and is not asked: the canonical CI detector
(`internal/cienv`, `GITHUB_ACTIONS=true` or a truthy `CI`) refuses, and so does
`CI` set to any value, `false` and `0` included. What runs is the
registry's fixed argv for the platform, never a shell string and never a command
composed from input, and only when the package manager resolves on `PATH`
outside the repository (`internal/core/tools`). The step runs in a process group
of its own, bounded at 15 minutes (its verify at 30 seconds), and a timeout kills
that group through the handle abcd holds. The kill has one limit: a process the
step moves into another group or session (`setsid`, `setpgid`) is out of its
reach and can outlive the run. abcd stops reading output 10 seconds after the
step exits or is killed, so such a process holding the output open cannot hold
the run past its bound, and a step that exits cleanly while leaving one behind
is reported as failed, saying so.

The non-interactive flags pre-answer the prompts: approve every resolvable
category, decide the adoption question either way, set the marker target, the
oracle backend, the deep-scan toggle and the repo visibility, select track-latest
dogfood mode, proceed despite a stale running binary (the default refuses before
the adoption question and before any write, the writability probe of a named
`PATH` directory included, and names the rebuild fix), name the directory for the `PATH` entry,
and opt the repo into the attribution prompt hook.

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

Blanket approval does **not** adopt an unmanaged repo or pin an unset git
identity: those still need their own answer. The identity-pin exclusion is
stated rather than assumed — the flag's own help names it, the install envelope
carries it as skipped-and-optional, and the completion output prints it with the
way to apply it (iss-166).

**Who commits is proposed, never assumed** (itd-131). Detection raises
`git_identity.mismatch` or `git_identity.unset` for the author,
`git_identity.committer` for a committer that diverges on its own (required
where the repo pins an identity, advisory where it does not), and
`git_identity.tool` wherever the author or the committer is a machine identity
(the harness's own default, a `[bot]` account, a vendor's address, a configured
automation's name such as `semantic-release-bot`), pinned or not, because the
human is the author of record either way. The machine
identities are one list, `internal/core/identity/tool-identities.txt`, which the
CI attribution gate reads too, role asymmetry included: a `noreply@` mailbox is
a machine as the author, and the forge's own committer stamp passes. Once the
config-change category is approved, the install proposes the human identity
(the pin, else the global git identity, read from disk and never looked up over
the network, per adr-38, and never a machine's) and sets `user.name` and
`user.email` in the repository's own `.git/config` only when the person confirms
at a terminal. Otherwise it writes nothing and its notes say why: under the
approve-everything flag; with no terminal, where it asks nothing at all, so a
piped or routine run can neither write unasked nor wait on an answer no one can
give; when an environment override or an `author.*` or `committer.*` key
outranks `user.*`, so the write would change nothing; and when there is no human
identity to propose. An autonomous routine is therefore detected here and
established by whatever launches it, before its first commit
(iss-2608210932052003). The pin is never recorded from a machine identity.

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
`.abcd/` (ruled 2026-09-23; see prepare-this-repo). Every name-guard
write is create-if-absent **and** contained: paths resolve through an `os.Root` opened
at the repo, so a symlink committed at the hooks directory or at the local tier
cannot land an artefact outside it. The private stub is written only where git
reports the path as ignored. A clone arms the hooks once by pointing git at the
hooks directory; abcd never sets that config, and no surface reports a committed
hook as a running one.

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
repo that has armed the citation gate, the gap count, and — on a repo — guard
health and the banlist block with its reach, closing on a next-step line for the
unmanaged kinds. In JSON form the same pass renders the detection envelope plus
vintage and staleness, and `superseded_root` when the note applies; the plugin
command reads those from exactly this render, so they are a contract with the
plugin surface rather than a convenience. The note is the one the version flag
carries, under the same conditions
([`12-version.md`](12-version.md#a-superseded-plugin-root-names-itself)).

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
  completion, **then** the repo carve-out is written, the identity pin is
  recorded where the git-identity gate is adopted, the visibility-driven ignore
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
- **Given** a tool a capability uses is not on `PATH`, **when** the dependency
  category is approved, **then** the person is shown the tool registry's
  explanation and asked per tool; the registry's fixed install step runs only
  on a yes typed at a terminal or relayed by a host naming the tool, the result
  reports what ran and whether its verify passed, and a no reports what the
  capability continues on (itd-63).
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
- **Given** a fresh machine with no `~/.abcd/`, **when** the install runs in a
  repo, **then** the user-scope directory is bootstrapped before the repo is
  registered, so the user is not blocked by missing user-scope state.
- **Given** a registered repo that has been moved on disk, **when** the install or
  the doctor runs, **then** detection notices the stale registered path and
  the install refreshes it: the root SHA is unchanged, so the entry is updated
  rather than duplicated.
- **Given** a pinned repo whose repo-local `user.name` and `user.email` differ
  from the pin, **when** the install runs at a terminal, **then** it proposes
  the pinned identity and writes repo-local config only after the person
  confirms; declined, git config is unchanged.
- **Given** a committer that differs from the author or the pin, or an author or
  committer that is a machine identity, **when** detection runs, **then** the
  divergence is reported as its own gap, and with no terminal the install asks
  nothing, writes nothing and says so.

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
| `--base-url` | string |
| `--env` | string |
| `--field` | string |
| `--file` | string |
| `--home` | string |
| `--key` | string |
| `--model` | stringArray |

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
| `--dev` | bool |
| `--docs-target` | string |
| `--install-tool` | stringSlice |
| `--oracle-backend` | string |
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
