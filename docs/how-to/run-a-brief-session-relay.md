# Run a brief session relay

A team edits its brief together live through a relay the team runs itself.
abcd runs no relay and holds no key: the relay is part of the `abcd` command,
run either on the host's own computer or as the team's own small web service.
Either way it keeps each session's edits in memory only, drops them when the
host closes the session or nobody has been connected for half an hour, writes
nothing to disk and logs nothing.

The people who join never see any of this. The session page finds the relay
itself, never asks anyone for a server address or a key, and opens straight
onto the brief's press release.

Pick one route, or set up both:

- [On your own computer, over Tailscale](#on-your-own-computer-over-tailscale):
  for a project whose brief should not leave your tailnet. Your computer runs
  the relay and serves the page itself, so nothing goes to abcdesign.app.
- [On the team's own relay, on Render](#on-the-teams-own-relay-on-render): one
  deploy serves all of the team's sessions, the page is at abcdesign.app, and
  anyone with the invite link can reach it.

Both routes sign team members in through the team's own
[GitHub App](#set-up-the-teams-github-app). Each team registers its own; abcd
owns no shared app.

## Set up live sessions once

Repository setup asks about live sessions: `abcd ahoy install` asks
"Set up live brief sessions?" and, on a yes, "Where may sessions run?"
(Local, Cloud or Both). It records the answer in `.abcd/config.json`. A no is
never asked again; deciding later asks again at the next setup run and when
someone runs `abcd session start`. To answer without being asked, pass
`--live-sessions local`, `cloud`, `both` or `none` to `abcd ahoy install`.

Then, in the project's checkout, in a Terminal window of its own:

```bash
abcd session setup
```

It works by hand at any time, after a no too; `--where local|cloud|both`
says where sessions may run when nothing is recorded yet. It does the steps
below in order and prints what is left to you:

1. It prints an address on your computer. Open it: GitHub opens with the
   team's sign-in app filled in. Check its name and click
   **Create GitHub App**. abcd keeps only the app's public client id, in
   `.abcd/relay/github-app.json`; it never keeps or shows the app's secrets.
   Creating an app for an organisation takes someone who may manage its apps.
2. Install the app on the repository, on the page it names.
3. For local sessions, tick **Enable Device Flow** on the app's settings page,
   which it names; GitHub's pre-filled form cannot tick it. For the cloud
   alone, leave it off.
4. For the cloud, commit and push the files it writes, deploy the relay with
   Render's **Deploy** button, paste the app's client secret into Render's
   form, and record the relay's address, as
   [the Render route](#on-the-teams-own-relay-on-render) describes.

## On your own computer, over Tailscale

1. In the project's checkout, start the session in a Terminal window of its
   own:

   ```bash
   abcd session start
   ```

   It listens on this computer alone, at `127.0.0.1:8787` (`--port` names
   another), serves sessions for the repository the checkout's `origin` names
   on GitHub, and prints two steps. It runs until you press Ctrl-C, and every
   session ends then. It needs the GitHub App's client id, which it reads from
   `.abcd/relay/github-app.json` (or `--client-id`), and Tailscale running on
   this computer, whose name it reads without changing anything.

2. The first step it prints shares it on your tailnet with Tailscale Serve:

   ```bash
   tailscale serve --bg 8787
   ```

   Run it once; Tailscale keeps it until you turn it off with
   `tailscale serve --https=443 off`. Use Serve only, never Funnel: Funnel
   would put your computer on the open internet.

3. The second step is the link to open, `https://<your computer's tailnet name>/<owner>/<repo>/brief/`.
   Sign in with GitHub: the page shows a short code, you confirm it on
   github.com, and no secret ever sits on your computer. Start the session,
   and the page shows a code, a QR code and a link to copy for inviting
   people.

Teammates on your tailnet open the invite link. Someone outside your tailnet,
such as a domain expert or a stakeholder, can join once you share this
computer with them through Tailscale's sharing; they join as a guest.

The relay answers only your computer's tailnet name and its own loopback
address, and admits only the page it serves itself.

## On the team's own relay, on Render

`abcd session setup` does steps 1 and 3 itself, with the team's client id
already in `render.yaml`, and prints the rest.

1. In the project's checkout, write the blueprint:

   ```bash
   abcd session blueprint --release <vMAJOR.MINOR.PATCH>
   ```

   Name the abcd release the relay should run: one that includes the
   `abcd session` command, since the relay runs it. Left out, the release is
   the abcd you are running; an older release is refused. The command writes
   `render.yaml` at the repository's root and `.abcd/relay/Dockerfile`. The
   Dockerfile downloads that release's linux/amd64 binary and checks it
   against the checksum the release published, which the command reads from
   the release page and pins in the file. It never overwrites a file you
   edited.

   `render.yaml` sits at the root because Render's Deploy button reads the
   Blueprint there and takes no other path. If your repository already has a
   `render.yaml` of its own, abcd leaves it alone and writes its Blueprint to
   `.abcd/relay/render.yaml` instead.

   `render.yaml` also names, in `ABCD_SESSION_ORIGINS`, the session pages the
   relay admits: `https://abcdesign.app`, unless you serve the page yourself,
   in which case name your site's https address with
   `--origin https://<your site>` (repeatable). Once the team's GitHub App is
   recorded in `.abcd/relay/github-app.json`, it carries the app's public
   client id too.

   The relay serves sessions only for this repository, which `render.yaml`
   names in `ABCD_SESSION_REPOS` from the checkout's `origin`. To serve more,
   edit that value: a comma list of `owner/name`, or an owner alone for all of
   its repositories. A relay that is not on your own machine refuses to start
   without it, so nobody else can hold its sessions with a repository of their
   own.

2. Commit the files and push them.

3. Deploy with Render's Deploy button, the link the command prints:
   `https://render.com/deploy?repo=https://github.com/<owner>/<repo>`. Render
   creates one free web service that runs `abcd session serve` behind its own
   proxy. For a private repository, install Render's own GitHub App on it
   first. With the Blueprint under `.abcd/relay/`, choose **New**, then
   **Blueprint** on Render instead, pick the repository, and set the
   Blueprint Path to `.abcd/relay/render.yaml`.

4. Render's form asks for `ABCD_SESSION_GITHUB_CLIENT_SECRET`, and for
   `ABCD_SESSION_GITHUB_CLIENT_ID` where none was recorded. On your
   [GitHub App](#set-up-the-teams-github-app)'s settings page, under
   **Client secrets**, generate a new client secret and paste it straight into
   that field: it never passes through abcd. With both, team members sign in
   with one click. Without them, nobody can sign in, and only guests can join.

5. Once Render shows the service's address, name it as the relay the
   repository trusts, and commit the file this writes:

   ```bash
   abcd session setup --relay-origin https://<your service>.onrender.com
   ```

   (`abcd session blueprint --release <vMAJOR.MINOR.PATCH> --relay-origin …`
   does the same.) It adds `.abcd/relay/relays.json`, which the session page
   reads from the repository to find the relay that belongs to it. Nobody
   types the address anywhere.

6. The command also prints the relay's sign-in callback,
   `https://<your service>.onrender.com/auth/callback`. On your
   [GitHub App](#set-up-the-teams-github-app)'s settings page, under
   **Identifying and authorizing users**, list it as a **Callback URL**, with
   wildcard matching off, and save. GitHub sends a sign-in only to a callback
   URL the app lists, so this ties every sign-in to your relay. Until it is
   listed, signing in on the relay does not work.

The blueprint turns automatic deploys off: a session lives in the relay's
memory, so a redeploy ends every open session. Deploy by hand between calls.
A free Render service also sleeps after a quiet spell and wakes on the next
visit, and a relay that slept holds no session.
To move to a newer abcd release, run `abcd session blueprint` again with that
release; a file you have not edited is replaced.

## Set up the team's GitHub App

A team member proves they can change the repository by signing in with GitHub,
through a GitHub App the team registers once. Nobody pastes a token: the relay
takes a GitHub App's sign-in and nothing else. An App's sign-in reaches only
the repositories the App is installed on, so even a relay you did not mean to
trust cannot use it on your other repositories. The relay asks GitHub for the
person's login and whether they may push to the repository, and keeps neither
the sign-in nor anything else once it has the answer.

`abcd session setup` registers it from a pre-filled form and records its
client id; the steps below are what it fills in, for registering it by hand.

1. On GitHub, register a new GitHub App under the team's organisation or your
   account. Give it exactly the repository permissions the page uses, and no
   others: **Contents** read and write, **Pull requests** read and write, and
   **Metadata** read. It needs no webhook. Install it on the repositories your
   sessions are for, and only those.
2. For sessions on the team's relay: set its callback URL to the relay's own
   address followed by `/auth/callback`, such as
   `https://<your service>.onrender.com/auth/callback`, with wildcard matching
   off. GitHub returns the person there after they sign in; the relay sends
   them back to the session page, and the page hands GitHub's code to that
   relay and no other. The session page's own address is not a callback.
   Generate a client secret and give both to the relay through its
   environment:
   - `ABCD_SESSION_GITHUB_CLIENT_ID`: the client id, which is public.
   - `ABCD_SESSION_GITHUB_CLIENT_SECRET`: the secret. The relay reads it by
     this name, adds it only to its own exchange with GitHub, and never logs
     it or sends it to a browser.
3. For sessions on your own computer: tick **Enable Device Flow** in the app's
   settings, and record its client id in the repository, with no secret:

   ```json
   {"client_id": "<the app's client id>"}
   ```

   as `.abcd/relay/github-app.json`. `abcd session start` refuses a file that
   holds a `client_secret`. Leave the device flow off if you only use the
   team's relay: anyone who knows the public client id can start a device
   sign-in and ask someone to confirm its code. Only `abcd session start`
   offers the device flow; the team's relay (`abcd session serve`) signs in
   through the web flow alone, and without the secret it offers no sign-in.

An OAuth app is not taken: its sign-in asks for access to every repository
the person can reach. A relay set up earlier with
`ABCD_SESSION_GITHUB_APP_TYPE=oauth-app` refuses to start; register a GitHub
App as above, give the relay its client id and secret, and remove the
variable. Where sign-in is not set up, the page says so and to ask the
technical facilitator; guests can still join.

## Name your relay in the repository

A link to the session page can name any relay, so the page on abcdesign.app
does not take the link's word for it. Before any sign-in or guest's name goes
to a relay, the page reads `.abcd/relay/relays.json` from the repository's
default branch:

```json
{"relays": ["https://<your service>.onrender.com"]}
```

Commit this file naming the https address of your team's relay: the one
`abcd session blueprint --relay-origin` writes. Each entry is an origin alone,
with no path. A page your own computer serves with `abcd session start` needs
no entry: it is its own relay.

- For a public repository the page reads the file without signing in, and
  uses the first relay it names. An invite link naming a relay the file does
  not name gets no sign-in, no guest's name and no connection: the page says
  the link does not lead to this project's own session.
- For a private repository, or one without the file, GitHub answers that
  there is no file to read, and the page cannot check the link's relay. It
  names the project and asks one plain question before a guest joins: "Only
  continue if someone on the project's team sent you this link." The answer
  holds for that browser tab, for at most twelve hours. Team members then
  sign in with GitHub as they would on a public repository: GitHub returns
  them only to the callback your app lists, and the page hands the sign-in to
  that relay alone, whatever relay the link named.
- With no relay in the link and none in the file, the page says that live
  sessions are not set up for this project yet, and to ask the technical
  facilitator.
- If GitHub does not answer at all (it refuses, limits the page's requests, or
  fails), the page cannot check the relay, and refuses it without asking:
  "try again in a moment".

Keeping the relay and its secret safe is the team's responsibility.

## Serve the session page

The page everyone opens is a few plain files that abcd writes for you to put
on a static host of your choice.

1. Write them into an empty directory:

   ```bash
   abcd session page --out site
   ```

   The command writes `index.html`, an identical `404.html`, the page's script
   and stylesheet, and the editor bundle under `vendor/`. It refuses a
   directory that holds anything else.

   If the site is served below a path rather than at the root of its domain,
   such as a project's GitHub Pages address `owner.github.io/<repo>/`, name
   that path with `--base-path <repo>`, and the page opens below it.

2. Publish that directory on any static host, such as GitHub Pages. The host
   must answer an unknown path with `404.html`: that is how
   `/<owner>/<repo>/brief/` reaches the page.

3. Open `https://<your site>/<owner>/<repo>/brief/`. The page finds the relay
   in the repository's `relays.json`. If you set up a sign-in app, its
   callback URL is still the relay's own `/auth/callback`, not your site.

4. Start the relay with `--origin https://<your site>`, or, on the team's
   relay, write the blueprint with `--origin https://<your site>`, which sets
   `ABCD_SESSION_ORIGINS` in `render.yaml`. By default a relay admits a page
   from `https://abcdesign.app` or from this computer; the origins you name
   replace that list, so the relay then admits your page alone.

The page will not run inside another site's frame. A static host cannot tell
browsers to refuse framing, so the page refuses it itself and shows only a
link that opens it in its own tab.

### The page's two designs

The session page comes in two designs, so your team can try both:

- **Guided**, the default: large cards with numbered areas, and a "What
  happens next" note on each step.
- **Shared**: a team bar along the top showing who is here, with the brief
  beneath it; the documents and the suggestions sit in panels at either side,
  and inviting people and sending to GitHub are icon buttons in the bar.

Press Ctrl+[ or Ctrl+] to switch between them; it is Ctrl on every computer,
a Mac included. The page remembers each person's choice in their own browser.
In either design, Alt+Shift+D shows or hides the documents and Alt+Shift+S
the panel on the right, and Escape closes whatever is open on top. On a
phone, the panels open as sheets from the bottom of the screen.

## Who can join

- Anyone the host gives the invite link or the join code to joins as a guest,
  with just a name. A guest sees the brief and everyone's changes, and
  suggests changes for a team member to apply or decline.
- A participant who signs in, and whom GitHub reports as able to push to the
  repository, joins as a team member and edits directly.

Someone whose connection drops, or who reloads the page, comes back as the
same participant without the code, even after the host has chosen a new one:
the relay gives each participant's page a rejoin token, which stops working
when the session ends.

Twenty wrong join codes or rejoin tokens from one address within ten minutes
stop that address from joining until the ten minutes pass. An IPv6 address
counts by its /48, the block a site is usually given. On your own machine,
behind Tailscale Serve, the address is each visitor's own tailnet address, so
one teammate's mistyped codes never lock out the rest. Past six hundred wrong
codes from everywhere together within ten minutes, joining by code pauses for
everybody for ten minutes; someone coming back with their rejoin token, or a
signed-in team member, still gets in.

A session lasts at most twelve hours from when the host opened it. Then it
ends for everyone, as if the host had ended it, and the host can open a new
one. A rejoin token brings back the role its participant had, so this is also
the longest a team member whose access was removed on GitHub keeps editing a
session they were already in.

The relay also bounds how fast anyone can send. One address opens at most three
sessions in ten minutes. On one connection, a team member's edits are held to
sixty a second, cursor and selection updates to twenty a second and a guest's
suggestions to one a second, each with a burst allowed. A message over its
limit is dropped with a `rate-limited` error, and the connection stays open.
A cursor update is at most 8 KiB. A guest's suggestion is at most 70 KiB, and
one connection makes at most a hundred, a session holding at most 1 MiB of
them. A connection that keeps sending messages the relay refuses is cut off
after a thousand. One address may sign in through the relay at most ten times
in ten minutes, and the relay holds at most 64 sign-ins waiting for a code to
be confirmed on GitHub, four of them from one address. A sign-in the person
stops on the page, or leaves the page during, is ended at once and frees its
place; past the four, the page says too many sign-ins are in progress and to
wait a few minutes.

It bounds connections too: 256 in all and 16 from one address. Past either, a
new connection is turned away, and a connection that has not hosted or joined
a session within thirty seconds is closed. A connection whose page stops
reading is dropped once 8 MiB is waiting for it.
