---
name: reserve
description: "Reserve an issue or intent on the shared remote, or release it: Writes one commit to its origin branch; refuses a record someone else holds, with exit 3."
argument-hint: "<record-id> --footprint <path> [--footprint <path>…] [--take-over] [--lease <duration>] | <record-id> --release | --holder"
block: agents
user-invocable: false
---

# `/abcd:reserve`

Reserve an issue or an intent on the shared remote before starting work on it,
so every other participant (a second person, a second computer, a hosted cloud
session) sees that it is taken before starting the same item. This verb is an
agent's, and so is `/abcd:peers`, which lists the reservations: a person sees
them on the board (bare `/abcd:abcd`) and on a record's page
(`/abcd:abcd <record-id>`), not by typing either.

**What it does to the remote.** The command's documented meaning is a push: it
writes one commit to the branch `abcd-reserved/<record-id>` on the remote named
`origin`, and to nothing else. The commit changes no file (its tree is the
empty tree); its trailers name the record, the holder, this working copy, the
footprint and the lease. The push is a plain fast-forward with the
repository's hooks running, never forced and never with `--no-verify`, and it
touches no HEAD, index or working tree here. When git's identity does not name
the holder (below), it also makes one read-only call to GitHub's user endpoint,
through `gh`, to learn whose account the forge token `gh` resolves from this session's
environment belongs to.

## Reserve before starting

Run, naming every repository-relative path the work will change, once per path:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve iss-2609020716570699 --footprint internal/core/claim/claim.go --footprint commands/reserve.md --json
```

The record is an issue (`iss-N`) or an intent (`itd-N`). The footprint is
required. `--lease <duration>` sets how long the reservation holds, from `1m`
to `24h`; the default is two hours. Running the same command again from the
same copy renews it.

The payload carries `act` (`reserved`, `renewed`, `taken_over`, `released` or
`already_released`), `record`, `holder`, `copy` (`this` or `another`, never a
copy's key), `footprint`, `claimed_at`, `effective_expiry`, `commit`, and
`replaced` when a take-over replaced a reservation: its `holder`, `copy`,
`commit`, `effective_expiry`, and `unreadable` when it could not be read. When
origin's default branch could not be read, `derived_unchecked` says why: the
act went on, but whether the record is already finished there was not checked.

Tell the user what was reserved and until when.

## When it refuses

**Exit 3 means back off: take other work.** The record is reserved by someone
else and still live, or lapsed less than five minutes ago, or its reservation
cannot be read and is still within its two-hour hold, or another push reserved
it while this one was on its way. Say who holds it and until when, and pick
something else. Never retry in a loop.

**Exit 3 also covers a shared path.** A footprint path that another live
reservation's footprint also holds, whoever holds it and for whatever record,
is refused (the user's own reservations from this copy excepted, since that
work is this copy's: the user's other computer still backs off), naming that path, the other record, its holder and until when. Two
paths are shared when they are equal or one is a directory holding the other.
Running the command again to renew is checked the same way, since it names
its footprint anew, so a reservation is never widened onto another; a build's
or a drain's own renewal keeps its footprint and is not checked while the
reservation is live, but is when it revives a lapsed one. Take other
work, or reserve with a footprint that does not overlap. The cap below and this
overlap are judged on the reservations fetched just before the push, and each
reservation is its own branch, so two people reserving at the same moment can
both pass them: they are best-effort under a race. Two reservations of the
same record cannot both land; the second push is refused.

**Exit 3 also covers your own other copy.** When the same person holds the
record from another computer or another checkout, the refusal says "you are
already building … on another computer". Pass `--take-over` only when the user
confirms the work moves here; it moves only the user's own reservation, and the
output names the copy it replaced. It never moves another person's.

**Exit 2 when the user holds ten.** One person may hold at most ten live
reservations, counted across every record by username. A reservation past that
is refused, naming the ten; a renewal, and a take-over of the user's own other
copy, of a reservation still live are never refused by the cap, while renewing
one that has lapsed counts as a new one. Release one first, if the user agrees.

**Exit 2 is every other refusal**: an id that is not an issue or intent, no
footprint, a path that is not repository-relative, a lease out of bounds, a
message over the 4096-byte cap, no GitHub account (the token could not be
asked or did not answer), no local tier (`.abcd/.work.local/`, which `/abcd:ahoy install` creates), no remote
named `origin`, a fetch or a push origin refused. Each refusal ends by saying
nothing was pushed or no reservation was made, and quotes git's own reason with
the home directory redacted. Relay it.

**What a reservation is not.** Reservations coordinate people who share push
access to the repository and act in good faith. They are not an access
control: anyone who can push can append a commit to another holder's
reservation branch, and the holder a reservation names is read from its own
trailer, set from that writer's `user.email`, machine configuration or token,
so it is self-declared. The forge's push permissions are the only boundary.

**Clocks.** A reservation's expiry is written by the holder's clock and
judged by the reader's, and the five-minute margin is all the disagreement it
absorbs. A holder whose clock runs behind loses that much of each lease: one
behind by more than half the lease plus the margin (65 minutes for the
default two hours) can be taken over between its own renewals, and a
reservation made once and never renewed by a clock behind by more than its
lease is open to a take-over as soon as another copy fetches it. A
reservation dated more than five minutes ahead of the reader counts from when
that copy first fetched it, so it holds at most a day from then. Keep the
machines that reserve on network time.

**The holder** is the user's GitHub username, from the first of these that
names it: `git config user.email` when it is a GitHub no-reply address;
`reservation.github_username` in `~/.abcd.noindex/config.json`; and the
account the session's own GitHub token belongs to, through one call to
GitHub's user endpoint. The repository's own configuration is never read for
it. When git's identity is not a person's (unset, or a machine's, as in a
hosted cloud session), the token alone names the holder and that account
authors the reservation commit; the identity git had is shown nowhere. When
nothing names the holder the refusal says why; relay it.

To learn the holder without reserving anything:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve --holder --json
```

It prints `{"login", "id", "noreply", "source"}` (`source` is `git`, `config`
or `token`; `id` and `noreply` when known), or as text `would reserve as
<login> (from <source>)`. It may make that one user-endpoint call in two
cases: git's identity is a person's but carries neither a no-reply address nor
a machine key; or git's identity is not a person's (unset, or a machine's), when
the machine key is skipped. It takes no record id and no other flag, and
reserves, writes and pushes nothing. Exit 0 when the account is known, 2 when
not. Its `noreply` is the account's address; a person's own git identity still
authors their commits.

## Release when done

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve iss-2609020716570699 --release --json
```

A release pushes a tombstone child and deletes nothing. It takes no footprint,
take-over or lease. Only the holder's own reservation from this copy is
released (exit 2 otherwise); a release already made exits 0 as
`already_released` and pushes nothing. A record that is resolved, won't-fixed,
shipped or superseded on origin's default branch counts as released whatever
its reservation says, and is not reserved again: a reservation or take-over of
it backs off (exit 3) saying it is finished on origin's default branch, with a
tip or without one, and pushes nothing.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
