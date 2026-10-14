---
name: session
description: "Run the relay a team's live brief session goes through, or write its deployment blueprint: Writes nothing bare; refuses an unknown sub-verb."
argument-hint: "[serve|blueprint]"
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

There are two ways to run it, both the team's own:

- **On the host's own machine**, with `serve`, reached by teammates on the
  host's Tailscale network through Tailscale Serve. Never Funnel: a guest
  outside the tailnet joins through the team's relay instead.
- **On the team's own relay**, deployed once to the team's own Render account
  from the files `blueprint` writes. One deploy serves all of the team's
  sessions.

## Status

Bare invocation prints the two sub-verbs and writes nothing. A running relay's
sessions live in that process's memory alone, so there is no state to report.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" session --help
```

## Serve

Starting the relay opens a network listener, so do it only when the user asks
to host a session from this computer. It runs in the foreground until stopped
with Ctrl-C, and every session it holds ends with it, so the user runs it in a
Terminal of their own rather than in this session:

```bash
abcd session serve
```

It listens on this computer alone (`127.0.0.1:8787`) and prints one line
saying where, with the `tailscale serve --bg <port>` command that puts it on
the user's tailnet at an https address; relay that line as printed. A
non-loopback `--listen` is a second, explicit choice, meant for the team's
relay; pass it only when the user names the address. `--behind-proxy` trusts
the proxy's `X-Forwarded-For` entry as each visitor's address, and belongs to
the team's relay behind its host's proxy. `--origin` admits a page served from
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

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
