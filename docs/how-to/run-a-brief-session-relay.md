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
   then.

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

2. Commit both files and push them.

3. On Render, choose **New**, then **Blueprint**, pick the repository, and set
   the Blueprint path to `.abcd/relay/render.yaml`. Render creates one free web
   service that runs `abcd session serve` behind its own proxy.

4. In the service's **Environment**, set `ABCD_SESSION_GITHUB_CLIENT_ID` and
   `ABCD_SESSION_GITHUB_CLIENT_SECRET` from your
   [sign-in app](#set-up-github-sign-in), or leave both empty to go without
   sign-in.

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

1. On GitHub, register a new OAuth app under the team's organisation or your
   account. Set its callback URL to the address the session page is served
   from, `https://abcdesign.app/`.
2. Note the app's client id, and generate a client secret.
3. Give both to the relay through its environment:
   - `ABCD_SESSION_GITHUB_CLIENT_ID`: the client id, which is public.
   - `ABCD_SESSION_GITHUB_CLIENT_SECRET`: the secret. The relay reads it by
     this name, adds it only to its own exchange with GitHub, and never logs
     it or sends it to a browser.

Keeping the relay and its secret safe is the team's responsibility.

## Who can join

- Anyone the host gives the join code to joins as a guest. A guest sees the
  brief and everyone's changes, and proposes changes for a team member to
  approve.
- A participant who signs in, and whom GitHub reports as able to push to the
  repository, joins as a team member and edits directly.

Twenty wrong join codes from one address within ten minutes stop that address
from joining until the ten minutes pass.
