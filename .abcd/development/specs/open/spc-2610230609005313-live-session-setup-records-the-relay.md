---
id: spc-2610230609005313
slug: live-session-setup-records-the-relay
intent: itd-2610091016262335
origin: researcher-authored
production_mode: hand-written
---
# Live-session setup on the cloud in the two steps the intent promises

## Summary

Filed 2026-10-23 by the remainder close of spc-2610141709430146, on abcd-ae's ruling as coordinator (dcn-2610230609300445). Its open questions were settled on 2026-11-24 under `facilitator.mode auto` (dcn-2611241230044952).

spc-2610141709430146 delivered live brief sessions. Its intent-fidelity
audit (Opus, with a Sonnet second; receipt rcp-a260a0a30612) found one
acceptance criterion not met, and this spec carries it. The criterion, in the
product thinker's words:

> (setup) **Given** Cloud, **when** the technical facilitator sets it up,
> **then** it takes two one-click steps: creating the team's GitHub App from a
> pre-filled manifest, and deploying the relay with Render's Deploy button,
> whose own form asks for the secrets. abcd writes the files and records the
> relay's address itself.

What shipped takes about six steps:
1. create the app;
2. install it;
3. generate a client secret and paste it into Render's form;
4. deploy;
5. re-run `abcd session setup --relay-origin <address>` by hand;
6. add the relay's callback URL on the app's settings page.

abcd writes `relays.json` only from an address the person types. This spec closes that gap, so that the setup is the two
one-click steps and abcd learns the relay's address itself.

The same audit found ac-6 (a guest from another tailnet joining a local
session through Tailscale) inconclusive: never exercised. Its live proof is
iss-2610161444081367, outside this spec.

## Approach

Settled 2026-11-24 by the technical facilitator (dcn-2611241230044952).

- **Setup assigns the relay's address; it does not learn it.** `abcd session setup` draws
  the Render service's name once (`abcd-relay-` and 20 base32 characters) and writes the
  address `https://<name>.onrender.com` into the app's manifest as its callback, into the
  blueprint, and into `.abcd/relay/relays.json`. A later run reuses the name
  `relays.json` holds. A `relays.json` that names any other relay is left alone, with the
  service's earlier name, so a relay deployed before this change is never renamed.
- **The relay holds itself to that address.** The blueprint starts it with its own
  address. Where the host states the service's public address and it differs, the relay
  refuses to start and names `abcd session setup --relay-origin`. An abcd release that
  does not know the flag fails to start, so the blueprint refuses to pin one older than
  the abcd writing it.
- **The page holds a listed relay to the repository's app.** A relay taken from the
  repository's `relays.json` must name the client id in `.abcd/relay/github-app.json`
  beside it and offer the web flow, or the page signs nobody in through it.
- **Creating and installing the app are one trip.** After GitHub creates the app, setup's
  page sends the browser to GitHub's install page with the repository chosen. For a
  repository whose id cannot be read without signing in, it sends the browser to the
  plain install page, never to the preselecting address without an id.
- **The secret stays one paste**, from the app's settings page into Render's form, whose
  only question it is. abcd reads, prints and stores no secret, as before.
- `--relay-origin` stays for a custom domain, another host, and a name Render did not
  give.
- Records: a dated amendment to adr-2610161400542916 in the change that lands steps 2
  and 3. No new ADR.

## Open Questions

- resolved: How abcd learns the relay's address. It assigns it before the app is created,
  and the relay refuses to run elsewhere; settled 2026-11-24 (Approach).
- resolved: Whether installing the app and listing the callback fold into the manifest
  step. The callback is in the manifest; the install follows in the same trip, with the
  click GitHub requires; settled 2026-11-24 (Approach).
- resolved: Whether the client secret can reach Render without being copied by hand. No:
  one paste into Render's own form stays, and abcd never sees it; settled 2026-11-24
  (Approach).

## Footprint

- packages: internal/surface/cli/session.go, internal/surface/cli/session_setup_test.go, internal/surface/cli/session_surface_test.go, internal/surface/session/server.go, internal/surface/session/server_test.go, internal/surface/session/appsetup.go, internal/surface/session/appsetup_test.go, internal/surface/session/web/session.js, internal/surface/session/web/web_test.go, internal/surface/session/web/signin_bind_test.go, internal/core/session/blueprint.go, internal/core/session/blueprint_test.go, internal/core/session/setup.go, internal/core/session/setup_test.go, internal/core/session/testdata/blueprint/render.yaml.golden, internal/core/session/testdata/blueprint/Dockerfile.golden, internal/core/session/testdata/blueprint/relays.json.golden, commands/session.md, docs/how-to/run-a-brief-session-relay.md, .abcd/development/brief/04-surfaces/37-session.md, .abcd/development/decisions/adrs/2610161400542916-the-brief-session-page-asks-nobody-for-a.md
- tests: internal/surface/session, internal/surface/session/web, internal/core/session, internal/surface/cli
- Generated output (`docs/reference/cli/commands.md`,
  `.abcd/development/release/surface.json`, `docs/reference/cli/commands/session.json`)
  is reserved by no one and is regenerated on the branch.

## Steps

1. The relay holds itself to the address it was given
   - packages: internal/surface/session, internal/surface/cli
   - tests: `session serve --own-origin` (also `ABCD_SESSION_RELAY_ORIGIN`) with the host's stated address equal starts; a different stated address refuses to start, exit 2, naming both addresses through the terminal sanitiser and `abcd session setup --relay-origin`; no stated address starts and says the address was not checked; an origin `NormalizeRelayOrigin` refuses is a usage refusal; `/config` gives the relay's own origin when it was given one and omits it otherwise; a loopback relay ignores the flag's absence
2. Setup assigns the relay's address and the blueprint carries it
   - packages: internal/core/session
   - tests: the name is `abcd-relay-` and 20 base32 characters from the given random source, and two draws differ; with no `relays.json` the blueprint names the service by it, carries `ABCD_SESSION_RELAY_ORIGIN` as a plain value, starts the relay with `--own-origin`, and writes `relays.json` naming the address; a second run reuses the name `relays.json` holds and reports every file unchanged; a `relays.json` naming any other relay keeps the service's earlier name and writes no own-origin value; `--relay-origin` wins over an assigned name and says the callback to list; a pinned release older than the running abcd is refused with nothing written; an edited file is still refused whole; the three golden files
   - needs: 1
3. Setup is one trip to GitHub, and the cloud is two steps
   - packages: internal/surface/session, internal/core/session, internal/surface/cli
   - tests: the blueprint's files are judged before GitHub is asked for anything, and a refusal there creates no app (iss-2610230519116460); the manifest of a cloud setup lists the assigned address's callback and a local one lists none; after the conversion the loopback page answers with a redirect to GitHub's install page carrying the owner's id and the repository's id; a repository whose id cannot be read is sent to the plain install page and never to the preselecting address; the redirect's host is github.com and nothing else; a failed conversion redirects nowhere; a cloud run with no `--relay-origin` writes `relays.json` and prints two steps to do (commit and push, then Render's button with the one secret) and no step that names an address to type or a callback to list; an app recorded by an earlier setup is told the one callback to list; a later run reads the relay's `/config` and says whether a relay answers there with this project's client id, without changing any file on a mismatch; no output or JSON field of any run carries a secret
   - needs: 1, 2
4. The page holds a listed relay to the repository's app
   - packages: internal/surface/session/web
   - tests: a relay taken from the repository's `relays.json` whose `/config` names another client id than `.abcd/relay/github-app.json` gets no sign-in, no token and no session, and the page says in plain words that the project's session is not answering as the project's own and to ask the technical facilitator, showing no address; the same for a listed relay offering any flow but the web flow; a repository with no `github-app.json` behaves as today; a relay the person confirmed (the list could not be read) behaves as today; a `github-app.json` that carries a `client_secret` is refused as the relay refuses it
5. The command page, the how-to, the brief's session chapter and the decision record say what the code does
   - packages: commands/session.md, docs/how-to/run-a-brief-session-relay.md, .abcd/development/brief/04-surfaces/37-session.md, .abcd/development/decisions/adrs
   - tests: `go test -run 'Session|Surface|Reference|CommandFacts' ./internal/surface/cli/` after regenerating; `make precheck` (docs-lint, record-lint)
   - needs: 3, 4
6. One live deploy shows the two steps
   - packages: .abcd/development/research/notes
   - tests: on a real Render account and a scratch repository, from a cut release that carries steps 1 to 4: setup, the two steps, a member signs in on abcdesign.app and a guest joins, with no address typed and no callback listed by hand; the note records the address Render gave beside the one assigned, and the count of clicks and pastes. A person makes the deploy; an agent cannot hold a Render account.
   - needs: 5
   - waits on: the first release cut after steps 1 to 4 land (the relay's image downloads a released binary)

## Size

Three pull requests, about 24 agent-hours and under an hour of a person's time. Steps 1
and 4 are small and touch one file of code each. Step 2 changes a pure renderer and its
three golden files. Step 3 is the largest: it reorders `abcd session setup`, which runs to
about 260 lines in one function. Step 6 is a note and the close, and waits on a cut.
