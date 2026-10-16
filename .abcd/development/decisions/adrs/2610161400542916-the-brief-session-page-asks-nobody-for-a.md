---
id: adr-2610161400542916
slug: the-brief-session-page-asks-nobody-for-a
status: accepted
date: 2026-10-16
supersedes: null
superseded_by: null
related_intents: [itd-2610091016262335]
related_rfcs: []
related_adrs: [adr-2610091016270711, adr-2610032150581128, adr-2610141949110653]
---

# ADR-2610161400542916: The brief session page asks nobody for a relay address or a token, and a local session signs in by GitHub's device flow

## Context

The product thinker tested the draft brief session page on 2026-10-16 and could not use it (iss-2610161332024554). Its first screen asked for a "relay address", then for a pasted GitHub personal access token. The brief appeared only after hosting and picking a file. The page's audience is product thinkers and domain experts, not the people who set the project up.

The same day the product thinker approved a new design. It is recorded, restated, in itd-2610091016262335's Decisions:

- The page never shows or asks for a relay address. It never asks the person for technical input of any kind.
- Nobody pastes a token. The paste is removed from the page and from the relay.
- Once a session is joined, the page opens straight onto the brief, press release first, in plain words.
- There are two ways to host, and both stay. In the cloud, the team's relay runs on its own hosting account and the page is at abcdesign.app; members use "Sign in with GitHub". Locally, for sensitive projects, the session runs on the host's own computer, reached over Tailscale Serve on the tailnet only. That computer runs the relay and serves the page itself, so nothing goes to abcdesign.app. Sign-in uses GitHub's device flow, so no secret sits on the host's computer.
- Org members sign in and edit. Everyone else joins as a guest with a name and suggests changes, as the draft has it.

adr-2610091016270711 decision 2 kept "pasting a personal access token is the fallback". The 2026-10-14 amendment of adr-2610032150581128 admits the relay as abcd's second listener and has team membership decided "by GitHub's own answer on the presented token". The spec had the page read the relay address from a field or a link, and confirm by showing its origin. This record replaces those parts. It keeps every other term of both records.

Three technical questions had a state-of-the-art answer, and the product thinker allows those to be decided on their grounds.

## Decision

We will build the session as follows.

1. **The page finds its relay itself.**
   - Served by the relay (a local session), the page takes its own origin as the relay. The page and the relay are one server, so there is nothing to check.
   - On abcdesign.app, the page uses the relay the invite link names, if the repository's committed `.abcd/relay/relays.json` names it. With no relay in the link, it uses the first relay the file names.
   - Where the file cannot be read without signing in (a private repository), the page asks one plain question before anything reaches the link's relay: whether someone on that project's team sent the link. A relay the person confirmed is remembered for that repository. (**Amended 2026-10-16:** for the browser tab and at most twelve hours; see the amendment below.)
   - With no relay anywhere, the page says that live sessions are not set up for this project yet, and to ask the technical facilitator.
   - No relay address appears on the page, in its words, in a field, or as link text.

   The trust rules of the draft hold: the repository is the anchor (security review F1); the list is read fail-closed (finding 5); a list the token can see decides; and the invite link carries a relay only when the guest's page could not find the same one itself.
2. **No token is pasted anywhere.** The page has no token field. The relay's `verify` takes only a GitHub App user access token, whose documented prefix is `ghu_` (https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/about-authentication-to-github#githubs-token-formats). It refuses any other token before asking GitHub, so a personal access token, an OAuth app's token or an installation token is never accepted.
3. **The relay offers one of two sign-ins, chosen by how it is set up.** Given the GitHub App's client id and secret, it offers GitHub's web flow, as the draft does (`/auth/token`). Given the client id alone, it offers GitHub's device flow:
   - `/auth/device` asks GitHub for a device code and keeps it in memory under a random handle. It returns only the handle, the user code, GitHub's verification address and the timings.
   - `/auth/device/token` polls GitHub for that handle, no faster than GitHub's interval, and returns the token once the person has confirmed the code on github.com.
   - The device code never reaches the browser. Both routes check the page's origin, and starting a device sign-in is limited per address as the exchange is.
   - The relay holds a bounded number of pending sign-ins, each dropped when GitHub's code expires.
4. **A cloud relay keeps the web flow** (decision on state-of-the-art grounds, below).
5. **A local session is one command, `abcd session start`.** It runs the relay on loopback with the device flow and serves the page from the relay itself. It admits only its own tailnet name and loopback, as hosts and as page origins. It prints the one Tailscale Serve step and the link to open. It reads the GitHub App's public client id from `.abcd/relay/github-app.json`, which repository setup writes, or from `--client-id`. It holds no secret.
6. **The relay does not use Tailscale Serve's identity headers** (decision on state-of-the-art grounds, below).

### Decided on state-of-the-art grounds

- **Cloud sign-in stays on the web redirect flow.** GitHub's device flow needs no client secret ("The `client_secret` is not needed for the device flow", https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps). GitHub's web flow still requires the secret for a GitHub App, PKCE or not (https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app). GitHub has no public-client concept, and its roadmap item for single-page-app support was paused on 2026-08-13 (https://github.com/github/roadmap/issues/1153). So the device flow would free the cloud relay of its one secret. It is still not chosen there, for three reasons:
  - It is open to remote phishing (RFC 8628 §5.4, https://www.rfc-editor.org/rfc/rfc8628#section-5.4). Anyone can start a device flow with the team's public client id and ask a member to enter the code. The web flow's redirect is bound to the app's registered callback, so a stolen code is useless without the secret. **Amended 2026-10-16:** that holds only where the registered callback is the relay that exchanges the code; see the amendment below.
  - GitHub limits device-flow code submissions to 50 an hour per application.
  - It sends every member to a second tab to type an eight-character code, where the web flow is one click.

  RFC 10017 (OAuth 2.0 for Browser-Based Applications, BCP 212) recommends a backend that holds the client's credentials. The relay's `/auth/token` is that backend, and it already holds the secret on the team's own account. On the host's own computer the trade reverses: the product thinker ruled that no secret sits there, so the device flow is the only GitHub sign-in that fits.
- **Tailscale Serve's identity headers are not used.** Serve adds `Tailscale-User-Login`, `Tailscale-User-Name` and `Tailscale-User-Profile-Pic` for a device that is not tagged, shared-in users included, and strips copies a client sends (https://tailscale.com/kb/1312/serve). They are still not used, for three reasons:
  - A Tailscale login is not a GitHub login, and it says nothing about write access to the repository, which is what membership means here.
  - The relay listens on loopback. Tailscale's page notes that a service on loopback can still be reached, headers and all, by other programs on the same computer.
  - adr-2610032150581128 already rules that a Tailscale identity is a label, never a key.

  Membership stays GitHub's answer on a GitHub App token. A guest's name stays the one they type.

### Amendment of 2026-10-16: the device flow is offered only where the relay serves its own page

Amended in the open, on review of the change that lands decision 3. Every term above holds, with these narrowings:

- Decision 3's device flow is offered only by a relay that serves its own session page, which is `abcd session start` (decision 5). `abcd session serve`, the team's relay, offers the web flow when it holds the secret and no sign-in without it: its `/config` reports sign-in off, and `/auth/device` answers that the device flow is not offered.
- The session page takes a relay's device flow only on a page that relay serves itself. Every local session serves its own page, so the page on abcdesign.app, or on any other origin, signs in through the web flow alone, and a relay that names the device flow there gets no sign-in.
- Beside the bound on pending device sign-ins, one client address holds at most four at once, counted as the exchange limit counts it and never more than that limit allows, so a few addresses cannot take every place.

### Amendment of 2026-10-16: the sign-in code goes only to the relay the app's callback names

Amended in the open, on review of the change that lands decision 3's web flow. It corrects a premise above and narrows decisions 1 and 3.

The premise was that the web flow's redirect, bound to the app's registered callback, keeps the code with the team's relay. With the session page as the registered callback, GitHub binds the code to the page, and the page then handed it to whichever relay it had been started with, which could be a relay the person had confirmed but GitHub had no part in naming. The binding stopped at the page. So:

1. **The callback is the relay's own.** The team's GitHub App lists `https://<relay>/auth/callback` as its callback URL, and the page sends that as the `redirect_uri`, with the page's own address carried in the state. GitHub returns the person to the relay; the relay sends them back to the page, naming its own origin, and only for a session page on an origin it admits and a repository it serves. The page posts the code and its verifier to that relay alone, and refuses a return that names no relay, or one other than the relay the sign-in started with; it never takes the exchange's relay from the invite link or its storage. The relay's `/auth/token` exchanges only a code given for its own callback.
2. **The token goes only to the relay GitHub returned to.** Decision 1's confirmation, for a relay the repository's list cannot speak for, still lets that relay run the web sign-in, so the members of a private repository sign in on abcdesign.app: the callback binding in 1 is the control, since GitHub sends the code only to a callback the team's app lists. The page presents the token for verification only to the relay the sign-in returned through, never to a relay an invite link named that GitHub did not return the person to. The confirmation is no longer remembered for the repository: it is kept for the browser tab and lapses after twelve hours, a session's longest life.
3. **Setup names the callback.** `abcd session setup --relay-origin` and `abcd session blueprint --relay-origin` print the exact callback URL to list, with wildcard matching off; a manifest posted once the relay's address is known sets it. For an app recorded before, setup also says to remove the session page's own address from the app's callback URLs if it is listed, since an earlier setup listed it.
4. **A stopped device sign-in ends at once.** The page tells the relay (`/auth/device/cancel`) when the person stops a device sign-in or leaves the page, and the relay frees its place. An address refused for holding its share is told, in `Retry-After`, when its first sign-in ends, and the page says too many sign-ins are in progress and to wait a few minutes.
5. **The callback forwards only to the page's own base path.** Each origin the relay admits may name the base path its session page is served below (`https://<host>/<base>/`, as `abcd session page --base-path` writes it); an entry naming none, and the default origins, mean the root. The callback forwards a code only to a session page under exactly that path on that origin, since a static origin can carry other publishers' pages beside the session page. The base path is part of the origin entry, not a separate relay-wide setting: it belongs to one place the page is published, and one relay may admit the page at abcdesign.app's root and a team's own copy below a project path.
6. **The callback is spelled as the browser spells an origin.** The relay's address drops the scheme's default port (443 for https), as a browser's origin does (https://url.spec.whatwg.org/#origin), so the callback setup names matches, exactly, the `redirect_uri` the page sends.

**Decided on state-of-the-art grounds:** the relay's own callback rather than the page's address with the relay in its query. GitHub compares a `redirect_uri` with a registered callback exactly only when that callback's wildcard matching is off, and otherwise by host, port and a path beneath the callback's ("When wildcard matching is disabled, the redirect URL must exactly match the callback URL", https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/about-the-user-authorization-callback-url). Its user-token page says a `redirect_uri` "can't contain any additional parameters" and says nothing about a query in a registered callback (https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app). A relay's own host is compared under either mode, so the binding does not depend on undocumented query matching. The authorization code then reaches only a destination GitHub binds, as OAuth 2.0's security best current practice asks of redirect URIs (RFC 9700 §2.1, https://www.rfc-editor.org/rfc/rfc9700#section-2.1).

## Alternatives Considered

- **Keep the relay field and the token paste, with better words.** This is the least work, but it leaves a product thinker facing a server address and a secret. The product thinker rejected it.
- **Device flow everywhere, so no relay holds a secret.** It removes the cloud relay's one secret. It is rejected for the cloud on the grounds above, and chosen for local sessions, where the product thinker ruled out a secret.
- **A shared GitHub App owned by abcd, so a local session needs no app registration.** The device flow needs a client id, and a client id is not a key. But GitHub only lets an app's token reach repositories where the app is installed (https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user), so each organisation would still install it. abcd would then own a registration that every team's sign-in depends on. That is a product decision, not taken here: it is put back to the product thinker. **Amended 2026-10-16:** the product thinker ruled that each team registers its own sign-in app at repository setup, for local sessions as for the cloud, and that abcd owns no shared app (itd-2610091016262335's Decisions). `abcd session setup` registers it from a pre-filled manifest.
- **Admit tailnet visitors as members by their Tailscale identity.** Rejected on the grounds above.
- **The chosen design:** the page finds its relay, nobody pastes anything, the cloud uses the web flow and a local session uses the device flow.

## Consequences

- A product thinker sees the brief's name, a sign-in button and, with an invite, a name field. Once joined, they see the press release.
- **Local sessions still need a GitHub App, registered once.** The device flow needs the client id of a GitHub App with "Enable Device Flow" switched on. A manifest cannot switch that on (https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest), and the app must be installed on the repository. No secret is kept and no server is deployed. But "no account and no cloud setup" overstates it: someone who may create apps for the organisation does this once. Repository setup (a later lane) carries that step.
- A team that allows both kinds of session uses one app, with the device flow switched on. That leaves the cloud open to the device-flow phishing above. The setup step switches the device flow on only when local sessions are allowed, and says why.
- The relay holds a little more state: pending device sign-ins, bounded in number and lifetime, in memory only, never logged.
- A pasted personal access token no longer works anywhere. A team without a GitHub App cannot sign in, and the page tells them, in plain words, to ask their technical facilitator.
- adr-2610091016270711 and adr-2610032150581128 each carry an amendment pointing here.
