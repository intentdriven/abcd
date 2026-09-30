---
name: ahoy
description: "Detect abcd's install state and list its gaps, or report one mode a flag names: Writes nothing; refuses any argument or two modes at once."
argument-hint: "[install | uninstall | doctor | --dry-run | --remote | remote apply | --providers | connect]"
block: people
---

# `/abcd:ahoy` install/update detector

`abcd --help` lists `ahoy` in the person's set-up group. `statusline`, the
harness-invoked row that `install` wires, is in the agents-and-hosts block of
`abcd --help --agent`, and its line there names this page.

Run abcd's install/update engine for the current repo and present the result.
Bare invocation, its `--dry-run`, `--remote` and `--providers` modes, and the
`doctor` sub-verb perform **zero writes**; `install`, `uninstall`, `remote apply`
and `connect` are the four that change something, and each says so before it
runs — `remote apply` is the only one that changes state outside this machine,
and it asks before it does.
A mode is a flag on the bare verb, one at a time; a distinct action is a
sub-verb.

Read `$ARGUMENTS` for the sub-verb or the mode. No argument, or `status`, is the
bare read-only detection pass below.

## Bare — read-only detection

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy --json
```

Then summarise the JSON for the user:

- `folder_kind` — `managed-repo`, `unmanaged-repo`, or `unmanaged-folder`.
- `plugin_root_status` and `root_sha` — where abcd is anchored.
- `signals.install_mode` — the PATH-entry install mode: `dev (tip build)` when
  the track-latest dogfood shim is installed, `pinned` for the abcd-owned copy
  of the verified release binary, empty when there is nothing on `PATH` yet.
  Either non-empty value may carry a trailing ` (shadowed on PATH)` when another
  `abcd` comes first on `PATH`. Report it so a dev install — or an entry abcd
  wrote that is not the one that runs — is never invisible.
- `signals.statusline` — the host's status line as abcd sees it: `installed`
  (abcd's own `<entry> statusline` command, entry present), `absent` (no
  status line configured), `foreign` (a status line that is not abcd's),
  `dangling` (abcd's, but the entry it names is gone — a required repair,
  because that blanks the status line in every repository), `unreadable`
  (the settings file is not a JSON object), or `no-harness` (no settings file
  was found, so nothing is offered).
- `vintage` and `staleness` — the running binary's build revision (in a source
  checkout) or pinned version, and whether it is up to date, stale, or of an
  undeterminable vintage relative to the on-disk reference. Report them so a
  binary running behind its own source is never silent. The comparison is
  disk-only — no network.
- `banlist` — the two-layer name guard, when the folder is a repo: `hook` and
  `merge_hook` (`installed` / `absent` / `foreign` / `unreadable`), whether this
  clone is armed (`hooks_path_armed`), `public_family`, and the private layer's
  state on this machine (`private_store`, and its shape — `private_keyed`,
  `private_entries`, `private_unparsed`). Never report a `foreign` hook as abcd's
  guard, and never report a committed hook as a running one. Relay `reach`
  verbatim: it is the one sentence stating what the private layer does NOT cover,
  and a paraphrase drops the half that matters.
- `gaps` — how many are outstanding, and for each actionable one its `title`,
  `category`, and `fix_hint`; call out which are `required`. A `dependency` gap
  carries `tool`, the tool registry's explanation: relay `what`, the
  `requirement` for `capability_name`, `without_it`, `does`, the exact `step`
  and `effects` (what the install does on the machine and over the network,
  Homebrew's own analytics included) rather than a bare command, so the
  person can judge the install.

If there are actionable gaps, tell the user to run `/abcd:ahoy install` to apply
them. If `folder_kind` is `unmanaged-folder`, note there is nothing to act on
(not a git repo, no abcd markers).

## `install` — apply the outstanding gaps

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install --json
```

**This writes.** It applies the actionable gaps the detection pass found — the
marker block (only where `--docs-target` names a conventions file; the
default, `skip`, names none), the `.abcd/` scaffolding, the owned `PATH` entry. Lead
the report with the returned `headline`, then each `summary` item in its own
three parts: `what` it is, `why` it matters, and the `action`, if any, the user
should take. These are abcd's own plain words for the product thinker and the
technical facilitator; relay them rather than rewording, and keep the `refs`
(the exact paths and identifiers each item explains) for anyone who asks. Then
report any `notes` — a note is a refusal, stating something abcd deliberately
did not do and why. The engine prompts before an
ambiguous adoption, so surface any prompt to the user rather than answering it
for them.

A default install writes abcd's name into none of the repository's
conventions files. The one mention of abcd it commits outside `.abcd/` is
the pair of name-guard hooks (`.githooks/pre-commit`,
`.githooks/pre-merge-commit`) and the fenced block in `.gitignore`: the hooks
run the binary, the fence says not to hand-edit it, and both carry the marker
that later runs recognise an adopted repository by. That is a deliberate,
sanctioned exception, so do not offer to rename or strip it.

The `PATH` entry goes to `~/.local/bin` (created when absent), or to an
abcd-owned entry already on `PATH`, which is adopted exactly where it stands.
`--bin-dir <dir>` names a different directory — the only way to reach a
system-wide one — and fails loudly when it is not writable. abcd never escalates
privileges, so never suggest re-running any of this under `sudo`; report the
failure and let the user pick a directory they own. If the report carries a
`path.bin_dir_not_on_path` gap, relay its one-line `export` fix verbatim and
leave the user's shell profile alone.

The entry install writes is the abcd-owned copy of the verified release binary,
taken from the persistent plugin data directory a session's hooks provision.
When no verified copy is there, install writes no entry at all rather than a
symlink into the plugin root, which stops working at the next plugin update, and
its note names the command to run first: the install one-liner in the README,
which downloads the release binary, verifies it against that release's own
checksums and records it, after which a re-run of `ahoy install` adopts it.
Relay that note verbatim. A `symlink.legacy` gap is a symlink into the plugin
root that an earlier release wrote: it works until the next plugin update, and
its fix hint says whether install replaces it now or which command comes first.
A `symlink.dangling` gap whose detail calls the entry abcd-owned — including an
entry `~/.abcd/path-entry` records — is repaired by install the same way. When
that entry is not the one install acts on (a link a plugin update stranded ahead
of the copy the one-liner wrote), install removes it with its record once the
entry it does act on is working, and a note names what it removed.

A `symlink.shadowed` gap (or a note saying the same) means another `abcd` comes
first on `PATH`, so the entry abcd just wrote is NOT what runs — typically a
binary an older install copied into a system directory. Relay it prominently:
the install is not finished from the user's point of view. abcd will not remove
that binary, and neither should you offer to; state the two remedies it gives
(delete the stale one, or install ahead of it with `--bin-dir`) and let the user
choose. When the occupant is a link whose target is gone, the gap says it runs
nothing (the shell skips it) and asks for it to be removed; relay that, not the
"not what runs" framing above.

Prompts read stdin whether or not stdin is a terminal, so an answer can be
relayed without one:

```bash
yes | "${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install
```

**One line answers one question.** The install asks one approval per gap
category present — often several — and every line after the last one you supply
reads end-of-input and DECLINES. `yes` is the reliable form because it never
runs out; a single `printf 'y\n'` answers the first question only and silently
declines the rest. The questions come in a fixed order (dependency,
safe-autocreate, config-change, status-line, oracle-routing, user-state, plugin-owned), so a
scripted stream of specific answers lines up with them. Each answer is echoed back, so the
transcript shows what was asked and what it was answered — read it back rather
than assuming. Under `set -o pipefail` the pipeline reports 141: `yes` takes
SIGPIPE when abcd stops reading, by design — judge the run by abcd's own output
and exit status, not the pipeline's.

**Every value question arrives explained.** A question that picks one of
several values (`visibility`, `docs_target`, `oracle_backend`, `scan_deep`, the
house-style question and each status-line element) is printed with abcd's own
explanation above it: one paragraph saying what is being decided, then one
line per answer saying what that answer means, including what it asks of the
user (keys, tools, cost). When you relay such a question, relay that
explanation verbatim with it; never describe an answer in your own words, and
never offer an answer the question does not list. The four config values also
name the flag that answers them without the question (`--visibility`,
`--docs-target`, `--oracle-backend`, `--scan-deep`), and the matching gap's fix
hint names it too: in a scripted run, pass the user's answer through the flag
rather than lining a piped answer up with the question. `--yes` chooses no
value, so a `--yes` run that still has a value to ask says so once, above the
first such question.

That is a channel for passing on an answer the technical facilitator has GIVEN
— set `abcd mode facilitator`, ask the technical facilitator first, then pipe;
it is never a licence to answer on their behalf. Note that `yes |` approves
EVERY question, so only reach for it once they have agreed to all of them.

**Stdin must end, or the prompt waits.** With stdin at end-of-input every
question declines, so a run that was told nothing writes nothing — but a stdin
that is held open and silent (a pipe from a still-running command) makes the
prompt WAIT for the answer that never comes, rather than declining. For a run
that must not block and must not prompt, close stdin or pre-answer everything:
`abcd ahoy install --yes --refuse-adopt < /dev/null`.

`--yes` approves every resolvable category but never adopts the optional
git-identity pin, because the pin records whatever git identity is currently
configured, never wires the status line (below), because that rewrites a
harness-wide setting, and never accepts a model-tier routing table (below),
because a table decides which model every delegated step asks for. When the result carries `optional_skipped`, report it and
offer the `yes |` form above as the way to apply it.

**The git identity question is a person's alone.** When the author or committer
a commit would carry diverges from the identity pin, or is a machine identity
(`git_identity.mismatch`, `git_identity.unset`, `git_identity.committer`,
`git_identity.tool`), the install proposes the human identity (the pin, else
the global git identity) and asks whether to commit to this repository as it,
but only at a terminal. Off a terminal it asks nothing and writes nothing, and a
piped `yes` does not answer it; neither does `--yes`. Relay the result's note to
the user and tell them to run `abcd ahoy install` in their own terminal, or to
set `user.name` and `user.email` themselves; never set them on the user's
behalf. For an autonomous routine the human identity is set by whatever launches
it, before its first commit.

**The tool question.** When a `dependency` gap is present and its category is
approved, each missing tool is its own question, and a piped answer never
answers it: installing runs a program on the machine. At a terminal the install
shows the explanation and asks `Install <tool> now by running <step>? [y/N]`.
Through this page, ask the technical facilitator with the host's question tool
instead: present the gap's `tool` explanation (what it is, whether this
capability needs it, what works without it, the exact step, what the install
does), and only on their yes run

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install --install-tool <tool> --json
```

`--install-tool` is the relayed yes, for the named tool only, and it also
answers the `dependency` category's question, which is then not asked (so a
piped answer stream has one question fewer); never pass it without the user's
answer. The result's `changes` reports what ran and whether
its verify passed; a no, a failed step or a missing package manager is a `notes`
line ending in what the capability continues on (`continuing on the native
secret scanner`). The step runs in its own process group, bounded at 15
minutes, and a timeout kills that group; a process the step moves out of the
group (`setsid`) escapes the kill and can outlive the run, so abcd stops
waiting on its output 10 seconds after the step ends and reports a step that
left one behind as failed. `--yes` never installs a tool, and a run with `CI` set
to any value, or with `GITHUB_ACTIONS=true`, installs none and is not asked. A name that is not a tool `ahoy install` checks
for is refused, naming the ones it does.

**The house-style question.** When the install seeds `.abcd/docs-lint.json`,
it asks `docs_lint.em_dash_in_list_item (blocking/warning) [warning]`: whether an
em dash inside a list item, abcd's own house style rather than a currency rule,
blocks the docs lint or only warns. Relay the question to the technical
facilitator and pass on their answer; never answer it for them. The answer is
written into the seeded config as that token's severity (`blocker` or `warn`),
where the technical facilitator can change it later. `--yes` does not ask and
seeds a warning, and the result's `notes` says so. End of input or a bare Enter takes the warning. An answer that is
neither word (the `y` of `yes |`) also seeds the warning, with a note naming
what was heard. The question comes after the category approvals and the
configuration values and before the status-line offer, and is asked only when
the config is being created: a repository that already has one keeps its own
severity.

**The artefact kind.** A repository with no `.abcd/config/artefact.json` carries
an `artefact.missing` gap: the launch verbs choose what to preview, check and
scaffold by the kind declared there, and refuse to guess it. Once config changes
are approved, a repository carrying `.claude-plugin/plugin.json` is declared
`kind: plugin` without a question. Any other is asked
`artefact_kind (plugin/binary/application) [application]`: relay it and pass on
the user's answer, never answering for them. End of input or a bare Enter takes
`application` — gate plumbing with an empty build job, assuming nothing about the
build. `--yes` does not ask and declares `application`, and the result's `notes`
says so. An answer naming none of the three (the `y` of `yes |`) also declares
`application`, with a note naming what was heard; the user edits the file to
declare another. The question is the last one the install asks, after the
status-line offer. A declaration that is present but
refused by the reader the launch verbs share (an unknown kind, a malformed file)
is an `artefact.invalid` gap instead: the file is the user's, so the install
reports it and never overwrites it.

**The status-line offer.** When the harness's user-level settings file exists
(`$CLAUDE_CONFIG_DIR/settings.json`, or `~/.claude/settings.json`) and its
`statusLine` is absent or is a command that is not abcd's, the install asks
whether to install abcd's status line — one paragraph of reason, then one
question, then one on/off prompt per element after the badge (repository,
branch, model, context, five-hour and seven-day usage, intent and issue
counts; default on). Present the reason to the user and relay their answer;
never answer it for them. Consent writes exactly two files: the user-level
setting `~/.abcd/statusline.json` (the bundled defaults with the switches
taken, plus `previous_command` recording whatever the harness ran before) and
the harness's `settings.json`, whose `statusLine` is pointed at
`'<entry>' statusline` with every other key preserved. In an abcd-managed
repository the line then becomes abcd's own row, led by a badge saying whether
abcd is here and whose answer the loop is waiting on; in every other repository
the previous command runs untouched. Declining writes nothing and records
nothing, so the next install offers again; `--yes` skips the offer and reports
it under `optional_skipped`; `yes |` answers it (and keeps every element on). A
`statusLine` of a type abcd does not understand, or a `settings.json` that
does not parse, is refused with a note and nothing is written on either side.
`ahoy uninstall` restores the previous command. The line can be switched off
or reconfigured at any time in `~/.abcd/statusline.json`.

**The model-tier routing offer.** abcd ships a proposal for the model tier and
fan-out bound each of its agents deserves (`frontier` for the verdicts a person
reads, `economy` for the rest), and none of it applies until it is accepted.
While `~/.abcd/oracle-routing.json` is absent the install renders the proposal
as a table, one row per agent with its tier and fan-out, in one question;
consent writes it there, owner-only. A second, separate question offers the same
table for the repository at `.abcd/config/oracle-routing.json`, which is
committed, applies to everyone working in the repository, and wins over each
machine's table. Present the table and relay the user's answer to each question;
never answer them for the user. Declining writes nothing and records nothing, so
the next install offers again; `--yes` skips both offers and reports
`oracle_routing.machine_offered` and `oracle_routing.repo_offered` under
`optional_skipped`; `yes |` accepts both. Either file can be edited row by row
afterwards, and the bare `abcd` board shows which layer each agent's row comes
from. With no provider configured every step still runs through the harness,
which is asked for the tier. `ahoy uninstall` leaves both files, because they
are the user's configuration.

`--attribution` is its own approval and works on an already-installed repo (the
step the adopt phase runs it in). It opts the repo into the committed
`prepare-commit-msg` prompt,
which seeds a commented disclosure line into every commit message an editor
opens. It is opt-in and never a default — the hook stamps a convention onto
every commit message, which is a repo's choice to make — and it writes no
value, because which tool assisted is a fact only the committer has. The choice
is recorded in `.abcd/config.json`, so a later `install` without the flag keeps
the hook and restores a hand-deleted one; a `prepare-commit-msg` hook abcd did
not write is reported, never replaced.

For dogfooding abcd itself, `abcd ahoy install --dev` installs a track-latest
shim instead of the pinned owned copy: the `PATH` entry rebuilds abcd from
the source tip on every call and fails loudly on a broken build. Re-running
`abcd ahoy install` without `--dev` switches back to the pinned owned copy.

## `uninstall` — reversible marker-only removal

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy uninstall --json
```

**This writes.** It removes the BEGIN/END marker block and abcd's own `PATH`
entry — the owned copy (or a legacy pinned symlink), found wherever it sits on
`PATH`, along with its provenance record; a dangling link that record names is
removed with it even when no plugin root resolves any more — and leaves
`.abcd/` intact, so the
repo's record survives. The persistent download cache is left to the harness's
own uninstall to delete. Report `marker.removed` and the entry note; the
receipt's `symlink.target` is already rendered in tilde form, so relay it as
given rather than expanding it. When the harness's `statusLine` is abcd's, it
is handed back to the command recorded in `~/.abcd/statusline.json` before
abcd took the row — or removed, when none was recorded — and the receipt's
`status_line` says which; a status line that is not abcd's is left alone, and
`~/.abcd/statusline.json` itself stays, because it is the user's
configuration. It never touches `hooks.json`. An entry that was
installed with `--bin-dir` into a directory outside `PATH` cannot be found by a
`PATH` scan — pass the same `--bin-dir <dir>` to `uninstall` to remove it.

## `doctor` — the full read-only report

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy doctor --json
```

Runs the same detection pass plus a read-only audit sweep and reports **every**
gap, including user-scope state the bare render leaves out. Writes nothing.
Report the folder kind, the detection-gap count, and the audit-gap count, then
the per-gap detail from the JSON. This is the sub-verb to reach for when the bare
render says a repo is healthy and the user's experience says otherwise.

## `--remote` — the repo's GitHub security settings

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy --remote --json
```

Reports GitHub's two native secret-scanning toggles on the repository this
checkout's own origin remote names — `secret_scanning` and
`secret_scanning_push_protection` — and the changes an apply would make. It
writes nothing, on the remote or in the tree. A toggle it could not read is
reported `unknown`, never `disabled`: the two need opposite responses.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy remote apply --json
```

**This writes, and it is the one abcd verb that changes state outside this
machine.** It enables both toggles — secret scanning first, because GitHub
refuses push protection on a repository whose secret scanning is off — and
mirrors the desired state into `.abcd/work/rulesets/repo-settings.json`, so a
later verify reads the same intent from the tree rather than from a web console.
Push protection blocks a secret at push time, earlier than any CI scan, and
secret scanning covers the default branch continuously; both are free on public
repositories.

Never run it unasked. Four gates stand before any change leaves the machine, and
each refuses rather than guesses: the folder must be a repo abcd manages, the
repository must be the one this checkout's own origin names (no other repository
can be addressed, and a name that is not a plain GitHub name is refused before it
reaches an API path), the repo's config must not carry the opt-out, and the
caller must CONFIRM the specific toggles named. A repo that sets
`scan.native_secret_scanning` to `false` in `.abcd/config.json` is left exactly
as it is and is not contacted at all.

The confirmation is the fourth gate, not a formality: an unanswered run declines
and changes nothing, so set `abcd mode facilitator` and present the question
and the repository it names to the technical facilitator before answering it.
`--yes` says yes in advance, and it is the technical facilitator's word to give —
never pass it on their behalf. A run that changed nothing exits NON-ZERO
(`refused` or `aborted`), so a failed invocation is never mistaken for a write
that landed; `opted_out` is the one non-change that exits clean, because leaving
the repo alone is what the repo asked for.

The API host is pinned to github.com on every request. `gh` would otherwise take
it from `GH_HOST` or from whichever host the caller is authenticated to, which
would send this verb's authenticated write to a machine the origin URL never
named.

The call goes through the GitHub CLI (`gh`), so the write is made by the user's
own authenticated identity and abcd never holds a token; if `gh` is absent the
verb refuses, and the refusal carries the tool registry's explanation of `gh`:
what it is, that these verbs require it, the exact install step and what that
install does. Relay it; the step is the user's to run. It is idempotent — a repository already in the desired
state takes no write, and a re-run rewrites nothing in the tree — and it stops
at the first failed step rather than attempting one that cannot succeed. Relay
`status`, the resolved `repo`, every `change`, and every `note`: a note is a
thing abcd deliberately did not do, and the reason.

## `--providers` and `connect` — the optional model provider

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy --providers --json
```

Explains the optional OpenAI-compatible provider adapter and writes nothing. An
aggregator (OpenRouter, for one) serves many vendors' models behind one address
and one key, and a local OpenAI-compatible server is reached the same way. abcd
would use one for decision models and cheap judgements pointed at it by name,
and only for the models a provider's list names: a model the person does not
list, a frontier model included, stays on the host, and abcd bundles no vendor
denylist. Everything works without one: with no provider configured, every
delegated step runs on the host. Relay `explanation`, each of
`providers` with its `key_state` (`set`, `not set`, `none`, or a refusal; never
the key) and `key_home` (the home it resolves from), the `denylist` (the `oracle.denylist` entries the configuration writes, empty when none is), the `routes`, every line of `diagnostics`, and the
`key_homes` prose verbatim: it recommends the platform keychain in prose, and
the choice stays the person's, so never present one home as the marked option.
Relay `dispatch` too: no delegating verb sends a step to a provider yet, so a
configured provider changes no step until provider dispatch lands.

The bare board names the same adapter as an optional gap
(`oracle_api.none_configured`) while none is configured, and a configuration the
adapter refuses as `oracle_api.config_refused`, naming the file and the key.
Declining is not running `connect`, and it changes nothing.

The setup is `abcd ahoy connect <provider> --base-url <url> --model <model>
[--model <model>…] --home <home> [--key <name>]`. **This writes, under
`~/.abcd/` and, for the keychain home, into the platform keychain.** Set
`abcd mode facilitator` and ask the technical facilitator which home through
your question tool, after relaying `key_homes`, and offer the three without
marking one: `external` takes `--env <VARIABLE>` or
`--file ~/<file>.json --field <dotted.field>` (abcd keeps only where the key
is); `abcd` and `keychain` take the key piped in on stdin from a file or a
variable. It verifies the provider with one call to the first model listed,
and only when that call succeeds keeps the key in that home and writes the
provider block (the base URL, the key's name and the models, the allowlist)
into `~/.abcd/config.json`. Nothing goes into the repository or the harness's
settings, and a failed verification writes nothing. A `~/.abcd` that is a
symlink (into a dotfiles checkout, say) is refused with nothing written, in
any home, naming the link: the key would otherwise land wherever it points.
A `--file` pointer is refused, naming the link, when any directory between the
home and the tool's file is a symlink, wherever it leads (a `~/.config` linked
elsewhere, say); `--env` stays open.
`--home none` sets up a server that takes no key.

The key is read from stdin and nowhere else, and never from a terminal, where it
would be echoed. **Never ask the person for the key and never pass it
yourself**: it would enter this conversation. Give them the command to run in
their own shell, with the key piped in from a file or a variable they hold, and
relay the result — `verified` (the provider, the model asked for, the model
it reported and the credential's name), each `wrote` path, and `dispatch`.

## `credential` — the credential store's walkthrough

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy credential --json
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy credential <name> --json
```

Every external credential abcd holds lives in one store, in the home the person
chooses once per credential. Bare, the sub-verb lists each credential an
adapter reads (`hosting.cloudflare` for the site setup, each configured
provider's key) with its `state` (`set`, `not set`, or a refusal) and `home`;
never a value. With a name it explains that credential and writes nothing:
relay `unlocks`, `without_it`, then `homes_prose` verbatim (it recommends the
platform keychain in the prose; never present one home as the marked option),
then the `homes` and the `setup` command for each.

The walkthrough is `abcd ahoy credential <name> --home <home>`, with the same
three homes as `connect`: `external` with `--env`, or `--file` and `--field`;
`abcd` and `keychain` with the value piped in on stdin. **This writes the
chosen home only after the reading adapter's own verification call succeeds**
(the provider's one short exchange, the hosting provider's account read). Set
`abcd mode facilitator` and ask the technical facilitator for the home through
your question tool; never ask for the value, and never pass it yourself: give
the person the command to run in their own shell and relay `name`, `home`,
`verified` and each `wrote` entry. A name another home already holds, or a
different value for a name already kept, is refused: abcd never replaces a
stored secret. The abcd home is refused when `~/.abcd` lies inside a git
working tree (the keychain and an external home stay open
there), and a platform with no keychain tool refuses the keychain home and
names the other two.

## `--dry-run` — the canonical detection envelope

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy --dry-run
```

Renders the canonical `DetectionResult` JSON envelope and writes nothing — the
same pass `install` would apply, shown rather than applied. `--dry-run` always
emits JSON, so it needs no `--json` flag. Use it when the user wants to see
exactly what an install would do before letting it run.

## Scoping note: `--identity` is CLI-only

`abcd ahoy --identity` exits non-zero when the author or the committer a commit
would carry diverges from the committed pin. It exists to be wired into a pre-commit hook or CI, where
its exit code is the whole point, so it stays a bare-CLI entrypoint rather than a
plugin mode; report it only if a user asks how the identity gate fails
closed.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
