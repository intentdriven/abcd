# `/abcd:session` — The Brief Session's Relay

`/abcd:session` sets up live brief sessions for a project (the team's own
GitHub sign-in app, from a pre-filled manifest), starts a live brief session
on the host's own computer, runs the relay a team's cloud sessions go through,
writes the blueprint that deploys it as the team's own relay, and writes the
static page the session is held on (itd-2610091016262335, spc-2610141709430146
steps 1 to 6). The page
never shows or asks for a relay address or a token
([adr-2610161400542916](../../decisions/adrs/2610161400542916-the-brief-session-page-asks-nobody-for-a.md)). abcd runs no relay
and holds no key
([adr-2610091016270711](../../decisions/adrs/2610091016270711-who-may-change-an-abcd-managed-project-s.md)):
the team runs it, on the host's own machine or on the team's own Render
account. It is abcd's second listener, admitted on the terms of
[adr-2610032150581128](../../decisions/adrs/2610032150581128-abcd-is-reachable-on-a-network-only.md)'s
amendments of 2026-10-14 and 2026-10-16 and brief invariant 7's inbound clause
([`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md)).

This chapter describes the relay, its blueprint and the page. The route back
to GitHub is the page's own: it sends a session's changes as one pull request or pre-filled issues (see Sending).

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
| `setup` | — | shipped |
| `start` | — | shipped |
| `serve` | — | shipped |
| `blueprint` | — | shipped |
| `page` | — | shipped |

## Where the relay listens

The relay runs in the foreground until interrupted and listens on loopback
(`127.0.0.1:8787`) unless the host names another address on that run. A
non-loopback address is the second, explicit choice the amendment requires,
and is meant for the team's relay, which listens on every address behind its
host's proxy and trusts that proxy's forwarded address. On a person's own
machine, the local session runs it on loopback, serves the page from it as
well, and
prints the Tailscale Serve command that puts it on the tailnet, with the link
to open. Funnel is never offered: someone outside the tailnet joins once the
host shares the computer with them in Tailscale, as a guest. A local session
answers
only the computer's tailnet name and its loopback address, as hosts (any other
gets 421) and as page origins.

It serves these routes and nothing else: the session's WebSocket at
`/session`, the web-flow sign-in's callback at `/auth/callback` and its
exchange at `/auth/token`, the device-flow sign-in at `/auth/device`,
`/auth/device/token` and `/auth/device/cancel` (a local session only), the sign-in app's public
client id and flow at `/config`, a health check at `/healthz`, and, for a
local session only, the page's embedded files from a fixed route table. A browser is admitted only from
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
cleaned of control and format characters. A participant who presents the
GitHub App user access token their sign-in gave them (`ghu_`; any other kind
is refused before GitHub is asked, so nobody pastes a token) becomes a team
member when GitHub reports push access, and the token is dropped straight
after the check. Team members'
updates are logged and forwarded; awareness (cursors and selections) is
forwarded from anyone and never logged. A team member registers the Yjs client
ids their page edits under, which the roster carries for credit.

Twenty wrong codes from one address within ten minutes shut that address out
of joining until the window passes; an IPv6 visitor is counted by its /48 (a tailnet address by itself, since a tailnet's devices share one /48), and
the forwarded address is trusted only behind a declared proxy, or, on a relay bound to loopback, as Tailscale Serve forwards a tailnet address from a loopback peer.

## What it holds, and for how long

Updates are opaque bytes behind a one-byte tag; the relay never parses them or
the brief. Everything is in memory: a session is dropped whole when the host
closes it, or when nobody has been connected for thirty minutes. Nothing is
written to disk and nothing is logged, no token, secret or brief text. The
sign-in app's secret, on a cloud relay, is read from
`ABCD_SESSION_GITHUB_CLIENT_SECRET` by name, added only to the relay's own
exchange with GitHub, and never sent to a browser; GitHub's answer is passed
on field by field, so nothing it echoes reaches the page. A local session's
relay, which serves its own page and holds the client id and no secret (it
never reads one), offers GitHub's device flow instead and keeps each device
code in memory under a random handle, at most 64 at once and four from one
client address, never sending it to a browser. A relay that serves no page
offers the device flow never: without the secret it offers no sign-in, and
the page on any origin but its own relay's signs in through the web flow
alone.

Limits: 256 KiB a message, 8 MiB of log a session, 32 participants a session,
16 sessions a relay, five sign-in attempts a connection, and a connection that
falls a thousand frames behind or misses a ping is dropped.

## The blueprint

The blueprint is `render.yaml` at the repository's root (or `.abcd/relay/render.yaml` beside a root `render.yaml` the team wrote), one free Render web service with
the docker runtime, a `/healthz` health check, automatic deploys off (a
redeploy ends every open session) and the sign-in app's secret, with its client id until setup has recorded the app,
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
HTML names its own files from the site's root, or from under the base path the page is built for, so a base path works too. Its
content-security policy loads scripts and styles from the page's own origin
only and lets it connect over https and wss, or to loopback for a local
relay; it sends no referrer, and every piece of brief, relay or participant
text is put on the page as text, never as markup.

- **Finding the relay.** Nobody types a relay address. A page a local session
  serves takes its own origin. On a static host the page uses the relay the invite
  link's fragment names, if the repository's `.abcd/relay/relays.json` names
  it (a relay the file does not name is refused: the page says the link does not lead to this project's own session), or else, with no relay in the link, the first relay the file names; where GitHub answers that there is no file to read
  before sign-in, it asks in plain words whether the project's team sent the
  link; that answer holds in that tab, for at most twelve hours. With none, where the file names no relay it says live sessions are not set up yet, and where there is no file it says it found no live session and to ask the host for the invite link; either way it says to ask the
  technical facilitator. The invite link carries the relay only when the
  guest's page could not find it alone.
- **Signing in.** "Sign in with GitHub" goes through a relay the repository's
  file names, a confirmed one where the file cannot be read, or a local
  session's own page. On a cloud relay it
  goes to GitHub's authorize page with PKCE and a random state, both held in
  session storage across the redirect, and with the relay's own
  `/auth/callback` as the redirect, the callback URL the team's app lists.
  GitHub returns the person to that relay, which sends them back to the page
  naming itself, only to a session page under the base path the page's
  origin entry names (the root unless the entry names one), and the page posts the code and verifier to that relay's
  `/auth/token` alone, refusing a return that names no relay or another one;
  the relay exchanges only a code given for its own callback, and the token
  is presented only to that relay. In a local session, the page shows the device flow's code
  to confirm on github.com and asks the relay after it. There is no token
  field. The token is held in the page's memory only, and is presented to the
  relay as `verify`.
- **Hosting.** A team member hosts; the page reads the default branch's brief
  through GitHub's API with the host's token and seeds one shared document
  with a text per file, once, recording the base commit, the branch, the
  route back (one pull request by default, or one issue per change) and a
  snapshot of the seeded state. The seed is sent before the invite shows: a
  QR code of the page's address with the code in the fragment, and the relay too only where a guest's page could not find it alone, the
  code in large letters, and buttons for a new code and to end the session.
- **Editing.** The session opens on the brief's press release. A list of
  the brief's documents, grouped by folder, and a Markdown editor bound to
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
- **Sending.** A team member opens Send to GitHub, which lists each changed
  passage with its credit, the text before and after, and the route the host
  chose: one pull request for the session, or a pre-filled issue per passage.
- **Two designs.** The page has two looks over the same elements and the same
  logic, which the product thinker chose on 2026-10-16 so the team can try
  them against each other. Guided, the default, has large cards, numbered
  areas, a "What happens next" note (before the session on each step the
  page can go on from and when sessions are not set up, never while it is
  checking or cannot go on; live, for guests only), and arrival cards whose
  main buttons share a line, on one control height per size and a 4 px spacing scale.
  Shared puts a live team bar over the document, with an icon-only Invite at
  its end, an icon-only Send to GitHub that widens to its label on hover or
  keyboard focus, and Suggestions alone on the right, mirroring Documents on
  the left; as the window narrows, labels fade and fold to their icons, least
  important first, never cut, wrapped or overlapping; it has no "What happens
  next", and every overlay has a Close X at its top right. Ctrl+Shift+[ and
  Ctrl+Shift+] (Ctrl+{ and Ctrl+}) switch between the designs on every
  platform, the Mac included (never Cmd), leaving Ctrl+[ and Ctrl+] to the
  editor's indent and outdent, with a short toast naming the design. On
  Windows and Linux those keys are the editor's fold and unfold, which the
  page takes; it shows no fold arrows, so a section cannot be folded there
  (a Mac folds with Cmd+Alt+[ and Cmd+Alt+]); the choice is kept per viewer in the
  browser's local storage, beside the display name, and an unreadable or
  unknown value falls back to Guided. In both, Alt+Shift+D shows or hides the
  documents and Alt+Shift+S the right panel, as their tooltips say, each
  toggle keeping one name and showing whether it is on; a hidden sidebar
  gives its room to the brief, and the state lasts through every change of
  screen and design. A phone, where the sidebars become sheets, keeps its
  own state apart from a wider window's, and so does the right panel at the
  slide-over widths, so closing it there never hides it on a wide window.
  Escape closes the overlay on top that can be seen; inside the editor, and
  while an input method is composing, Escape is theirs. A dialog takes focus
  when it opens and gives it back to what opened it, and focus on a control
  a change of layout hides moves to the control that does its job now. Both follow the light or dark setting and
  reduced motion, reach every control from the keyboard with a visible focus,
  and use the system's own font, inline SVG icons and one cobalt accent token
  (#2457d6, or #7aa2ff in dark).

## Exit codes

`0` done; `1` a failure after the relay started, when no app is recorded (stopped, the hour ran out, GitHub's conversion failed three times, or the app came back owned by another account than the organisation), or while the page or the blueprint was being
written; `2` refused, with nothing
started or written, but for the live brief session question a person at a terminal answers first, whose answer is kept, and a blueprint that cannot be written once the app is recorded, which exits 2 with the app and the answer kept.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd session`

Sub-verbs: `abcd session blueprint`, `abcd session page`, `abcd session serve`, `abcd session setup`, `abcd session start`.

Flags: none.

### `abcd session blueprint`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--origin` | stringArray |
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

### `abcd session setup`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--relay-origin` | string |
| `--release` | string |
| `--where` | string |

### `abcd session start`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--client-id` | string |
| `--port` | int |

<!-- surface-appendix:end -->
