---
name: update
description: "Swap the PATH-installed binary for a verified release, or with --check only compare: Writes the swapped binary; refuses a binary it cannot prove is abcd's."
argument-hint: "[tag | news]"
block: people
---

# `/abcd:update`

Complete a chosen update of the PATH-installed binary. The verb's documented
meaning IS the fetch: it resolves the latest release (or takes an explicit
tag), verifies the platform binary against the same release's
`checksums.txt` and against the SLSA build provenance the release workflow
signed for it, and swaps the PATH copy atomically, printing a receipt that
opens with `abcd updated from <old> to <new>` and names the path, origin,
digest and the provenance signer. abcd never checks for or
applies updates on its own — this verb reaches the release origin only when
invoked, as does the one other flag that reaches it, `--fetch-baseline` on
`launch` and `launch ship`, which downloads a released plugin archive to
compare against. Its one sub-verb, `news`, fetches and swaps nothing: it says
what an update changed and offers the setup that waits
([below](#news--what-an-update-changed-and-the-setup-that-waits)).

**Why two checks.** `checksums.txt` sits on the same release page as the
binary, so it only proves the download arrived intact: whoever could replace
the binary could replace the manifest with it. The build provenance is the
check that does not trust the page. The update looks up the binary's SHA-256
in GitHub's attestations API (`https://api.github.com/repos/intentdriven/abcd/attestations/sha256:<digest>`,
read without a token), fetches the Sigstore trusted root through TUF from
`https://tuf-repo-cdn.sigstore.dev` (anchored on the root built into the
binary, cached nowhere), and verifies the signed bundle in process: its
certificate must chain to Sigstore's Fulcio CA, its entry must be in the Rekor
transparency log, its signer must be `intentdriven/abcd`'s
`.github/workflows/release.yml` on `main`, run on a GitHub-hosted runner under
GitHub Actions' OIDC issuer, and its SLSA provenance must name exactly the downloaded bytes.
The signature alone says who built the bytes, not which release they are, so
the check is also bound to the release: the tag is resolved to its commit
through GitHub's git API (`https://api.github.com/repos/intentdriven/abcd/git/ref/tags/<tag>`,
following an annotated tag to its commit), and both the certificate's source
digest and the provenance's source commit must be that commit. An older
release's genuine build re-uploaded under a newer tag is refused. A file that
already matches the release's `checksums.txt` passes the same check before it
is reported current. Any failure refuses before anything is staged, a tag
whose commit cannot be resolved included. Every address is reached over the
same pinned transport as the release download.

The same line announces a swap the plugin bootstrap makes. When a hook that
discards its output made the swap, the next session start shows the line once,
and claims that release with an empty file: `update-shown-<new>` in the plugin
data directory's `cache/`, or `.update-shown-<new>` in the plugin root when there
is no data directory. That claim is the only thing the session start writes.

**Only asking.** When the user wants to know whether a newer release exists
without taking it, run the check, which fetches the latest release's tag once
and swaps nothing:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" update --check --json
```

Relay `version`, `vintage` and the `check` object: its `verdict`, the `latest`
tag and its `source`, and — when an update is available — `next_step`, verbatim,
which names the command that takes the update for this install's shape.

abcd never fetches implicitly (adr-38): the network is touched only when the
caller runs a command whose documented job includes a remote call. The release
origin is reached by `update --check`, `update`, and `launch --dry-run
--fetch-baseline` or `launch ship --fetch-baseline`; GitHub's attestations API
and Sigstore's TUF repository by `update`, to verify the build provenance; a
cited source by `docs
cite refresh`; a URL by `library ingest <url>`; a provider's verification call by
`ahoy connect` and `ahoy credential <name> --home`; a model provider by a
delegating verb routed to one (`reading ingest --dispatch` among them); the
site host and GitHub by `site setup`; and GitHub, through `gh` or `git`, by
`ahoy --remote`, `ahoy remote apply` and the implement loop's landing.

To take the update, run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" update --yes --json
```

An explicit tag pins the release; the bare form resolves the latest and the
receipt names what it resolved (`--yes` skips the terminal confirmation,
which cannot be answered here):

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" update v0.7.0 --json
```

Report the receipt's `action`, `tag`, `digest`, `attested_by` (the verified
provenance signer), `target_path`, and `old_version` → `new_version` from the
JSON. If `env_ignored` is present,
relay it — it names proxy/CA environment overrides the fetch deliberately
refused to honour.

`ownership` names the proof that let abcd replace the file: `release-manifest`
(its digest is published), `running-executable` (the file is the binary that
ran the command), or `path-entry-record` (`~/.abcd.noindex/path-entry` records it as
this machine's install). The last two carry no `old_version` — no published
release names those bytes any more — and the receipt reports `old_digest`
instead; relay it as an unpublished build with its digest, not as a missing
value.

**Expect a refusal in a plugin session, and relay it as the answer, not an
error.** Under `--json` a refusal is one document: the receipt, with
`"abcd": "error"`, `error` and `exit_code` beside `action` and `refusal`. Every
refusal is a named shape with a remedy in `refusal`:

- `plugin-root` — the binary belongs to the plugin install, and `abcd update`
  never touches a plugin root. Tell the user to take a plugin update in the
  host.
- `dev-shim` — the PATH entry is the track-latest dev shim; `abcd ahoy
  install` switches modes first. This names the install shape, not the version
  string: a binary that `abcd --version` reports as `dev` is any locally built
  one, and a link to such a binary is `foreign`, not `dev-shim`.
- `owned-dangling` — abcd's own entry points at a binary that is gone (a
  plugin update strands it); `abcd ahoy install` replaces it with a verified
  copy of the current release, or names the command to run first when none is
  available.
- `owned-superseded` — the entry is abcd's own pin into a plugin vintage the
  harness has moved past, so `abcd` answers an older release than the plugin
  holds; `abcd ahoy install` replaces it with the current release.
- `package-manager` — the binary resolves into a Homebrew Cellar; relay the
  printed `brew upgrade abcd`.
- `foreign` / `unprovenanced-file` — abcd never clobbers a binary it cannot
  prove is its own; relay the described occupant and remedy. The remedy
  reinstalls OVER the file: never suggest deleting it first, because the verb
  cannot run once it is gone.
- `absent` — nothing on PATH to update; point at the install remedy.
- `unverified-provenance` — the binary matched `checksums.txt`, but no build
  provenance signed by `intentdriven/abcd`'s release workflow on `main`
  from the release tag's commit verifies for it: none is published, it was signed by
  another repository or workflow, it names other bytes, it was built from
  another release's commit, the tag's commit could not be resolved, or
  GitHub's API or the Sigstore TUF repository could not be reached (offline
  included). Nothing was swapped. Relay the
  detail, which names what failed, and the remedy: retry once both are
  reachable, or install by hand — download the asset, check it with
  `gh attestation verify <asset> --repo intentdriven/abcd --cert-identity
  <release.yml on main> --source-ref refs/heads/main
  --deny-self-hosted-runners --source-digest <the tag's commit>`, then copy it
  over the PATH copy.
- `not-this-release` — the binary matched `checksums.txt` and its build
  provenance verifies, but its own build metadata (the main-module version Go
  stamps from the tag the release workflow built at) does not claim the
  requested tag: it names another release, or no release at all, as a source
  or modified build does. Nothing was swapped, and no release is recorded. Relay the
  detail and the remedy, which keeps the binary in place; there is nothing to
  retry.
- `older-than-mark` — a bare `abcd update` resolved a latest release older
  than `~/.abcd.noindex/release-mark`, the newest release this machine has
  installed or verified: the publisher may have withdrawn the newer one.
  Nothing was fetched. Relay the remedy, `abcd update <tag>` naming the older
  tag, as the person's choice to make; never run it for them.

Every install or verification this verb completes records its release in
`~/.abcd.noindex/release-mark`. A bare update only raises the mark; a tag the
person names records itself, older or not, which is the one way to lower it.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout. In a checkout the
person has listed in `~/.abcd.noindex/dogfood-roots`, or a worktree that
checkout records, `"${CLAUDE_PLUGIN_ROOT}/abcd" hook dogfood-binary` prints
the path of a build of the reviewed branch's tip, as the remote that line
declares last reported it, once it is ready: run that path in place of the
plugin-root binary, and the plugin-root binary while it prints nothing.

## `news` — what an update changed, and the setup that waits

After an abcd update, session start adds one fixed line to the session's
context: "abcd: an abcd update has news for this session. After the person's
first message, run /abcd:update news, whose page runs the abcd this plugin
ships (never a bare abcd: the one on PATH can be an older release), then begin
your first reply with the lines it prints, word for word, before anything
else." It carries no version and no count. When you see it, wait for the
person's first message, then run this plugin's own binary, never a bare
`abcd`:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" update news --json
```

The session start that printed the line ran this same binary, so it has the
verb. The `abcd` first on `PATH` can be an older release, which has no `news`
and reads the word as the name of a release to fetch.

It writes nothing. Begin your first reply with its `lines`, before anything
else, each word for word as it stands: never restate, summarise, reorder or
reformat them. When `lines` is empty, say nothing about it. The lines come in
this order:

- one line per release this computer has not been shown, newest first, with
  how many changes it made and the record ids of those that need the person
  to act;
- where the binary cannot describe some releases, one line naming them, where
  their notes are published, and that the summary is partial (`partial` is
  true);
- last, where the update brings this repository setup it never answered, the
  offer (`offer`): the release it was set up with, the one running, and how
  many setup questions and changed defaults wait.

Every word comes from the binary: checked release numbers, counts, abcd's own
titles and the digest of changes the binary carries. Nothing is read from the
repository's own changelog and nothing is fetched.

**The offer is the person's to answer.** Put `offer.line` to them with the
host's question tool, quoting it whole, with the two answers yes and not now.
Then record their reply, and only their reply:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" update news --answer yes      # or --answer not-now
```

The answer is kept once per repository and release in abcd's home folder
(`~/.abcd.noindex/after-update/`), never in the repository, and marks the news
as shown. Not now is not asked again until the next release; `abcd ahoy` still
lists the upgrade as a gap, with the command that runs the setup that waits. A
second answer for the same
release is refused with exit 2 and changes nothing. On yes, run the setup that
waits, `abcd ahoy install --only-waiting`, as
[What the binary install does](ahoy.md#what-the-binary-install-does) on the ahoy page says, with an
answers file:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install --only-waiting --answers <file> --answered-in "Claude Code" --json
```

It puts only the questions this repository never answered and the settings
whose default changed, never one answered or deferred with decide later. The
yes approved the run, so it asks no approval. Start the file with no answers:
a setting value the run would ask (the visibility, the docs target, deep
scanning) stops it with exit 2 before any write, naming that one question; put
it to the person, add their answer, and run again. A question the run asks
later, such as the artefact kind or an offer, is checked when it is asked, and
a stop there leaves the steps before it done.

**The news alone.** Where no offer was made, run
`"${CLAUDE_PLUGIN_ROOT}/abcd" update news --seen` once the person has
replied to the reply that showed it, so the next session does not show it
again. In a session with no person to answer (a headless or scripted run),
run neither `--answer` nor `--seen`: the binary cannot tell such a run from a
person's, and a mark written there would hide the news and the offer from the
person's next session.

**On request.** When the person asks for details, run
`"${CLAUDE_PLUGIN_ROOT}/abcd" update news --full` and relay it: every
change of the releases since the last update, those that need action first,
each with the record it cites, then what waits in this repository. It writes
nothing, so it can be shown again at any time, and it is the way to see the
summary on a host that runs no session-start hook. `--answer` takes neither
`--full` nor `--seen`.

**User input:** $ARGUMENTS
