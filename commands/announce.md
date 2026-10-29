---
name: announce
description: "Post a fixed-form announcement on the shared remote: Writes one commit to your status branch; refuses free text, a record not yet public and an embargoed one."
argument-hint: "<kind> <item> [--expires-in <duration>]"
block: agents
user-invocable: false
---

# `/abcd:announce`

Post a short fixed-form announcement on the shared remote, so every other
session sees what this one is doing: on the same computer, under a colleague's
account on another computer, or in a hosted cloud session. This verb is an
agent's, beside `/abcd:reserve`. An announcement informs and grants nothing:
it never approves, lends or hands over anything. What is really a hold is a
reservation (`/abcd:reserve`), not an announcement.

**What it does to the remote.** The command's documented meaning is a listing,
a fetch and a push: it lists the status branches on origin first (`git
ls-remote`, which changes nothing here), fetches the first 500 named by lower-case
GitHub usernames, in origin's order and with your own among them, each by
name, with origin's default branch, then writes one commit to your own status branch, `abcd-status/<your GitHub
username in lower case>` on the remote named `origin`, and to nothing else.
The status branches are named on the fetch's standard input (`git fetch
--stdin`), which needs git 2.29 or later: Once origin holds a status branch,
an older git fetches nothing, and the refusal names the status fetch and the
git version.
The commit changes no file (its tree is the empty tree); its trailers carry the
kind, the item, you as sender, this working copy, the time sent and the
expiry, and nothing else. The push is a plain fast-forward with the
repository's hooks running, never forced and never with `--no-verify`, and it
touches no HEAD, index or working tree here. When git's identity does not name
the sender (below), it also makes one read-only call to GitHub's user endpoint,
through `gh`, to learn whose account the forge token `gh` resolves from this
session's environment belongs to, as `/abcd:reserve` does. Announcements are
public and permanent: a status branch is never deleted.

## Announce

Run, naming one kind and one item:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" announce merging '#1012' --json
```

The kind is one of `merging`, `merged`, `asking`, `answered` and
`handed-over`. The item is the record it concerns, an issue (`iss-N`) or an
intent (`itd-N`), or a pull request (`#1012`, quoted, or the bare `1012`,
since a shell reads an unquoted `#1012` as a comment). `--expires-in <duration>` sets
how long it stands, from `1m` to `24h`, to the whole second (a fraction of a
second is dropped); the default is 30 minutes. There is no
free text: a question points at the item, and the conversation happens there.

The payload carries `kind`, `item` (a pull request always as `#N`), `sender`,
`branch`, `sent_at`, `expires_at`, `commit` and `attempts` (1, or more when a
push lost a race and was sent again).

Tell the user what went out and until when.

## abcd's own pushes announce first

You do not need to announce before abcd pushes for you. `/abcd:land`, and the
landing of a build or drain lane (`/abcd:implement` step, which `/abcd:build`
and `/abcd:drain` drive), post `merging <item>` through this same code before
they push, and push only once the announcement is on `origin`, so a
colleague's `/abcd:status` shows it before the branch exists there. Land
announces each issue, intent or pull request the branch's own commits name in
a `Refs:`, `Resolves:` or `Delivers:` trailer; a lane announces its own
record. An item you already announced `merging`, still standing for a minute
or more, is not announced again, unless your newest announcement of it is now
another (`merged`, say): a colleague sees only that newest one, so `merging`
is sent anew. A record not yet on the default branch is
not announced (the refusal above), and the verb says so and pushes. Any other
failure refuses the push with the reason, and nothing is pushed. When a branch
names several items and a later one fails, the announcements already sent for
the earlier ones stay standing, and running the verb again resumes without
sending them again while they stand.

A push a person types by hand with `git push` posts no announcement. Neither
does any hook. Run this verb first when a colleague should see it.

## When it refuses

**Exit 2 is every refusal.** Nothing is pushed on a refusal except "not sent"
below, the one refusal that follows pushes, every one of them refused by
origin. Relay the reason; each names the field and the limit, or what to set.

- **Not the fixed form.** A kind outside the list, an item that is neither an
  issue or intent id nor a pull-request number, an expiry outside `1m` to
  `24h`, or any further word after the item.
- **Not public yet.** A record that is not on origin's default branch as just
  fetched is refused, since an announcement is public the moment it is pushed:
  merge or push the record first. When this clone names no default branch of
  origin, abcd asks origin which it is (`git ls-remote --symref origin HEAD`,
  which changes nothing here) and fetches that branch for this check. Any
  record is refused when origin cannot be asked or the fetch could not read the
  branch. A pull request needs neither.
- **Under embargo.** A record the embargo store on this machine holds for an
  advisory not yet published is refused, through the same check
  `/abcd:embargo` runs. When that store exists but cannot be checked, every
  announcement is refused, never let through.
- **Whose account.** The sender is your GitHub username, worked out as
  `/abcd:reserve`'s holder is, from the first of these that names it: `git
  config user.email` when it is a GitHub no-reply address;
  `reservation.github_username` in `~/.abcd.noindex/config.json`; and the
  account the session's own GitHub token belongs to, through one call to
  GitHub's user endpoint. The repository's own configuration is never read for
  it. When git's identity is not a person's (unset, or a machine's, as in a
  hosted cloud session), the token alone names the sender and that account
  authors the announcement commit; the identity git had is shown nowhere.
  When nothing names the sender (the token could not be asked or did not
  answer), the refusal names the failure's class and never the identity it
  rejected. Relay it as it is; it asks the user to change nothing.
- **Not sent.** A push that lost a race to another of the user's sessions is
  rebuilt on the tip that won and pushed again, up to three times. After that,
  or when origin refuses the push for any other reason, the refusal starts
  `not sent:` and says why. Each refused push is followed by one more listing
  and fetch, to tell a push that landed anyway from a lost race. Do not retry
  in a loop. A push git reports as
  refused that origin took anyway is reported sent, not sent twice.
- **May have reached origin.** When a push was made and what became of it
  cannot be read back (the fetch after it failed, or the rebuilt commit could
  not be written), exit 2 says a refused push may still have reached origin
  and names the status branch: fetch and look at it before announcing again.
- No local tier (`.abcd/.work.local/`, which `/abcd:ahoy install` creates), no
  remote named `origin`, a fetch origin refused, or, once origin holds a status
  branch, a git older than 2.29.

**What an announcement is not.** Announcements coordinate people who share
push access to the repository and act in good faith. They are not an access
control, and they are seen only after a fetch, so nothing that must never be
missed rests on one alone.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
