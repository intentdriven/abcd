---
id: adr-2610032150581128
slug: abcd-is-reachable-on-a-network-only-through-the-dashboard
status: accepted
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610032150577708, itd-2610032150580455, itd-2610040740108331]
related_rfcs: []
related_adrs: [adr-38, adr-2609091248200336]
---

# ADR-2610032150581128: abcd is reachable on a network only through the dashboard, only while the person runs it, and only for people the person lets in

## Context

The product thinker asked on 2026-10-03 for a web dashboard of an abcd-managed project, reachable from a phone, an iPad and a computer on the local network, and later from anywhere through Tailscale (itd-2610032150577708, itd-2610032150580455). Until now abcd never listens on a network: it makes outbound calls only when asked (adr-38, brief invariant 7) and keeps its files in folders the person handed it. A page anyone on the same network can open is a new trust boundary: a guest on the home Wi-Fi, a device that resolves a hostile page's name to the person's computer, or a link copied into the wrong chat can all reach it. The product thinker confirmed on 2026-10-03 that this rule is its own record, apart from the feature.

## Decision

Proposed by the facilitator; the details are put to the product thinker at the dashboard's planning interview, informed by the state-of-the-art pass:

Accepted on 2026-10-05, in the change that lands the first step of spc-2610040741034208 (itd-2610032150577708 decision 22: a rule is in force when the code that enforces it lands), with brief invariant 7's inbound clause. The interview of 2026-10-04 below settles what this Decision left open, as amended by decisions 21 and 23.

Two people are named apart, as the role model does: the person who starts the dashboard at the computer (often the technical facilitator) and the people they let in (the product thinker on a phone, an iPad or another computer).

- abcd listens on a network only while someone has started the dashboard on purpose, and stops when they stop it. Whether it may run unattended for days (which reach from anywhere, itd-2610032150580455, needs) is decided at the dashboard's interview; it never starts by itself.
- The dashboard is the only listener abcd has. A later one (an MCP server over HTTP, which brief invariant 3 anticipates) is admitted only under these same conditions, by amending this ADR in the open.
- Only the people let in can open it: access is granted by something the person who started it hands over (for example a one-time pairing link), never by being on the same local network alone. Whether being on the person's own private Tailscale network counts as let in is decided at that draft's interview.
- What the page may change, if anything, is decided at the interview; until it is, the page shows and changes nothing.
- Nothing the page serves leaves the person's own devices: no third-party scripts, fonts or trackers.

### The product thinker's interview of 2026-10-04

The dashboard's first version is reached through Tailscale only (itd-2610032150577708 decision 9): it listens on this computer's own Tailscale addresses alone, over plain HTTP with no certificate (decision 23: a certificate would publish the computer's name for good, and Tailscale already encrypts the connection), never through Serve or Funnel (decision 21: Serve hides which device connects); it is started by hand and runs until stopped (decision 12). On Tailscale the person's own devices are let in already (decision 4), which overrides the security review's "pairing still applies" below for the tailnet: the identity is taken from Tailscale's own lookup of the connecting device, never from a header a local program could write. The local-network requirements below (an explicit address, the one-time code, the capped device lifetime of seven days, the idle switch-off, the Wi-Fi warning) govern the later home-network draft itd-2610040740108331. The only writes are notes and "Still right" on the brief, from a device the dashboard let in, with cross-origin protection, held in the checkout's local tier until the technical facilitator reviews them (decisions 11, 16 and 17).

### Requirements the spec carries

_From the security review of 2026-10-03 (Fable 5.1, verdict "block as drafted" until these are carried); each has one defensible answer, so none is put to the product thinker._

- It listens on this computer alone unless an address is chosen on that run; never every address at once, and never an address reachable from the internet without a second, explicit choice. The start line names every address it listens on, and abcd checks that it can reach itself there (a firewall that silently drops the traffic is reported).
- Letting a device in is its own step. Its link is single-use, expires within ten minutes, is never written where a transcript, a log or a screen-share could keep it, and is in the scanner's pattern set.
- A device let in holds a session abcd can list and revoke, with a capped lifetime; pages are never cached.
- A device not let in sees the pairing page and nothing else, and its requests read nothing from the project.
- Only the expected host names are answered; others are refused before anything is read.
- Server timeouts, size limits and a cap on live-update connections; one shared reading of the project, however many devices watch.
- The page is drawn by the record site's one renderer, from records looked up by id; no file is served from a requested path; the content-security policy loads nothing from outside the computer and the page cannot be framed.
- Nothing is changed through the page over plain HTTP.
- On the local network it runs in the foreground, stops itself when idle and after a hard limit, and says so; running unattended is allowed only behind Tailscale Serve (never Funnel), where pairing still applies and the Tailscale identity is a label, never a key.
- The start line says the page is only as private as the network: anyone who can read the Wi-Fi can read it. The person confirms on each run that the network is their own.

## Alternatives Considered

- Never listen, and publish a static site instead (the superseded itd-139's route), or rebuild the existing record site on change and serve the folder: rejected by the record review of 2026-10-03. The site build needs a composition only `site setup` plants, takes minutes, and rewrites a committed file on every build; it shows the facilitator's tiles and record ids, not what waits on the product thinker; and serving the folder still needs a listener and this rule.
- Listen on the network with no access control, as many local development servers do: simplest, and the commonest cause of exposed local dashboards.
- Listen only on this computer and reach it from other devices through a tunnel: safe, but a phone on the home network cannot open it without extra software, which the request rules out until Tailscale.
- The proposed rule above: reachable on the network, on purpose, and only for people let in.

## Consequences

- abcd gains its first listening surface. Changes to it are reviewed as the principle adversarial-review-scales-with-blast-radius requires, and the spec states its bind address, its access check and its limits.
- The rule is held by tests, not by prose: one package alone may open a listener (the shape of invariant 15's store-path test), and a test proves the page loads nothing from outside the person's computer.
- This refines adr-38 and brief invariant 7 with an inbound clause; it does not supersede adr-38, whose outbound rule stands.
- Pairing state lives under abcd's home (adr-2609091248200336), never beside the project.
- Brief invariant 7's network trichotomy gains an inbound clause when this is accepted; the brief's invariants are edited in the change that accepts it.
- Tailscale (itd-2610032150580455) is a second way in under the same rule.
