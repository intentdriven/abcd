---
name: session
description: "Run the relay a team's live brief session goes through, or write its blueprint or its page: Writes nothing bare; refuses an unknown sub-verb."
argument-hint: "[serve|blueprint|page]"
block: people
---

# `/abcd:session`

The relay a team's live brief session goes through. During a call, a team
opens its brief on abcdesign.app, the host opens a session, and everyone else
joins by a QR code or a six-character code. The page is static; the session
itself runs on a relay the team runs, never on anything abcd runs. The relay
holds each session's edits in memory only, as opaque bytes, and drops them
when the host closes the session or nobody has been connected for half an
hour. It writes nothing to disk and logs nothing: no token, no secret, no
brief text.

The page itself is a set of static files `page` writes, for the team to serve
from any static host.

There are two ways to run the relay, both the team's own:

- **On the host's own machine**, with `serve`, reached by teammates on the
  host's Tailscale network through Tailscale Serve. Never Funnel: a guest
  outside the tailnet joins through the team's relay instead.
- **On the team's own relay**, deployed once to the team's own Render account
  from the files `blueprint` writes. One deploy serves all of the team's
  sessions.

## Status

Bare invocation prints the three sub-verbs and writes nothing. A running relay's
sessions live in that process's memory alone, so there is no state to report.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" session --help
```

## Serve

Starting the relay opens a network listener, so do it only when the user asks
to host a session from this computer. It runs in the foreground until stopped
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
connection itself comes from loopback. `--origin` admits a page served from
another origin besides `https://abcdesign.app` and loopback. `--json` prints
the start line as an object with `url`, `loopback`, `behind_proxy`, `sign_in`
and `hint`.

Team members sign in with the team's own GitHub sign-in app. The relay reads
its secret from `ABCD_SESSION_GITHUB_CLIENT_SECRET` by name, and its public
client id from `--client-id` or `ABCD_SESSION_GITHUB_CLIENT_ID`. Never ask the
user for the secret, and never print it or pass it on a command line. Without
them, sign-in is not offered and a team member pastes a personal access token
instead; the relay asks GitHub for that token's login and whether it may push
to the repository, and drops the token straight after.

It refuses (exit 2, nothing started) a `--listen` that is not an IP address
(or `localhost`) and a port, an `--origin` that is not an exact
`scheme://host[:port]`, and an address it cannot listen on.

## Blueprint

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" session blueprint --release <vMAJOR.MINOR.PATCH>
```

It writes `.abcd/relay/render.yaml`, a Render Blueprint declaring one free web
service, and `.abcd/relay/Dockerfile`, which downloads the named abcd
release's linux/amd64 binary and checks it against the SHA-256 that release's
`checksums.txt` publishes. It reads that checksum from the release page when it
writes the files (the one network read it makes, named on stderr) and pins it
in the Dockerfile. `--release` defaults to this abcd's own version when that is
a release. A file left exactly as abcd wrote it is replaced; one that was
edited, or that abcd did not write, is never overwritten, and then nothing is
written (exit 2). It refuses outside a checkout abcd manages, and for a
release that publishes no checksum for the linux/amd64 binary.

Relay what it prints: each file and whether it was written, replaced or left
unchanged, the release and checksum pinned, and the next steps. The user
commits both files, chooses New, then Blueprint, on Render, picks the
repository, sets the Blueprint path to `.abcd/relay/render.yaml`, and sets
`ABCD_SESSION_GITHUB_CLIENT_ID` and `ABCD_SESSION_GITHUB_CLIENT_SECRET` in the
service's Environment. `--json` returns `release`, `asset`, `sha256`,
`checksums_url` and `files`, each with its `path` and `state`.

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
only to the relay the person names and to `api.github.com` with the person's
own token, and it sends the person to `github.com` for sign-in and to open the
pre-filled issues a session sends. The directory is the one the user names;
never publish or deploy it for them.

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
