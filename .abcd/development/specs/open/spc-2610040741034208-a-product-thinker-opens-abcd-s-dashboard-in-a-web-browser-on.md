---
id: spc-2610040741034208
slug: a-product-thinker-opens-abcd-s-dashboard-in-a-web-browser-on
intent: itd-2610032150577708
origin: researcher-authored
production_mode: hand-written
---
# a-product-thinker-opens-abcd-s-dashboard-in-a-web-browser-on

## Summary

This spec delivers
[itd-2610032150577708](../../intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard-in-a-web-browser-on.md)
(the product thinker's dashboard, in a browser through Tailscale) under its
standing rule,
[adr-2610032150581128](../../decisions/adrs/2610032150581128-abcd-is-reachable-on-a-network-only-through-the-dashboard.md).
The settled records are designed to here, never reopened: the intent's
decisions 1 to 20, its eight product criteria (P1 to P8 below, in the
intent's order) and seven technical checks (T1 to T7), its three scope
conditions, and the ADR's Decision with its interview section and its
requirements. One finding of this design does reopen T1's wording, and it is
put back to the product thinker rather than decided here (decision D1).

Six things land, one step each. A `dashboard` verb starts and stops one
server on this computer, reachable only from devices on the person's own
Tailscale network, which it identifies by asking Tailscale itself who each
connecting device is. The server draws a summary page (what waits on you,
what is being built now, the next few items) and one page per record, from
the record site's one renderer, in process, with no site build. The brief
gets one reading page per chapter, headed with when it changed and when the
viewer last confirmed it, and how many intents and decisions rely on it.
The viewer can select words of a chapter and leave a note, or tap "Still
right"; both are held in the checkout's local tier, anchored so that a note
survives rewording and is never lost. Everything touched is earmarked for
the technical facilitator: the next session says how many items wait, a
review verb hands them back one at a time, the facilitator's proposal comes
back to the dashboard and to the product thinker's next session, and the
brief changes only when the product thinker confirms.

## Scope

In:

- A new `dashboard` verb on the CLI and the plugin surface: `start`, `stop`,
  `status` (the bare verb), `review` (the facilitator) and `confirm` (the
  product thinker), and a hidden `serve` that only `start` launches.
- `internal/surface/dashboard` (new): the one package that opens a listener;
  the listener, the connection gate, the HTTP server and its pages.
- `internal/adapter/tailscale` (new): the `tailscale` command as an adapter,
  for this computer's addresses and name, and the lookup of
  a connecting device.
- `internal/core/dashboard` (new, transport-agnostic): the shared snapshot,
  the summary model, the brief chapter table, the note and stamp store, the
  anchoring and re-attachment, and the review queue's state transitions.
- `internal/core/site`: an exported in-process render of one document
  through the existing `Renderer` (`RenderDocument`), and the brief
  citation scan (`ScanBriefCitations`) beside `scanMentions`.
- `internal/surface/cli`: the verb, the session-start greeting line and the
  board's facilitator row for waiting items, both counts only, on the inbox's
  pattern (`inboxGreeting` and `boardInbox` in `report.go`).
- Pages and record: `commands/dashboard.md`, `docs/how-to/dashboard.md`, the
  regenerated `docs/reference/cli/commands.md`, a brief surface chapter
  `04-surfaces/36-dashboard.md`, invariant 7's inbound clause in
  `02-constraints/03-invariants.md`, a glossary entry that keeps "the
  dashboard" apart from "the roadmap dashboard", and the ADR moved to
  accepted with its interview paragraph amended as D1 settles.

Out:

- The home network without Tailscale, with its one-time code, its seven-day
  device lifetime, its idle switch-off and its Wi-Fi warning: the draft
  itd-2610040740108331 (decision 9). The ADR's requirements that govern only
  that draft are mapped below to it, not built here.
- Acting from the dashboard (rewriting the brief, approving intents,
  answering abcd's questions, confirming a proposal): itd-2610040740122709.
  The only writes the page makes are notes and "Still right" (decision 11).
- The facilitator's view and other team members: itd-2610040740135705.
- Any project but abcd's own record (decision 13, scope condition 3).
- `site.Build`, `site setup` and `ui.json`: the dashboard neither runs the
  build nor reads or writes the composition (the record review's finding 4).
- A published or static copy of any page (itd-139 is superseded, decision 2).

## Design

### How the dashboard is reached, and how a device is identified

The interview settled that the first version is reached through Tailscale
only, that a device on the person's own Tailscale network is let in already,
and that the identity comes from Tailscale's own lookup of the connecting
device, never from a header a local program could write (decisions 4 and 9;
T2). T1 adds the means: loopback only, published by Tailscale Serve.

Reading Tailscale's source for this design (`ipn/ipnlocal/serve.go`,
`proxymap/proxymap.go`, main branch, read 2026-10-04) shows the two cannot
both hold. Serve's reverse proxy reaches a loopback backend from tailscaled
itself, and it does not register that connection in the table tailscaled
keeps for proxied connections (the code carries a TODO saying so), so a
lookup of the backend's peer address names no device. Behind Serve the only
statement of who is connecting is the `Tailscale-User-Login` header, which
any program on the computer can send to a loopback port.

The options, each judged against T1, T2 and the rule that a new dependency
needs the person's sign-off:

1. **Serve to loopback, trusting Serve's identity headers.** Meets T1; fails
   T2 by construction. Rejected.
2. **Serve to a Unix socket readable only by the person.** Close to T1's
   intent, since nothing on the network stack listens; but the identity is
   still a header, so T2 fails for any program the person runs, and the
   sandboxed Tailscale app on macOS has not been shown able to reach a
   socket in the person's folders. Rejected.
3. **`tsnet`, an embedded Tailscale node.** Meets T2 strongly; but it is a
   large new dependency needing sign-off, it adds a second device to the
   tailnet that someone must authorise, and its listener is not loopback
   either. Kept as the fallback if option 4 fails its first-step check.
4. **TLS on this computer's own Tailscale addresses, with Tailscale's
   certificate, and Tailscale's lookup on each connecting address.** The
   listener binds only the computer's tailnet addresses (its 100.64.0.0/10
   and fd7a:115c:a1e0::/48 addresses), serves HTTPS with the certificate
   `tailscale cert` issues for the computer's own name, and asks `tailscale
   whois` about each connection's real peer address, which the tailnet has
   already authenticated. No header is trusted, no Serve or Funnel
   configuration is made, and the standard library suffices (`crypto/tls`,
   `os/exec`).

- **D1 (facilitator, decided; the product thinker confirmed T1's rewording on 2026-10-04, the intent's decision 21):**
  option 4. It is the only standard-library route that meets T2 and the
  ADR's "never from a header", and it keeps T1's ends: nothing answers off
  the tailnet, Funnel is never involved, one package alone listens. It
  changes T1's means, which the product thinker accepted, so step 1 does not
  land until the product thinker confirms T1 reworded as: "Given the
  dashboard runs, when its listeners are enumerated, then it listens on this
  computer's own Tailscale addresses only, over HTTPS with Tailscale's
  certificate, never through Funnel, and a test proves one package alone
  opens a listener." The ADR's interview paragraph is amended to match in
  step 1. Should the product thinker prefer `tsnet` (option 3), the
  dependency is the sign-off and step 1 changes its listener alone.

- **D1a (the product thinker, 2026-10-04, the intent's decision 23): no
  certificate in the first version.** Told that `tailscale cert` publishes
  the computer's tailnet name in the public certificate-transparency logs for
  good, and that Tailscale already encrypts and authenticates every
  connection, the product thinker chose plain HTTP on the computer's own
  Tailscale addresses. Option 4 holds with its TLS removed: the listener binds
  the tailnet addresses only and serves HTTP; the gate and the lookup (D2) are
  unchanged; D4 is retired. The browser labels the page "Not secure", and
  writes are judged by `http.CrossOriginProtection`'s Origin-against-Host
  check (no `Sec-Fetch-Site` over plain HTTP). A certificate stays possible
  later, with the product thinker's consent to the publication.

- **D2 (facilitator, decided): the gate is at the connection, before any byte is sent.**
  On accept, the server checks that the peer address is a tailnet address
  and that the lookup names an untagged node; otherwise it closes the
  connection having sent no byte. So a device off the tailnet, or anything
  reaching the address some other way, gets no answer at all, which is P2
  read literally. Lookups are cached by peer address for 30 seconds and at
  most four run at once; a connection waiting on one counts against the
  connection cap. A tagged node is refused: a tag names a machine, not a
  person, and decision 4 lets in the person's devices. Every untagged node
  the lookup names is let in, whoever's it is, as decision 4's accepted cost
  says.

- **D3 (facilitator, decided): the `tailscale` command is the adapter.**
  abcd does not speak Tailscale's local API itself: how to reach it differs
  between the macOS app, the macOS standalone build and Linux, and the
  command already knows. The command is resolved once at `start`, from
  `PATH` and then the macOS app's bundled command, and refused when it sits
  in a world-writable folder or inside the working tree, the conditions the
  plugin hooks already refuse for `abcd` itself. Step 1 records the
  Tailscale version floor its probes established, with the source beside
  the constant.

- **D4 (retired by D1a; kept as written for the later certificate): the certificate is held in memory only.**
  `start` runs `tailscale cert --cert-file - --key-file - <name>` and parses
  both from its output; no key is written to disk. The server fetches it
  again when it is within 14 days of expiry. As with Serve, issuing a
  certificate publishes the computer's tailnet name in the public
  certificate-transparency logs; the docs page says so, and the name is all
  that is published.

Step 1 opens with a dated receipt in the local tier, taken against a real
tailnet before any code: the lookup names a phone's node by its address,
and fails for a non-tailnet address and for a loopback one, on macOS (app)
and Linux. No certificate is requested (D1a). If the lookup does not behave so, step 1 stops and the choice goes back to the
product thinker with option 3.

### The `dashboard` verb

- `abcd dashboard start [--port N]`: refuses outside a managed checkout,
  when a dashboard already runs on this computer (one per computer, D5),
  when Tailscale is not running, when the port is one Tailscale's own Serve
  or Funnel configuration names. It
  launches the hidden `serve` as a detached child in its own process group,
  waits on an inherited pipe for the child to report it is listening, then
  fetches its own address through the tailnet with a one-time value only
  this run knows, and only then prints one line:
  "abcd dashboard: open http://<name>:<port> on a device on your Tailscale
  network; anyone on that network can open it. `abcd dashboard stop` stops
  it." If the self-fetch fails, it stops the child and says that the
  address could not be reached from this computer. The default port is 8080.
- `abcd dashboard stop`: reads the run file, checks that its process is the
  one `start` launched (process id, start time and executable, re-read just
  before the signal), sends it SIGTERM, waits for the listener to close, and
  removes the run file. It never signals by name or pattern.
- `abcd dashboard status [--json]` (and the bare verb): whether it runs,
  where, since when, and the devices that opened it in this run, each by its
  Tailscale device name and person.
- `abcd dashboard serve`: hidden; refuses unless its readiness pipe is open,
  so nothing but `start` starts the server (P8, "never starts by itself").
  abcd installs no login item, launch agent or service unit.
- `review` and `confirm` are described under the handback below.

- **D5 (facilitator, decided): one dashboard per computer, as a detached
  child, with its run file under abcd's home.** `start` returning lets the
  person close the Terminal, which "running until stopped" (decision 12)
  needs. The run file and the devices-seen list live at
  `~/.abcd/dashboard/` through `internal/abcdhome`, as the ADR places such
  state; they name no project content.
- **D6 (facilitator, decided): no device list of abcd's own on the
  tailnet.** A device is let in by being on the person's Tailscale (decision
  4), so it is shut out the same way, in Tailscale; `status` lists who
  opened the dashboard so the person can see whom to remove, and the docs
  page says how. Revocation from the computer (decision 5) belongs to the
  home-network draft, where abcd itself lets devices in.

### The server

One `http.Server` on the gated listener, in `internal/surface/dashboard`:

- Host check first: a request whose `Host` is not the computer's tailnet
  name (with or without the port) is answered 421 before any handler runs,
  which defeats a page that rebinds a name to this address.
- Limits: `ReadHeaderTimeout` 5 s, `ReadTimeout` 15 s, `WriteTimeout` 30 s
  (lifted per stream through `http.ResponseController` for live updates
  alone), `IdleTimeout` 60 s, `MaxHeaderBytes` 16 KiB, request bodies capped
  at 64 KiB with `http.MaxBytesReader`, at most 32 open connections.
- Every response carries `Content-Security-Policy: default-src 'self';
  script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self';
  object-src 'none'; base-uri 'none'; form-action 'self';
  frame-ancestors 'none'`, `X-Content-Type-Options: nosniff`,
  `Referrer-Policy: no-referrer`, `Cache-Control: no-store` and
  `Cross-Origin-Opener-Policy: same-origin`.
- Routes are a fixed table. Records are found by id in the snapshot; brief
  chapters by key in the snapshot's chapter table; the stylesheet and the one
  script are embedded in the binary and served by fixed name. No handler
  joins a request path onto a folder.
- The only routes that accept anything but GET are `POST /notes` and
  `POST /still-right` (step 4).

- **D7 (facilitator, decided): identity before content.** The steps put the
  gate in step 1 and the first project content in step 2, the reverse of the
  outline the planning suggested (pages, then Tailscale), so that no build
  of the dashboard ever serves the record to an unidentified connection,
  including one from another account on the same computer.

### One snapshot for every viewer

`internal/core/dashboard` builds an immutable snapshot of the project, and
the server holds the current one behind an atomic pointer that every request
reads. One goroutine rebuilds it when a cheap fingerprint changes, polled
every two seconds: HEAD, and the sizes and modification times of the record
folders, the brief and the dashboard store. The snapshot holds the status
block (`statusblock.Read` with the loop's lane and peer readers, as the bare
board calls it, since this computer's own state is not published), the
record graph (`lint.LoadRecordGraph`, with the issue ledger included), the
file dates (`site.LoadHistory`, reloaded only when HEAD moves), the mode
state (`mode.ReadAt`), the brief chapter table with its citation counts, and
the dashboard items with their anchors re-attached.

- **D8 (facilitator, decided): live updates are one event stream per page,
  capped.** `GET /events` sends one `changed` event when the snapshot's
  generation moves, and the page reloads itself. At most 16 streams are open,
  four per device; one past the cap is refused with 503 and the page falls
  back to reloading when it returns to the foreground. A connection the gate
  refuses never reaches the snapshot.

### Pages

All prose is drawn by the record site's one renderer, in process. The site's
`Renderer` already renders a document body (`renderBody`, used by the record
pages and by `CheckRecordBody`); step 2 exports one entry,
`(*Renderer).RenderDocument(rel, content string) (string, error)`, which
strips the frontmatter and renders the body, and the dashboard calls it with
its own link and image hooks.

- **D9 (facilitator, decided): render in process, never through the build.**
  `site.Build` needs a composition only `site setup` plants, takes minutes,
  and rewrites `ui.json`; the dashboard needs one document at a time. The
  dashboard's link hook maps a link to a record file to that record's page,
  a link to a brief chapter to its chapter page, keeps an absolute `https:`
  link as a link, and draws any other repository path as plain text, since
  no file is served from a path. Its image hook draws the alt text: the CSP
  and the no-path rule leave no route for an image file, and the record's
  prose carries almost none. Its two interface strings (the copy button's
  labels) are the dashboard's own; it reads no `ui.json`.

The pages, laid out in one column for a phone, following the system's dark
setting, with tap targets of at least 24 px (WCAG 2.2 AA):

- **Summary (`/`)**, P1: "Waiting on you" first, then "Being built now",
  then the next three items, each a plain title linking to its page. No
  record id, command or lane word reaches the page: a golden test over the
  page's text refuses any record handle and any backticked word.
- **Item (`/item/<id>`)**, P3: for an intent, its announcement (the press
  release), its checklist of what done means (the acceptance criteria), its
  decisions, the problems linked to it (issues joined by a typed link or a
  body mention, each a title linking to its own page) and the work notes
  (its spec's body); its other sections follow under one disclosure. Any
  other record (an issue, a spec, a decision) is its whole body. Every id is
  checked against the record id validators and looked up in the snapshot;
  anything else is 404.
- **Brief (`/brief` and `/brief/<part>/<chapter>`)**, P4: see below.

- **D10 (facilitator, decided): what "waiting on you" holds.** The
  facilitator's proposals on the viewer's own notes that wait for the
  viewer's confirmation, each quoting the words and the proposal; and, when
  the mode state names the product thinker, one line saying abcd has a
  question for them that their next session asks. The question's text is
  not shown: it is not in the record, and the board's design review settled
  that the role is what can be read (itd-2610031214560142, "kept for the
  spec"). When nothing waits, the section says so in one line.

### The brief chapter page

The chapter table is built from the brief folder at snapshot time: each
numbered chapter in each numbered part, plus `00-meta.md`, keyed by part and
chapter file stem. Glossary entries are not chapters. Each chapter is one
scrolling page:

- Its heading line reads "changed <date> · last confirmed by you <date or
  never>". "Changed" is the day the last commit touched the file, or
  "changed today, not yet committed" when the working tree differs from
  HEAD. "Last confirmed by you" is the latest "Still right" stamp made by the
  person viewing.
- "Relied on by N intents and M decisions" sits under the heading, folded
  until tapped, opening onto their titles, each linking to its page.
  Disclosure is used only for these appendages (the citations, the notes
  list and the orphans); the chapter's own text is never folded.
- The chapter's notes are listed after its text, and the words they mark are
  highlighted in place by the CSS highlight API where the browser has it,
  without changing the text.

- **D11 (facilitator, decided): the citation count comes from a body scan,
  not from `record.json`.** The research summary placed the count in
  `record.json`'s mentions, but that pass reads only record handles (`adr`,
  `itd`, `iss`, `spc`), and no brief chapter is a record node. So
  `ScanBriefCitations` reads intent and decision bodies once each, through
  the same containment root and guarded read as `scanMentions`, and counts
  a record once per chapter for a link resolving to the chapter's file, or,
  for the invariants chapter, for the words "brief invariant <n>".

### Notes and "Still right"

Selecting words in a chapter offers "Note"; the form takes an optional
text, of at most 2,000 characters, and an optional kind, question (the
default), disagree or out of date (decision 14). A note with no text is a
mark. "Still right" is one button under the chapter heading.

The anchor is taken against the chapter's text as the server renders it
(its visible text with whitespace runs collapsed, the copy buttons left
out), so the server, not the browser, is the authority on where words are.
It holds the file and the commit (HEAD, and the SHA-256 of the chapter's
bytes as rendered, since the working tree may differ), the selected words
exactly (at most 1,000 characters), up to 32 characters before and after
them, and their start and end offsets. On save, the server checks the words
against its own text at those offsets, finds them by quote if the offsets
disagree, and refuses with "the chapter changed while you were reading;
reload" if it cannot find them at all.

On every snapshot each note is re-attached against the chapter as it is now,
in this order: by position, when the words at the stored offsets are the
words stored; by the exact words, the prefix and suffix choosing among
several matches and the nearest to the old position winning a tie; by
similar words, the best window of about the same length whose word-level
similarity is at least 0.75; otherwise it is an orphan. The stored anchor is
never rewritten; the attachment is computed, so reading writes nothing. An
orphan is listed under "Notes whose words are gone", its old words struck
through, with the note; nothing deletes a note.

- **D12 (facilitator, decided): anchoring as above.** It follows the
  annotation model the 2026-10-04 research recommends (a quote with context,
  a position, the file and the commit) with the order and the never-lose
  rule it found. The 0.75 threshold is a named constant with its fixture
  table, and it is the one number expected to move after use.
- **D13 (facilitator, decided): a note may have no text.** The criterion's
  "marked" is then a selection saved as a question with nothing written,
  which still goes to the facilitator.

The store is one JSON file per item, in the primary worktree's local tier at
`.abcd/.work.local/dashboard/items/<id>.json`, the primary worktree found as
the banlist already finds it (`banlist.PrimaryWorktreeRoot`). An item is a
note or a stamp, with its id (minted by `recordid.Minter` in a family of its
own, `dbn`, which is not a record family), the chapter, the anchor, the
person and device the lookup named, the time, its state (earmarked,
proposed, resolved), the proposal when there is one, and `resolved_by`. The
server only creates items, each with an exclusive create; it never rewrites
one. The verbs move an item between states under the store's lock with an
atomic write. At most 1,000 unresolved items are held; past that the page
says the facilitator has many items to read first.

- **D14 (facilitator, decided): the store is the primary worktree's local
  tier, and resolved items stay there.** The primary worktree, because the
  facilitator's session may run in a linked worktree of the same checkout,
  and the criterion places the items in the checkout's local tier. They do
  not graduate into a committed file: the notes are the product thinker's
  private words and abcd's repository is public; the public trace is the
  brief edit a confirmed proposal makes, committed as any other change.

### Writes

The two write routes accept only JSON (`application/json`, decoded with
`jsonstrict.Decode`), only from a connection the gate let in (D2), and only
through Go's `http.CrossOriginProtection`, which over plain HTTP judges by
the request's Origin against its Host (D1a). The text of a note is drawn as text,
escaped, everywhere it appears; it is never rendered as Markdown.

### The handback to the facilitator, and back

An item reaches the facilitator by two counts-only lines, on the inbox's
pattern:

- the session-start hook prints to the session (stdout), when any wait:
  "abcd: 2 dashboard notes wait for the facilitator's review and 1 proposal
  waits for the product thinker; `abcd dashboard review` walks them."
- the board's facilitator rows gain "dashboard: 2 to review, 1 to confirm —
  `abcd dashboard review`", beside its inbox row; the product thinker's view
  of the board carries no such line (itd-2610031214560142, decision 6).

`abcd dashboard review` hands back the oldest earmarked item, one at a time:
the chapter, the words as they now stand (or the old words, when it is an
orphan), the kind and the note, under an `untrusted:` line, as the inbox
frames a report. The facilitator answers with one of: propose a rewording
(`review <item> --propose-rewording <text>`), answer without a change
(`--answer <text>`), or later (`--later`, which leaves it queued and asked
again next session). A "Still right" stamp is acknowledged (`--acknowledge`)
and resolves, since the tap is itself the product thinker's confirmation. In
a Terminal the verb asks through the plain-Terminal question layout; with
`--json` the plugin page asks one question at a time.

A proposal then shows on the product thinker's dashboard under "Waiting on
you", and their next session's greeting counts it. `abcd dashboard confirm`
hands back the oldest proposal, quoting the words, the note and the
proposal; the product thinker answers yes, no or later. On yes to a
rewording, the verb replaces the anchored words in the chapter file, only
when they re-attach by position or exact words at that moment, and records
`resolved_by` (the role, the time and the answer); otherwise it refuses,
names why, and the item stays proposed. On yes to an answer, the item
resolves. On no, the item returns to the facilitator's queue with the
refusal in its history. The person commits the brief edit as any other
change.

- **D15 (facilitator, decided): counts only on the session channel.** A
  session-start hook's output enters the session's context, and the note
  text came in over a network page that anyone on the tailnet can reach; the
  hook's own design already keeps repository-derived words off that channel.
  The words travel only through the review verb, framed as data.
- **D16 (facilitator, decided): confirming happens in the product thinker's
  session, not on the dashboard,** because decisions 6 and 11 allow only
  notes and "Still right" from it; acting from the dashboard is
  itd-2610040740122709. Nothing in the brief changes before a yes, and the
  verb is the one thing that changes it then.

## Trust boundary

Each requirement of adr-2610032150581128, where it is enforced, and the test
that proves it. Tests named here are in `internal/surface/dashboard` unless
another package is given.

- **Listens only while started on purpose; never starts by itself.** The
  hidden `serve` refuses without `start`'s readiness pipe; nothing installs
  a login item. `TestOnlyStartStartsTheServer`,
  `TestStopLeavesNothingListening`, `TestStopChecksTheProcessBeforeSignalling`.
- **The dashboard is the only listener abcd has.** A source scan of every
  non-test Go file for `net.Listen`, `net.ListenTCP`, `net.ListenConfig`,
  `tls.Listen`, `http.ListenAndServe` and their kin allows one package.
  `TestOnlyTheDashboardOpensAListener` and `TestListenerScannerIsArmed`, the
  shape of `TestOnlyTheHomeResolverNamesTheHome`.
- **Only the people let in can open it** (on the tailnet, per decision 4,
  the identity from Tailscale's lookup, never a header): the connection gate
  (D2). `TestListensOnTailnetAddressesOnly`,
  `TestConnectionWithoutIdentityGetsNoBytes`, `TestTaggedNodeIsRefused`,
  `TestTailscaleHeadersAreIgnored` (a forged `Tailscale-User-Login` is
  refused, at the connection with a failing lookup and as a relay with a
  passing one, and reaches no route). Two cases decision 4 did not settle
  are refused until the product thinker decides (the facilitator's safe
  default, `.abcd/work/DECISIONS.md` 2026-10-05). A device shared into the
  tailnet from another account (the lookup's `Node.Sharer` set) belongs to
  someone else and is refused at the connection with no byte, like a tagged
  one: `TestASharedInNodeIsRefused`, and `TestWhoIsReportsASharer`
  (`internal/adapter/tailscale`). A request another of the person's nodes
  relays with its own Serve, or puts on the open internet with Funnel,
  comes from that node, which the lookup names as the person's; it is
  dropped with no byte, before the host check, when it carries any header
  only a proxy adds: the ones Tailscale's Serve and Funnel set
  (`X-Forwarded-Host` on every proxied request, `X-Forwarded-For`,
  `X-Forwarded-Proto`, `Tailscale-Funnel-Request`, `Tailscale-User-Login`,
  `Tailscale-User-Name`, `Tailscale-User-Profile-Pic`,
  `Tailscale-Headers-Info`, `Tailscale-App-Capabilities`; read from
  `ipn/ipnlocal/serve.go` at tailscale commit 9128778b6515 and Tailscale's
  Serve documentation, 2026-10-05), any other `Tailscale-` or
  `X-Forwarded-` header, `Forwarded` and `Via`:
  `TestARelayedRequestIsRefused`. **Residual:** a raw TCP relay (Serve's
  TCP forwarding; with the PROXY protocol turned on, Go's HTTP server
  answers the preamble 400 Bad Request and serves nothing) adds no header and reaches the
  dashboard as the relaying node itself, so it cannot be told apart from
  that node opening the dashboard and is let in like it. This computer itself, at one of the listening
  addresses, may make `start`'s single-use self-check and nothing else, and
  is refused before any lookup once it has answered (D7: another account on
  the same computer, or a Serve or Funnel configured after `start`):
  `TestThisComputerIsRefusedOnceTheSelfCheckAnswered`.
- **What the page may change**: notes and "Still right" only (decision 11).
  `TestOnlyTwoRoutesWrite` enumerates the route table.
- **Nothing leaves the person's devices**: no third-party script, font or
  tracker. `TestPagesLoadNothingFromOffTheHost` parses every page and the
  stylesheet for anything loaded and refuses an absolute or protocol-relative
  address; `TestSecurityHeadersOnEveryResponse` holds the CSP.
- **Listens on this computer alone unless an address is chosen; never every
  address; the start line names every address; abcd checks it can reach
  itself.** Under D1 the addresses are the computer's own tailnet addresses,
  never the unspecified address. `TestListensOnTailnetAddressesOnly`,
  `TestStartLineNamesAddressAndWhoCan`, `TestStartChecksItCanReachItself`.
- **Letting a device in is its own step, with a single-use link**: the
  home-network draft itd-2610040740108331; on the tailnet a device is let in
  by Tailscale (decision 4).
- **A device let in holds a session abcd can list and revoke; pages are
  never cached.** No-store on every response, in
  `TestSecurityHeadersOnEveryResponse`. Sessions and revocation from the
  computer are the home-network draft's (D6); `status` lists the devices
  seen, in `TestStatusListsDevicesSeen` (`internal/surface/cli`).
- **A device not let in sees nothing and its requests read nothing from the
  project.** The gate runs before TLS, and the snapshot is reached only from
  a handler. `TestConnectionWithoutIdentityGetsNoBytes` counts snapshot
  reads through a spy and expects none.
- **Only the expected host names are answered, before anything is read.**
  `TestUnexpectedHostIsRefusedBeforeAnyRead`.
- **Timeouts, size limits, a cap on live updates, one shared reading.**
  `TestServerLimits`, `TestLiveUpdateConnectionsAreCapped`,
  `TestOneSnapshotServesEveryViewer`.
- **Drawn by the one renderer from records looked up by id; no file served
  from a requested path; CSP self only; no framing.**
  `TestPagesRenderThroughTheSiteRenderer` (`RenderDocument` equals the
  record page's body for a fixture, in `internal/core/site`),
  `TestRecordsAreLookedUpByIDOnly` (traversal and encoded paths are 404 and
  open no file), `TestSecurityHeadersOnEveryResponse`.
- **Nothing is changed over plain HTTP.** Under D1a the dashboard serves
  plain HTTP, so the requirement reads: nothing is changed except by a
  connection the gate let in, through cross-origin protection.
  `TestWritesNeedTailnetIdentity` (step 1: no write route exists, and a write
  from a connection the lookup did not name gets no byte),
  `TestWritesNeedCrossOriginProtection` (step 4). `TestPlainHTTPIsNeverAnswered`
  was written for the TLS listener and is retired with D4.
- **On the local network: foreground, idle stop, Wi-Fi warning; running
  unattended only behind Serve.** The home-network draft. Under D1 nothing
  is published through Serve or Funnel at all; `start` refuses a port a
  Serve or Funnel configuration names, in `TestStartRefusesAServedPort`.
- **Pairing state lives under abcd's home.** The run file and the devices
  seen do, through `internal/abcdhome`, held by its existing boundary test.
- **Notes and stamps in the checkout's local tier until reviewed** (T7).
  `TestNotesLiveInTheLocalTier`, `TestServerNeverRewritesAnItem`
  (`internal/core/dashboard`).
- **The session channel carries counts only** (D15).
  `TestSessionStartGreetingCountsOnly` (`internal/surface/cli`) writes a
  note holding an instruction and asserts stdout carries none of its words.

## How each acceptance criterion is met

- **P1** (summary in plain words): steps 1 and 2.
  `TestSummaryShowsWaitingNowNextInPlainWords`, and the dated receipt from a
  phone on the tailnet in step 6.
- **P2** (off the tailnet nothing answers): step 1, D2.
  `TestConnectionWithoutIdentityGetsNoBytes`, and the receipt from a phone
  off the tailnet in step 6.
- **P3** (the whole private record behind an item): step 2.
  `TestItemPageShowsTheWholePrivateRecord`.
- **P4** (one chapter page with its dates and reliance, folded): step 3.
  `TestBriefChapterPageHeading`, `TestBriefCitationCounts`.
- **P5** (a note stays attached, or is shown struck through): step 4.
  `TestNoteReattaches`, `TestOrphanIsShownStruckThrough`.
- **P6** (touched items reach the facilitator's next session one at a time;
  the proposal comes back; the brief waits for a yes): step 5.
  `TestReviewHandsBackOneItemAtATime`, `TestProposalReachesDashboardAndSession`,
  `TestBriefUnchangedUntilConfirmed`.
- **P7** ("Still right" shows today and goes to review): step 4 for the
  date, step 5 for the review. `TestStillRightShowsToday`,
  `TestStillRightResolvesOnAcknowledgement`.
- **P8** (one start line; stop answers nothing; never starts by itself):
  step 1. `TestStartLineNamesAddressAndWhoCan`,
  `TestStopLeavesNothingListening`, `TestOnlyStartStartsTheServer`.
- **T1** (listeners; one package listens): step 1, as reworded under D1.
- **T2** (identity from Tailscale's lookup, never a header): step 1.
- **T3** (host names, timeouts, caps, one snapshot, capped live updates):
  step 1 for the first three, step 2 for the last two.
- **T4** (one renderer, by id, no path, CSP, nothing off the host): step 2.
- **T5** (writes only from a tailnet device the gate let in, with cross-origin protection; reworded by decision 23): step 4.
- **T6** (anchored by quote, offsets and commit; orphans kept): step 4.
- **T7** (in the local tier until reviewed): step 4.

## Decisions taken in this design

D1 to D16 above are the facilitator's, each with its reason beside it. D1
alone changes something the product thinker accepted (T1's means) and is
held for their word before step 1 lands; D16 restates decisions 6 and 11
rather than adding to them.

## Footprint

- packages: internal/surface/dashboard, internal/core/dashboard, internal/adapter/tailscale, internal/core/site, internal/surface/cli, commands/dashboard.md, docs/how-to, docs/reference, .abcd/development/brief/02-constraints, .abcd/development/brief/04-surfaces, .abcd/development/brief/glossary, .abcd/development/decisions/adrs
- tests: TestOnlyTheDashboardOpensAListener, TestListenerScannerIsArmed, TestListensOnTailnetAddressesOnly, TestConnectionWithoutIdentityGetsNoBytes, TestTaggedNodeIsRefused, TestTailscaleHeadersAreIgnored, TestASharedInNodeIsRefused, TestARelayedRequestIsRefused, TestWhoIsReportsASharer, TestThisComputerIsRefusedOnceTheSelfCheckAnswered, TestUnexpectedHostIsRefusedBeforeAnyRead, TestServerLimits, TestSecurityHeadersOnEveryResponse, TestStartLineNamesAddressAndWhoCan, TestStartChecksItCanReachItself, TestStartRefusesAServedPort, TestStopLeavesNothingListening, TestStopChecksTheProcessBeforeSignalling, TestOnlyStartStartsTheServer, TestStatusListsDevicesSeen, TestSummaryShowsWaitingNowNextInPlainWords, TestItemPageShowsTheWholePrivateRecord, TestRecordsAreLookedUpByIDOnly, TestPagesRenderThroughTheSiteRenderer, TestPagesLoadNothingFromOffTheHost, TestOneSnapshotServesEveryViewer, TestLiveUpdateConnectionsAreCapped, TestBriefChapterPageHeading, TestBriefCitationCounts, TestNoteReattaches, TestOrphanIsShownStruckThrough, TestOnlyTwoRoutesWrite, TestWritesNeedTailnetIdentity, TestWritesNeedCrossOriginProtection, TestNotesLiveInTheLocalTier, TestServerNeverRewritesAnItem, TestStillRightShowsToday, TestSessionStartGreetingCountsOnly, TestBoardRowCountsDashboardItems, TestReviewHandsBackOneItemAtATime, TestProposalReachesDashboardAndSession, TestBriefUnchangedUntilConfirmed, TestConfirmAppliesRewordingOnlyWhenWordsReattach, TestStillRightResolvesOnAcknowledgement; the surface parity test; dated receipts in the local tier; docs-lint and record-lint clean

## Steps

1. The verb and the gate: start, stop and status, the tailnet listener, the identity check and the hardening
   - criteria: P2, P8, T1 (as reworded under D1), T2, and T3's host names, timeouts and size caps; serves one fixed page naming the person let in and nothing of the project
   - held: until the product thinker confirms T1's rewording (D1); opens with the dated Tailscale receipt, and stops for the product thinker if it fails
   - security review: required before landing (listener, identity, process control)
   - record: adr-2610032150581128 moved to accepted with its interview paragraph amended to D1, and invariant 7 given its inbound clause, in the same change
   - packages: internal/surface/dashboard, internal/adapter/tailscale, internal/surface/cli, commands/dashboard.md, docs/reference, .abcd/development/decisions/adrs, .abcd/development/brief/02-constraints
   - tests: TestOnlyTheDashboardOpensAListener, TestListenerScannerIsArmed, TestListensOnTailnetAddressesOnly, TestConnectionWithoutIdentityGetsNoBytes, TestTaggedNodeIsRefused, TestTailscaleHeadersAreIgnored, TestASharedInNodeIsRefused, TestARelayedRequestIsRefused, TestWhoIsReportsASharer, TestThisComputerIsRefusedOnceTheSelfCheckAnswered, TestUnexpectedHostIsRefusedBeforeAnyRead, TestServerLimits, TestSecurityHeadersOnEveryResponse, TestWritesNeedTailnetIdentity, TestStartLineNamesAddressAndWhoCan, TestStartChecksItCanReachItself, TestStartRefusesAServedPort, TestStopLeavesNothingListening, TestStopChecksTheProcessBeforeSignalling, TestOnlyStartStartsTheServer, TestStatusListsDevicesSeen, each against a fake `tailscale` on PATH and watched fail first; the surface parity test with the new page; the command reference regenerated
2. The summary and item pages, from one snapshot through the site's renderer
   - criteria: P1, P3, T3's shared snapshot and capped live updates, T4
   - security review: required before landing (routing, rendering of record text, live-update streams)
   - packages: internal/core/dashboard, internal/core/site, internal/surface/dashboard
   - tests: TestSummaryShowsWaitingNowNextInPlainWords (golden, no record handle and no command word), TestItemPageShowsTheWholePrivateRecord, TestRecordsAreLookedUpByIDOnly, TestPagesRenderThroughTheSiteRenderer, TestPagesLoadNothingFromOffTheHost, TestOneSnapshotServesEveryViewer, TestLiveUpdateConnectionsAreCapped; the site's existing record-page tests unchanged
3. The brief chapter pages: one reading page each, with dates and reliance
   - criteria: P4 ("last confirmed by you" reads "never" until step 4)
   - security review: not required; it adds GET routes through step 2's lookup table and changes neither the gate nor the router
   - packages: internal/core/dashboard, internal/core/site, internal/surface/dashboard
   - tests: TestBriefChapterPageHeading (changed date, uncommitted wording, "never"), TestBriefCitationCounts (a path link, a "brief invariant" mention, a record counted once), the chapter table's keys refusing an unknown chapter
4. Notes and Still right: the anchored store in the local tier, and the two writes
   - criteria: P5, P7 (the date), T5, T6, T7
   - security review: required before landing (the only writes, and text from the network stored and drawn)
   - packages: internal/core/dashboard, internal/surface/dashboard
   - tests: TestNoteReattaches (position, exact words, similar words, orphan, as a table), TestOrphanIsShownStruckThrough, TestOnlyTwoRoutesWrite, TestWritesNeedTailnetIdentity, TestWritesNeedCrossOriginProtection, TestNotesLiveInTheLocalTier, TestServerNeverRewritesAnItem, TestStillRightShowsToday, a note holding markup drawn as text
5. The handback: the session greeting, the board row, review and confirm
   - criteria: P6, P7 (the review)
   - security review: required before landing (the session channel, network-written text reaching a session, and the one verb that edits the brief)
   - packages: internal/core/dashboard, internal/surface/cli, internal/surface/dashboard, commands/dashboard.md, docs/reference
   - tests: TestSessionStartGreetingCountsOnly, TestBoardRowCountsDashboardItems, TestReviewHandsBackOneItemAtATime (with the untrusted line), TestProposalReachesDashboardAndSession, TestBriefUnchangedUntilConfirmed, TestConfirmAppliesRewordingOnlyWhenWordsReattach, TestStillRightResolvesOnAcknowledgement; the page tests for commands/dashboard.md; the surface parity test
6. The docs page, the brief chapter, the receipts, and the close
   - criteria: the end-to-end receipts for P1, P2 and P5; the intent's close
   - packages: docs/how-to, .abcd/development/brief/04-surfaces, .abcd/development/brief/glossary, internal/core/dashboard (tests only)
   - tests: docs-lint and record-lint clean; dated receipts in the local tier from a phone on the tailnet (the summary, an item, a note saved and re-attached after a rewording) and from one off it (no answer); a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610040741034208` (the intent already declares `impact: additive`) with a `Delivers: itd-2610032150577708` trailer
