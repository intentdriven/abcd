---
name: ahoy
description: "Detect abcd's install state and list its gaps, or report one mode a flag names: Writes nothing; refuses any argument or two modes at once."
argument-hint: "[install | uninstall | doctor | --dry-run | --remote | remote apply | --providers | connect | credential]"
block: people
---

# `/abcd:ahoy` install/update detector

`abcd --help` lists `ahoy` in the person's set-up group. `statusline`, the
harness-invoked row that `install` wires, is in the agents-and-hosts block of
`abcd --help --agent`, and its line there names this page.

Run abcd's install/update engine for the current repo and present the result.
Bare invocation, its `--dry-run`, `--remote` and `--providers` modes, and the
`doctor` sub-verb perform **zero writes**; five change something. `install`,
`uninstall`, `remote apply` and `connect` each say so before they run —
`remote apply` is the only one that changes state outside this machine, and it
asks before it does — and `credential --home` writes the chosen home only after
the credential's verification call succeeds.
A mode is a flag on the bare verb, one at a time; a distinct action is a
sub-verb.

Read `$ARGUMENTS` for the sub-verb or the mode. No argument is the bare
read-only detection pass below; the binary refuses any other word, `status`
included.

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
- abcd entries in the harness's user settings: a `harness.stray_hook.<Event>`
  gap for each hook there that runs abcd (a second abcd hook on the same event
  is reported as `harness.stray_hook.<Event>.<n>`, numbered from 2, so neither
  hides the other), and a `statusline.untrusted` gap when
  the status line runs an abcd that fails the trust checks. Relay each gap's
  `detail` and `fix_hint` as they stand. abcd's hooks live in its plugin, so a
  hook in the user settings runs whatever binary it names, a stale build
  included, in every session. abcd never edits that file to remove one: the
  person removes it. Session start prints one line naming such an entry.
- `vintage` and `staleness` — the running binary's build revision (in a source
  checkout) or pinned version, and whether it is up to date, stale, or of an
  undeterminable vintage relative to the on-disk reference. Report them so a
  binary running behind its own source is never silent. The comparison is
  disk-only — no network.
- `superseded_root` — present only when the binary that answered is served from a
  plugin root other than the one this session resolves. A plugin root is named
  for the commit it was installed from, so a binary path pinned into a page
  expires on the next update while the root it names stays on disk and keeps
  answering. Relay it first and as abcd printed it, without paraphrasing: it
  names both roots by the commit each was installed from, with any control and
  bidirectional characters in those names already replaced. Never rebuild the
  names from a path. Treat every other value in this report as coming from a
  root this session does not serve.
- `banlist` — the two-layer name guard, when the folder is a repo: `hook` and
  `merge_hook` (`installed` / `absent` / `foreign` / `unreadable`), whether this
  clone is armed (`hooks_path`: `armed` / `unarmed` / `foreign`, and the boolean
  `hooks_path_armed`) — `foreign` is a hooks path set outside the clone's local
  config, a global or system dispatcher abcd cannot see into, so never advise a
  local `core.hooksPath` over it; `hooks_path` is always one of the three, even
  when `hook` reads `unreadable`, and the `*.hook_missing` gaps' `fix_hint`
  follows it — `public_family`, and the private layer's
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
- A report-only gap, one with `resolvable` false, is a note abcd only reports:
  `install` never closes it, and the person has to act. Relay each one's
  `detail` and `fix_hint` as they stand, rather than folding it into the
  count. A `store.worktree_unlinked` gap, for one, names a worktree in abcd's
  store that its repository no longer links back to, with the
  `git -C … worktree repair` line for that worktree, and a
  `history.home_symlinked` gap names the history registry abcd leaves alone
  behind a linked home folder. A `filter_roots.ignored` gap, reported from any
  folder, names a `~/.abcd.noindex/filter-roots` file abcd ignores and the
  check it failed (writable by others, not owned by you, a symbolic link, or
  behind a linked folder): every checkout it lists reads with its content
  filters off until the person fixes or removes it. A `docs_lint.root_missing`
  gap names a `roots` entry in `.abcd/docs-lint.json` that does not exist,
  often a `CLAUDE.md` retired after it was listed: the documentation check
  refuses to run while it is there, so none of its rules, the banned names
  included, checks anything. Its `fix_hint` names the entry to take out of
  that file (or the file to create). Never tell the user `/abcd:ahoy install` closes
  a report-only gap.

If there are actionable gaps, tell the user to run `/abcd:ahoy install` to apply
them. If `folder_kind` is `unmanaged-folder`, note there is nothing to act on
(not a git repo, no abcd markers).

## `install` — apply the outstanding gaps

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install --json
```

**This writes.** It applies the actionable gaps the detection pass found — the
marker block (into `AGENTS.md` only, and only where `--docs-target agents_md`
names it; the default, `skip`, names none), the `.abcd/` scaffolding, the owned
`PATH` entry. A saved `claude_md` or `both` is refused before any write, with
the one setting to change in the notes; relay that note as it stands. A
`refused` install (that setting, a `.abcd` that is not a real directory, or a
stale binary) writes nothing and exits 2 after rendering its result, so the
notes still say why. Before
anything else, relay each line of the returned `warnings` as it stands: each
names an agent tool's own conventions file holding the user's words, which that
tool reads in place of `AGENTS.md` and abcd never edits or removes, or what
hides `AGENTS.md` from outside the repository's own files (a personal
conventions file at the root, one in a folder above the repository, or an
agent tool older than the release that reads `AGENTS.md` on its own); the text
render prints them first too. Then lead
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
entry `~/.abcd.noindex/path-entry` records — is repaired by install the same way. When
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
safe-autocreate, config-change, status-line, oracle-routing, drain-rule, conventions-file, user-state, plugin-owned), so a
scripted stream of specific answers lines up with them. Each approval lists
what it would change above its question, one line per change naming the file
or the setting, and asks in plain words ("Change the settings listed above?").
Every change is listed, on every route, however long the list. When you relay an
approval, quote its list with it, exactly as written. The drain-rule and
conventions-file questions are asked only at a terminal, so a piped stream
never meets them. Each answer is echoed back, so the
transcript shows what was asked and what it was answered — read it back rather
than assuming. Under `set -o pipefail` the pipeline reports 141: `yes` takes
SIGPIPE when abcd stops reading, by design — judge the run by abcd's own output
and exit status, not the pipeline's.

**Every value question arrives explained.** A question that picks one of
several values (`visibility`, `docs_target`, `scan_deep`, the
house-style question and each status-line element) is printed with abcd's own
explanation above it: one paragraph saying what is being decided, then one
line per answer saying what that answer means, including what it asks of the
user (keys, tools, cost). The explanation is the one for this repository: the
`public` visibility answer adds that git cannot hide records it already tracks
only where `.abcd/` holds tracked files. When you relay such a question, relay that
explanation verbatim with it; never describe an answer in your own words, and
never offer an answer the question does not list. The config values asked
also name the flag that answers them without the question (`--visibility`,
`--docs-target`, `--scan-deep`), and the matching gap's fix
hint names it too: in a scripted run, pass the user's answer through the flag
rather than lining a piped answer up with the question. A question no flag
answers, such as the artefact kind, ends its explanation instead on a
`(Change later: <where>)` line saying where the answer is changed afterwards.
`--yes` chooses no value, so a `--yes` run that still has a value to ask says so
once, above the first such question.

**The AI reviewer is recorded, not asked.** `oracle_backend` has one answer
abcd ships an adapter for, `host-delegated` (the assistant the user is working
in), so install records it without a question and says so in one note, which
names `--oracle-backend` as the way to choose another reviewer once one
arrives. Relay that note; do not ask the user to choose a reviewer. The flag
still sets the value in a scripted run, and the question returns on its own once
a second answer has an adapter.

That is a channel for passing on an answer the technical facilitator has GIVEN
— ask the technical facilitator first, under a `Tech` chip, then pipe;
it is never a licence to answer on their behalf. Note that `yes |` approves
EVERY question, so only reach for it once they have agreed to all of them.

**At a terminal the questions are drawn; through this page they are relayed
in an answers file.** When stdin, stdout and stderr are all terminals, the
install draws each question itself (the material, the question, each answer
with its meaning, and decide later last), answered by the arrow keys or by
its number, and Ctrl-C ends the run with exit 130, keeping the answers given
before it. Every answer is recorded, with the question as it was asked and
where it was answered, in `.abcd/.work.local/interviews/setup-<stamp>.json`,
and the answers that change the machine (the status line, the machine's
routing table) in `~/.abcd.noindex/interviews/`; a run that ends aborted or
refused changed nothing and records nothing. Through this page, put each
question to the user with the host's question tool, quoting the question
exactly as the install writes it, then pass their answers in an answers file
and say where they were given:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install --answers <file> --answered-in "Claude Code" --json
```

The file is one JSON object: `{"schema_version": 1, "interview": "setup",
"answers": [{"id": "adopt", "value": "yes"}, {"id": "visibility", "value":
"private", "note": "<optional>"}]}`. Each answer names its question by id
(a value question's key, `adopt`, `approve.<category>` for a kind of change,
or the offer's gap id) and gives one of the values the question offers, or
`later` to decide later; an unknown or repeated key is refused. With
`--answers`, each question is written as plain text on stderr. A value
the question does not offer, or a missing answer to the adoption, an
approval or a setting value the run would ask (the visibility, the docs
target, deep scanning), stops the run with exit 2 before anything is
written, naming the question's id, the flag that answers it, if any, and the
line to add; nothing is recorded. Put that question to the user, add their
answer, and run again. The questions the run itself decides to ask (the
status line's elements, the offers, the artefact kind) are checked when
they are asked: a stop there leaves the steps before it done, says so, and
the next run asks only what is still open. The git identity, the
drain rule and installing a tool are never answered from a file. `--answers`
replaces the piped answer stream for that run.

**A value flag needs the settings approval.** `--visibility`, `--docs-target`,
`--oracle-backend` and `--scan-deep` are held to the config-change approval
like any other settings change: a flag that would change a saved setting puts
the config-change question even when nothing else would, and only a yes, given
at the prompt, as `approve.config-change` in the answers file, or by `--yes`,
saves it. `later`, a no or end of input drops the flag: nothing it would change
is saved or moved, and the run names each dropped flag in a note.

**Stdin must end, or the prompt waits.** With stdin at end-of-input every
question declines, so a run that was told nothing writes nothing — but a stdin
that is held open and silent (a pipe from a still-running command) makes the
prompt WAIT for the answer that never comes, rather than declining. For a run
that must not block and must not prompt, close stdin or pre-answer everything:
`abcd ahoy install --yes --refuse-adopt < /dev/null`.

`--yes` approves every resolvable category but never adopts the optional
git-identity pin, because the pin records whatever git identity is currently
configured, never wires the status line (below), because that rewrites a
harness-wide setting, never accepts a model-tier routing table (below),
because a table decides which model every delegated step asks for, and never
adds the drain eligibility record (below), because the record decides what an
unattended agent may change in the repository. When the result carries `optional_skipped`, report it and
offer the `yes |` form above as the way to apply it, except `drain_rule.offered`
and `conventions.retire_offered`, which only a person at a terminal is asked.

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

**The seeded documentation check.** A repository with no
`.abcd/docs-lint.json` gets one. Its `roots` list `docs` when that folder
exists and `README.md` when that file exists, at install time, so the check it
arms runs rather than refusing over a root that is not there; with neither,
`roots` is empty, and `abcd lint docs` reads nothing and says so in a warning at
exit 0 until the user lists their documentation there. An existing config is
never changed.

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
counts; default on). The question shows the one change: the file, and the
`statusLine` entry's value now and after. Present the reason and the change to
the user and relay their answer; never answer it for them.

`statusLine` is the only entry abcd writes in the harness's user settings, and
only after a yes. Consent writes exactly two files: the user-level setting
`~/.abcd.noindex/statusline.json` (the bundled defaults with the switches
taken, plus `previous_command` recording whatever the harness ran before) and
the harness's `settings.json`, whose `statusLine` is pointed at
`'<entry>' statusline` with every other key preserved. Before the harness file
changes, a copy of it goes into `~/.abcd.noindex/backups/`, which keeps the
newest 10; no copy, no write. After the change the file is read back, and one
that does not say what was written is put back from the copy. `<entry>` is the
PATH install `~/.abcd.noindex/path-entry` records, and only when it passes the
trust checks: the binary, a link's target when the entry is a link, is owned
by the user and writable by nobody else, every directory it is reached through
is writable by neither its group nor every account and is owned by the user or
root, and none of them is inside the working tree; it is never the plugin's
own copy, whose
versioned folder a plugin update deletes. A current command that already runs
abcd's status verb is never recorded as the previous one.

In an abcd-managed repository the line then becomes abcd's own row, led by a
badge saying whether abcd is here and whose answer the loop is waiting on; in
every other repository the previous command runs untouched. Declining writes
nothing and records nothing, so the next install offers again; `--yes` skips
the offer and reports it under `optional_skipped`; `yes |` answers it (and
keeps every element on). A `statusLine` of a type abcd does not understand, a
`settings.json` that does not parse, or a PATH install that fails the trust
checks is refused with a note, and nothing is written on either side. The line
can be switched off or reconfigured at any time in
`~/.abcd.noindex/statusline.json`.

**The status-line repair.** When abcd's own status line runs an abcd that is
gone (`statusline.dangling`) or one that fails the trust checks
(`statusline.untrusted`), install repairs it as config-change work, `--yes`
included: it points the line at the trusted recorded PATH install, or, with
none, hands it back to the recorded previous command, or removes it when none
was recorded. The repair keeps a copy and reads the file back like the wiring
does. A status line that runs abcd for some other purpose is the person's own,
and is only reported.

**The model-tier routing offer.** abcd ships a proposal for the model tier and
fan-out bound each of its agents deserves (`frontier` for the verdicts a person
reads, `economy` for the rest), and none of it applies until it is accepted.
While `~/.abcd.noindex/oracle-routing.json` is absent the install says the proposal in
counts, in one question: how many agents, how many at each tier, and their
fan-out bounds, naming no agent, because a row per agent does not fit one
question. Consent writes the full table there, owner-only, where every row can
be read and edited. A second, separate question offers the same
table for the repository at `.abcd/config/oracle-routing.json`, which is
committed, applies to everyone working in the repository, and wins over each
machine's table. Present that text and relay the user's answer to each
question; never answer them for the user. Each routing question, like every question
this page asks through your question tool, follows the asking rules in
`commands/intent.md` (the block marked `generated: asking-rules`): the text
being decided first, quoted in full, and the question last. Declining writes nothing and records nothing, so
the next install offers again; `--yes` skips both offers and reports
`oracle_routing.machine_offered` and `oracle_routing.repo_offered` under
`optional_skipped`; `yes |` accepts both. Either file can be edited row by row
afterwards, and the bare `abcd` board shows which layer each agent's row comes
from. With no provider configured every step still runs through the harness,
which is asked for the tier. `ahoy uninstall` leaves both files, because they
are the user's configuration.

**The drain eligibility record offer.** `abcd drain` takes an open issue alone
only under a rule the repository records for itself, and refuses to run until an
accepted decision record in `.abcd/development/decisions/adrs/` states it in
four frontmatter fields (`drain_categories`, `drain_severities`,
`drain_security`, `drain_remedy`). While no accepted record carries them, the
install states abcd's strict baseline in one question (take an issue only when
its category is `tech-debt`, `documentation`, `inconsistency`, `drift`, `bug` or
`ux`, its severity is `nitpick` or `minor`, it carries a remedy and nothing open
blocks it; every security, major and critical issue is a person's); consent
mints it through the decision store's own seam as an accepted record, which is
committed with the repository. Relay the user's answer; never answer it for the
user. Declining writes nothing and records nothing, so the next install offers
again. It is asked only at a terminal, as the git identity question is: off one
(a pipe, a routine, CI) neither its category nor the offer is asked, so a
scripted answer stream keeps its order and a scripted yes never writes the
record; that run, and a `--yes` run, report `drain_rule.offered` under
`optional_skipped`. The offer only ever writes the baseline:
loosening a floor is an edit a person makes to the record, and `abcd drain`
names every floor loosened. A repository whose record states the rule badly is
not offered a second one; `abcd drain` names what is wrong with the one it has.

**An agent tool's own conventions file.** Some agent tools read a file of their
own in place of `AGENTS.md` when it exists: `CLAUDE.md` or `.claude/CLAUDE.md`,
`GEMINI.md`, and `.rules`, `.cursorrules` or `.github/copilot-instructions.md`.
The install classifies each one at the repository root with one guarded read,
never through a link. One holding the user's words (or one it cannot read
whole) is never edited, moved, merged or removed; it raises the warning
`conventions.owner_file`, reported in `warnings`. One that only repeats
`AGENTS.md` (a link to it, that link saved as a plain file, an exact copy, a
lone `@AGENTS.md` line, or a file empty once abcd's block is taken out) raises
`conventions.retire_offered`, and after the conventions-file approval the
install asks, one question per file, `retire`, `keep` or `later` (the default).
Relay each question as it stands and never answer it for the user. On `retire`
the file is checked again and removed from the working tree only if it still
only repeats `AGENTS.md`, for the user to commit; a file that changed while the
question was open is left and named in the notes. The install does not edit
`.abcd/docs-lint.json`: a retired file still listed in its `roots` is reported
by the next `ahoy` as `docs_lint.root_missing`, for the user to remove. `keep` and `later` write and
record nothing, so the next install asks again. Like the drain rule offer it is
asked only at a terminal; a piped run and a `--yes` run report
`conventions.retire_offered` under `optional_skipped`.

**What hides `AGENTS.md` from outside the repository.** The install, and only
the install, also warns about a personal `CLAUDE.local.md` at the root, a
`CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` in any folder above the
repository (not the user-level `.claude/CLAUDE.md` in the home folder), and a
`claude` command on `PATH` older than the release that reads `AGENTS.md` on its
own. Each is a warning in `warnings`, never a refusal. The folders above are
asked only whether a file of that name exists: nothing in them is read, and
abcd takes no settings from them. Relay each warning as it stands; never offer
to edit, move or remove one of those files.

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
is handed back to the command recorded in `~/.abcd.noindex/statusline.json` before
abcd took the row — or removed, when none was recorded — and the receipt's
`status_line` says which, naming the copy of the file kept in
`~/.abcd.noindex/backups/` first; a status line that is not abcd's is left
alone, and `~/.abcd.noindex/statusline.json` itself stays, because it is the
user's configuration. Run `uninstall` before removing abcd: a status line left
pointing at a removed abcd is blank in every repository. It never touches
`hooks.json`. An entry that was
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
and changes nothing, so present the question and the repository it names to
the technical facilitator, under a `Tech` chip, before answering it.
`--yes` says yes in advance, and it is the technical facilitator's word to give —
never pass it on their behalf. A run that changed nothing exits NON-ZERO
(`refused` exits 2, as a refused install does; `aborted` exits 1), so a failed
invocation is never mistaken for a write that landed; `opted_out` is the one
non-change that exits clean, because leaving the repo alone is what the repo
asked for.

The API host is pinned to github.com on every request. `gh` would otherwise take
it from `GH_HOST` or from whichever host the caller is authenticated to, which
would send this verb's authenticated write to a machine the origin URL never
named.

The call goes through the GitHub CLI (`gh`), so the write is made by the user's
own authenticated identity and abcd never holds a token. If `gh` is absent, the
verb explains it from the tool registry (what it is, that these verbs require
it, the exact install step and what that install does) and offers to install
it, running the step only on a yes typed at a terminal. `--yes` never answers
that offer, and neither does a piped answer, so through this page the offer is
declined: the verb refuses, and its notes carry the explanation and the command.
Relay them; the install is the user's to run, by hand or by running the verb at
a terminal. `ahoy --remote` never offers the install, since it writes nothing,
and names this verb as the one that does. It is idempotent — a repository already in the desired
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
list, a frontier model included, is never asked for, and the record shows what
answered; abcd bundles no vendor denylist. Everything works without one: with no provider configured, every
delegated step runs on the host. Relay `explanation`, each of
`providers` with its `key_state` (`set`, `not set`, `none`, or a refusal; never
the key) and `key_home` (the home it resolves from), the `denylist` (the `oracle.denylist` entries the configuration writes, empty when none is), the `routes`, every line of `diagnostics`, and the
`key_homes` prose verbatim: it recommends the platform keychain in prose, and
the choice stays the person's, so never present one home as the marked option.
Relay `dispatch` too: a delegating verb whose agent's `oracle.roles` entry
points at a provider sends the step there itself and ingests the answer, and
every other step runs on the host. A provider whose block names a key takes
only the self-contained agents, the four cold-reading positions (ruling DR5 of
2026-09-29); every other agent pointed at one is refused before any call.
`oracle.bundled_context_providers` in `~/.abcd.noindex/config.json` is the person's
override, naming providers that may take bundled-context requests for
file-reading agents; it is read from the machine alone, a repository declaring
it is refused, and it admits an agent only once abcd builds that agent's
bundle, which none has yet.

The bare board names the same adapter as an optional gap
(`oracle_api.none_configured`) while none is configured, and a configuration the
adapter refuses as `oracle_api.config_refused`, naming the file and the key.
A route the configuration read skips (a repository's route to a provider that
holds a key, a repository's route that is not `<provider>/<model>` or whose
name is not a plain lower-case name, a repository's route to a model a provider
holding no key does not list, a route to a provider this machine has not
configured, or a role outside the roster) is the optional gap `oracle_api.route_skipped`, its
`detail` one line per skipped route; relay each line. A skipped
repository route never displaces the machine's: where `~/.abcd.noindex/config.json`
routes the same name, that route applies.
Declining is not running `connect`, and it changes nothing.

### The guided path: `connect --guide`

When the person wants to connect a service from this session, guide them
rather than asking for every value. The guide works the values out one
question a turn, writes nothing, never asks for the key, and ends by printing
the one command for the person to paste into a terminal on this machine. It
never runs that command, and no flag makes it: `--guide` with `--home`,
`--model`, `--key`, `--env`, `--file` or `--field` is refused.

Start it, with the provider's name and the address when the person has given
them (both are optional; the guide asks for the address and names the provider
after the host):

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy connect [<provider>] --guide [--base-url <url>] --json
```

Each run prints one turn. A turn carries `ask`, the question, and `tool`,
the same question as your question tool's input: ask it through your question
tool exactly as `tool` gives it, one question a turn, never reworded, never
with a preview and never with an option marked. A question with a typed part
(`ask.questions[0].typed`) takes the person's own text in the tool's row for
typing; where it lists too few options for the tool, `tool` carries one more,
"Type my own answer", which only points at that row. Then pass the answer
back, the turn's `resume` member unchanged on stdin and the answer as the tool
returned it (an option's label, or the text typed). The `resume` member goes
in a quoted heredoc, never as an argument: it carries the models the service
listed, up to 32 KiB of their names, and the count of any it does not carry.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy connect --guide --resume - --answer '<answer>' --json <<'RESUME'
<resume JSON>
RESUME
```

Do no other bookkeeping: the guide replays every answer from the first
question each turn, refuses (exit 2, naming the question) an answer a
question does not take or a `resume` that was edited, and asks a typed
question again, saying why, when the text cannot be used.

The questions, in order: the address, when none was given; whether to look
up the models the service lists, showing the scheme and host first (only a yes
sends one request, carrying no key, following no redirect, and giving up after
ten seconds); the model, offering the models the person's other connections
already use that the service lists, at most three, and narrowing the list as
they type part of a name, with no second request (when the service lists more
models than the guide carries and a part matches none carried, it asks for the
model's full name); where the service publishes
no list, or the person declines the look-up, a typed model name and the reason;
whether the service takes a key; where the key lives, offering the three
homes and never a key the person saved in the system keychain by hand; and,
for an environment variable, its name, offering at most three variables whose
names end in `_API_KEY`, never their values. A service that lists its models
only for a key skips the model question and the key question: the command
then has no `--model`,
`done.picks_in_terminal` is true, and the person picks in the terminal once
the command has the key.

A turn carrying `stopped` is the person's decide later: relay it verbatim,
and keep `resume` to pick up there. A turn carrying `done` is the end: relay
`done.command` on a line of its own and every line of `done.writes` verbatim,
the paths the command writes, then tell the person to paste the command into a
terminal on this machine, where it asks for the key on hidden input when the
home stores one. **Never ask for the key, never pass it, and never run the
command for the person.**

### By hand

The setup is `abcd ahoy connect <provider> --base-url <url> --model <model>
[--model <model>…] --home <home> [--key <name>]`. **This writes, under
`~/.abcd.noindex/` and, for the keychain home, into the platform keychain.** Ask the
technical facilitator which home through your question tool, under a `Tech`
chip, after relaying `key_homes`, and offer the three without
marking one: `external` takes `--env <VARIABLE>` or
`--file ~/<file>.json --field <dotted.field>` (abcd keeps only where the key
is); `abcd` and `keychain` take the key piped in on stdin from a file or a
variable, or, when the person runs the command in a terminal, pasted on hidden
input: abcd prints one line on stderr, "Paste the key for <provider> and press
Enter. It is not shown.", and reads the key with echo off. An empty paste is
refused as an empty pipe is. It verifies the provider with one call to the first model listed,
and only when that call succeeds keeps the key in that home and writes the
provider block (the base URL, the key's name and the models, the allowlist)
into `~/.abcd.noindex/config.json`. Nothing goes into the repository or the harness's
settings, and a failed verification writes nothing. A `~/.abcd.noindex` that is a
symlink (into a dotfiles checkout, say) is refused with nothing written, in
any home, naming the link: the key would otherwise land wherever it points.
A `--file` pointer is refused, naming the link, when any directory between the
home and the tool's file is a symlink, wherever it leads (a `~/.config` linked
elsewhere, say); `--env` stays open.
`--home none` sets up a server that takes no key.

Run in a terminal (stdin, stdout and stderr all terminals) with no `--model`,
the command lists the service's models with the key it holds: the pasted key,
the value the `external` pointer names, or none for `--home none`. The list is
one request to the service's model list, which follows no redirect and gives
up after ten seconds. Only the names abcd accepts as model identifiers are
offered, in the plain-Terminal list, typing part of a name to narrow it, and
the model picked is verified with one real completion before anything is
written; a list is never taken as the verification. A list that cannot be
read, a list with no usable name, decide later, and a completion that fails
each write nothing and say which; Ctrl-C at the list exits 130 with nothing
written. Off a terminal, a run with no `--model` is refused, naming both ways
on: `--model`, or running the command in a terminal to pick there.

The key is read from stdin and nowhere else: piped, or at a terminal on hidden
input, never echoed. Ctrl-C during the paste exits 130, the terminal restored
and nothing written. **Never ask the person for the key and never pass it
yourself**: it would enter this conversation. Give them the command to run in
their own terminal, where they paste the key on hidden input or pipe it in from
a file or a variable they hold, and relay the result — `verified` (the provider, the model asked for, the model
it reported and the credential's name), each `wrote` path, and `dispatch`. A
route the configuration read skips is named on stderr, in the text and the JSON
form alike, and the setup stands: relay that line too.

## `credential` — the credential store's walkthrough

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy credential --json
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy credential <name> --json
```

Every external credential abcd holds lives in one store, in the home the person
chooses once per credential. Bare, the sub-verb lists each credential an
adapter reads (`hosting.cloudflare` for the site setup, each configured
provider's key) with its `state` (`set`, `not set`, or a refusal) and `home`;
never a value. A route the configuration read skips (a repository's route to
a provider that holds a key, or one that is not `<provider>/<model>` or whose
name is not a plain lower-case name) is named on stderr and the listing goes on: relay
that line too, as with a name that is a provider's credential, whose read of
the configuration names it the same way. With a name it explains that credential and writes nothing:
relay `unlocks`, `without_it`, then `homes_prose` verbatim (it recommends the
platform keychain in the prose; never present one home as the marked option),
then the `homes` and the `setup` command for each.

The walkthrough is `abcd ahoy credential <name> --home <home>`, with the same
three homes as `connect`: `external` with `--env`, or `--file` and `--field`;
`abcd` and `keychain` with the value piped in on stdin. **This writes the
chosen home only after the reading adapter's own verification call succeeds**
(the provider's one short exchange, the hosting provider's account read). Ask the
technical facilitator for the home through your question tool, under a `Tech`
chip; never ask for the value, and never pass it yourself: give
the person the command to run in their own shell and relay `name`, `home`,
`verified` and each `wrote` entry. A name another home already holds, or a
different value for a name already kept, is refused: abcd never replaces a
stored secret. The abcd home is refused when `~/.abcd.noindex` lies inside a git
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
