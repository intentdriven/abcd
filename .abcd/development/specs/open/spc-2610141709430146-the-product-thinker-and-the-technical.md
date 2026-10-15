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

**One relay, two routes.** `abcd session serve` is a verb in the abcd binary. It listens on loopback by default, and it serves four things:
- the session's WebSocket at `/session`
- the sign-in exchange at `/auth/token`
- the GitHub App's public client id at `/config`, with `app_type` always `github-app` when a client id is set, since the relay takes no other kind of sign-in app (see Sign-in), and an authorize URL that carries no scope, which a GitHub App does not use ("GitHub Apps do not request scopes during authorization", https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app). `serve` refuses to start with `ABCD_SESSION_GITHUB_APP_TYPE`, which once chose an OAuth app, set to anything but `github-app`.
- a health check at `/healthz`

A relay serves sessions only for the repositories it is given: each `--repo owner/name`, every repository of each `--owner`, and each entry of `ABCD_SESSION_REPOS`, a comma list of either. Nobody verifies against, or hosts, any other repository, so a stranger signed in to GitHub cannot hold a team's relay with a repository of their own. A relay listening beyond loopback refuses to start without one; on loopback it serves the checkout's origin repository, or any repository outside a GitHub checkout.

The relay reaches the team by one of two routes:
- **The host's own machine.** The host exposes the relay with `tailscale serve`, which reaches teammates on the tailnet and gives the browser the https address it needs. Funnel is never used, because it would put the person's computer on the internet. A guest outside the tailnet joins through the team's relay instead (adr-2610032150581128, amendment of 2026-10-14).
- **The team's own relay.** `abcd session blueprint` writes a Render Blueprint and a Dockerfile under `.abcd/relay/`. The Dockerfile fetches the released abcd binary for linux/amd64 and checks it against the release's checksum before running the same verb. One deploy serves all of the team's sessions. The Blueprint sets `ABCD_SESSION_REPOS` to the checkout's origin repository, a plain value, and `ABCD_SESSION_ORIGINS` to the page origins the relay admits: `https://abcdesign.app`, or the https origins `blueprint --origin` names. `serve` reads `ABCD_SESSION_ORIGINS`, a comma list, whenever no `--origin` is given. Given `--relay-origin`, the verb also writes `.abcd/relay/relays.json`, `{"relays": ["<https origin>"]}` with a stamp key the page ignores, which the page reads from the repository as its trust anchor for relays.

The relay holds:
- each session's Yjs updates, as opaque bytes, in memory only. A session's log is dropped when the host closes the session, or when nobody has been connected for the idle limit.
- the team's GitHub sign-in app secret, read from the environment by name. The secret is never logged and never sent to a browser.

The relay never parses the brief and never writes to disk.

**The wire.** A JSON control message is a text frame. A Yjs update or awareness message is a binary frame with a one-byte tag:
- tag 0, an update, sent by a team member's page;
- tag 1, an awareness message, sent by anyone;
- tag 2, an attributed update, sent only by the relay and refused from a page: the tag, the sender's participant number as four big-endian bytes (the N of its roster id `pN`), then the update's bytes exactly as the member sent them after its tag 0. Every update is logged and sent on in this form, live and in the replay. The roster's `authors` map names the login of each participant number the log carries, including members who have left, so a page checks that the Yjs client ids inside an update are credited to its sender's login. The relay still never parses the update.
- tag 3, an attributed awareness message, sent only by the relay and refused from a page: the tag, the sender's participant number as four big-endian bytes, then the awareness message's bytes exactly as the participant sent them after its tag 1, mirroring tag 2. Every awareness message is sent on in this form, and none is logged. A page keeps an awareness state only when its client id is tied to that sender: credited to a member's login, or, for a guest, who has no client ids credited, the one client id the guest's first awareness message named while the guest is in the roster. It labels the cursor from the roster, a member by their login and a guest by the name the relay stamped, never by the name the message claims.

The roster also names the login that opened the session (`host`), while the host is away too. The shared `meta` map (`base_snapshot`, `base_sha`, `branch`, `repo` and `mode`) counts a value only when the Yjs item that holds it was written by a client id credited to that login: the first four as the host's page seeded them, any later write void, and `mode` as the host last set it. A page warns when it sees a void write, and reads the base, the pull request's target and the commit's expected head through that one rule.
A page that refuses an update (one holding a client id not credited to its sender) also sets aside, with a note, whatever Yjs then holds pending on that update's clocks: every page refuses the same update, so nothing could ever supply them.

These control messages exist:
- `host`: a team member opens a session for owner/repo. The relay answers with a six-character join code from rascals' 31-symbol alphabet. The code works while the host keeps it open, and the host can rotate it.
- `join`: anyone presents the code. Failed joins are rate-limited per address: an IPv4 address, or an IPv6 /48, or one tailnet address, 20 wrong codes in ten minutes each. Past 600 wrong codes from everywhere within ten minutes, joining by code pauses for everybody for ten minutes, as the rascals server does: every code tried meanwhile gets the same `paused` answer, right or wrong, and counts for its own address alone. A rejoin token and a verified team member are not held up.

  **Accepted tradeoff (security review 2, finding 9).** About thirty addresses, each spending its own 20 wrong codes, can pause joining by code for everybody for ten minutes, and they can repeat that. The pause is what keeps a guesser with many addresses from searching the 887 million codes, so it stays. What it costs is bounded: only a new guest joining by code waits, and the session itself goes on; a participant comes back by their rejoin token, and a team member is exempt, since signing in admits them whatever the pause. Exempting members does not help a guesser: a member's wrong codes do not count towards the pause, and a member can open the session directly. The budget was raised from 200 to 600 so the pause takes about thirty addresses rather than ten. The forwarded address is trusted only behind a declared proxy, or, on a relay bound to loopback alone, as Tailscale Serve forwards it: one tailnet address, on a connection from loopback. Behind a proxy it is the rightmost `X-Forwarded-For` entry, one trusted hop: Render's proxy appends to the header, so the client controls the leftmost entry (https://render.com/articles/host-pocketbase-on-render), and Render's advice is to trust one hop (https://render.com/articles/how-render-handles-ddos-attacks). On a loopback relay a process on the host could forge a tailnet address in that header; that is accepted, since such a process already holds the host's machine.
- `verify`: a participant presents a GitHub token for a repository the relay serves. The relay asks GitHub for the login, and for the repository's `permissions.push`. A participant with write access or above becomes a team member, and the relay drops the token straight after the check.
- `rejoin`: a participant whose connection dropped presents the rejoin token the relay gave them with `hosted` or `joined`: 128 random bits, held in memory with the participant as a digest. It restores the same participant (role, name and verified login) without the code, so rotating the code shuts nobody out; closing, purging or expiring the session voids it. The role restored is the one the relay recorded, not asked of GitHub again. **Decision (facilitator, on security review 2, finding 8):** a member whose write access GitHub has since revoked keeps their role at most until the session ends or expires; a session expires 12 hours after it was opened, however busy, and ends for everyone as a host's close does, with `closed {reason: "expired"}`. Failed rejoins count with failed joins. The page keeps the token in memory and in the tab's session storage, so a reload rejoins too.

The relay replays the session's update log to every newcomer, as one item on the connection's send queue that the writer sends entry by entry as the connection takes them, so a log longer than the queue never drops a newcomer. It then:
- forwards Yjs updates only from team members. A guest's binary updates are refused, which is how guests stay read-only on the text.
- forwards a guest's `suggest` message to the team members, stamped with the guest's display name.
- broadcasts the roster of verified identities.

Limits bound all of this: message size (256 KiB), an awareness message's size (8 KiB); log bytes, each entry charged 64 bytes beyond its own; log entries (50,000); participants per session; sessions per relay; idle time; a session's life (12 hours); connections (256 per relay and 16 per address, past which the upgrade is refused with 503); and 30 seconds for a connection to host, join or rejoin before it is closed, its send queue held to 64 items until then. Each connection's send queue holds at most 1,024 items and 8 MiB of frames; a log replay is charged no bytes, its entries being the session's own. A connection past either is dropped at once, its queue with it, so a page that stops reading holds nothing more (security review 2, finding 1). A guest's suggestions are bounded in size (70 KiB each), in count (100 per connection) and in total (1 MiB per session), since a member's page records each in the document and so in the log (finding 2).

Rates bound it too: each connection's updates (60 a second, in bursts of 200), awareness messages (20 a second, in bursts of 40) and suggestions (1 a second, in bursts of 5), each session's awareness bytes from every participant together (128 KiB a second, in bursts of 512 KiB), its sign-in attempts (5), the sessions one address opens (3 in 10 minutes), and the sign-in exchanges at `/auth/token` from one address (10 in 10 minutes, keyed as wrong codes are, then 429 with `Retry-After` before GitHub is asked; finding 10). A message over its rate is dropped with a `rate-limited` error naming its `kind` (`update`, `awareness`, `host` or `suggest`), and the connection stays open: the page resends recent updates, tells a guest to send a dropped suggestion again, and answers a dropped cursor with nothing. Every control message, and every frame the relay refuses, also spends one budget per connection (50 a second, in bursts of 100): past it, the message is dropped unanswered before the relay's lock is taken, and a connection refused 1,000 times is closed (finding 7). An accepted update or awareness message spends nothing of it, and a rate limit's own refusal does not count as one. Origins are checked against an allowlist: abcdesign.app and loopback pages by default, or exactly the origins `--origin` names, or, without `--origin`, those `ABCD_SESSION_ORIGINS` names, which replace them.

**Sign-in.** The page builds the GitHub authorize URL from the relay's public client id, with PKCE and a state value it keeps in session storage. The redirect comes back to the page. The page posts the code and the PKCE verifier to `/auth/token`, and the relay adds the secret and does the exchange: this is the token-mediating backend in the IETF's browser-based-apps guidance. The token stays in the page's memory. Pasting a personal access token is the fallback, and the same `verify` admits it.

A link's fragment can name any relay, so the repository is the page's trust anchor for relays (security review F1). Before any sign-in, token or display name reaches a relay, the page reads the repository's committed `.abcd/relay/relays.json` (`{"relays": ["https://…"]}`, https origins, other keys ignored) from `api.github.com` on the default branch, without a token for a public repository, and refuses a relay the file does not name: no `/config`, no sign-in, no `verify`, no join and no connection. Only GitHub's 404 means there is no list the page can see. A 403, a 429, a 5xx or a network error means the list could not be checked: the relay is refused, with "could not check the relay list; try again later" and no confirmation offered, and the next click reads the list again (security review 2, finding 5). Where GitHub answers 404 without a token (a private repository, or no file), the page shows the relay's origin and the person must confirm it ("Only continue if your team runs this relay"; for a guest's join, "Only continue if this repository's team gave you this relay") before any sign-in, token or join reaches it; a confirmed relay is remembered per repository in the browser, a relay from the fragment is never remembered without that confirmation, and the relay card is never hidden for a relay that is not cleared. Once a token exists, the page asks `api.github.com` alone for the file again, with the token; a file that lists relays without this one refuses `verify` and forgets the confirmation.

Sign-in goes through a GitHub App only. **Decision (facilitator, on security review 2, finding 3, on state-of-the-art grounds):** a link can open the page for a stranger's repository, whose own `relays.json` names the stranger's relay, so the list alone cannot stop a relay collecting tokens; what bounds the harm is the token. A GitHub App's user token reaches only the repositories the app is installed on, within its fine-grained permissions (Contents and Pull requests read and write, Metadata read), and GitHub recommends Apps over OAuth apps (https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps), while an OAuth app's `scope=repo` reaches every repository the person can. So the authorize URL never carries a scope, the oauth-app choice is gone, and a relay whose `/config` says anything other than `github-app` gets no sign-in button: the person may paste a token only. The page names the repository prominently wherever a relay is trusted: a listed relay shows "This relay is named by owner/repo. Only continue if you work on owner/repo." beside the sign-in, and the confirmation for an unlisted one opens with "This page is for owner/repo."

**Frames.** A static host such as GitHub Pages cannot send `frame-ancestors` or `X-Frame-Options`, and a `<meta>` Content-Security-Policy cannot carry `frame-ancestors`. That is accepted (security review 2, finding 11b), with mitigations: the page refuses to run in a frame (`window.top !== window.self`) and shows only a link that opens it in its own tab, so no relay confirmation, sign-in or session is ever drawn where another page could overlay it; the sign-in itself happens on github.com, which refuses to be framed.

**The page.** abcd generates the page itself. `abcd session page --out <dir>` writes plain files from assets embedded in the binary:
- one HTML file, served for every path through GitHub Pages' 404 fallback, which is how `/<owner>/<repo>/brief/` resolves on a static host
- the page script
- the stylesheet
- `collab.js`, a vendored bundle

`collab.js` holds Yjs 13.6.33, y-codemirror.next 0.3.6, y-protocols 1.0.7, CodeMirror 6 with Markdown, and qrcode-generator 2.0.4, all MIT. It was built outside the repository with esbuild 0.28.2. Every package version is at least two weeks old. Its provenance (each package, version and npm integrity hash, the entry module and the bundle's SHA-256) is committed beside it, and a Go test fails if the bytes drift. No Node toolchain enters the repository, consistent with adr-47.

The host's page loads the brief through the GitHub API with the host's token. It records the default branch's commit as the session's base and seeds one Y.Text per brief file exactly once, before any invite. It then shows a QR code that encodes the page URL, with the relay address and the code in the URL fragment, so neither reaches GitHub's servers.

Participants:
- see each other's edits and cursors live (awareness)
- if they are guests, propose a replacement for a selected passage; a team member applies or declines it, and the record names the approver. A member's page records only a suggestion's validated fields, each bounded: a path among the brief files the session started with; relative positions cleaned and at most 256 bytes as JSON; the passage's context at most 200 characters each side; the passage and the replacement at most 64,000 characters; and all of it at most 70 KiB, the relay's cap, which the guest's page checks before sending (security review 2, finding 2). The relay's answer to a suggestion, a `rate-limited` one included, does not say which message it answers, so a guest's page has one suggestion in flight at a time: a refusal or a rate limit clears it and tells the guest it was not sent, and only an accepted one is listed as waiting for a team member. The files a session sends back are confined to the base snapshot read through the host-written settings (`effectiveMeta`), never the member-writable `meta` map as such (finding 6).

The document has `gc` off, so each change can be credited. Credit comes from the Yjs client ids each member's page registers in the roster.

**Back to GitHub.** The host picks the route per session.
- **Issues.** Each changed passage becomes a pre-filled issue link, carrying:
  - the file path
  - the passage before and after
  - the base commit
  - the credited author, or the approver and the guest

  The base commit is named only when it is 40 hex digits, and a guest's display name, in an issue body or a pull request body, goes in a code span, its fence longer than any run of backticks in it, so it cannot write a link, an image, code or emphasis (security review 2, finding 11a), nor be autolinked: GitHub links a bare URL or domain in a name even with every Markdown character escaped, but nothing inside a code span. A commit's trailer names the guest as plain text.

  Each issue's title starts `Brief:` and its body ends with the fixed line `Sent from an abcdesign.app brief session.` The team's agent session finds a session's issues by that marker, not by a label: a pre-filled label makes GitHub answer 404 to anyone without triage permission, and a label the repository lacks is unreliable.

  The link's length is measured as GitHub receives it, after encoding, against a margin under 8 KB. A longer passage is offered as a pull request instead.
- **One pull request.** The submitter's token calls `createRef`, then `createCommitOnBranch` (the commit's author is the submitter), then `createPullRequest`. The commit message carries a `Co-authored-by:` line for every credited team member. For each approved guest change it carries a `Suggested-by:` line naming the guest, with the `Approved-by:` line beside it.

Every request in a session goes to the team's relay, to GitHub, or to the static page's host. None reaches a server abcd runs.

The hosting of abcdesign.app is recorded in the plan but not made in this change. abcd's own GitHub Pages site is planned to become abcd's record (itd-2610142145236613 and adr-2610142147242544, planned on their own branch), and a repository has one Pages site, so the static abcdesign.app page is published from a repository of its own: that repository's workflow downloads the released abcd and runs `abcd session page`. Creating that repository is an outward act, done at the cut with the product thinker's go. <!-- record-lint: forward-looking -->

## Footprint

- packages: internal/core/session (new), internal/surface/session (new), internal/surface/cli, commands/session.md, docs/
- tests: internal/core/session, internal/surface/session, internal/surface/cli, evals/ smoke

## Steps

1. The relay: session rooms, join codes, roles, update log, suggestions, sign-in exchange and limits, behind `abcd session serve`, with the WebSocket library github.com/coder/websocket v1.8.15 (ISC, no dependencies)
   - packages: internal/core/session, internal/surface/session, internal/surface/cli, commands/session.md
   - tests: internal/core/session, internal/surface/session
   - landed: fc68350ac
2. The relay blueprint and the Tailscale route: `abcd session blueprint` writes `.abcd/relay/render.yaml` and `.abcd/relay/Dockerfile`, pinned to a released binary and its checksum
   - packages: internal/core/session, internal/surface/cli, docs/
   - tests: internal/core/session, internal/surface/cli
   - landed: fc68350ac
3. The page: `abcd session page --out` emits the static page and the vendored, provenance-pinned editor bundle; joining by QR or code, live editing, presence, guest suggestions and approvals, and credit
   - packages: internal/surface/session/web, internal/surface/cli
   - tests: internal/surface/session
   - landed: f835b6f38
4. Back to GitHub: issue links under the measured length limit, and one pull request with co-author, Suggested-by and Approved-by lines
   - packages: internal/surface/session/web
   - tests: internal/surface/session
   - landed: a30ec9432
