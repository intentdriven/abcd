---
id: spc-2610141709430146
slug: the-product-thinker-and-the-technical
intent: itd-2610091016262335
origin: researcher-authored
production_mode: hand-written
---
# the-product-thinker-and-the-technical

## Summary

A team opens its brief during a call and edits it together live: on abcdesign.app/<owner>/<repo>/brief/, or, for a sensitive project, on the host's own computer over the team's tailnet. The session runs on a relay the team owns, and changes go back to GitHub as issues or one pull request, never through anything abcd runs. The page never shows or asks for a relay address and never asks for a token (adr-2610161400542916). It opens onto the brief, press release first, in plain words for product thinkers and domain experts.

**One relay, two routes.** The relay is a verb in the abcd binary. `abcd session serve` runs it for the team's cloud relay, and `abcd session start` runs it for a local session (see Local session). It listens on loopback by default, and it serves these routes:
- the session's WebSocket at `/session`
- the web-flow sign-in exchange at `/auth/token`, offered when the relay holds the GitHub App's client id and secret
- the device-flow sign-in at `/auth/device` and `/auth/device/token`, offered only by a local session's relay, which serves its own page and holds the client id alone (see Sign-in)
- the GitHub App's public client id at `/config`, with `app_type` always `github-app` when a client id is set, since the relay takes no other kind of sign-in app (see Sign-in); `flow`, `web` or `device`; and, for the web flow, an authorize URL that carries no scope, which a GitHub App does not use ("GitHub Apps do not request scopes during authorization", https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app). `serve` refuses to start with `ABCD_SESSION_GITHUB_APP_TYPE`, which once chose an OAuth app, set to anything but `github-app`.
- a health check at `/healthz`
- for a local session only, the page itself (see Local session)

A relay serves sessions only for the repositories it is given: each `--repo owner/name`, every repository of each `--owner`, and each entry of `ABCD_SESSION_REPOS`, a comma list of either. Nobody verifies against, or hosts, any other repository, so a stranger signed in to GitHub cannot hold a team's relay with a repository of their own. A relay listening beyond loopback refuses to start without one; on loopback it serves the checkout's origin repository, or any repository outside a GitHub checkout.

The relay reaches the team by one of two routes:
- **The host's own machine.** `abcd session start` runs the relay and serves the page from it. The host exposes it with `tailscale serve`, which reaches teammates on the tailnet and gives the browser the https address it needs. Funnel is never used, because it would put the person's computer on the internet. Someone outside the tailnet reaches a local session only when the host shares the computer with them through Tailscale, and joins as a guest (adr-2610032150581128, amendments of 2026-10-14 and 2026-10-16).
- **The team's own relay.** `abcd session blueprint` writes a Render Blueprint and a Dockerfile under `.abcd/relay/`. The Dockerfile fetches the released abcd binary for linux/amd64 and checks it against the release's checksum before running the same verb. The verb refuses a release older than the first to carry `abcd session` (a constant the release cut sets, now v0.17.0; the running abcd's own release is always taken), since that container would exit at once. One deploy serves all of the team's sessions. The Blueprint sets `ABCD_SESSION_REPOS` to the checkout's origin repository, a plain value, and `ABCD_SESSION_ORIGINS` to the page origins the relay admits: `https://abcdesign.app`, or the https origins `blueprint --origin` names. `serve` reads `ABCD_SESSION_ORIGINS`, a comma list, whenever no `--origin` is given. Given `--relay-origin`, the verb also writes `.abcd/relay/relays.json`, `{"relays": ["<https origin>"]}` with a stamp key the page ignores, which the page reads from the repository as its trust anchor for relays.

The relay holds:
- each session's Yjs updates, as opaque bytes, in memory only. A session's log is dropped when the host closes the session, or when nobody has been connected for the idle limit.
- on a cloud relay, the team's GitHub sign-in app secret, read from the environment by name. The secret is never logged and never sent to a browser. A local session's relay holds no secret.
- each device sign-in under way: GitHub's device code, under a random handle, in memory only, at most 64 at once and four from one client address, each dropped when GitHub's code expires or the sign-in ends. The device code never reaches a browser.

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
- `verify`: a participant presents the GitHub token their sign-in gave them, for a repository the relay serves. The relay takes only a GitHub App user access token, whose documented prefix is `ghu_` (https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/about-authentication-to-github#githubs-token-formats), and refuses any other before asking GitHub, so a pasted personal access token is never accepted. It asks GitHub for the login, and for the repository's `permissions.push`. A participant with write access or above becomes a team member, and the relay drops the token straight after the check.
- `rejoin`: a participant whose connection dropped presents the rejoin token the relay gave them with `hosted` or `joined`: 128 random bits, held in memory with the participant as a digest. It restores the same participant (role, name and verified login) without the code, so rotating the code shuts nobody out; closing, purging or expiring the session voids it. The role restored is the one the relay recorded, not asked of GitHub again. **Decision (facilitator, on security review 2, finding 8):** a member whose write access GitHub has since revoked keeps their role at most until the session ends or expires; a session expires 12 hours after it was opened, however busy, and ends for everyone as a host's close does, with `closed {reason: "expired"}`. Failed rejoins count with failed joins. The page keeps the token in memory and in the tab's session storage, so a reload rejoins too.

The relay replays the session's update log to every newcomer, as one item on the connection's send queue that the writer sends entry by entry as the connection takes them, so a log longer than the queue never drops a newcomer. A replay is bounded to a minute, and a close the relay asks for during one stops it at the next entry and is written at once. A connection the relay closes leaves its session at once but stays counted in the connection caps until its socket has ended, which the transport ensures within 5 seconds of the close, so a close waiting behind a replay frees no place for another socket (final review A). It then:
- forwards Yjs updates only from team members. A guest's binary updates are refused, which is how guests stay read-only on the text.
- forwards a guest's `suggest` message to the team members, stamped with the guest's display name.
- broadcasts the roster of verified identities.

Limits bound all of this: message size (256 KiB), an awareness message's size (8 KiB); log bytes, each entry charged 64 bytes beyond its own; log entries (50,000); participants per session; sessions per relay, an expired session holding none while it waits for the sweep (final review G); idle time; a session's life (12 hours); connections (256 per relay and 16 per address, past which the upgrade is refused with 503); and 30 seconds for a connection to host, join or rejoin before it is closed, its send queue held to 64 items until then. Each connection's send queue holds at most 1,024 items and 8 MiB of frames; a log replay is charged no bytes, its entries being the session's own. A connection past either is dropped at once, its queue with it, so a page that stops reading holds nothing more (security review 2, finding 1). A guest's suggestions are bounded in size (70 KiB each), per participant on their seat, so a rejoin does not reset them (100 suggestions and 256 KiB; final review B), and in total (1 MiB per session, the backstop for a guest who joins anew by code), since a member's page records each in the document and so in the log (finding 2).

Rates bound it too: each connection's updates (60 a second, in bursts of 200), awareness messages (20 a second, in bursts of 40, and 32 KiB a second, in bursts of 64 KiB, below the session's rate; final review C) and suggestions (1 a second, in bursts of 5), each session's awareness bytes from every participant together (128 KiB a second, in bursts of 512 KiB), its sign-in attempts (5), the sessions one address opens (3 in 10 minutes), and the sign-in exchanges at `/auth/token` and device sign-ins started at `/auth/device` from one address (10 in 10 minutes together, keyed as wrong codes are, then 429 with `Retry-After` before GitHub is asked; finding 10). A device sign-in's poll asks GitHub no faster than GitHub's interval, whatever the page sends. A message over its rate is dropped with a `rate-limited` error naming its `kind` (`update`, `awareness`, `host` or `suggest`), and the connection stays open. An update or awareness frame over its connection's rate is judged before the relay's lock is taken and answered once a second at most, the drops between silent (final review E). Every refusal of a `suggest` names kind `suggest`, and a suggest dropped past the message budget below is answered too, once a second, so a guest's page always learns its suggestion went; the page also gives one up after 15 seconds without an answer (final review F). On a drop: the page resends recent updates, tells a guest to send a dropped suggestion again, and answers a dropped cursor with nothing. Every control message, and every frame the relay refuses, also spends one budget per connection (50 a second, in bursts of 100): past it, the message is dropped unanswered before the relay's lock is taken, and a connection refused 1,000 times is closed (finding 7). An accepted update or awareness message spends nothing of it, and a rate limit's own refusal does not count as one. Origins are checked against an allowlist: abcdesign.app and loopback pages by default, or exactly the origins `--origin` names, or, without `--origin`, those `ABCD_SESSION_ORIGINS` names, which replace them.

**Sign-in.** The relay offers one of two sign-ins through the team's GitHub App, chosen by how it is set up (adr-2610161400542916):
- **Web flow, on a cloud relay** (client id and secret). The page builds the GitHub authorize URL from the relay's public client id, with PKCE and a state value it keeps in session storage. The redirect comes back to the page. The page posts the code and the PKCE verifier to `/auth/token`, and the relay adds the secret and does the exchange: the token-mediating backend of RFC 10017 (OAuth 2.0 for Browser-Based Applications). GitHub's web flow still needs the secret for a GitHub App, PKCE or not.
- **Device flow, on a local session** (client id alone, and only on the page the relay serves itself; `abcd session serve` never offers it, and the page on any other origin takes the web flow alone). The page posts to `/auth/device`. The relay asks `https://github.com/login/device/code` for a device code, keeps it, and answers with a handle, the user code, GitHub's verification address (only `https://github.com/login/device` is shown) and the timings. The page shows the code in large type, with a button that opens github.com/login/device in a new tab and one that copies the code. It polls `/auth/device/token` with the handle, and tells `/auth/device/cancel` when the person stops the sign-in or leaves the page, which frees its place at once; the relay asks GitHub no faster than its interval, adding five seconds on `slow_down`, and returns the token once the person confirms. GitHub's own words name the app and the account on the confirming page. A device code expires after 15 minutes. GitHub's device endpoints are not called from the browser, because GitHub documents no CORS for them.

Either way the token stays in the page's memory, and the same `verify` admits it. No token is ever pasted: the page has no field for one, and the relay takes only a GitHub App user access token. A relay with no client id offers no sign-in. The page then says, in plain words, that sign-in is not set up for this project yet and to ask the technical facilitator. A guest can still join.

**Why the cloud keeps the web flow** (state-of-the-art grounds, adr-2610161400542916). The device flow needs no secret ("The `client_secret` is not needed for the device flow", https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps), but:
- it is open to remote phishing (RFC 8628 §5.4) with the app's public client id;
- GitHub limits it to 50 code submissions an hour per app;
- it sends every member to a second tab to type a code.

The secret already sits on the team's own relay. On the host's own computer, where the product thinker ruled out a secret, the device flow is the one that fits. **Tailscale Serve's identity headers are not read:** a Tailscale login says nothing about write access, and a local program could write them (adr-2610032150581128).

**Where the page finds its relay.** The page never shows a relay address, never asks for one, and never asks the person for technical input. In order:
1. A page the relay serves itself (a local session) carries `<meta name="abcd-session-relay" content="self">`, and takes its own origin as the relay. The page and the relay are one server, so the relay list does not apply.
2. On abcdesign.app, the page reads the repository's relay list (below). The relay is the one the invite link's fragment names, when the list names it; with no relay in the link, the first relay the list names.
3. Where the list cannot be read without signing in (a private repository), the relay is the one the link names, or one the person confirmed earlier for this repository. Before any sign-in, name or suggestion reaches it, the page asks one plain question, naming the repository and no address: "This page can't check this invite against owner/repo's settings. Only continue if someone on the owner/repo team sent you this link." A confirmed relay is remembered for that repository in the browser.
4. With no relay from any of these, the page says that live sessions are not set up for this project yet, and to ask the technical facilitator. It offers nothing to type.

A link's fragment can name any relay, so the repository is the page's trust anchor for relays (security review F1). Before any sign-in, token or display name reaches a relay, the page reads the repository's committed `.abcd/relay/relays.json` (`{"relays": ["https://…"]}`, https origins, other keys ignored) from `api.github.com` on the default branch, without a token for a public repository, and refuses a relay the file does not name: no `/config`, no sign-in, no `verify`, no join and no connection. The page then says the invite link does not lead to this project's own session and to ask the host for a new one, showing no address. Only GitHub's 404 means there is no list the page can see. A 403, a 429, a 5xx or a network error means the list could not be checked: the relay is refused, with "the page could not reach GitHub to check this project's settings; try again in a moment" and no confirmation offered, and a Try again button reads the list again (security review 2, finding 5). Once a token exists, the page asks `api.github.com` alone for the file again, with the token; a file that lists relays without this one refuses `verify` and forgets the confirmation.

The invite link carries the relay in its fragment only when the guest's page could not find the same one itself: never for a local session, and on abcdesign.app only when the relay is not the first the list names. The invite card shows the QR code, the join code and a Copy invite link button; the link itself is not printed on the page.

Sign-in goes through a GitHub App only. **Decision (facilitator, on security review 2, finding 3, on state-of-the-art grounds):** a link can open the page for a stranger's repository, whose own `relays.json` names the stranger's relay, so the list alone cannot stop a relay collecting tokens; what bounds the harm is the token. A GitHub App's user token reaches only the repositories the app is installed on, within its fine-grained permissions (Contents and Pull requests read and write, Metadata read), and GitHub recommends Apps over OAuth apps (https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps), while an OAuth app's `scope=repo` reaches every repository the person can. So the authorize URL never carries a scope, the oauth-app choice is gone, and a relay whose `/config` says anything other than `github-app` gets no sign-in button. The page names the repository prominently beside the sign-in ("You are signing in to work on owner/repo"), and the confirmation for a relay the list cannot speak for opens with the repository's name.

**Frames.** A static host such as GitHub Pages cannot send `frame-ancestors` or `X-Frame-Options`, and a `<meta>` Content-Security-Policy cannot carry `frame-ancestors`. That is accepted (security review 2, finding 11b), with mitigations: the page refuses to run in a frame (`window.top !== window.self`) and shows only a link that opens it in its own tab, so no relay confirmation, sign-in or session is ever drawn where another page could overlay it; the sign-in itself happens on github.com, which refuses to be framed.

**The page.** abcd generates the page itself. `abcd session page --out <dir>` writes plain files from assets embedded in the binary:
- one HTML file, served for every path through GitHub Pages' 404 fallback, which is how `/<owner>/<repo>/brief/` resolves on a static host (a local session's relay serves the same files itself; see Local session)
- the page script
- the stylesheet
- `collab.js`, a vendored bundle

`collab.js` holds Yjs 13.6.33, y-codemirror.next 0.3.6, y-protocols 1.0.7, CodeMirror 6 with Markdown, and qrcode-generator 2.0.4, all MIT. It was built outside the repository with esbuild 0.28.2. Every package version is at least two weeks old. Its provenance (each package, version and npm integrity hash, the entry module and the bundle's SHA-256) is committed beside it, and a Go test fails if the bytes drift. No Node toolchain enters the repository, consistent with adr-47.

The host's page loads the brief through the GitHub API with the host's token. It records the default branch's commit as the session's base and seeds one Y.Text per brief file exactly once, before any invite. It then shows a QR code that encodes the invite link: the page URL with the code, and the relay only where the link needs it, in the URL fragment, so neither reaches GitHub's servers.

**The start screen** is in plain words, for product thinkers and domain experts. It names the project, then offers:
- with a code in the link, "Join the live session": "Sign in with GitHub" for the project's team, who edit directly, and a name field with "Join as a guest" for everyone else, who suggest changes;
- without a code, "Start a live session": "Sign in with GitHub", then the choice of how the work goes back and "Start the session"; and a smaller "Have a code?" form for a code given by voice.

The words "relay", "token" and "personal access token" appear nowhere on the page. Once a session is joined or started, the page opens the brief's press release (the brief file whose name holds `press-release`, or else the first file).

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

**Local session.** `abcd session start` is the host's one command for a session on their own computer, for a project whose brief should not leave the team's tailnet:
- It reads this checkout's origin repository on github.com, refusing a checkout without one, and serves sessions for that repository only.
- It reads the GitHub App's public client id from `--client-id`, from `ABCD_SESSION_GITHUB_CLIENT_ID`, or from `.abcd/relay/github-app.json` (`{"client_id": "…"}`, which repository setup writes). Without one, it refuses and says that sign-in is not set up for this project. It never reads or holds a secret, so it offers the device flow.
- It asks Tailscale, through the read-only adapter, for this computer's tailnet name (`tailscale status`), and refuses when Tailscale is not running. It changes nothing in Tailscale.
- It listens on loopback (`127.0.0.1:8787`, or `--port`) and admits exactly `https://<tailnet name>` and its own loopback address, as page origins and as `Host` headers. Any other host gets 421 before anything is read. That is the DNS-rebinding guard adr-2610032150581128 asks for.
- It serves the page's embedded files from a fixed route table: `/session.js`, `/session.css`, `/vendor/collab.js`, `/vendor/LICENSES.txt`, and the HTML for `/<owner>/<repo>/brief/` of the repository it serves, carrying the `abcd-session-relay` marker. Nothing is read from a requested path. The page's responses carry `frame-ancestors 'none'` and `X-Frame-Options: DENY`.
- It prints the one Tailscale step (`tailscale serve --bg <port>`, which Tailscale keeps across restarts), the link to open (`https://<tailnet name>/<owner>/<repo>/brief/`), and that people outside the tailnet can join as guests once the host shares the computer with them in Tailscale. It runs until interrupted, and every session ends with it.

Every security property of `serve` holds for it unchanged: the limits, the rejoin tokens, suggestion attribution, the origin checks and the `ghu_`-only `verify`.

**Setup (repository setup, a later lane).** Setting up live sessions belongs to repository setup (`abcd ahoy`), built by a separate lane after this one. That lane builds:
1. **The first question:** "Set up live brief sessions?" with Yes, No and Later. Later is asked again at the next setup run, and when someone runs `abcd session start` or `blueprint` on a project not yet set up. No is recorded and never asked again, but the setup step stays runnable by hand.
2. **The second question,** after a Yes: "Where may sessions run?" with Local, Cloud and Both.
3. **The team's GitHub App,** created through GitHub's App manifest flow. That is a POST form to `https://github.com/settings/apps/new`, or `https://github.com/organizations/<org>/settings/apps/new` for an organisation, carrying a pre-filled manifest:
   - the app's name;
   - the callback URL, for the cloud: the page's own address (`https://abcdesign.app/<owner>/<repo>/brief/`). An app already recorded is kept; where the cloud is allowed, setup then names this callback URL to add on the app's settings page, since an app recorded for local sessions alone has none;
   - permissions: Contents and Pull requests read and write, Metadata read;
   - no webhook;
   - `public: false`.

   The conversion (`POST /app-manifests/{code}/conversions`, within the hour) returns the client id, the secret and a private key. abcd keeps only the client id, in `.abcd/relay/github-app.json`. The private key, the webhook secret and the client secret are discarded unwritten and never shown: a secret printed in a terminal can end up in a transcript. For the cloud, the person generates a client secret on the app's settings page and pastes it straight into Render's form (step 5), so no secret passes through abcd. That is one step more than the product thinker's two, and the lane says so when it asks. The person then installs the app on the repository: one more click, on the page GitHub opens.
4. **Device flow for Local.** A manifest cannot switch on the device flow, so for Local or Both the person ticks "Enable Device Flow" in the app's settings, on the page abcd opens. abcd says why the tick is needed: anyone holding the public client id can start a device sign-in. It is left off for Cloud alone.
5. **Deploying for Cloud.** The relay is deployed with Render's Deploy button (`https://render.com/deploy?repo=<repository URL>`). Its Blueprint declares the secret as `sync: false`, so Render's own form asks for it at creation, and abcd never sees it again; the client id, public, is written into it as a plain value. **Settled (facilitator, on state-of-the-art grounds, 2026-10-16):** the button takes only the repository's URL, with an optional `/tree/<branch>`, and Render documents no parameter naming another Blueprint path (https://render.com/docs/deploy-to-render); Render reads the Blueprint at the repository's root, and only its New Blueprint form takes a custom Blueprint Path (https://render.com/docs/infrastructure-as-code). So `blueprint` writes `render.yaml` at the root, the Dockerfile staying under `.abcd/relay/`. A root `render.yaml` the team wrote is never touched: the Blueprint then goes to `.abcd/relay/render.yaml`, deployed through the New Blueprint form with that Blueprint Path.
6. **The relay's address,** recorded by abcd itself once Render has given it (`blueprint --relay-origin`, which writes `.abcd/relay/relays.json`). abcd reads it from Render's answer, or the person pastes it into the terminal: never into the page.

abcd never runs a server that holds project content or a key. A local session's GitHub App is the team's own, registered once at setup as above: the product thinker ruled on 2026-10-16 that each team registers its own app and abcd owns no shared one (itd-2610091016262335's Decisions).

The setup step is `abcd session setup`, runnable by hand whatever the first question's answer. The answer is `sessions.live` in `.abcd/config.json` (`local`, `cloud`, `both`, or `none` for a no), which `abcd ahoy install --live-sessions` also sets. `setup` runs the manifest flow through a page on this computer's loopback address: the page posts the manifest to GitHub, and GitHub's redirect returns to that address with the code, which `setup` converts, reading the client id, the app's slug and its owner and nothing else.

abcdesign.app is published from a repository of its own, intentdriven/abcdesign, created on 2026-10-15 at the product thinker's go. abcd's own GitHub Pages site is planned to become abcd's record (itd-2610142145236613, adr-2610142147242544), and a repository has one Pages site. That repository holds no code: its one workflow downloads the latest abcd release daily or on demand, verifies its checksum and build provenance, runs `abcd session page`, and deploys to its GitHub Pages under the abcdesign.app domain, using only the run's own token. A release without `abcd session page` publishes nothing. Its main branch takes changes only through a pull request, its Actions are limited to GitHub's own, and its workflow token is read-only by default. Pointing abcdesign.app's DNS at GitHub Pages and verifying the domain for the organisation are the product thinker's steps, with a token scoped to that zone.

## Footprint

- packages: internal/core/session (new), internal/surface/session (new), internal/surface/cli, commands/session.md, docs/, internal/core/ahoy (step 6)
- tests: internal/core/session, internal/surface/session, internal/surface/cli, evals/ smoke (smoke_session_test.go runs `abcd session page --out`)

## Steps

1. The relay: session rooms, join codes, roles, update log, suggestions, sign-in exchange and limits, behind `abcd session serve`, with the WebSocket library github.com/coder/websocket v1.8.15 (ISC, no dependencies)
   - packages: internal/core/session, internal/surface/session, internal/surface/cli, commands/session.md
   - tests: internal/core/session, internal/surface/session
   - landed: cf20d3ded
2. The relay blueprint and the Tailscale route: `abcd session blueprint` writes `.abcd/relay/render.yaml` and `.abcd/relay/Dockerfile`, pinned to a released binary and its checksum
   - packages: internal/core/session, internal/surface/cli, docs/
   - tests: internal/core/session, internal/surface/cli
   - landed: a2ab10f7a
3. The page: `abcd session page --out` emits the static page and the vendored, provenance-pinned editor bundle; joining by QR or code, live editing, presence, guest suggestions and approvals, and credit
   - packages: internal/surface/session/web, internal/surface/cli
   - tests: internal/surface/session
   - landed: f835b6f38
4. Back to GitHub: issue links under the measured length limit, and one pull request with co-author, Suggested-by and Approved-by lines
   - packages: internal/surface/session/web
   - tests: internal/surface/session
   - landed: a30ec9432
5. The page asks nobody for a relay address or a token (adr-2610161400542916, iss-2610161332024554): the page finds its relay, opens onto the press release in plain words, and signs in through the web flow on a cloud relay or the device flow on a local session; `verify` takes only a GitHub App user access token; `abcd session start` runs a local session that serves its own page
   - packages: internal/surface/session, internal/surface/session/web, internal/surface/cli, commands/session.md, docs/
   - tests: internal/surface/session, internal/surface/session/web, internal/surface/cli
   - landed: 9baf9b93c
6. Setting up live sessions at repository setup: the two questions, the GitHub App manifest, the device-flow switch for local sessions, Render's Deploy button and the recorded relay address (a later lane)
   - packages: internal/core/ahoy, internal/surface/cli, commands/ahoy.md, docs/
   - tests: internal/core/ahoy, internal/surface/cli
