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
(its id, reserved CVE id or title); the refusal names the kind and the line,
never the match, and the remedy is to record the verdict again with a reason
that does not carry it.

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
characters (a zero-width space, a soft hyphen) are dropped, and a token broken
at a hyphen across a line end reads whole. Exit 0 is clean (or no store exists on this machine), exit 2
is a finding. `intake_stale` says the last intake is more than a day old: run
intake first.

Limits the user must hear: the check cannot recognise a paraphrase or a count
("fixes several private reports"), which stays the author's to avoid; a
title shorter than 12 characters is not matched on its own, though its
advisory's id still is; and an advisory id of the placeholder form, each
four-character group one character repeated (`GHSA-xxxx-xxxx-xxxx`,
`GHSA-2222-cccc-xxxx`), is taken for a documentation or test placeholder and
not refused as unlisted, so a real id of that form would pass unremarked. An
advisory the store holds is still refused by its id whatever its form.

## Publish

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo publish --json [--release <vX.Y.Z>]
```

Once the release carrying the fixes is cut, this shows the publication plan
and sends nothing but reads. For each advisory judged a vulnerability (the
release defaults to the highest `vMAJOR.MINOR.PATCH` tag): its record resolved
in the release, its fix commit, the first release tag holding the fix, the
patched version that gives, and the affected range in GitHub's form.
`left_out` names each advisory the plan does not publish and why: its record
is not resolved in the release, its fix is in no release tag, its package is
not set on GitHub, or its text carries a secret, a private name, or another
advisory still under embargo. Show the user the whole plan, then
`approve_with`.

**Approval, one per act.** Pass `--approve <digest>` only after the person, in
this session, has read that plan and said yes to it, and never on your own. A
digest for any other plan sends nothing. With it abcd publishes one advisory
first as a canary and reads it back, then the rest the same way, stopping at
the first mismatch, and confirms none is left in draft. A stopped run exits 1
naming what is published and what is not; running publish again plans only
what is still unpublished. An advisory GitHub already reports published or
closed is left out of a plan, and its `published_at` or `closed_at` recorded
from that, so a publication whose record failed is completed by the next run.

## What stays the person's act

In this release `embargo` does not fix, land or announce anything. These stay
with the person:

- fixing each vulnerability, and landing the fixes (with their records) so
  they are in the release that publishes the advisories;
- requesting a CVE for an advisory, on GitHub;
- putting back any record details they redacted to land a fix, once the
  advisory is published;
- every verdict, and every yes that lets abcd close or publish an advisory.

abcd closes or publishes an advisory only on the person's own approval of what
it showed them.
