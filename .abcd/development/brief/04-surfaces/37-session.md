# `/abcd:session` — The Brief Session's Relay

`/abcd:session` runs the relay a team's live brief session goes through,
writes the blueprint that deploys it as the team's own relay, and writes the
static page the session is held on (itd-2610091016262335,
spc-2610141709430146 steps 1 to 3). abcd runs no relay
and holds no key
([adr-2610091016270711](../../decisions/adrs/2610091016270711-who-may-change-an-abcd-managed-project-s.md)):
the team runs it, on the host's own machine or on the team's own Render
account. It is abcd's second listener, admitted on the terms of
[adr-2610032150581128](../../decisions/adrs/2610032150581128-abcd-is-reachable-on-a-network-only.md)'s
amendment of 2026-10-14 and brief invariant 7's inbound clause
([`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md)).

This chapter describes the relay, its blueprint and the page. The route back
to GitHub is the spec's later step.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `serve` | — | shipped |
| `blueprint` | — | shipped |
| `page` | — | shipped |

## Where the relay listens

The relay runs in the foreground until interrupted and listens on loopback
(`127.0.0.1:8787`) unless the host names another address on that run. A
non-loopback address is the second, explicit choice the amendment requires,
and is meant for the team's relay, which listens on every address behind its
host's proxy and trusts that proxy's forwarded address. On a person's own
machine the relay is reached through Tailscale Serve, on the tailnet only, and
the start line prints the Serve command that does it. Funnel is never offered:
a guest outside the tailnet joins through the team's relay.

It serves four routes and nothing else: the session's WebSocket at `/session`,
the sign-in exchange at `/auth/token`, the sign-in app's public client id at
`/config`, and a health check at `/healthz`. A browser is admitted only from
`https://abcdesign.app`, a loopback page or another origin the host names, each
an exact origin; a WebSocket or exchange with no origin, or another one, is
refused with 403.

## Who may do what

A connection is a guest until GitHub says otherwise. The host signs in, and
GitHub must report that the login may push to the repository before a session
opens; the relay answers with a six-character join code drawn from a
cryptographic source over 31 symbols with the look-alike characters left out.
The host can draw a new code at any time, and the old one stops working. A
host whose page reloads keeps the role by signing in as the same login.

Anyone with the code joins as a guest and receives the session's update log
replayed, then every update as it arrives. A guest's update frame is refused,
which is how a guest stays read-only on the text; a guest's `suggest` message
is routed to the team members alone, stamped with the guest's display name,
cleaned of control and format characters. A participant who presents a GitHub
token, signed in or pasted, becomes a team member when GitHub reports push
access, and the token is dropped straight after the check. Team members'
updates are logged and forwarded; awareness (cursors and selections) is
forwarded from anyone and never logged. A team member registers the Yjs client
ids their page edits under, which the roster carries for credit.

Twenty wrong codes from one address within ten minutes shut that address out
of joining until the window passes; an IPv6 visitor is counted by its /64, and
the forwarded address is trusted only behind a declared proxy.

## What it holds, and for how long

Updates are opaque bytes behind a one-byte tag; the relay never parses them or
the brief. Everything is in memory: a session is dropped whole when the host
closes it, or when nobody has been connected for thirty minutes. Nothing is
written to disk and nothing is logged, no token, secret or brief text. The
sign-in app's secret is read from `ABCD_SESSION_GITHUB_CLIENT_SECRET` by name,
added only to the relay's own exchange with GitHub, and never sent to a
browser; GitHub's answer is passed on field by field, so nothing it echoes
reaches the page.

Limits: 256 KiB a message, 8 MiB of log a session, 32 participants a session,
16 sessions a relay, five sign-in attempts a connection, and a connection that
falls a thousand frames behind or misses a ping is dropped.

## The blueprint

The blueprint is `.abcd/relay/render.yaml`, one free Render web service with
the docker runtime, a `/healthz` health check, automatic deploys off (a
redeploy ends every open session) and the sign-in app's two variables
unsynced, and `.abcd/relay/Dockerfile`, which downloads the named release's
`abcd-linux-amd64` and checks it against the SHA-256 that release's
`checksums.txt` publishes, read when the files are written and pinned in the
file, before running the relay on every address behind Render's proxy. Each
file's first line carries a digest of its body, so a
rerun replaces a file left as abcd wrote it and refuses, writing nothing, one
that was edited or that abcd did not write. It refuses outside a checkout abcd
manages and for a release that publishes no checksum for the linux/amd64
binary.

## The page

The page is written into a directory the person names, from files embedded in
the binary: `index.html` and an identical `404.html`, `session.js`,
`session.css`, and the vendored editor bundle `vendor/collab.js` with its
licences
([adr-2610141949110653](../../decisions/adrs/2610141949110653-the-brief-session-s-browser-editor-ships.md)).
It writes those six files and nothing else and removes nothing. The directory
may be absent, empty, or one an earlier run wrote, whose files are replaced;
one holding anything else, or the page's names without the `index.html` it
wrote, is refused with nothing written. The bundle's bytes and its entry
module are checked against the committed provenance by a test.

A static host's 404 fallback serves the page at `/<owner>/<repo>/brief/`; the
HTML names its own files three levels up, so a base path works too. Its
content-security policy loads scripts and styles from the page's own origin
only and lets it connect over https and wss, or to loopback for a local
relay; it sends no referrer, and every piece of brief, relay or participant
text is put on the page as text, never as markup.

- **Joining.** A joiner's relay address and code arrive in the URL fragment of
  the host's QR code, so neither reaches the static host; a host types the
  relay address once and the page remembers it. Only an https relay is
  accepted, or plain http on this computer.
- **Signing in.** "Sign in with GitHub" goes to GitHub's authorize page with
  PKCE and a random state, both held in session storage across the redirect;
  the page posts the returned code and verifier to the relay's `/auth/token`.
  Pasting a personal access token is the fallback. The token is held in the
  page's memory only, and is presented to the relay as `verify`.
- **Hosting.** A team member hosts; the page reads the default branch's brief
  through GitHub's API with the host's token and seeds one shared document
  with a text per file, once, recording the base commit, the branch, the
  route back (one pull request by default, or one issue per change) and a
  snapshot of the seeded state. The seed is sent before the invite shows: a
  QR code of the page's address with the relay and code in the fragment, the
  code in large letters, and buttons for a new code and to end the session.
- **Editing.** A file list grouped by folder, and a Markdown editor bound to
  the shared text, with each participant's cursor and name. A team member's
  edits are sent as they are typed and the page registers its client id for
  credit; edits made while the connection is down are sent when it returns,
  and the page rejoins with the same code and replays the log.
- **Guests.** A guest's editor is read-only. A guest selects a passage and
  proposes a replacement, sent with the passage's relative positions and a
  text quote of it. Team members see it in a suggestions panel: Apply replaces
  the passage, found by its positions or else by its quote and refused when
  neither still matches, and records the guest, the approving member, the text
  before and after, and the time, in one change; Decline drops it, and both
  are recorded so every member's panel agrees.

Sending the session back to GitHub is the next step; the page shows that it
is coming.

## Exit codes

`0` done; `1` a failure after the relay started, or while the page was being
written; `2` refused, with nothing
started or written.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd session`

Sub-verbs: `abcd session blueprint`, `abcd session page`, `abcd session serve`.

Flags: none.

### `abcd session blueprint`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--relay-origin` | string |
| `--release` | string |

### `abcd session page`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--base-path` | string |
| `--out` | string |

### `abcd session serve`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--behind-proxy` | bool |
| `--client-id` | string |
| `--listen` | string |
| `--origin` | stringArray |
| `--owner` | stringArray |
| `--repo` | stringArray |

<!-- surface-appendix:end -->
