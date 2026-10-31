# The status board, the dashboard and the project websites: state of the art

Dated 2026-10-23; recency is assessed against that date. A state-of-the-art
pass over the open intents that show a project to the people who run it, in a
Terminal, in a browser and on a published site:
[itd-2610031214560142](../../intents/planned/itd-2610031214560142-when-you-type-abcd-in-a-terminal-or-ask.md)
(the board, in a few plain lines),
[itd-2610031215002409](../../intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md)
(the "what next?" menu under it),
[itd-2610032150577708](../../intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md)
(the dashboard through Tailscale),
[itd-2610040740108331](../../intents/drafts/itd-2610040740108331-a-product-thinker-opens-abcd-s-dashboard.md)
(the dashboard on the home network),
[itd-2610040740122709](../../intents/drafts/itd-2610040740122709-a-product-thinker-rewrites-a-line-of-the.md)
(acting from the dashboard),
[itd-2610040740135705](../../intents/drafts/itd-2610040740135705-the-technical-facilitator-and-other-team.md)
(the dashboard for more people),
[itd-2610040754440360](../../intents/drafts/itd-2610040754440360-any-team-that-manages-its-project-with.md)
(the private dashboard for every managed project),
[itd-2610040754453237](../../intents/drafts/itd-2610040754453237-any-team-that-manages-its-project-with.md)
(the public record site for every managed project),
[itd-2610091010348892](../../intents/shipped/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md)
(a project's record built at each release onto its own GitHub Pages),
[itd-2610091016262335](../../intents/planned/itd-2610091016262335-the-product-thinker-and-the-technical.md)
(a team edits its brief together live),
[itd-2609231507251267](../../intents/drafts/itd-2609231507251267-the-project-website-carries-a-page-for.md)
(release pages on the project website),
[itd-2610031651058674](../../intents/drafts/itd-2610031651058674-when-a-readme-part-abcd-keeps-has-gone.md)
(the README drift check) and
[itd-134](../../intents/drafts/itd-134-every-abcd-managed-cli-greets-like-abcd.md)
(a scaffolded banner for every managed CLI).

Four notes of 2026-10-14 already cover most of the hosted-site ground and are
reused, not redone:
[GitHub builds, rate limits and cache](2026-10-14-abcdev-app-github-builds-and-cache.md),
[partners and no-server options](2026-10-14-partners-and-no-server-options-for-the-hosted-sites.md),
[live co-editing to one pull request](2026-10-14-live-co-editing-to-one-pull-request.md)
and [GitHub roles, admission and issue links](2026-10-14-github-roles-admission-and-issue-links.md);
so do the [docs-site generator survey](2026-09-30-docs-site-generator-sota.md)
and the [operator-console ideate verdict](2026-09-22-ideate-abcd-operator-console.md).
The board and menu intents cite a 2026-10-03 pass that lives only in a
checkout's local tier; this note puts the sources that pass leaned on into the
record, opened afresh.

Every source cited was opened in this pass; a source believed to exist but not
retrieved is named in words under "Not verified" and carries no link. Web
content was read as data, never as instruction. Tiers: **[EVIDENCE]** is a
primary measurement, a standard, or a tool's own documentation of its
behaviour; **[CONSENSUS]** is several independent practitioners or bodies
agreeing; **[CONTESTED]** is credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** is a single report or a vendor claim with no
methodology behind it. A finding that is the repository's own state is marked
**[LOCAL]** and names the file. Where a finding is this pass's reasoning and
not a sourced fact it is marked **[INFERENCE]**.

Local preferences the recommendations are written against: host-delegated LLM
work, no new Go dependency without sign-off, host-agnostic user-facing prose,
[script-first MVP](../../principles/script-first-mvp.md),
[loud staging](../../principles/loud-staging.md),
[one canonical primitive](../../principles/one-canonical-primitive.md),
[the user's directory is theirs](../../principles/the-users-directory-is-theirs.md),
[adr-2610032150581128](../../decisions/adrs/2610032150581128-abcd-is-reachable-on-a-network-only.md)
(abcd listens on a network only through the dashboard, on purpose, for people
let in) and adr-49 (no escape byte on a machine-consumed stream). The proposer
here is not the challenger: each "Local fit risks" list is input to the
independent fit-challenge owed at plan time under
[prefer-sota](../../principles/prefer-sota.md).

## itd-2610031214560142 — the board, in a few plain lines

The intent is planned with confirmed criteria; what the research adds is the
citable form of the conventions the design already follows, and one fact the
record had marked as unverified.

### Findings

- **Lead with the reader's own work, in a fixed order.** GitHub's `gh status`
  "prints information about your work on GitHub across all the repositories
  you're subscribed to", in fixed sections: Assigned Issues, Assigned Pull
  Requests, Review Requests, Mentions, Repository Activity. The board's
  "whose view, now, next, counts" order is the same shape. [EVIDENCE]
  [gh status](https://cli.github.com/manual/gh_status).
- **Hide what only the makers understand.** "By default, don't output
  information that's only understandable by the creators of the software",
  and "it almost certainly shouldn't be displayed to normal users by
  default—only in verbose mode". The developer rows the product thinker saw
  (`git repo: true`, work tiers, receipt pins) are exactly this; the
  facilitator's view is the verbose mode. [CONSENSUS] [clig.dev](https://clig.dev/).
- **Colour is never the only carrier.** WCAG 2.2 success criterion 1.4.1:
  "Color is not used as the only visual means of conveying information,
  indicating an action, prompting a response, or distinguishing a visual
  element." The word-and-symbol rule in the press release is this criterion
  applied to a terminal. [EVIDENCE]
  [Understanding SC 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html).
- **When to drop colour.** clig.dev: disable colour when stdout or stderr is
  not a TTY, when `NO_COLOR` "is set and it is not empty (regardless of its
  value)", or when `TERM` is `dumb`. The `NO_COLOR` standard adds that
  "user-level configuration files and per-instance command-line arguments
  should override the `NO_COLOR` environment variable", which is the order
  itd-112's shipped ladder already uses (`--no-color` first). [EVIDENCE]
  [clig.dev](https://clig.dev/), [no-color.org](https://no-color.org/).
- **The host session renders flat text in its screen-reader mode.** In that
  mode the host prints "no box-drawing characters", "no color-only cues", and
  "tables in Claude's replies read as `Header: value` sentences instead of a
  box-character grid". A board that is a markdown list, not a table, survives
  both the ordinary render and this mode unchanged. [EVIDENCE]
  [Use Claude Code with a screen reader](https://code.claude.com/docs/en/accessibility.md).
- **A conflict with the ruling, left intact.** clig.dev also says "Suggest
  commands the user should run", because it "helps them learn how to use your
  program and discover new functionality". The product thinker ruled that the
  product thinker's view names no command (decision 6 and the menu intent).
  The ruling stands; the menu is where the next step lives, as the record
  says. [CONSENSUS, overridden by ruling] [clig.dev](https://clig.dev/).

### Options weighed

1. Keep the design as planned; cite the conventions above in the spec.
2. Add a `--plain` form beside `--json` for a pipe, as clig.dev suggests.
   Not needed: the text form already carries no escape byte in a pipe (A2).

### Recommendation

**Path 2 — native, shaped to the `gh status` and clig.dev conventions, with
one renderer behind the Terminal, pipe and markdown forms as the seam.** No
dependency. The record's `[F]` marks on the session-render facts can be
replaced by the citations above at the spec.

### Local fit risks

- The screen-reader finding is about one host; the rule itself (list, not
  table) is host-agnostic, and the plugin page must not name the host beyond
  the one mention the record allows.
- clig.dev's "suggest commands" is a standing counter-argument to a view with
  no command; if the product thinker's falsifier (A4) trips, this is the first
  thing to revisit.

## itd-2610031215002409 — the "what next?" menu

### Findings

- **The host's question tool limits are documented, not only in a schema.**
  "Each `AskUserQuestion` call supports 1-4 questions with 2-4 options each";
  `header` is a "short label for the question (max 12 characters)"; each
  option has a `label` and a `description`; the answer maps the question text
  to the chosen label, with an optional free-text `response`. This settles the
  record's `[F]` on "the question tool takes no more than four" in the
  record's favour. [EVIDENCE]
  [Handle approvals and user input](https://code.claude.com/docs/en/agent-sdk/user-input.md).
- **Only prompt where someone can answer.** "Only use prompts or interactive
  elements if `stdin` is an interactive terminal (a TTY)", "if `--no-input` is
  passed, don't prompt", and "never require a prompt". The record's B3 is this
  rule. [CONSENSUS] [clig.dev](https://clig.dev/).
- **A machine-level off switch is the forge CLI's shape too.**
  `GH_PROMPT_DISABLED`: "set to any value to disable interactive prompting in
  the terminal"; `GH_ACCESSIBLE_PROMPTER` (preview): "set to a truthy value to
  enable prompts that are more compatible with speech synthesis and braille
  screen readers"; `GH_SPINNER_DISABLED` replaces the spinner "with a textual
  progress indicator". All are the person's environment, never the
  repository's, matching decision 8. [EVIDENCE]
  [gh environment](https://cli.github.com/manual/gh_help_environment).
- **Numbered lists are the accessible form everywhere.** The host's screen
  reader mode: "menus you'd normally navigate with the arrow keys … become
  numbered lists. Claude Code announces each option as a numbered line, then
  a `Select with numbers` prompt". The Go prompt library huh "has a special
  rendering option designed specifically for screen readers",
  `form.WithAccessible(true)`, which "will drop TUIs in favor of standard
  prompts", read from an `ACCESSIBLE` environment variable in its own example.
  [CONSENSUS]
  [screen reader](https://code.claude.com/docs/en/accessibility.md),
  [huh](https://github.com/charmbracelet/huh).
- **The menu's write is unusual but not unprecedented in kind.** `gh status`
  itself writes nothing; the dashboards surveyed earlier either show or act.
  No surveyed CLI marks a status "waiting on you" merely by drawing a menu;
  the record already calls this the one exception and owes the chapter edit.
  [INFERENCE from absence]

### Options weighed

1. Keep the menu as decided: arrow keys or numbers in a Terminal, the same
   object to the host's tool, off by a machine setting.
2. Add an environment-variable alias for the off switch (the `gh` shape) so a
   CI image or a shell profile can turn it off without the settings file.

### Recommendation

**Path 2 — native over the standard library and golang.org/x/term (already
signed off), shaped to the host tool's 2–4 options and 12-character header,
with the question object as the seam.** Option 2 is cheap and worth taking at
the spec: the setting is the record; an environment variable reads the same
switch. No new dependency; huh is cited as a convention, not adopted (the
record already counts its module cost).

### Local fit risks

- The 12-character header and 2–4 options are a host's limits, carried into
  abcd's own question layout as the common shape; a second host with wider
  limits would not widen abcd's.
- An environment-variable alias is a second place the switch can be set; the
  record's "one setting" wording would need to name both.

## itd-2610032150577708 — the dashboard through Tailscale

Step 1 of the spec is built **[LOCAL]** (`internal/surface/dashboard/`: the
tailnet-only listener, the lookup gate, the relay-header refusal, Go's
cross-origin protection, the fixed route table). The research serves the two
open questions and the notes criterion.

### Findings

- **A shared-in device is attributed to the sharer.** "As of Tailscale v1.4,
  shared machines appear in the other tailnet as the sharer, not the owner of
  the machine"; "sharing gives the recipient access to only the shared machine
  in your tailnet, and nothing else"; "a shared machine is visible only to the
  individual recipient user"; shared machines "are quarantined by default" and
  "cannot start connections" unless the recipient allows it; sharing "strips
  tags, groups, and subnet information". So a device someone else shared into
  the product thinker's tailnet resolves, in the lookup, to a user who is not
  one of the tailnet's own, which is a property the gate can test rather than
  guess at; and by default such a device cannot even open a connection to the
  dashboard until the recipient lifts the quarantine. [EVIDENCE]
  [Share machines](https://tailscale.com/kb/1084/sharing).
- **The lookup carries a profile and the policy's capabilities.** The local
  client's `WhoIs` answers with `Node`, `UserProfile` and `CapMap`, "a map of
  capabilities to their values"; "in successful whois responses, Node and
  UserProfile are never nil". `UserProfile.LoginName` is "for display purposes
  only (provider is not listed)", so a login string is a label, not a key: the
  gate's identity is the node, as built. [EVIDENCE]
  [apitype.WhoIsResponse](https://pkg.go.dev/tailscale.com/client/tailscale/apitype#WhoIsResponse),
  [tailcfg.go](https://github.com/tailscale/tailscale/blob/v1.104.2/tailcfg/tailcfg.go).
- **Serve hides the client and adds identity headers; Funnel adds none.**
  Serve "adds a few Tailscale identity headers to the request sent to your
  backend" (`Tailscale-User-Login`, `Tailscale-User-Name`,
  `Tailscale-User-Profile-Pic`), "removes these headers from incoming requests
  if the client sends them, to prevent spoofing", and does not populate them
  for tagged devices; "Funnel traffic doesn't include identity headers"; Serve
  "requires you to enable HTTPS certificates in your tailnet". Decision 21's
  route (listen on the node's own addresses) and the relay-header refusal
  match the documented behaviour. [EVIDENCE]
  [Tailscale Serve](https://tailscale.com/kb/1312/serve),
  [Tailscale Funnel](https://tailscale.com/kb/1223/funnel).
- **A certificate publishes the machine name, as decision 23 says.** The
  public ledger "includes the fully qualified domain name of your devices";
  "do not enable the HTTPS feature if any of your machine names contain
  sensitive information". Certificates are Let's Encrypt, 90 days, obtained
  with `tailscale cert` or `tailscale.com/client/local.Client.GetCertificate`.
  [EVIDENCE] [Enabling HTTPS](https://tailscale.com/kb/1153/enabling-https).
- **Cross-origin protection over plain HTTP, as built.** Go 1.25's
  `http.CrossOriginProtection` judges by `Sec-Fetch-Site`, falling back to
  comparing `Origin` with `Host`; "requests without Sec-Fetch-Site or Origin
  headers are currently assumed to be either same-origin or non-browser
  requests" and are allowed; GET, HEAD and OPTIONS always pass. So a write
  from a non-browser client on the tailnet is stopped only by the gate and
  the session, never by this layer; the spec's "accepted only from a device
  the dashboard let in" is the load-bearing check. [EVIDENCE]
  [net/http CrossOriginProtection](https://pkg.go.dev/net/http#CrossOriginProtection).
- **Notes that survive rewording: the annotation field's method.** The
  reference implementation stores three selectors per note (a range, a
  character position, and a quote with a 32-character prefix and suffix) and
  re-anchors in order: range, then position, then "context-first fuzzy
  matching" (find the prefix near the expected start, the suffix near the
  expected end, compare the text between against the saved quote within a
  threshold), then a fuzzy search for the quote alone. The W3C selectors the
  co-editing note already cites are the data model; this is the matching
  order to copy, and it is what makes the criterion "stays attached even after
  the brief is reworded; if the sentence is gone, shown struck through" an
  implementable rule rather than a hope; what to show when all four fail is
  not in the write-up, and the orphan is this record's own rule. [EVIDENCE]
  [Fuzzy anchoring](https://web.hypothes.is/blog/fuzzy-anchoring/),
  [W3C Web Annotation selectors](2026-10-14-live-co-editing-to-one-pull-request.md)
  (opened in the earlier pass).
- **Per-person capabilities exist in the policy file, for later.** Grants
  carry an `app` field of the form `domainName/capabilityName` with JSON
  parameters that Tailscale "only validates … is valid JSON"; the application
  reads them from the request. golink grants admin rights through
  `"tailscale.com/cap/golink": [{"admin": true}]` and "looking up the user's
  capabilities via the Tailscale daemon whois API". Not needed for the
  product-thinker-only first version; it is the seam itd-2610040740135705
  can use. [EVIDENCE] [Grants](https://tailscale.com/kb/1324/grants),
  [golink](https://github.com/tailscale/golink).
- **Browsers are starting to gate public-to-local requests, not local pages.**
  Chrome's Local Network Access permission covers "any request from the
  public network to a local network or loopback destination", where local
  means RFC 1918, link-local and IPv6 unique-local ranges; local-to-local
  gating is future work. The dashboard is a local page with a self-only
  content-security policy, so it is unaffected; a public page trying to reach
  it would be prompted, which is a defence for free. The Tailscale IPv6
  address is unique-local and so counts as local; the IPv4 range is not in
  the list the post gives. [EVIDENCE]
  [Local Network Access](https://developer.chrome.com/blog/local-network-access).

### Options weighed

For the first open question (a device shared into the tailnet):

1. Refuse it, as the facilitator's default does, now testable by "the lookup's
   user is not one of this tailnet's own".
2. Let it in when the recipient lifted the quarantine, since the person
   already chose to let that device reach their machines.

For the second (a request passed on by Serve or Funnel): refuse on the
headers, as built; Funnel is additionally refused by never holding a
certificate. For notes: store the three selectors and re-anchor in the
documented order.

### Recommendation

**Path 2 — native, shaped to the Tailscale lookup, the annotation field's
anchoring order and Go's own cross-origin layer, with the identity lookup as
the seam.** Option 1 stays the default until the product thinker rules; the
research turns it from "refuse what we cannot tell" into "refuse a user who
is not ours", which is the question to put to them. No new dependency: the
lookup already runs the tailscale CLI's `whois --json` through
`internal/adapter/tailscale` (there is no `tailscale.com` module in `go.mod`;
the `apitype.WhoIsResponse` citation stands as the JSON shape the CLI
prints), and the anchoring is string matching.

### Local fit risks

- "Refuse a user who is not ours" needs the list of the tailnet's own users,
  which the lookup does not supply; it would come from the admin API (an
  outbound call, adr-38) or from treating "appears as the sharer" plus a
  login domain difference as the signal. The fit-challenge should decide
  which is admissible.
- The fuzzy re-anchoring threshold is a judgement; a note re-attached to the
  wrong sentence is worse than an orphan. Loud staging says show the orphan
  and the old words rather than guess.

## itd-2610040740108331 — the dashboard on the home network

### Findings

- **Plain HTTP on a LAN address is not a secure context, so a whole class of
  browser features is unavailable.** Secure contexts are documents "delivered
  over an HTTPS connection, or … from a loopback (local) address"; the
  trustworthy hosts are `127.0.0.0/8`, `::1/128`, `localhost` and
  `*.localhost`. A `192.168.x.x` or `.local` origin over HTTP qualifies under
  none of them. Restricted to secure contexts: the Web Authentication API
  (passkeys), Service Workers, the Async Clipboard API, the Web Crypto API,
  Geolocation, Notifications, Web Share and the Storage API. So a passkey or
  WebAuthn pairing on the home network is impossible by construction; the
  one-time code the product thinker chose is the right class of mechanism.
  [EVIDENCE]
  [Secure contexts](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts),
  [Features restricted to secure contexts](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts/features_restricted_to_secure_contexts).
- **The local-server field agrees: expose on a LAN only with authentication
  and HTTPS.** Syncthing's GUI listens on `127.0.0.1:8384` by default; to
  reach it from other devices "you should then also set a username and a
  strong password for authentication and check the option to use HTTPS", and
  binding your computer's LAN address "will NOT restrict access to only
  devices on your local network!".
  code-server: "web views rely on service workers, and service workers are
  only available in a secure context", with the options being loopback, a
  real certificate, "a trusted self-signed certificate" (mkcert), or
  disabling browser security. Jupyter Server: token authentication "on by
  default", carried in a URL (`?token=…`) after which "a cookie will be set in
  your browser"; `allow_remote_access` defaults to `False`, which returns 403
  when "the Host header indicates a non-local domain" (its DNS-rebinding
  guard), with `local_hostnames` defaulting to `['localhost']`;
  `shutdown_no_activity_timeout` ("Shut down the server after N seconds with
  no kernels running and no activity") defaults to `0`, off. [CONSENSUS]
  [Syncthing GUI listen address](https://docs.syncthing.net/users/guilisten.html),
  [code-server FAQ](https://coder.com/docs/code-server/FAQ),
  [Jupyter Server security](https://jupyter-server.readthedocs.io/en/latest/operators/security.html),
  [Jupyter Server config](https://jupyter-server.readthedocs.io/en/latest/other/full-config.html).
- **Writes over plain HTTP: the cross-origin layer is weaker than it looks.**
  Go's protection allows any request with neither `Sec-Fetch-Site` nor
  `Origin`, and over HTTP the safe-method rule still holds; a non-browser
  client on the Wi-Fi that has captured the session cookie can write. On the
  tailnet the transport is encrypted and the peer is named; on the home
  network neither holds. The ADR's "nothing is changed through the page over
  plain HTTP" is the conservative reading of this fact. [EVIDENCE +
  INFERENCE] [net/http CrossOriginProtection](https://pkg.go.dev/net/http#CrossOriginProtection).
- **Chrome's Local Network Access does not stand in the way.** It gates
  requests "from the public network to a local network or loopback
  destination"; a page opened at the computer's own LAN address is not a
  public page. It would, though, prompt a public page that tried to reach the
  dashboard, which complements the Host check. [EVIDENCE]
  [Local Network Access](https://developer.chrome.com/blog/local-network-access).
- **A `.local` name needs mDNS.** The Go standard library has no multicast
  DNS responder; a friendly `abcd.local` address would mean a dependency and
  a multicast listener, two things the record does not have today. The
  "explicit address" the ADR requires is therefore an IP and port, or the
  computer's own hostname where the home router resolves it. [INFERENCE; the
  mDNS RFC was not opened in this pass]

### Options weighed

For writes on the home network:

1. Reads only, as the ADR says; notes and "Still right" need the tailnet (or
   a later HTTPS route). Simplest; nothing to install on a phone.
2. A local certificate authority the person installs on each device once (the
   mkcert pattern code-server names), after which the LAN origin is a secure
   context and writes travel encrypted. Publishes nothing, unlike Let's
   Encrypt; costs an install step per device and a trust root on each.
3. Writes over plain HTTP guarded by cookie, Host check and cross-origin
   layer. The field does not do this, and the layer admits header-less
   clients.

For the idle switch-off: Jupyter's precedent is a setting with a default; the
ADR requires the switch-off, so the default cannot be "never".

### Recommendation

**Path 3 — deliberate divergence from the field's "HTTPS on the LAN" advice,
chosen given the parent's decision 23 (a ruling against a certificate that
publishes the machine's name in a public register, which a locally minted
authority does not do) and the cost of installing a trust root on each
device (this pass's inference, not a ruling in the draft, for the product
thinker): reads only over the home network (option 1), with option 2 named
as the later route to writes.** Put to the product thinker at planning: the idle switch-off
as a setting beside the status-line settings, defaulting to one hour idle and
eight hours hard (the research supplies no measured number; the two values
are the ones the record already lists), and whether a home-network device may
be let in for seven days when it can never write. No new dependency.

### Local fit risks

- Option 2 later would mean abcd minting a certificate authority on the
  person's machine and asking them to trust it on a phone: a trust root is a
  heavier thing than a seven-day cookie, and the user's directory is theirs.
- A reads-only home network makes the "mark a line and leave a note" promise
  tailnet-only; the press release of this draft should say so (loud staging).
- `.local` naming is a convenience the field offers and this design declines;
  the start line must make the raw address easy to type or scan.

## itd-2610040740122709 — acting from the dashboard

### Findings

- **The forge's own "propose, then someone with write access applies" is the
  closest pattern.** On a pull request "other people can suggest specific
  changes", and "you can apply these suggested changes directly in a pull
  request if you have write access"; applying one or a batch "creates a
  single commit", on which "each person who suggested a change included in
  the commit will be a co-author" and "the person who applies the suggested
  changes will be a co-author and the committer". Authority sits with the
  applier; credit goes to the proposer. [EVIDENCE]
  [Incorporating feedback](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/incorporating-feedback-in-your-pull-request).
- **Documents do the same.** In Google Docs' suggesting mode "you can suggest
  changes to a document without changing the original text"; "the owner of
  the file will get an email about your suggestions and can decide whether to
  keep them"; suggestions are accepted or rejected one at a time or all at
  once. [CONSENSUS]
  [Suggest edits in Google Docs](https://support.google.com/docs/answer/6033474).
- **The kernel's tags separate "proposed", "agreed" and "checked".**
  `Suggested-by:` "indicates that the patch idea is suggested by the person
  named"; `Acked-by:` "is meant to be used by those responsible for or
  involved with the affected code" and "does not necessarily indicate
  acknowledgement of the entire patch"; `Reviewed-by:` "indicates that the
  patch has been reviewed and found acceptable". Git recognises any `Key: value` block at the end of a message
  as trailers. These give the rewrite and the approval distinct, recordable
  shapes: a rewrite is a suggestion the facilitator applies; an approval is an
  acknowledgement, not a review. [EVIDENCE]
  [Submitting patches](https://www.kernel.org/doc/html/latest/process/submitting-patches.html),
  [git interpret-trailers](https://git-scm.com/docs/git-interpret-trailers).
- **Writes stay on the encrypted way in.** From the sibling section above:
  the cross-origin layer admits header-less clients on any network, so the
  gate and the session carry the check everywhere; what the tailnet adds is
  an encrypted transport and a peer the daemon names, and that, not the
  cross-origin layer, is why an acting dashboard is tailnet-only, as this
  draft already says. [EVIDENCE + INFERENCE]

### Options weighed

For a rewrite: (1) a suggestion the facilitator applies, credited to the
product thinker; (2) a direct write reviewed after. For approving an intent:
(1) the sign-off itself; (2) a request the facilitator completes.

### Recommendation

**Path 2 — native, shaped to the forge's suggested-changes flow: a rewrite is
a suggestion the facilitator applies and the product thinker is credited on
the resulting change; an approval is a recorded acknowledgement the
facilitator completes, never the planning sign-off itself.** This answers
both open questions in the direction decision 17 already points (everything
the product thinker touches reaches the facilitator's review first). No
dependency; the suggestion is the note store of itd-2610032150577708 with a
proposed replacement text attached.

### Local fit risks

- "Credited on the change" means a trailer on a commit a human authors; the
  attribution gate's rules on who may be named (and the private-names banlist)
  bind what the dashboard writes.
- Direct writes reviewed after (option 2) would make the dashboard the second
  writer of a record file, against one-writer-per-file.

## itd-2610040740135705 — the dashboard for more people

### Findings

- **GitHub cannot tell the product thinker from the facilitator.** Custom
  repository roles are Enterprise Cloud only; teams exist only in
  organisations; a personal repository has only an owner and write
  collaborators; "if narrowing is wanted, an optional config file in the repo,
  keyed on numeric user ids, is the only mechanism that works everywhere".
  That is said of GitHub user ids for GitHub-side admission; it rules out
  GitHub as the role source and does not transfer to the tailnet lookup's
  profile id, a different identifier space. [EVIDENCE]
  [GitHub roles note](2026-10-14-github-roles-admission-and-issue-links.md)
  (opened in the earlier pass).
- **The tailnet can carry per-person roles, attested by the daemon.** A
  grant's `app` capability (`domainName/capabilityName`, JSON parameters)
  reaches the application through the lookup's `CapMap`; golink's
  `"tailscale.com/cap/golink": [{"admin": true}]` is the worked example, and
  "application capabilities are defined by the applications themselves, not
  by Tailscale". That is a role declaration that no header can forge and no
  browser can change, but it lives in the tailnet's policy file, which the
  tailnet's admin edits, not the project. [EVIDENCE]
  [Grants](https://tailscale.com/kb/1324/grants),
  [golink](https://github.com/tailscale/golink),
  [apitype.WhoIsResponse](https://pkg.go.dev/tailscale.com/client/tailscale/apitype#WhoIsResponse).
- **Each device is already let in separately.** The lookup names a node and a
  profile per connection; the devices-seen file lists them **[LOCAL]**
  (`internal/surface/dashboard/seen.go`). A per-person view needs no new
  admission step on the tailnet, only a mapping from the profile to a role.
- **Views, not visibility, is the cheap first cut.** The facilitator's view in
  the board intent is "today's rows plus spec-id and in-flight rows", the same
  read of the project arranged differently; `gh status` likewise shows one
  person's slice of shared data. Hiding records from a role is a second,
  harder thing (every page comes from one renderer by id). [INFERENCE from
  the record and the board findings]

### Options weighed

1. Roles declared in the project's abcd configuration, keyed on the login
   the lookup reports; the view follows the role; nothing is hidden.
2. As 1, plus the role read from a tailnet app capability when present
   (`<abcd's domain>/cap/abcd`), for tailnets whose admin prefers the policy
   file.
3. Roles from GitHub: ruled out by the roles note.

### Recommendation

**Path 2 — native, shaped to golink's capability model, with the role source
as the seam: the project's configuration first (option 1), a tailnet
capability as an optional second source later (option 2).** The first release
chooses a view per role and hides nothing; who sees what, beyond the view,
is a later question with a cost the renderer's one-source design makes real.
No new dependency.

### Local fit risks

- A role keyed on a login string keys on a display label ("for display
  purposes only"); a stable key would be the user id the lookup's profile
  carries. The spec should pick the id.
- A role list in the repository is shared working data read by a per-machine
  listener; a person's role on the tailnet is the tailnet admin's to grant.
  Two authorities for one fact is the thing to challenge.

## itd-2610040754440360 — the private dashboard for every managed project

### Findings

- **The dashboard already reads a project, not abcd.** The listener, gate and
  control live under abcd's home and "name no project content"; the pages
  come from the record site's renderer by record id **[LOCAL]**
  (`internal/surface/dashboard/control.go`, the spec's technical checks). What
  a second project needs is therefore a tree that the renderer and the status
  read can open, not a second dashboard.
- **A fixture tree is the Go convention for exactly this.** "The go tool will
  ignore a directory named "testdata", making it available to hold ancillary
  data needed by the tests." A sparse sample project is a `testdata` tree with
  a `.abcd/` of a few records, opened by a test that starts the dashboard
  against it. [EVIDENCE] [cmd/go](https://pkg.go.dev/cmd/go#hdr-Test_packages).
- **Theme ecosystems demand a runnable example beside the thing.** Hugo's
  theme submission requires a `theme.toml` and screenshots; the example-site
  requirement widely associated with it was not on the page opened, so it is
  not claimed here (see Not verified). [EVIDENCE for what was opened]
  [Hugo theme submission](https://github.com/gohugoio/hugoThemesSiteBuilder/tree/main#readme).

### Options weighed

1. The sample lives in abcd's repository under a `testdata` tree and a test
   proves the dashboard runs on it.
2. The sample is a second public repository abcd manages for the purpose.
   Honest, but a second checkout to keep current and a second place for
   records to drift.

### Recommendation

**Path 2 — native, shaped to the `testdata` fixture convention: the proof is a
test that starts the dashboard on a committed sparse tree (option 1).** How an
adopting project starts it is the same verb with no abcd-specific setup; the
test is what shows that claim true or false. No dependency.

### Local fit risks

- A fixture under `testdata` is not a "project a team manages"; the
  fit-challenge should ask whether the discipline itd-140 names accepts a
  fixture as the second project, or needs a real checkout.

## itd-2610040754453237 and itd-2610091010348892 — the public record on a project's own GitHub Pages

The two are one question at two scopes: the planned intent builds abcd's
workflow for an opted-in project at each release; the draft asks what any
team publishes by default and what stays private. The findings are shared.

### Findings

- **The official actions and their limits.** The deploy job needs
  `pages: write` and `id-token: write` ("necessary to request the OIDC JWT
  token") and "should target the `github-pages` environment". The artefact
  must "be a single gzip archive containing a single tar file", "under 10GB
  in size (we recommend under 1 GB!)", must "not contain any symbolic or hard
  links", and the "officially supported maximum size limit is 1GB", with
  larger sites often failing on the 10-minute deployment timeout;
  `retention-days` defaults to `1`. The current majors, from each
  repository's tags on 2026-10-23, are `actions/configure-pages@v6`,
  `actions/upload-pages-artifact@v5` and `actions/deploy-pages@v5`
  (v5.0.1); GitHub's starter workflow still pins `configure-pages@v5` and
  `upload-pages-artifact@v3`, with `concurrency: group: "pages",
  cancel-in-progress: false`. `upload-pages-artifact` has an
  `include-hidden-files` input, default `false`, so dotfiles are left out of
  the artefact unless it is set.
  Pages limits: sites "may be no larger than 1 GB", a soft 100 GB a month of
  bandwidth, deployments "timeout if they take longer than 10 minutes", and
  the ten-builds-an-hour limit "does not apply if you build and publish your
  site with a custom GitHub Actions workflow". [EVIDENCE]
  [deploy-pages](https://github.com/actions/deploy-pages),
  [upload-pages-artifact](https://github.com/actions/upload-pages-artifact),
  [starter workflow](https://raw.githubusercontent.com/actions/starter-workflows/main/pages/static.yml),
  [Pages limits](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits).
- **A release-triggered deploy runs on the tag, and the default environment
  rule can refuse it.** GitHub "recommend[s] that you add a deployment
  protection rule so that only the default branch can deploy to this
  environment"; "the deployment branch or tag rule is matched against the
  `GITHUB_REF` of the workflow run" (the deployments-and-environments
  reference); "name patterns must be configured for branches or tags
  individually". One project's docs workflow, triggered on
  release, failed with "Tag 'v1.0.1' is not allowed to deploy to
  github-pages" and was reworked to build from the default branch via
  `repository_dispatch`. For a workflow that runs on `release: published`
  the fix is either a tag rule (`v*`) on the `github-pages` environment, set
  by the owner in repository settings, or a job that checks out the release's
  commit while running on the default branch. Setup must say which, because
  "nothing else in the repository changes" cannot cover a settings page.
  [EVIDENCE]
  [Configuring a publishing source](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site),
  [Deployments and environments reference](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments),
  [Managing environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments),
  [redis-om-spring #660](https://github.com/redis/redis-om-spring/pull/660) (one project's report, [ANECDOTE] for the error text).
- **The site lives under a path, and the action tells you which.**
  `actions/configure-pages` outputs `base_url` ("https://octocat.github.io/my-repo"),
  `origin`, `host` and `base_path` ("/my-repo" or an empty string); its
  `enablement` input "Try to enable Pages for the repository if it is not
  already enabled" needs a token beyond `GITHUB_TOKEN` (a PAT with `repo`, or
  an App with `administration:write` and `pages:write`). So abcd's renderer
  must take a base path (project sites are `<owner>.github.io/<repositoryname>`),
  and enabling Pages with the "GitHub Actions" source is a one-time step the
  owner does by hand or abcd's setup does with the owner's own forge login,
  never with a key abcd holds. [EVIDENCE]
  [configure-pages action.yml](https://raw.githubusercontent.com/actions/configure-pages/main/action.yml),
  [What is GitHub Pages](https://docs.github.com/en/pages/getting-started-with-github-pages/what-is-github-pages).
- **A release made with `GITHUB_TOKEN` does not trigger.** The earlier note
  records that a release created with `GITHUB_TOKEN` does not start a
  `release`-triggered workflow (GitHub's general rule for events the default
  token raises). The publishing-source page's "Commits pushed by a GitHub
  Actions workflow that uses the `GITHUB_TOKEN` do not trigger a GitHub Pages
  build" is the branch-publishing source's rule, not the Actions source's
  this route uses. A project whose release workflow creates the release with
  the default token needs the site job inside that workflow, not a second
  one listening for the event. [EVIDENCE]
  [Configuring a publishing source](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site),
  [GitHub builds note](2026-10-14-abcdev-app-github-builds-and-cache.md).
- **What stays private is decided by the plan, not by abcd.** An
  access-controlled Pages site needs GitHub Enterprise Cloud ("we use a
  unique subdomain for the privately published site"); on other plans a
  Pages site built from a private repository is public (earlier note). For a
  private repository the opt-in question is therefore "publish your record to
  the world?", and the default must be no. What a public project publishes
  is the committed tree only: `.abcd/development/` and `.abcd/work/`; the
  local tier is never in the repository, so it is never on the site. The
  record's "every word comes from the repository" rule and the site's
  single-source gate already enforce this. [EVIDENCE + LOCAL]
  [Changing Pages visibility](https://docs.github.com/en/pages/getting-started-with-github-pages/changing-the-visibility-of-your-github-pages-site),
  [partners note](2026-10-14-partners-and-no-server-options-for-the-hosted-sites.md).
- **Acceptable use.** Pages is not for "sensitive transactions like sending
  passwords or credit card numbers" or commercial SaaS; "the visitor's IP
  address is logged and stored for security purposes" by GitHub. A record
  site is informational and within the terms; the "report" link pointing at
  GitHub, as ruled, matches who holds the logs. [EVIDENCE]
  [Pages limits](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits),
  [What is GitHub Pages](https://docs.github.com/en/pages/getting-started-with-github-pages/what-is-github-pages).

### Options weighed

1. One workflow abcd scaffolds, running on `release: published`, with the
   environment tag rule documented as the owner's one setting.
2. The site job appended to the project's existing release workflow (where
   abcd scaffolds that too), so a `GITHUB_TOKEN`-created release still builds
   the site and the run is on the default branch's ref.
3. A reusable workflow in abcd's repository the project calls. Centralises
   fixes but makes every project's build depend on abcd's default branch.

### Recommendation

**Path 1 for hosting — GitHub Pages through the project's own workflow and
the three official actions, exactly as the ruling says — with the renderer
native and base-path-aware.** Take option 2 where abcd owns the release
workflow and option 1 where it does not; in both, setup prints the one
settings step (enable Pages with the Actions source; add the tag rule if the
job runs on the tag) as instructions to a person, verified and linked. The
draft for every team adds nothing to this but the default: public
repositories opt in; a private repository is told its site would be public
and is not offered the switch until a later intent. No dependency.

### Local fit risks

- "Nothing else in the repository changes" (the first criterion) is true of
  the tree and false of the settings page; the acceptance criterion should
  say so or setup must do that step through the owner's login (an outbound
  call under adr-38's "when asked" rule).
- A reusable workflow (option 3) is the convenient fix channel and the
  single point of failure; the record's "the project's own workflow builds
  it" wording favours copying the workflow in, which then needs abcd's
  setup to update it on a later run.
- The renderer today composes abcd's own site; a base path and a project
  parameter are the itd-140 boundary made concrete.

## itd-2609231507251267 — release pages on the project website

### Findings

- **The changelog convention is one entry per version, linkable, dated,
  latest first.** Keep a Changelog's principles: "There should be an entry
  for every single version", "Versions and sections should be linkable", "The
  latest version comes first", "The release date of each version is
  displayed"; a pulled release is shown as `[YANKED]`; and "using commit log
  diffs as changelogs is a bad idea: they're full of noise". [CONSENSUS]
  [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/).
- **The forge's release page is per tag, with a stable latest link and a
  feed.** "You can share every release you create on GitHub with a unique
  URL"; adding `releases/latest` to a repository URL links the latest; the
  repository's `releases.atom` is an Atom feed titled "Release notes from
  <repo>" with one entry per release linking to its tag page (confirmed on a
  public repository's feed). Automatically generated notes list "merged pull
  requests, a list of contributors to the release, and a link to a full
  changelog", grouped by label through `.github/release.yml`: the thing the
  record deliberately does not do (a press release is user moments, not a
  pull-request list). [EVIDENCE]
  [About releases](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases),
  [Linking to releases](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases),
  [a releases Atom feed](https://github.com/golang/go/releases.atom),
  [Automatically generated release notes](https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes).
- **The site already links release tags.** The timeline renders
  `/releases/tag/v<version>` per release and the install surface
  `/releases/latest` **[LOCAL]** (`internal/core/site/timeline.go`,
  `compose.go`). A release page per `RELEASE.md` and archive entry slots
  under the same addressing.
- **A conflict with the parent's ruling, left intact.** Keep a Changelog's
  "an entry for every single version" is about the changelog, which abcd
  already derives for every cut; the ruling that a fixes-only release writes
  no page concerns a second document. The two are compatible as long as the
  release page links the changelog section for the releases it skips.
  [CONSENSUS, scoped]

### Options weighed

1. One page per archived release at a stable path keyed on the version,
   latest first on an index, each page dated and linking its tag; an Atom feed
   of the pages beside it, mirroring the forge's.
2. One long page of all releases (the Keep a Changelog file shape). Simpler,
   but the index is the thing the product thinker reads and a long page buries
   the latest.

### Recommendation

**Path 2 — native, shaped to Keep a Changelog's ordering rules and the
forge's per-tag page and feed (option 1).** The feed is optional and cheap
from the same data; the page composes nothing, as ruled. No dependency.

### Local fit risks

- A feed is an outward-facing text: the attribution rules (no tool footer, no
  session URL) and the public-naming rules bind its entries as they bind the
  page.
- The press-release personas are fictional (parent decision 6); a feed reader
  shows an entry stripped of its page context, so each entry should carry the
  one-line "fictional persona" note or none of the quotes.

## itd-2610031651058674 — the README drift check

### Findings

- **The marker already carries a checksum.** Each part is wrapped in
  `<!-- BEGIN ABCD:<name> sha256=<64 hex> -->` over a normalised body
  **[LOCAL]** (`internal/core/ahoy/readme_marker.go`). That is cog's design:
  its `-c` option will "checksum the output to protect it against accidental
  change", written as `[[[end]]] (sum: …)`, and on a hand edit "the checksum
  won't match, and cog will stop to avoid overwriting the edited output".
  [EVIDENCE] [cog](https://cog.readthedocs.io/en/latest/running.html).
- **"Would change if run again" is the staleness check the field uses.**
  cog's `--check`: "Check that the files would not change if run again",
  paired with `--diff` to show why; embedme's `--verify`: "Verify that running
  embedme would result in no changes. Useful for CI"; terraform-docs injects
  between `<!-- BEGIN_TF_DOCS -->` and `<!-- END_TF_DOCS -->` and leaves the
  rest of the file alone. None of these distinguishes a hand edit from a
  stale derivation in its report except cog, and cog only at write time.
  [CONSENSUS]
  [cog](https://cog.readthedocs.io/en/latest/running.html),
  [embedme](https://github.com/zakhenry/embedme),
  [terraform-docs output](https://terraform-docs.io/user-guide/configuration/output/).
- **Exit codes separate "drift" from "broken".** Prettier's `--check` exits
  `0` ("Everything formatted properly"), `1` ("Something wasn't formatted
  properly") and `2` ("Something's wrong with Prettier"). abcd's lint already
  exits 2 on an error finding and 1 on warnings alone; a drifted part is the
  warning class, a duplicated or unclosed marker the error class. [CONSENSUS]
  [Prettier CLI](https://prettier.io/docs/cli).
- **No colour in a pipe.** The check runs in CI; clig.dev's rule (no colour
  when not a TTY, honour `NO_COLOR` and `TERM=dumb`) and adr-49's invariant
  apply to its output. [CONSENSUS] [clig.dev](https://clig.dev/).

### Options weighed

1. Two findings per part, from the data already in the marker: *hand-edited*
   (body hash differs from the marker's) and *stale* (hashes agree, but the
   body regenerated from the project differs); exit 1 on either, 2 on a
   broken marker; an optional diff.
2. A single "would change" finding (the cog `--check` shape). Loses the
   distinction the parent intent's decision 6 asks for.

### Recommendation

**Path 2 — native, shaped to cog's checksum-plus-check pair and Prettier's
exit codes (option 1).** The two findings fall out of the marker the parent
already writes; the check is a reader of that marker, never a writer. No
dependency.

### Local fit risks

- The check regenerates each part to compare; regeneration must not make the
  badge-address or outbound calls the setup path makes (adr-38: a check reads
  the disk).
- Exit 1 in CI fails the build on a hand edit the owner meant to keep; the
  owner's remedy (remove the marker, or run setup) must be in the finding's
  text, as at-every-refusal-emit-what-the-tool-knows requires.

## itd-134 — every managed CLI greets like abcd

### Findings

- **The colour contract is settled and language-neutral.** `NO_COLOR`:
  suppress colour when the variable is "present and not an empty string
  (regardless of its value)"; flags and user configuration override it; it
  covers "only ANSI color, not other styling". clig.dev adds the TTY test and
  `TERM=dumb`. Truecolor detection: "checking `$COLORTERM` is recommended"
  for the values `truecolor` or `24bit`, with the caveat that it "is not
  forwarded via sudo, ssh, etc." and the advice that programs "should fall
  back to using 8-bit color". abcd's own ladder (itd-112, shipped) already
  encodes this order. [CONSENSUS]
  [no-color.org](https://no-color.org/), [clig.dev](https://clig.dev/),
  [termstandard/colors](https://github.com/termstandard/colors).
- **ASCII art needs a text alternative.** WCAG technique H86, "Providing
  text alternatives for emojis, emoticons, ASCII art, and leetspeak": ASCII
  art "can be very confusing to people who are accessing the internet using
  screen readers" and "should also have a text explanation of what the
  picture is". For a terminal banner the words beside the art (name, version,
  tagline) are that alternative, and the art must never carry meaning the
  words do not. [EVIDENCE] [H86](https://www.w3.org/WAI/WCAG22/Techniques/html/H86).
- **A rendered artefact beats generated code, for a CLI in any language.**
  The field's convention is a committed text file the program prints when
  the ladder allows; this pass found no standard for the file's shape, so the
  shape is abcd's to choose. [INFERENCE from absence; the figlet font format
  was not opened]
- **Screen-reader modes drop decoration wholesale.** The host prints "no
  box-drawing characters" and "no color-only cues" in its screen-reader mode;
  a managed CLI driven from a host session must emit no banner at all, which
  is itd-112's rule (never a byte on a machine-consumed stream). [EVIDENCE]
  [screen reader](https://code.claude.com/docs/en/accessibility.md).

### Options weighed

1. The generator emits four committed files: plain text, 16-colour ANSI,
   256-colour (8-bit) ANSI, the fallback termstandard advises when
   `COLORTERM` is absent, and 24-bit ANSI, plus a one-paragraph note of the
   rule a CLI follows (TTY, `NO_COLOR`, `TERM=dumb`, `COLORTERM`) to pick
   one.
2. One plain file only; colour left to each language's own libraries. Loses
   the identity the intent promises.
3. Generated code per language. Rejected by the intent's own text.

### Recommendation

**Path 2 — native, shaped to the `NO_COLOR` and `COLORTERM` conventions
(option 1): rendered artefacts, not code, with the selection rule written
beside them.** The per-repo accent stays open: no convention for deriving a
colour from a name was found in this pass. No dependency.

### Local fit risks

- Four files in the managed repository is four things setup must keep
  current; the README drift check's marker-and-checksum shape is the model
  for telling a hand-edited banner from a stale one.
- The words are the accessible form; a generator that lets the tagline be
  absent emits art with no alternative, against H86's advice (a sufficient
  technique, not a requirement).

## itd-2610091016262335 — a team edits its brief together live

The co-editing note already settles the editor and sync stack (raw Markdown in
CodeMirror over Yjs v13, one load from a recorded base commit, notes by
relative position plus quote anchor, a three-way merge at submit). This pass
researches the two hosting routes the product thinker chose and the crediting
rules.

### Findings

- **The Tailscale route reaches only people on the host's tailnet.** A
  device not on the tailnet cannot open an address there; letting a teammate
  in means sharing a machine with them (an invitation; "unused invite links
  expire after 30 days"; the share is "visible only to the individual
  recipient user" and quarantined by default). Funnel would share the session
  "for anyone to access—even if they don't use Tailscale", but it is in beta,
  "can only listen on ports 443, 8443, and 10000", requires "HTTPS enabled
  and valid HTTPS certificates for your tailnet" (which publishes the machine
  name in the certificate ledger), and its traffic "is subject to
  non-configurable bandwidth limits". So "scan a QR code from any
  phone" and "the host's own machine over Tailscale" are compatible only for
  a team already on one tailnet; for everyone else the relay is the route.
  [EVIDENCE] [Share machines](https://tailscale.com/kb/1084/sharing),
  [Funnel](https://tailscale.com/kb/1223/funnel),
  [Enabling HTTPS](https://tailscale.com/kb/1153/enabling-https).
- **The free relay host sleeps.** On Render's free plan a web service spins
  down after "15 minutes without receiving any inbound traffic", a cold start
  "takes about one minute", a workspace has 750 free instance hours a month,
  WebSocket messages count as activity and a new connection wakes the
  service; a free Postgres "expire[s] 30 days after creation" and the free
  key-value store is in-memory only. A relay blueprint on this tier must hold
  rooms in memory, expect the first joiner to wait a minute, and keep no
  database. [EVIDENCE] [Render free tier](https://render.com/docs/free).
- **Crediting a guest's approved change has a precedent in the forge's
  suggested changes.** Applying a suggestion makes the suggester "a co-author
  of the commit" and the applier "a co-author and the committer". The
  kernel's tags offer the distinction the record wants: `Suggested-by:` for
  the idea, `Acked-by:` or `Reviewed-by:` for the person who let it in. The
  record's `Approved-by:` is not in the kernel's tag set, but it has
  precedent: Mergify's declarative merge-commit format lets a project "opt
  into trailers like `co-authored-by`, `approved-by`, or `merged-by`", and
  FreeBSD's committer's guide requires an "Approved by:" line on approved
  MFCs. Git accepts any `Key: value` trailer, so it is legal; the attribution
  gate does not yet know it. [EVIDENCE]
  [Incorporating feedback](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/incorporating-feedback-in-your-pull-request),
  [Submitting patches](https://www.kernel.org/doc/html/latest/process/submitting-patches.html),
  [git interpret-trailers](https://git-scm.com/docs/git-interpret-trailers),
  [Mergify commit message format](https://docs.mergify.com/changelog/2026-05-06-declarative-commit-message-format-for-merge-commits/),
  [FreeBSD committer's guide](https://docs.freebsd.org/en/articles/committers-guide/).
- **The secure-context rule binds the session page too.** A session served
  over plain HTTP from a LAN or tailnet address has no Async Clipboard, no
  Web Crypto and no Service Worker (see the home-network section). The relay
  route is HTTPS by its host; the Tailscale route without a certificate is
  not, so the editor must not depend on those APIs. [EVIDENCE]
  [Features restricted to secure contexts](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts/features_restricted_to_secure_contexts).

### Options weighed

1. Ship both routes as ruled, with the Tailscale route documented as "for a
   team on one tailnet" and the relay as the general case.
2. Relay only in the first release; Tailscale later. Narrower; drops a ruled
   default.
3. Keep `Approved-by:` as ruled, or rename to a kernel-style tag. A naming
   choice the product thinker made; recorded here as a question, not changed.

### Recommendation

**Path 2 — native, on the stack the co-editing note recommends, with the two
ruled routes (option 1) and the relay blueprint written for an in-memory,
sleeping host.** The QR-and-Tailscale tension is put to the product thinker at
planning rather than resolved here. The web app's dependencies (Yjs,
CodeMirror) are a new dependency set outside the Go module and need
explicit sign-off; the Go side adds none.

### Local fit risks

- A relay the team deploys is still a server holding readable content while a
  session is live (co-editing note, risk 1); the ruling admits it because the
  team runs it, and the invite text must say so (loud staging).
- `Approved-by:` is a trailer the attribution gate does not know; the gate
  and the commit-msg hook need to accept it (it has prior use in Mergify's
  and FreeBSD's conventions), or the name should follow a kernel one.
- A host on Tailscale without a certificate serves the session over plain
  HTTP to the tailnet; the dashboard's "no writes over plain HTTP" rule is
  per-ADR about the dashboard, and the fit-challenge should decide whether it
  reaches this surface.

## Not worth adopting

- **Passkeys or WebAuthn for letting a home-network device in.** Unavailable
  outside a secure context, which a plain-HTTP LAN page is not.
- **Tailscale Funnel for any of these surfaces.** Public to the internet,
  beta, needs a certificate that publishes the machine name, and strips the
  identity the dashboard's gate depends on.
- **Tailscale Serve's identity headers as the dashboard's key.** Serve
  removes client-sent copies, but the backend still sees only the local
  proxy; the ruling to listen on the node's own addresses and ask the daemon
  stands.
- **mDNS (`.local`) naming for the home-network dashboard.** A dependency and
  a multicast listener for a convenience; the explicit address the ADR asks
  for is enough for a QR code.
- **Automatically generated release notes as the release page.** A
  pull-request list grouped by label; the record's page is composed from
  press releases and must cite records, not pull requests.
- **A prompt library (huh) for the menu.** Cited for its accessible mode as a
  convention; the record already counts its module cost against one menu.
- **A reusable workflow in abcd's repository as the only Pages route.** Every
  project's site would then build against abcd's default branch; the ruling
  says the project's own workflow builds it.
- **Roles from GitHub for the dashboard's views.** Settled by the roles note:
  no plan-independent signal exists.

## Not verified

- Whether Tailscale's lookup reports a shared-in device's `UserProfile` with
  the sharer's login (the sharing page says the machine "appears as the
  sharer"; the lookup's behaviour for such a node was not on any page
  opened). Test it before the gate relies on it.
- The `tailcfg.PeerCapMap` and `PeerCapability` doc comments: the package
  page and the source file were opened but the definitions sit past the
  portion read; the `CapMap` meaning is taken from `apitype.WhoIsResponse`.
- Which deployment-branch policy GitHub applies to a `github-pages`
  environment it creates for a custom workflow: the publishing-source page
  recommends a default-branch rule and the environments page says an
  environment created by a workflow has no rules; the one failure report is a
  single project's. Check the environment after the first run.
- Hugo's `exampleSite` convention: the submission guideline page opened
  names `theme.toml` and screenshots but not an example site.
- The mDNS RFC and whether any Go standard-library package answers `.local`
  names: asserted from knowledge of the standard library, not from a page
  opened.
- A convention for deriving a per-project accent colour from a name
  (identicon-style hashing): not researched in this pass.
- Jupyter Server's `IdentityProvider.token` and cookie options: the
  configuration page marks the `ServerApp` fields deprecated in favour of
  these, which were not on the page.
- Chrome's exact release that shipped the Local Network Access prompt: the
  post opened says "launching in Chrome 142"; third-party pages giving dates
  were not opened.
- The CodeMirror collaboration design, Yjs forum threads and the GitLab
  conversion measurement: cited through the co-editing note of 2026-10-14,
  not reopened.
