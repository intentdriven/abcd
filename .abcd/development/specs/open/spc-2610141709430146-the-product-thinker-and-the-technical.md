---
id: spc-2610141709430146
slug: the-product-thinker-and-the-technical
intent: itd-2610091016262335
origin: researcher-authored
production_mode: hand-written
---
# the-product-thinker-and-the-technical

## Summary

A team opens its brief at abcdesign.app/<owner>/<repo>/brief/ during a call and edits it together live. The page is static. The session runs on a relay the team owns, and changes go back to GitHub as issues or one pull request, never through anything abcd runs.

**One relay, two routes.** `abcd session serve` is a verb in the abcd binary. It listens on loopback by default, and it serves three things:
- the session's WebSocket at `/session`
- the sign-in exchange at `/auth/token`
- a health check at `/healthz`

The relay reaches the team by one of two routes:
- **The host's own machine.** The host exposes the relay with `tailscale serve`, which reaches teammates on the tailnet and gives the browser the https address it needs. Funnel is never used, because it would put the person's computer on the internet. A guest outside the tailnet joins through the team's relay instead (adr-2610032150581128, amendment of 2026-10-14).
- **The team's own relay.** `abcd session blueprint` writes a Render Blueprint and a Dockerfile under `.abcd/relay/`. The Dockerfile fetches the released abcd binary for linux/amd64 and checks it against the release's checksum before running the same verb. One deploy serves all of the team's sessions.

The relay holds:
- each session's Yjs updates, as opaque bytes, in memory only. A session's log is dropped when the host closes the session, or when nobody has been connected for the idle limit.
- the team's GitHub sign-in app secret, read from the environment by name. The secret is never logged and never sent to a browser.

The relay never parses the brief and never writes to disk.

**The wire.** A JSON control message is a text frame. A Yjs update or awareness message is a binary frame with a one-byte tag. These messages exist:
- `host`: a team member opens a session for owner/repo. The relay answers with a six-character join code from rascals' 31-symbol alphabet. The code works while the host keeps it open, and the host can rotate it.
- `join`: anyone presents the code. Failed joins are rate-limited per address, and the forwarded address is trusted only behind a declared proxy.
- `verify`: a participant presents a GitHub token. The relay asks GitHub for the login, and for the repository's `permissions.push`. A participant with write access or above becomes a team member, and the relay drops the token straight after the check.

The relay replays the session's update log to every newcomer. It then:
- forwards Yjs updates only from team members. A guest's binary updates are refused, which is how guests stay read-only on the text.
- forwards a guest's `suggest` message to the team members, stamped with the guest's display name.
- broadcasts the roster of verified identities.

Limits bound all of this: message size, log bytes, participants per session, sessions per relay and idle time. Origins are checked against an allowlist, with abcdesign.app and loopback as the defaults.

**Sign-in.** The page builds the GitHub authorize URL from the relay's public client id, with PKCE and a state value it keeps in session storage. The redirect comes back to the page. The page posts the code and the PKCE verifier to `/auth/token`, and the relay adds the secret and does the exchange: this is the token-mediating backend in the IETF's browser-based-apps guidance. The token stays in the page's memory. Pasting a personal access token is the fallback, and the same `verify` admits it.

**The page.** abcd generates the page itself. `abcd session page --out <dir>` writes plain files from assets embedded in the binary:
- one HTML file, served for every path through GitHub Pages' 404 fallback, which is how `/<owner>/<repo>/brief/` resolves on a static host
- the page script
- the stylesheet
- `collab.js`, a vendored bundle

`collab.js` holds Yjs 13.6.33, y-codemirror.next 0.3.6, y-protocols 1.0.7, CodeMirror 6 with Markdown, and qrcode-generator 2.0.4, all MIT. It was built outside the repository with esbuild 0.28.2. Every package version is at least two weeks old. Its provenance (each package, version and npm integrity hash, the entry module and the bundle's SHA-256) is committed beside it, and a Go test fails if the bytes drift. No Node toolchain enters the repository, consistent with adr-47.

The host's page loads the brief through the GitHub API with the host's token. It records the default branch's commit as the session's base and seeds one Y.Text per brief file exactly once, before any invite. It then shows a QR code that encodes the page URL, with the relay address and the code in the URL fragment, so neither reaches GitHub's servers.

Participants:
- see each other's edits and cursors live (awareness)
- if they are guests, propose a replacement for a selected passage; a team member applies or declines it, and the record names the approver

The document has `gc` off, so each change can be credited. Credit comes from the Yjs client ids each member's page registers in the roster.

**Back to GitHub.** The host picks the route per session.
- **Issues.** Each changed passage becomes a pre-filled issue link, carrying:
  - the file path
  - the passage before and after
  - the base commit
  - the credited author, or the approver and the guest

  The link's length is measured as GitHub receives it, after encoding, against a margin under 8 KB. A longer passage is offered as a pull request instead.
- **One pull request.** The submitter's token calls `createRef`, then `createCommitOnBranch` (the commit's author is the submitter), then `createPullRequest`. The commit message carries a `Co-authored-by:` line for every credited team member. For each approved guest change it carries a `Suggested-by:` line naming the guest, with the `Approved-by:` line beside it.

Every request in a session goes to the team's relay, to GitHub, or to the static page's host. None reaches a server abcd runs.

The hosting of abcdesign.app is recorded in the plan but not made in this change. Deploying the page to GitHub Pages under the abcdesign.app domain publishes, so it waits for the product thinker's go at the cut.

## Footprint

- packages: internal/core/session (new), internal/surface/session (new), internal/surface/cli, commands/session.md, docs/
- tests: internal/core/session, internal/surface/session, internal/surface/cli, evals/ smoke

## Steps

1. The relay: session rooms, join codes, roles, update log, suggestions, sign-in exchange and limits, behind `abcd session serve`, with the WebSocket library github.com/coder/websocket v1.8.15 (ISC, no dependencies)
   - packages: internal/core/session, internal/surface/session, internal/surface/cli, commands/session.md
   - tests: internal/core/session, internal/surface/session
2. The relay blueprint and the Tailscale route: `abcd session blueprint` writes `.abcd/relay/render.yaml` and `.abcd/relay/Dockerfile`, pinned to a released binary and its checksum
   - packages: internal/core/session, internal/surface/cli, docs/
   - tests: internal/core/session, internal/surface/cli
3. The page: `abcd session page --out` emits the static page and the vendored, provenance-pinned editor bundle; joining by QR or code, live editing, presence, guest suggestions and approvals, and credit
   - packages: internal/surface/session/web, internal/surface/cli
   - tests: internal/surface/session
4. Back to GitHub: issue links under the measured length limit, and one pull request with co-author, Suggested-by and Approved-by lines
   - packages: internal/surface/session/web
   - tests: internal/surface/session
