---
name: embargo
description: "Keep private security advisories off the public ledger until closed or published: Writes nothing bare, only the embargo store; refuses an unknown sub-verb."
argument-hint: "intake | triage <iss-N|GHSA-id> <verdict> --by <who> [--reason <why>] [--exists-without <why> --cannot-help <why>] | close <iss-N|GHSA-id> [--approve <digest>] | check [--range <a>..<b>] [--file <path>|-]... [--allow <GHSA-id>]... | publish [--release <vX.Y.Z>] [--approve <digest>]"
block: people
---

# `/abcd:embargo` private security advisories

When someone reports a security hole privately, `embargo` keeps it private
until its fix ships, then closes or publishes it with the person. Each
advisory becomes a private record in the **embargo store**, a worktree in the
machine's worktree store (`~/.abcd.noindex/worktrees/<root-sha>/embargo/`),
created detached and never committed. The record never enters this
repository's public ledger, and the link from an advisory to its record lives
in the store's state, not in the record. Every sub-verb checks the store
before it reads or writes it, and refuses, writing nothing, when the store is
on a branch, tracks a private record, sits inside the checkout, or is open to
other accounts on the machine: the store's folder is kept at `700` and each
private record at `600`, and a refusal names the `chmod` that restores them.

Every request to GitHub is the person's own `gh`, signed in to an account that
may read and edit the repository's security advisories; abcd holds no token.
A refusal for want of a scope names it and the person's own remedy
(`gh auth refresh -s repository_advisories:write`).

Pick the sub-verb from the user's input. With none, ask which step they are
on; there is no bare listing.

## Intake

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo intake --json
```

Reads the advisories GitHub holds in triage and draft and mints one private
record for each one not seen before. `minted` lists each new record with its
advisory and severity (high is major, medium and low are minor, critical is
critical); `held` lists the advisories already held, which are never recorded
twice; `needs_severity` lists the advisories GitHub gives no severity, which
get no record yet but are held under embargo all the same, so the check
refuses their id, reserved CVE id and title: tell the user to set the severity
on GitHub and run intake again. Nothing is written to the public ledger.

## Triage

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo triage --json <iss-N|GHSA-id> <verdict> --by "<who decided>" [...]
```

The verdict is the person's decision. Ask for it; never choose it yourself.

| Verdict | Needs | What happens |
| --- | --- | --- |
| `vulnerability` | `--by` | the record stays open in the store, to be fixed |
| `not-abcds` | `--by`, `--exists-without "<why>"`, `--cannot-help "<why>"` | the record is won't-fixed in the store and the closing text is drafted |
| `not-a-vulnerability` | `--by`, `--reason "<why>"` | the record is won't-fixed in the store and the closing text is drafted |
| `public-track` | `--by`, `--reason "<why>"` | the record is won't-fixed in the store; `capture_command` is the `abcd capture` command to run in the public checkout |

The two halves of `not-abcds` are the two halves of the rule that abcd fixes
what it adds: why the problem would exist without abcd, and why abcd cannot
help without changing how the repository works. The time is stamped from the
clock. A second triage keeps the earlier verdict in its history. An unknown
verdict, or one missing its reason, is refused with nothing written.

## Close

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo close --json <iss-N|GHSA-id>
```

For an advisory whose verdict is `not-abcds` or `not-a-vulnerability`. GitHub
offers no comment on a repository advisory, so the explanation is posted at the
head of the advisory's description, with the reporter's own text kept whole
below a separator line. Without `--approve` it sends nothing but a read: show
the user `text` exactly as returned, then `approve_with`. The reporter reads
that text, so it is refused before it is shown when it carries a secret, a
name the private-names layer bans, or another advisory still under embargo
(its id, reserved CVE id or title, a closed advisory's title included); the
refusal names the kind and the line, never the match, and the remedy is to
record the verdict again with a reason that does not carry it.

**Approval, one per act.** Pass `--approve <digest>` only after the person, in
this session, has read that text and said yes to it. Never supply the digest on
your own, and never reuse an approval for another advisory or another text:
the digest binds the text and the advisory as read, so an edited draft or an
advisory changed on GitHub sends nothing, and the remedy is to show the text
again. With the approval abcd closes the advisory, reads it back to confirm it
is closed with that text, and records `closed_at`. A read-back that does not
confirm it exits 1 naming which half did not hold. An advisory GitHub already
reports closed (a close whose read-back or record failed, or one closed by
hand) is not sent anything: `closed_at` is recorded from GitHub,
`reconciled` is true, and `text_held` says whether its description begins
with the drafted text; one GitHub reports published is refused, and its
`published_at` recorded.

## Check

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo check --json --range <a>..<b>
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo check --json --file <path> [--file -]
```

Run it before anything is made public: a push, a pull-request body, a
changelog section, a release page, a record. It reads each commit's message
and the lines it adds (a merge's against its first parent), and each text, and
refuses every advisory not yet published by its id, a CVE id reserved for it,
its title (not once the advisory is closed, since its reporter chose it and
nothing will publish it; its id and CVE id stay refused), and the store's
location on this machine; any other advisory id is
refused too unless `--allow <GHSA-id>` names it as another project's, already
public. Each finding names its `kind`, `source`, `where` and `line` and never
repeats the match. Text is folded before it is matched: fullwidth and other
compatibility letters read as plain ones, every dash as a hyphen, invisible
format characters (Unicode category Cf, such as a zero-width space or a soft
hyphen) are dropped, and a token broken at a hyphen across a line end reads
whole. Exit 0 is clean (or no store exists on this machine), exit 2 is a
finding. `intake_stale` says the last intake is more than a day old: run
intake first.

Limits the user must hear: the check cannot recognise a paraphrase or a count
("fixes several private reports"), which stays the author's to avoid; a
title shorter than 12 characters is not matched on its own, though its
advisory's id still is; and an advisory id of the placeholder form, each
four-character group one character repeated (`GHSA-xxxx-xxxx-xxxx`,
`GHSA-2222-cccc-xxxx`), is taken for a documentation or test placeholder and
not refused as unlisted, so a real id of that form would pass unremarked. An
advisory the store holds is still refused by its id whatever its form.

The fold catches less than a reader would. It does not catch look-alike
letters from other scripts (a Cyrillic or Greek letter standing in for a
Latin one); invisible characters outside category Cf (the combining grapheme
joiner U+034F, variation selectors, the Hangul filler U+3164); an id or title
written as HTML entities or with markdown backslash escapes; or a word whose
line wrap inserted a hyphen that was not in it (a title word split as
`over-` and `flow`). `--range` reads commit messages, the lines each commit
adds and the names of the files it adds, and nothing else: branch names and
annotated tag messages are not checked.

The check reads every embargo store under this account's home, whichever
repository each belongs to, so a second clone, a shallow clone or a repository
with merged-in history is checked against the records held for it. A store
whose checkout was deleted is still read and its records still refused; the
`note` counts such stores, naming none. A store this repository registered
under another home is refused (run abcd with that home, or, if that home no
longer exists, run `git worktree prune`), and a store's place that cannot be
examined, or whose state file cannot be read, refuses the check. `no_store:
true` means no store exists under this home at all. Because every store is
read, `under_embargo` counts the ids held across all of them, so it can be
non-zero in an unrelated repository that holds nothing itself, and it stays
off public logs there too.

A consumer of the output must treat `no_store: true` as "not checked", never
as clean: nothing was compared against anything. Keep the output off public
logs, its `under_embargo` count included: how many advisories are held is
itself a disclosure.

## Publish

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo publish --json [--release <vX.Y.Z>]
```

Once the release carrying the fixes is cut, this shows the publication plan
and sends nothing but reads. For each advisory judged a vulnerability (the
release defaults to the highest `vMAJOR.MINOR.PATCH` tag): its record resolved
in the release, its fix commit, the first release tag holding the fix, the
patched version that gives, and the affected range in GitHub's form.

The security rule keeps a security record out of every committed tree until
its advisory is published, so the release usually carries no public record of
it. Then the fix itself is the evidence: the private record resolved in the
store's own ledger, naming its fix commit. Resolve it there once the fix has
landed, from the store's folder, naming the commit by its full id:

```bash
cd ~/.abcd.noindex/worktrees/<root-sha>/embargo
"${CLAUDE_PLUGIN_ROOT}/abcd" capture resolve <iss-N> "<what fixed it>" --impact fix --commit <full commit id>
```

The resolve keeps the record's mode, so it stays this account's alone. The
plan takes that commit only when it is a full object name of this repository's
hash that git resolves to exactly itself as a commit, so an abbreviation, a
tag's name or anything shaped like an option is left out with the reason; the
first release tag holding the commit gives the patched version, as for a public
record. Such an entry carries `store_evidence: true`. A record resolved in the
release is always taken first, whatever the store holds.

`left_out` names each advisory the plan does not publish and why: its record
is resolved neither in the release nor in the store with a fix commit the plan
accepts, its fix is in no release tag, its package is not set on GitHub, or its
text carries a secret, a private name, or another advisory still under embargo
(a closed one's title included: it stays private on GitHub). Show the user the
whole plan, then `approve_with`.

**Approval, one per act.** Pass `--approve <digest>` only after the person, in
this session, has read that plan and said yes to it, and never on your own. A
digest for any other plan sends nothing. With it abcd publishes one advisory
first as a canary and reads it back, then the rest the same way, stopping at
the first mismatch, and confirms none is left in draft. A stopped run exits 1
naming what is published and what is not; running publish again plans only
what is still unpublished. An advisory GitHub already reports published or
closed is left out of a plan, and its `published_at` or `closed_at` recorded
from that, so a publication whose record failed is completed by the next run.

Each advisory published from store evidence carries `restore_with`, the
command, run from the checkout's root, that copies its private record,
resolved, from the store into the public ledger's `resolved/` folder now its
advisory is public. abcd has no verb that restores a record; show the user the
command, and the person runs it and lands the record in a change of its own.
The command is never lost to a stopped run: the failure names it for each
advisory published before the stop, and a later plan that finds such an
advisory published on GitHub names it again on its `left_out` entry.

## What stays the person's act

In this release `embargo` does not fix, land or announce anything. These stay
with the person:

- fixing each vulnerability, landing the fixes so they are in the release
  that publishes the advisories, and resolving each private record in the
  store with its fix commit;
- restoring each record into the public ledger once its advisory is
  published, with the command publish prints;
- requesting a CVE for an advisory, on GitHub;
- putting back any record details they redacted to land a fix, once the
  advisory is published;
- every verdict, and every yes that lets abcd close or publish an advisory.

abcd closes or publishes an advisory only on the person's own approval of what
it showed them.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.
