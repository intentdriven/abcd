---
id: adr-2610050711287503
slug: abcd-reaches-the-harness-s-user-settings-only-for-the-status
status: accepted
date: 2026-10-05
supersedes: null
superseded_by: null
related_intents: [itd-200]
related_rfcs: []
related_adrs: [adr-2609091248200336, adr-2610030720195401]
---

# ADR-2610050711287503: abcd reaches the harness's user settings only for the status line, with consent

## Context

On 2026-10-05, after the person had renamed `~/.abcd` to `~/.abcd.noindex`, the old folder reappeared three times within an hour, and each time the guard refused every shell command and every question in two sessions at once. The cause was a `SubagentStop` entry in the person's harness user settings that ran a locally built `abcd-darwin-arm64`, old enough to reject `--version`, with `hook subagent-stop`. Whenever a subagent stopped, that build staged the transcript under the old home and recreated it. The plugin already registered its own `SubagentStop` hook with the current binary, so the settings entry was a stale duplicate, and nothing in abcd noticed it (iss-2610050556323779).

The same file holds the one entry abcd writes there on purpose: `statusLine`, which `abcd ahoy install` points at `'<entry>' statusline` after the person consents (itd-200). The status line is essential for product thinkers, and a plugin cannot set it: the harness's plugin manifest reference admits only `agent` and `subagentStatusLine` in a plugin's settings. Every hook abcd needs, by contrast, can live in the plugin's own `hooks/hooks.json`.

The product thinker ruled the same day (iss-2610050556383525). The status line runs on every refresh in every session, so the ruling names seven risks the write must be guarded against: a stale or missing binary at the wired path; a status verb that writes state; a line left behind after an uninstall; the previous status command recorded once and gone stale; a concurrent write corrupting the settings file so the harness ignores all of it; a wired path a project or another account can replace; and settings synced to a machine without that path.

## Decision

abcd does not touch the person's harness configuration folder, and nothing in it calls abcd, with one exception the person consents to: the `statusLine` entry in the harness's user settings. Hooks live only in the plugin.

The one write is guarded:

- It happens only after an explicit yes to an offer that shows the entry's value now and after, and `--yes` never gives that yes.
- A copy of the file goes into `~/.abcd.noindex/backups/` first, which keeps the newest ten; no copy, no write.
- The file is replaced atomically with every other key untouched, read back, and put back from the copy when it does not say what was written.
- Only the PATH install `~/.abcd.noindex/path-entry` records is wired, and only when it passes the trust checks the plugin's hook shims apply (owned by the caller, writable by nobody else, outside the working tree), never the plugin's versioned copy.
- A command that reaches abcd's own status verb, under any build name, is never recorded as the previous one.
- `abcd statusline` writes nothing and is time-bounded (500ms for abcd's own work, 5s for the previous command), held by tests.
- `abcd ahoy uninstall` restores the previous command, and the install guide says to run it before removing abcd.

abcd reports, and never edits to remove, any hook in those settings that runs abcd, and an abcd status line whose binary fails the trust checks. Session start names such an entry in one line. The one repair abcd makes there is to its own status line, when that line dangles or runs an untrusted abcd, and it is done under the same guards with config-change approval.

## Alternatives Considered

- One consented, guarded write for the status line, and report-only for everything else (chosen): it keeps the line product thinkers need, which a plugin cannot set, and puts every hook where a plugin update replaces it with the current binary.
- No writes to the harness settings at all, with the person wiring the status line by hand: removes the risk, but leaves the person to copy a path and a verb, and nothing would notice when that path goes stale.
- abcd removes stray abcd hooks from the settings itself: would have fixed the incident automatically, but edits the person's own configuration without a consent that describes the change, the stance adr-2609091248200336 and adr-2610030720195401 take for the person's folders and settings.
- Keep hooks in the user settings as well as in the plugin: duplicates every hook, and the copy in the settings names a binary no plugin update refreshes, which is the defect that recreated the old home.

## Consequences

- The status line survives plugin updates, because it names the recorded PATH install rather than a versioned plugin folder; a machine without that install is refused before anything is offered.
- The backups folder grows by at most ten copies, and a copy that cannot be pruned is noted without failing the write.
- A stray hook stays in place until the person removes it; abcd's part is to name it in `abcd ahoy` and at session start for as long as it is there.
- Settings synced to a machine where the wired path is missing or untrusted read as `statusline.dangling` or `statusline.untrusted` there, and install repairs the line on that machine.
- Any future need to call abcd from the harness's user settings is a new decision; the default answer is the plugin's own hooks.
