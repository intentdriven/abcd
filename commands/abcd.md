---
name: abcd
description: "Render the status board, or say what one record id is and its next move: Writes nothing; refuses any other positional argument."
argument-hint: "[<record-id>]"
block: people
---

# `/abcd` where-am-i

Run the abcd binary's read-only status board for the current repo and show it to
the user. This command performs **zero writes**.

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" --format markdown
```

Paste its output to the user unchanged, in one fenced block: add nothing inside the fence, and retell none of it outside the fence. The output is the view for the product thinker, a few plain lines: what is being built, the next three things to build, and how many more are ready or parked. It is drawn for a person to read as it stands.

## The version

The board's last line is the installed abcd version, in both views and both
forms (`abcd v0.13.4`, a list item in the markdown form), and `--json` carries
it as `version`. It is the version `abcd --version` reports, read off the
binary with no network. When the user asks how abcd was installed, how old it
is, or whether it has drifted from the reference it should match, run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" --version --json
```

and tell them the `name`, `version`, `vintage` and `staleness`, plus
`install_mode` when it is present. The key is omitted when no abcd-owned `PATH`
entry resolves (nothing installed yet, a foreign or dangling entry, or an
unresolved plugin root); then say abcd is not on `PATH` yet and point at
`ahoy install` (see *Binary resolution* below) rather than inventing a mode.
Whether a newer release exists is the update verb's check
(`/abcd:update`); ask it only when the user does.

**If `superseded_root` is present, say it FIRST.** It means the binary that just
answered, and so the version on the board, is served from a plugin root other
than the one this session resolves: the version reported is true of that root
and false of this machine. A plugin root is named for the commit it was
installed from, so an absolute binary path pinned into a page expires on the
next update while the root it names stays on disk and keeps answering. Relay
the note as abcd printed it, without paraphrasing: it names both roots, each by
the commit its root was installed from, and abcd has already replaced any
control and bidirectional characters in those names. Never rebuild the names
from a path, and never re-decorate them. Then re-run the board through this
session's own plugin root (reload the plugin surface if the path this page gave
you is the stale one) before reporting a version at all. The note is silent
when the answering binary sits in a source checkout of abcd, whose currency the
`staleness` field already reports.

## The view for the facilitator

When the user asks for the full board, run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" --view facilitator --format markdown
```

and paste it the same way. For the fields below, when you need to reason over
them rather than show them, run `"${CLAUDE_PLUGIN_ROOT}/abcd" --json`: it carries
every field whichever view it names (`view`).

The full board names the directory (`dir`, named
home-relative as `~/…`, or by its directory name outside HOME, never by an
absolute path), whether it is a git repo, whether the abcd development record is
present, and which `.abcd/` work tiers exist.

In a repository abcd manages the board also carries one line of presence — the
`statusline` object in the JSON (`state`, `plain`, `elements`), rendered as a
`presence:` line in the text form. It is the same row the host's status line
shows, in plain words: the badge first (`abcd-managed`, `waiting on the
technical facilitator` or `waiting on the product thinker`, from the state
`/abcd:mode` stores), then the
repository, the branch and the record's counts. Relay the `plain` text when the
state is not `managed`: it says whose answer the loop is waiting on. The field
is omitted in a repository abcd does not manage. The board reads the state and
never changes it; `/abcd:mode` is the writer.

When a sibling worktree or a local branch of this checkout holds a record that
differs here, the board also carries a `peers` object (`live`, `ids`), rendered
as a `peers:` line. Relay it, and point at `/abcd:peers` for the whole picture.
It is omitted when no peer holds anything that differs.

The row itself is produced by `abcd statusline`, the verb the harness runs on
every status refresh with its JSON payload on stdin. In a managed repository it
prints abcd's row; anywhere else it runs the status command that was recorded
at install time and passes its output through unchanged, so the user's own line
is untouched. It writes nothing, and it is time-bounded: abcd's own work stops
at 500ms and the recorded command at 5s, and at either bound it prints what it
has and exits 0. `/abcd:ahoy install` offers and wires it, after a yes and only
to the trusted PATH install abcd recorded, never to the plugin's own copy;
nothing here invokes it.

When reports from managed repositories wait in the user account's inbox, the
board carries an `inbox` object (`reports`, `senders`), rendered as an `inbox:`
line. Relay the count and point at `/abcd:inbox`, which lists them; the field is
omitted when nothing waits.

Once a model-tier routing table is accepted — `.abcd/config/oracle-routing.json`
in the repository or `~/.abcd.noindex/oracle-routing.json` on the machine — the board
carries an `oracle` array, one object per agent (`agent`, `winner`, `layers`,
each layer with `layer`, `origin`, `tier` and `fan_out`), rendered as an
`oracle:` heading and one line per agent: every layer that holds a row as
`layer=tier`, highest precedence first, the one that applies marked `*`. Relay
the agents whose winning row is not the bundled one. The field is omitted when
no table is accepted, and every delegated step then runs through the harness at
`host-decides`. An orphan row (a name that is no agent) and a fan-out clamped
to the agent's ceiling are reported on stderr; a routing file that cannot be
read omits the lines and says why there.

When review folders sit under `.abcd/work/reviews/`, the board carries a
`reviews` object (`threshold`, `default_ref`, `rows`). Each row names the
folder (`folder`, `kind` — `review` or `receipt` — `scope`, and `spec` when the
folder is a spec's review), the commit it read (`review_of_commit`), and how
many commits the default branch has moved since (`commits_since`); `stale` is
true past the threshold of twenty. A row whose `state` is `unreachable` names a
commit this history no longer holds, and one that is `unpinned` predates the pin
rule; neither has a count. The text render is a `reviews:` heading and one line
per dated review, stalest first, a stale one marked `!`, then one `receipts:`
line: a release receipt gates the release it names and is never re-run, so the
line gives how many receipts there are, how far behind the oldest release it
gated is, and how many pins this history no longer holds, and `--json` lists
each receipt as a row. Relay the reviews marked `!` as reviews to re-run before
they are trusted, and a receipt's count as how old that release is, not as work
to do. The field is omitted when the tree holds no folder.

In a repository abcd manages the board ends with the Now / Next / Later block,
the `status` object in the JSON (`now`, `next`, `later`, `order`), rendered as
a `status:` heading with the three counts, one line per intent under `Now:` and
`Next:`, and a closing `Later: N intents` line that gives Later as a count
alone: its rows are in the JSON and on the site's Status page. Now is every
intent a build run has in a lane (each row's `lane` names the run, the lane,
its next stage and the role it waits on, its `branch` once cut, and
`in_flight` while that branch exists and the spec is open), then the intent
marked `next_up`; Next
is every other planned intent the readiness gate reports READY, since the
`next_up` intent is listed under Now alone; Later is every
planned intent the gate refuses, its `failing_checks` named, then the drafts.
A planned intent with a spec carries it as `spec_id` on its row. A planned
intent that names the release it must land by carries it as
`target_release` on its row, in any list, and its text line shows `target
<value>` in the brackets. An intent in a lane is listed under Now only, never also under Next or Later.
Next and the `next_up` intent are read
in `abcd build next`'s pick order (`order` is `pick`): the readiest first by the
pick's score, the oldest among equals, and the head passes over an intent that
`abcd build next` refuses from the record alone (an open question, an
unanswered claim section, a hold, an unsettled blocker, no step left to build)
or that is already in a lane, or that another checkout holds (build next's
peers check), so `next_up` is always the intent `abcd build next` would pick.
Relay Now first: it is what is being built and
what comes next. The block is computed each time and nothing stores it.

## Record-id dispatch

Bare answers *what can I do*; `abcd <id>` answers *what is this, and what is
my next move*. A positional matching `^(iss|itd|spc|adr|adm|srp|rfm)-[0-9]+$` locates the
record in its store — any status folder or bucket — and renders it read-only:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" <record-id> --json
```

The digits take one of two shapes, and both resolve: A short ordinal from
before ids were minted (`iss-188`), or the sixteen-digit timestamp the
minting verbs allocate (`iss-2609120452369809`). A ledger part-way through
adoption holds both, and neither is migrated away: Nothing renumbers an
existing record, because an id is a citation. An `N` in a placeholder such as
`<iss-N>` stands for either shape.

Summarise the `id`, `family`, `status`, `title`, `path`, the `links` edges
(`spec_id`, `intent`, `intents`, `related_intents`, `related_issues`, `resolved_by.*`,
`superseded_by` as present; `intents` is every member a bundle's shared spec lists), and each entry in `next_moves` — the concrete lifecycle move
(e.g. a draft intent points at the planning interview and `intent plan`; an
open issue points at `capture promote` / `resolve` / `wontfix`; decisions are
read). An admission (`adm-N`) and a surprise (`srp-N`) have no folder, so their
`status` is `admitted` or `recorded`; an admission's `links` are its `run`,
`proposal`, `proposal_path` and the standing `disposition`, a surprise's are
`occasioned_by` and `occasion_path`, and neither carries a next move. A reframe
(`rfm-N`) reads `open` or `complete`; its `links` are `occasioned_by`,
`occasion_path`, the three `*_before` fingerprints and, once complete, the three
`*_after` fingerprints and `changed`, and an open one's next move is
`capture reframe --complete <rfm-N>`. The reading families
(`rdi-N`, `dsp-N`, `rdg-N`) are not dispatched. For an issue id the JSON also carries `ledger` — the `checkout` and
`branch` whose ledger was read — because the same id can sit in another
worktree's ledger in another state; name it when you report. A shipped intent's
move reads its fidelity-review marker: an owed review names its receipt and the
re-emit command (`abcd intent audit <itd-N>`); a shipped intent with no marker
owes one too, and the re-emit mints its receipt; a dead-lettered review is
reported unreviewed with its reason; an ingested review leaves nothing to do. A
shape-matching id found in no store exits non-zero naming the stores
searched — unless a peer holds it (a sibling worktree or a local branch, see
`/abcd:peers`), in which case the refusal names that peer's branch, path and
folder instead; relay it, and do not recreate the record here. An issue whose
file is present but was skipped on read exits non-zero naming the file and the
skip reason, which carries the remedy where there is one; relay both, and do
not recreate the record. Any other positional is refused as an unknown command (exit 2) —
there is no `status` alias.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**When no binary resolves.** If every rung fails, the fix is **not** to install
Go. A compiler is a dependency of neither supported install route, which both
provision a prebuilt, checksum-verified binary. Tell the user to recover in this
order, no toolchain needed:

1. Restart a session with network access. `hooks/bootstrap.sh` re-provisions the
   plugin-root binary at the start of every session that can reach the release
   origin; an empty `.bootstrap.attempt` marker with no binary beside it means a
   previous provisioning began and did not finish, so a networked restart lands it.
2. Reinstall the plugin from its marketplace when its remote is stale (for example
   one predating an organisation rename). Re-adding the marketplace re-points it at
   the live release origin; the install guide gives the exact steps.
3. Install the CLI binary with the one-liner in the README, which downloads and
   SHA-256-verifies the same prebuilt binary into `~/.local/bin`.

`go run ./cmd/abcd` and `go build ./cmd/abcd` serve only a source checkout of this
repo (contributors) or a platform carrying no released binary; they are never a
prerequisite for a plugin or CLI user.

**User input:** $ARGUMENTS
