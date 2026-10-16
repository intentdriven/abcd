---
name: session
description: "Start a live brief session on this computer, or run, deploy or write the team's relay and page: Writes nothing bare; refuses an unknown sub-verb."
argument-hint: "[start|serve|blueprint|page]"
block: people
---

# `/abcd:session`

The relay a team's live brief session goes through. During a call, the host
starts a session on the project's brief, and everyone else joins by the invite
link, its QR code or a six-character code. Members of the project's team sign
in with GitHub and edit; anyone else joins as a guest with a name and suggests
changes. The page never asks anyone for a relay address or a token: it finds
the relay itself, and opens onto the brief's press release. The session runs
on a relay the team runs, never on anything abcd runs. The relay
holds each session's edits in memory only, as opaque bytes, and drops them
when the host closes the session or nobody has been connected for half an
hour. It writes nothing to disk and logs nothing: no token, no secret, no
brief text.

The page itself is a set of static files `page` writes, for the team to serve
from any static host. The abcdesign.app address goes live when abcd publishes
the page there; until then a team serves the page itself with `page` and admits
its origin with `--origin`.

There are two ways to run a session, both the team's own:

- **On the host's own computer**, with `start`, for a project whose brief
  should not leave the team's tailnet. The computer runs the relay and serves
  the page itself, reached through Tailscale Serve; nothing goes to
  abcdesign.app. Never Funnel: someone outside the tailnet joins as a guest
  once the host shares the computer with them in Tailscale.
- **On the team's own relay**, deployed once to the team's own Render account
  from the files `blueprint` writes and run there by `serve`, with the page on
  abcdesign.app. One deploy serves all of the team's sessions.

## Status

Bare invocation prints the four sub-verbs and writes nothing. A running relay's
sessions live in that process's memory alone, so there is no state to report.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" session --help
```

## Start

Starting a session opens a network listener, so do it only when the user asks
to host a live session from this computer. It runs in the foreground until
stopped with Ctrl-C, and every session ends with it, so never start it from
this session: tell the user to run `abcd session start` in a Terminal of their
own, in the project's checkout. It prints:

1. the one Tailscale step, `tailscale serve --bg <port>`, which shares it on
   the user's tailnet at an https address and which Tailscale keeps until it is
   turned off;
2. the link to open, `https://<this computer's tailnet name>/<owner>/<repo>/brief/`,
   where the host signs in with GitHub and starts the session, and the page
   then shows the invite: a code, a QR code and a link to copy.

Relay those lines as printed. It serves this checkout's origin repository on
github.com only. Team members sign in with GitHub's device flow: the page
shows a short code that the person confirms on github.com, so no secret sits on
the computer. The GitHub App's public client id comes from `--client-id`,
`ABCD_SESSION_GITHUB_CLIENT_ID`, or `.abcd/relay/github-app.json`
(`{"client_id": "…"}`), which repository setup writes; the file must hold no
secret. It asks Tailscale for this computer's tailnet name (read-only; it
changes nothing in Tailscale), and it answers only that name and its own
loopback address, as hosts and as page origins. It listens on
`127.0.0.1:8787`, or `--port`. `--json` prints `url`, `link`, `repo`,
`tailnet`, `serve_step` and `sign_in`.

It refuses (exit 2, nothing started) outside a checkout whose origin is on
github.com, without a client id ("sign-in is not set up for this project"),
with a `github-app.json` that holds a secret or no usable client id, when
Tailscale is not installed or not running, and on a port it cannot listen on.

## Serve

`serve` runs the relay for the team's cloud deployment, which the blueprint's
Dockerfile starts. Starting it opens a network listener, so do it only when
the user asks. It runs in the foreground until stopped
with Ctrl-C, and every session it holds ends with it, so never start it from
this session: tell the user to run `abcd session serve` in a Terminal of their
own. It listens on this computer alone (`127.0.0.1:8787`) and prints one line
saying where, with the `tailscale serve --bg <port>` command that puts it on
the user's tailnet at an https address; relay that line as printed. A
non-loopback `--listen` is a second, explicit choice, meant for the team's
relay; pass it only when the user names the address. `--behind-proxy` trusts
the proxy's `X-Forwarded-For` entry as each visitor's address, and belongs to
the team's relay behind its host's proxy. On loopback without it, a visitor
is counted by the tailnet address Tailscale Serve forwards, and only when the
connection itself comes from loopback. By default it admits a page from
`https://abcdesign.app` or loopback; `--origin` (repeatable) replaces that
list, so the relay then admits exactly the origins named, and without
`--origin` so does `ABCD_SESSION_ORIGINS`, a comma list of origins, which the
blueprint sets on the team's relay. `--json` prints the
start line as an object with `url`, `loopback`, `behind_proxy`, `sign_in`,
`repos`, `owners`, `any_repository` and `hint`.

It serves sessions only for the repositories it is given: each `--repo
owner/name`, every repository of each `--owner` (both repeatable), and each
entry of `ABCD_SESSION_REPOS`, a comma list of either. Nobody signs in
against, or hosts, any other. Listening beyond this computer it refuses to
start without one. On loopback, with none given, it serves this checkout's
origin repository, or any repository when run outside a GitHub checkout; the
start line says which.

Team members sign in with the team's own GitHub App, the only kind of
sign-in app the relay takes, and nobody pastes a token: the relay takes only a
GitHub App user access token (`ghu_`), asks GitHub for its login and whether it
may push to the repository, and drops it straight after. The relay reads the
app's public client id from `--client-id` or `ABCD_SESSION_GITHUB_CLIENT_ID`,
and its secret from `ABCD_SESSION_GITHUB_CLIENT_SECRET` by name. With both,
members sign in through GitHub's web flow (`/auth/token`); with the client id
alone, through GitHub's device flow (`/auth/device`); without a client id,
sign-in is not offered and only guests can join. `ABCD_SESSION_GITHUB_APP_TYPE`,
which once chose an OAuth app, may be unset or `github-app`; set to anything
else the relay refuses to start. Never ask the user for the secret, and never
print it or pass it on a command line.

It refuses (exit 2, nothing started) a `--listen` that is not an IP address
(or `localhost`) and a port, an `--origin` that is not an exact
`scheme://host[:port]`, a `--repo` that is not `owner/name`, an `--owner` or
`ABCD_SESSION_REPOS` entry that is neither, an `ABCD_SESSION_ORIGINS` entry
that is not an origin, `ABCD_SESSION_GITHUB_APP_TYPE` set to anything but
`github-app`, a relay beyond loopback with no repository to serve, and an
address it cannot listen on.

The relay bounds what it holds: 256 connections, 16 from one address (an IPv4
address or an IPv6 /48), past which a new one is refused with 503; 30 seconds
for a connection to host, join or rejoin; 1,024 queued items and 8 MiB per
connection, past which that connection is dropped; a member's updates to 60 a
second in bursts of 200; an awareness message to 8 KiB, one connection's
awareness to 32 KiB a second (bursts of 64 KiB), and a session's to 128 KiB a
second; a guest's suggestions to 70 KiB each, 100 and 256 KiB per participant
(a rejoin does not reset them) and 1 MiB per session; every control message
and refused frame to 50 a second per connection, closing one refused 1,000
times; an update or cursor over its rate is dropped and answered once a second
at most; a connection the relay closes ends within 5 seconds and stays counted
until it has, and a log replay is bounded to a minute; sign-in
exchanges and device sign-ins started to 10 per address in ten minutes
together, and 64 device sign-ins waiting at once; a session's life to 12 hours,
after which it ends for everyone with `closed {reason: "expired"}`; and
joining by code pauses for everybody for ten minutes past 600 wrong codes from
everywhere in ten minutes.

## Blueprint

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" session blueprint --release <vMAJOR.MINOR.PATCH> [--relay-origin <https origin>]
```

It writes `.abcd/relay/render.yaml`, a Render Blueprint declaring one free web
service, and `.abcd/relay/Dockerfile`, which downloads the named abcd
release's linux/amd64 binary and checks it against the SHA-256 that release's
`checksums.txt` publishes. It reads that checksum from the release page when it
writes the files (the one network read it makes, named on stderr) and pins it
in the Dockerfile. `--release` defaults to this abcd's own version when that is
a release. `render.yaml` sets `ABCD_SESSION_REPOS` to this checkout's origin
repository on github.com, the one the relay serves, and `ABCD_SESSION_ORIGINS`
to the session page origins the relay admits: `https://abcdesign.app`, or each
https origin `--origin` (repeatable) names. With `--relay-origin`, the
deployed relay's https origin once Render has given it one, it also writes
`.abcd/relay/relays.json`, naming that relay as the one the repository
trusts; the session page reads it. A file left exactly as abcd wrote it is
replaced; one that was edited, or that abcd did not write, is never
overwritten, and then nothing is written (exit 2). It refuses outside a
checkout abcd manages, in a checkout whose origin is not on github.com, a
`--relay-origin` or `--origin` that is not `https://host[:port]`, a release
older than the first to include `abcd session` (the relay runs it; this
abcd's own release is always taken), and a release that publishes no checksum
for the linux/amd64 binary.

Relay what it prints: each file and whether it was written, replaced or left
unchanged, the release and checksum pinned, and the next steps. The user
commits both files, chooses New, then Blueprint, on Render, picks the
repository, sets the Blueprint path to `.abcd/relay/render.yaml`, and sets
`ABCD_SESSION_GITHUB_CLIENT_ID` and `ABCD_SESSION_GITHUB_CLIENT_SECRET` in the
service's Environment. `--json` returns `release`, `asset`, `sha256`,
`checksums_url`, `repos`, `origins` and `files`, each file with its `path` and
`state`.

## Page

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" session page --out <dir> [--base-path <path>]
```

It writes the session page into `<dir>`: `index.html` and an identical
`404.html`, `session.js`, `session.css`, and `vendor/collab.js` with
`vendor/LICENSES.txt`, the vendored editor bundle and its licences. The files
are embedded in the binary and read nothing from the checkout. A static host
whose 404 fallback serves `404.html`, such as GitHub Pages, is how
`/<owner>/<repo>/brief/` reaches the page. The page loads its own files from
the site's root, or from under `--base-path` when the site is served below one
(a project's GitHub Pages address, `owner.github.io/<repo>/`), so it then opens
at `<base-path>/<owner>/<repo>/brief/`; the base path is plain path segments,
and an address without the trailing slash is sent on to it. The page connects
only to the relay the person names and to `api.github.com`, and it sends the
person to `github.com` for sign-in and to open the pre-filled issues a session
sends. The directory is the one the user names; never publish or deploy it for
them. (A session started with `start` serves the same files itself.)

The page never shows or asks for a relay address. A page `start` serves takes
its own origin as its relay. On a static host, it uses the relay the invite
link names, or, with none, the first relay the repository's committed
`.abcd/relay/relays.json` names (`{"relays": ["https://<relay>"]}`), which it
reads from `api.github.com` on the default branch, without a token for a
public repository and with the person's token once there is one. A link can
name any relay, so the page trusts the repository, not the link: a relay the
file does not name gets no sign-in, no token, no join and no connection, and
the page says the link does not lead to this project's own session. Only
GitHub's 404 means there is no list to read (a private repository before
sign-in, or no file): then a relay from the link reaches nothing until the
person answers one plain question naming the repository ("Only continue if
someone on the owner/repo team sent you this link"); the browser remembers the
answer for that repository. Any other failure to read the list (a 403, a 429,
a 5xx, no network) refuses the relay with "try again in a moment", and asks
nothing. With no relay at all, the page says live sessions are not set up for
this project yet and to ask the technical facilitator. So the team commits
`relays.json` naming its cloud relay's https origin, the one
`blueprint --relay-origin` writes.

Sign-in goes through a GitHub App alone: when a user asks how to set it up, it
is a GitHub App with the repository permissions Contents read and write, Pull
requests read and write, and Metadata read, and only those, installed on the
repository; for `start`, its "Enable Device Flow" setting is switched on. Its
sign-in carries no scope and reaches only the repositories it is installed on.
A relay that offers no GitHub App sign-in gets no sign-in button, and the page
says sign-in is not set up yet and to ask the technical facilitator; guests can
still join. There is no token field. The page refuses to run inside another
page's frame: a static host cannot forbid framing, so the page shows only a
link to open it in its own tab.

It writes those six files and nothing else, and removes nothing. `<dir>` may be
absent, empty, or a directory an earlier run wrote, whose files are replaced.
It refuses (exit 2, nothing written) a missing `--out`, a `--base-path` that is
not plain path segments, a symlink or a file, and a directory holding anything
else or holding the page's names without the `index.html` it wrote. Relay the
files it wrote and the next step it prints.
`--json` returns `out`, `files`, `bytes`, `base_path` and `replaced`.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
