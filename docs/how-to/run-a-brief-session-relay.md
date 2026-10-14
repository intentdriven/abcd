# Run a brief session relay

A team edits its brief together live through a relay the team runs itself.
abcd runs no relay and holds no key: the relay is the `abcd session serve`
command, run either on the host's own machine or as the team's own small web
service. Either way it keeps each session's edits in memory only, drops them
when the host closes the session or nobody has been connected for half an
hour, writes nothing to disk and logs nothing.

Pick one route:

- [On your own machine, over Tailscale](#on-your-own-machine-over-tailscale):
  nothing to deploy, and every participant must be on your Tailscale network.
- [On the team's own relay, on Render](#on-the-teams-own-relay-on-render): one
  deploy serves all of the team's sessions, and anyone with the join code can
  reach it.

Both routes use the same [GitHub sign-in app](#set-up-github-sign-in), which is
optional.

## On your own machine, over Tailscale

1. Start the relay in a Terminal window of its own:

   ```bash
   abcd session serve
   ```

   It listens on this computer alone, at `127.0.0.1:8787`, and prints the next
   command. It runs until you press Ctrl-C, and every session it holds ends
   then. Run in the project's checkout, it serves sessions for that
   repository alone, the one its `origin` names on GitHub; name others with
   `--repo owner/name` or `--owner <owner>`.

2. Put it on your tailnet with Tailscale Serve, in a second window:

   ```bash
   tailscale serve --bg 8787
   ```

   Tailscale prints the https address teammates on your tailnet reach the relay
   at. Use Serve only, never Funnel: Funnel would put your computer on the open
   internet. A guest outside your tailnet joins through the team's relay
   instead.

3. When the session is over, stop the relay with Ctrl-C and stop serving with
   `tailscale serve --https=443 off`.

## On the team's own relay, on Render

1. In the project's checkout, write the blueprint:

   ```bash
   abcd session blueprint --release v0.14.1
   ```

   Name the abcd release the relay should run. The command writes
   `.abcd/relay/render.yaml` and `.abcd/relay/Dockerfile`. The Dockerfile
   downloads that release's linux/amd64 binary and checks it against the
   checksum the release published, which the command reads from the release
   page and pins in the file. It never overwrites a file you edited.

   The relay serves sessions only for this repository, which `render.yaml`
   names in `ABCD_SESSION_REPOS` from the checkout's `origin`. To serve more,
   edit that value: a comma list of `owner/name`, or an owner alone for all of
   its repositories. A relay that is not on your own machine refuses to start
   without it, so nobody else can hold its sessions with a repository of their
   own.

2. Commit both files and push them.

3. On Render, choose **New**, then **Blueprint**, pick the repository, and set
   the Blueprint path to `.abcd/relay/render.yaml`. Render creates one free web
   service that runs `abcd session serve` behind its own proxy.

4. In the service's **Environment**, set `ABCD_SESSION_GITHUB_CLIENT_ID` and
   `ABCD_SESSION_GITHUB_CLIENT_SECRET` from your
   [sign-in app](#set-up-github-sign-in), or leave both empty to go without
   sign-in.

5. Once Render shows the service's address, name it as the relay the
   repository trusts, and commit the file this writes:

   ```bash
   abcd session blueprint --release v0.14.1 --relay-origin https://<your service>.onrender.com
   ```

   It adds `.abcd/relay/relays.json`, which the session page reads from the
   repository to know which relay belongs to it.

The blueprint turns automatic deploys off: a session lives in the relay's
memory, so a redeploy ends every open session. Deploy by hand between calls.
A free Render service also sleeps after a quiet spell and wakes on the next
visit, and a relay that slept holds no session.
To move to a newer abcd release, run `abcd session blueprint` again with that
release; a file you have not edited is replaced.

## Set up GitHub sign-in

A team member proves they can change the repository by signing in with GitHub,
through a sign-in app the team registers once. Without one, a team member
pastes a GitHub personal access token instead. Either way the relay asks
GitHub for the person's login and whether they may push to the repository, and
keeps neither the token nor anything else once it has the answer.

1. On GitHub, register a new GitHub App under the team's organisation or your
   account. Set its callback URL to the address the session page is served
   from, `https://abcdesign.app/`. Give it the repository permissions the
   page uses: read and write on contents and on pull requests. Install it on
   the repositories your sessions are for. A GitHub App's sign-in holds only
   what both the app and the person may do, on the repositories it is
   installed on.
2. Note the app's client id, and generate a client secret.
3. Give both to the relay through its environment:
   - `ABCD_SESSION_GITHUB_CLIENT_ID`: the client id, which is public.
   - `ABCD_SESSION_GITHUB_CLIENT_SECRET`: the secret. The relay reads it by
     this name, adds it only to its own exchange with GitHub, and never logs
     it or sends it to a browser.

A team that already has an OAuth app may use it instead: set
`ABCD_SESSION_GITHUB_APP_TYPE` to `oauth-app` as well. The default, and what
the blueprint sets, is `github-app`.

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

3. Open `https://<your site>/<owner>/<repo>/brief/` and give the page your
   relay's https address. If you set up a sign-in app, its callback URL is
   your site's address.

4. Start the relay with `--origin https://<your site>`. By default a relay
   admits a page from `https://abcdesign.app` or from this computer; the
   origins you name replace that list, so the relay then admits your page
   alone.

## Who can join

- Anyone the host gives the join code to joins as a guest. A guest sees the
  brief and everyone's changes, and proposes changes for a team member to
  approve.
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
one teammate's mistyped codes never lock out the rest. Past two hundred wrong
codes from everywhere together within ten minutes, joining by code pauses for
everybody for ten minutes; someone coming back with their rejoin token, or a
signed-in team member, still gets in.

The relay also bounds how fast anyone can send. One address opens at most three
sessions in ten minutes. On one connection, a team member's edits are held to
sixty a second, cursor and selection updates to twenty a second and a guest's
suggestions to one a second, each with a burst allowed. A message over its
limit is dropped with a `rate-limited` error, and the connection stays open.

It bounds connections too: 256 in all and 16 from one address. Past either, a
new connection is turned away, and a connection that has not hosted or joined
a session within thirty seconds is closed.
