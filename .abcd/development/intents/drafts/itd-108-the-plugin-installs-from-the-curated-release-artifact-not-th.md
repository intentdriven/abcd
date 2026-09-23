---
id: itd-108
slug: the-plugin-installs-from-the-curated-release-artifact-not-th
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-67, itd-105]
severity: major
warrants_assumed:
- "GitHub serves `releases/latest/download/<asset>` as a stable redirect to the newest release's asset of that name, so a URL cut into the catalog once keeps resolving to the current release."
- "The host harness resolves an archive plugin source's update signal without a per-release edit to the catalog — either from the downloaded zip's bytes or from a version the artifact carries. WHICH of these holds is the intent's central open question, not an assumption."
- "Released zip archives stay far below the harness's 256 MiB archive ceiling (today's curated payload is ~1.6 MB)."
---

# The plugin installs from the curated release artifact, not the repo, and every cut release reaches users automatically

## Press Release

> **Installing the abcd plugin now delivers the plugin, not the source tree that
> builds it.** Today the plugin cache holds a second full clone of the repo — 11 MB
> of `internal/`, `cmd/`, `evals/`, `go.sum` and the `.abcd/` design record — so
> that the harness can read 296 KB of commands, agents and hooks. With this intent
> the marketplace's PLUGIN source points at a curated zip published with each
> release: about 340 KB, no source, no git history, and no Go toolchain or git
> needed on the machine. Because the catalog names the *latest* release rather
> than a pinned one, cutting a release is the whole publication step — the next
> update a user takes brings the new surface, and the bootstrap that already
> tracks `releases/latest` brings the matching binary into the same fresh plugin
> root. Surface and binary arrive from one release, together, with no manual step
> between the tag and the user.

> **What this deliberately does NOT do**, stated here because the obvious reading
> is wrong: the marketplace SOURCE stays `REPPL/abcd-cli`, so adding the
> marketplace still clones the repo (~16 MB including git history and `.abcd/`).
> The design record therefore keeps reaching users, and this intent does not stop
> it. That is an accepted cost, taken deliberately in the grill below: the
> alternative front door is a 99-character catalog URL typed into the harness as a
> slash-command argument, against 38 characters for the repo form, and the repo is
> public so the residue is weight rather than exposure.

> "I install a plugin, I expect a plugin," said Alice, who added abcd on a
> laptop with no Go toolchain. "I got a source tree, somebody's architecture
> decision records, and a 54-second wait the first time I ran a command."
> Bob, who maintains abcd, had the mirror-image complaint: "Cutting a release
> published binaries, and then the plugin surface just… didn't move. I'd tag,
> and nothing reached anyone until I remembered the second step." Carol, who
> reviews what their team installs, put it plainest: "I could not tell you which
> of those 11 megabytes in the plugin cache the harness actually executes."

## Why This Matters

[[adr-28-single-repo-curated-release]] already decided this. It says `.abcd/**`
"stays in-tree" but is "excluded from the release artifact **by packaging**,"
and that "the repo is the marketplace." The packaging half exists and works:
`.abcd/config/launch-payload.json` names the payload (`.claude-plugin`,
`commands`, `agents`, `hooks`, `scripts`, `docs`, `README.md`, `LICENSE`), and
`abcd launch --dry-run` computes it today — 54 files, 0 scan hardfails.

What is missing is the last hop. `.github/workflows/release.yml` publishes the
four binaries and `checksums.txt` and nothing else, so the curated bundle is
computed, gated, and then discarded. Meanwhile
`.claude-plugin/marketplace.json` carries `"source": "./"` — a relative-path
source, meaning "the marketplace repo *is* the plugin." The curated artifact and
the install path have never been connected, which is what `AGENTS.md` concedes
when it says the launch bundler "denies the namespace structurally, though that
filter has yet to run on a cut release."

Measured on a first manual install (2026-08-10):

| what a user gets | size | why |
| --- | --- | --- |
| `marketplaces/abcd-marketplace` | 16 MB | whole repo incl. 4.5 MB `.git`, to read one catalog file |
| `plugins/cache/.../<sha>` | 22 MB | whole repo again (11 MB) + the bootstrapped binary (11 MB) |
| what the harness reads | **296 KB** | `commands/` + `agents/` + `hooks/` + `.claude-plugin/` |

The second half — *automatically* — is the part with no existing answer at all.
The binary already has this property: `hooks/bootstrap.sh` fetches
`releases/latest`, so a new cut reaches users with no edit anywhere. The plugin
surface has the opposite property: it tracks the repository tip through a
relative-path source, which is why [[itd-105]] had to invent a version-skew
notice to report surface-ahead-of-binary drift in the first place. If the
catalog names the latest release instead, both halves resolve to the same cut by
construction — a plugin update creates a fresh plugin root, and a fresh plugin
root is exactly what makes the bootstrap fetch the matching binary. The skew
this project currently *reports* becomes a skew it cannot *have*.

## Decisions (grilled 2026-08-11)

The maintainer resolved the shape questions in a grill session; these are
commitments, not options.

- **The marketplace source stays `REPPL/abcd-cli`; only the PLUGIN source
  becomes an archive.** The install line is typed into the harness as a
  slash-command argument, where a catalog URL runs to 99 characters against 38
  for the repo form. Users are the audience and contributors are incidental, so
  the short front door wins. Accepted cost, stated rather than discovered later:
  the marketplace clone remains (~16 MB including git history), so the plugin
  cache falls from ~22 MB to the payload plus the binary while the catalog clone
  does not move. The "abcd stops shipping its design record to strangers" half of
  the press release therefore does NOT hold under this shape — and since the repo
  is public, that residue is weight, not exposure.
- **No second repository.** A public-facing `abcd` beside `abcd-cli` was
  considered and rejected: it is the dev→public mirror
  [[adr-28-single-repo-curated-release]] retired, and it hides nothing, because
  `abcd-cli` is already public.
- **[[adr-28-single-repo-curated-release]] needs no amendment.** An earlier
  reading of this intent assumed the catalog would move off the repo, which would
  have falsified "the repo is the marketplace". With the catalog still committed,
  that clause stays literally true.
- **The archive is unpinned, and the attestation extends to cover it.** A
  committed catalog cannot carry a per-release `sha256` without a per-release
  commit, and a bot commit to the default branch during a release run trips
  `release.yml`'s no-branch-commit tripwire. So the catalog names a stable
  `releases/latest/download/` URL and pins nothing. The integrity story is the
  signing `release.yml` ALREADY performs — `actions/attest-build-provenance` over
  the binaries and `checksums.txt` — extended to the plugin zip, so both halves of
  the distribution carry the same guarantee instead of one having none.
  _Overtaken 2026-09-23: the product thinker's ruling E1 chose the pinned archive (address and `sha256`, rewritten by every ship), recorded in [adr-2609231048308186](../../decisions/adrs/2609231048308186-the-catalog-pins-the-latest-release-s-plugin-archive.md)._
- **An offline signing key is recorded as a future extension, not adopted now.**
  The attestation proves "built by this workflow in this repo", which defends a
  stolen upload token and not a full repository compromise, because the identity
  root is still the forge. A key held offline (minisign) closes that and costs a
  key to keep. It becomes relevant, and is deliberately deferred.
- **One bundle, less `docs/assets/**`.** The plugin zip is the release payload
  `launch-payload.json` already declares, rather than a second manifest to keep in
  step. But two PNGs — `intro.png` at 1.07 MB and `logo.png` at 182 KB — are
  1.25 MB of a 1.6 MB payload, against 42 KB for all eight markdown files, and
  nothing the harness loads reads the shipped `docs/` at all (the `docs/` paths in
  `commands/` and `agents/` address a lifeboat's internal record or a target
  repo's pages, never this one's). Excluding the assets takes the payload to
  ~340 KB and keeps the prose.
  Two mechanics the implementer needs and the review surfaced: `launchPayloadConfig`
  (`internal/core/launch/includes.go`) carries an `includes` list and no exclusion
  syntax, so this is expressed as NARROWER INCLUDES, never as an exclude; and
  `README.md` — itself in the payload — embeds two `docs/assets/img/*.png` tags,
  so trimming the assets leaves a payload README with broken images unless those
  tags are pointed at absolute URLs in the same change.
- **No migration is owed, because there are no users yet.** The archive source
  requires harness v2.1.224 or later; below it the install fails, and older hosts
  still fail to load the marketplace. With no installed base the question is moot
  and is not designed for. The version floor is still stated in the README, and
  the multi-harness install story is captured separately as a getting-started
  concern rather than carried here.

## Superseded direction (pre-grill)

Kept only to show what the grill moved. Every bullet below is now governed by
the Decisions above where the two disagree.

- **Mechanism: an `archive` plugin source.** It is the only source type that
  needs neither git nor npm on the user's machine, and the harness looks for
  `.claude-plugin/` at the archive root or one folder down, which the curated
  bundle already satisfies.
- **Automation: a stable, unpinned `releases/latest/download/` URL**, so the
  catalog is written once and never rewritten per cut. This is what makes
  "cut a release" the entire publication step.
- **Catalog delivery: by direct URL to `marketplace.json`**, so adding the
  marketplace fetches one file rather than cloning 16 MB. This keeps ADR-28's
  "the repo is the marketplace" literally true — the catalog stays a file in
  this repo; only the fetch method changes.
- **The binary keeps its own path.** The zip carries the surface; the bootstrap
  keeps fetching the platform binary from the same latest release. Four
  platform binaries in one archive would quadruple every download to save a
  step that already works.

## What's In Scope

- Publishing the curated bundle as a zip release asset from the existing release
  workflow, and extending the build-provenance attestation to cover it.
- Repointing `.claude-plugin/marketplace.json`'s PLUGIN source at that asset. The
  marketplace source stays `REPPL/abcd-cli` and the catalog stays committed, so
  the README's `add` line does not change.
- Amending `.abcd/config/launch-payload.json` so the declared payload is what a
  plugin needs — see the Decisions on `docs/assets`. One manifest, not two.
- The record consequence, narrowly: `AGENTS.md` states that `.abcd/**` is
  "present in every repository checkout — **marketplace installs** and release
  source archives included." Under the decided shape that stays TRUE of a
  marketplace install, whose clone remains, and becomes false only of the plugin
  payload. The sentence is re-scoped, not deleted. Neither
  [[adr-28-single-repo-curated-release]] nor
  [[adr-19-plugin-json-version-carve-out]] needs amending, and the change should
  say so explicitly so a later reader does not re-open them.
- A stated minimum host version in the README, since `archive` sources require
  harness v2.1.224 or later and fail loudly below it.

## What's Out of Scope

- The release/versioning pipeline itself — how versions are derived, bumped,
  and tagged stays with [[itd-73-derived-versioning]] and
  [[adr-31-derived-versioning-from-intents]]. This intent consumes a cut, it
  does not change how one is decided.
- The binary's own distribution and trust bar — [[itd-105]] settled that
  (same-origin checksums, build-from-source as the full-trust escape hatch)
  and this intent does not reopen it.
- Windows support, which remains its own future intent.
- The binary's user-invoked update verb — [[itd-130]] (builds on this intent
  and [[itd-105]]). Its commitment that matters HERE: `abcd update` never
  replaces a plugin-root binary — in that context it refuses and names the
  host's plugin-update path — so the one-cut surface/binary coherence this
  intent establishes stays structural rather than depending on updater
  discipline.
- Restructuring `docs/` itself, or where its images live in the repository. The
  manifest is amended to stop shipping `docs/assets/**` in the payload; how the
  repository organises those assets is untouched.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a machine with no Go toolchain and no git, **when** a user adds the
  abcd marketplace and installs the plugin, **then** the install succeeds and
  the PLUGIN ROOT contains the curated payload only — no `internal/`, no `cmd/`,
  no `.abcd/`, no `go.mod`, and no git history. (The marketplace clone is out of
  scope by decision and is not what this criterion measures.)
- **Given** a completed install, **when** the user runs any `/abcd:*` command,
  **then** it executes against the plugin-root binary without falling back to a
  source build — because no source is present to build from.
- **Given** an installed user at release N, **when** release N+1 is cut and
  published and the user takes a plugin update, **then** the new surface and the
  N+1 binary are both present, **and** no maintainer edited the catalog, the
  manifest, or any workflow between the tag and that update.
- **Given** a release cut, **when** the release workflow completes, **then** the
  curated zip is attached to the release with its digest recorded next to the
  binaries', **and** the workflow's no-branch-commit tripwire still passes —
  the release job pushed nothing to any branch.
- **Given** the published zip, **when** its contents are compared against
  `.abcd/config/launch-payload.json`, **then** they match the bundle
  `abcd launch --dry-run` reports, so the preview and the published artifact
  cannot disagree.
- **Given** a host older than the minimum version an `archive` source requires,
  **when** a user attempts the install, **then** the failure names the required
  host version rather than presenting as a broken plugin.

## Open Questions

- **Does an unpinned archive at a stable URL actually re-check for a new
  release?** The remaining empirical unknown, and the only thing between this
  shape and settled. The catalog never changes and the URL never changes, so the
  harness must fetch or head the zip to notice a new cut. The published version
  ladder says an archive source resolves its version from the `sha256` pin, or
  from the digest of the downloaded file when no pin is set — which implies a
  fetch — but "implies" is not "observed". Settle it against a real pre-release
  cut before Cut B is called done, not by reading the documentation twice. It
  must be a FULL release: GitHub's `releases/latest` excludes prereleases, so a
  prerelease leaves `releases/latest/download/` pointing at the previous cut and
  the experiment reports "no update" whatever the harness does — a false negative
  that would wrongly void the unpinned decision. The no-users-yet decision makes
  a throwaway full release free.
  **Narrowed by local experiments, 2026-08-20** (agent-run, recorded here for
  the Cut B decision): against harness v2.1.237, an archive-source plugin at
  a stable URL with no `sha256` pin was installed from a local rig, the
  served bytes were swapped in place, and `plugin update` was run — in BOTH
  version shapes, since the shipped artifact declares a version (adr-19)
  while a bare zip does not. Observed, no-version shape: the harness
  re-downloaded on every update invocation (server access log), installed
  new bytes into a fresh cache dir named by the digest's first 12 hex chars,
  and determined "already at the latest version" on a no-change run by
  re-downloading too. Observed, declared-version shape (the one abcd ships):
  the re-download still happens on every update — there is no pre-download
  version skip — but the INSTALL decision is version-keyed: same declared
  version with new bytes is fetched, compared, and DISCARDED ("already at
  the latest version", old content retained); a version bump installs.
  Corroborating basis for the mechanism claim: the update-engine code
  embedded in the harness binary was read directly, and its archive branch
  fetches before any version comparison — implementation-observed on one
  harness version, not a documented contract. Three consequences for this
  intent: (1) the unpinned decision holds — cutting a release propagates,
  because every cut bumps the declared version; (2) an operational
  invariant, recorded here so it is never discovered in production: **a
  re-published plugin asset under an unchanged version never reaches
  already-installed users** — fixing a botched zip requires a version bump,
  i.e. a new cut; (3) a bare `marketplace update` refreshes the CATALOG only
  and never touches the archive — the re-fetch is a plugin-update act
  (explicit, or the session auto-update pass). Accepted risk, stated rather
  than implied: the whole publication story rests on harness behaviour that
  is observed but not contractual, and adr-38 forbids any ambient canary
  that would detect a future harness regression — the only detector is a
  user running `version --check`. Residue still owed at Cut B: the GitHub
  `releases/latest/download/` redirect chain and the background auto-update
  pass were not in the local loop, so the real-cut check below stands, now
  as confirmation rather than as the open question. The next sentence's
  cache-forever branch is therefore believed dead; it remains recorded
  because the real-cut check is what retires it.
  If it turns out the harness caches by URL and never re-fetches, the unpinned
  decision above is void — and note the fallback is foreclosed TWICE, not once: a
  per-release version write into the committed catalog trips `release.yml`'s
  no-branch-commit tripwire AND violates
  [[adr-20-manifest-version-lockstep]]'s source polarity, which requires
  `/plugins/0/version` ABSENT in the working tree. That branch has no in-record
  recovery and would need adr-19/adr-20 amended, so it is a genuine dead end
  rather than a cheaper alternative.
- **Does extending the attestation to a new asset cost anything?** The decision to
  cover the plugin zip assumes adding it to the attest step's `subject-path` is
  sufficient. The signing path itself is NOT unproven — v0.4.2 carries a
  verifiable build-provenance attestation for its binaries, so the machinery runs
  on a real cut. Only the extension is unexercised.
- **Nothing verifies the zip at install time, and that gap is accepted rather than
  closed.** `bootstrap.sh` checksums the binary and refuses on mismatch, per
  [[itd-105]]'s bar; the harness would verify an archive against a `sha256` pin,
  which the Decisions decline. So the two halves carry the same attestation but
  not the same operative check — the binary is machine-verified on every install
  and the zip is verified by nobody. Recorded so the asymmetry is a known
  position, not an oversight, and so the future offline-key work has a stated
  target.

## Prerequisites

- **iss-205 is resolved; the sequencing constraint it imposed is met.** Every
  file in `commands/` resolves the binary from `${CLAUDE_PLUGIN_ROOT}` first,
  `PATH` second, with `go run ./cmd/abcd` only in a source checkout, and a
  detector test under `internal/core/launch` fails any `commands/` file that
  regresses the ladder. A curated payload without `cmd/` or source therefore
  no longer kills the command surface; what this intent still owes is
  verifying that ladder on a real archive install before the payload slims.
- **iss-206 gains a second cause.** An archive install's cache directory is not
  named for a commit, so `plugin_sha` stays `unknown` for a reason unrelated to
  the 40-hex gate. Its fix should stop assuming a commit sha rather than widen
  the gate. Note also that if this intent lands as designed, the skew notice's
  reason for existing largely evaporates — surface and binary come from one
  release — which is worth weighing before investing in repairing it.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
