# `/abcd:launch` — Curated Release

Cut a release without deciding anything by hand that the record already
decides. `launch` reads what shipped since the last tag, derives the version
from those records' declared impact, composes the changelog section and the
release page from them, and refuses the cut outright when the record and the tree disagree. The version
is never typed, the changelog is never hand-written, and a release that would
publish a compatibility lie does not happen.

Two things bound what it will do. It publishes nothing: the shipped verb writes
a dated changelog heading and the release page and stops, and CI and a human take it from there.
And it never ships the design record: the payload is default-deny with the whole
`.abcd/` namespace excluded structurally, so no include line can put it back.

The preview always exits 0, because a preview never blocks, and its one write is
its pre-flight report in the gitignored local tier. Bare `abcd launch` refuses
with a hint to ask for it. A repository with no launch payload
(`.abcd/config/launch-payload.json`) has nothing to preview, and the preview
says so and names the release path it does have: the scaffolded release
workflows, the dated CHANGELOG heading the cut writes, and the auto-release
workflow that tags it.

> **Phase ownership** ([adr-33](../../decisions/adrs/0033-launch-phase-ownership-tiered.md)): the curated-release cut — packaging with `.abcd/**` excluded plus the secret/PII scan — ships in [Phase 1](../../roadmap/phases/phase-1-ahoy.md). The pre-flight gate suite (itd-65) runs on the preview and on the cut's render path; the remaining release automation is separately scheduled (itd-66 render parity, itd-70 retention, itd-72 publishing); itd-73 derived versioning ships with the release cut.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `archive` | gate | shipped |
| `scaffold` | — | shipped |
| `ship` | gate | shipped |


**The cut makes the release, and that is where it stops.** It derives the
version and the record set from what shipped since the newest tag, runs the
surface guardrail, and, handed the host-composed prose, validates it
against that record set — a completeness bijection, so the notes and the records
name the same things. It admits only additions and fixes: the composer sees the
records that shipped and never the previous release's surface, so the other
changelog sections are refused by name and each dated section states under its
heading what the notes list and do not claim (iss-2609011207114761). It then
writes the dated `CHANGELOG.md` heading that the auto-release workflow turns
into a tag. In a repository whose version-location contract declares
`"publishes_plugin_archive": true`, it then pins the release's plugin archive in
the catalog (§ 3, *The pinned plugin archive*); without the declaration it leaves
the catalog untouched and says so. Given a payload directory, it stages the
versioned release payload, with the
derived version stamped into the payload's manifests and lockstep-proved before
return.

**A feature release arrives with a release page.** The same payload carries
`RELEASE.md`, composed from the press releases of the user-facing intents shipped
since the last tag: headline intents told as prose, the rest listed by title,
persona quotes carried word for word. The binary holds it to the changelog's
rule (every intent in that set cited once, nothing else, nothing planned),
checks each quote against its source, and runs the outbound policy and the
persona registry over it. The outgoing page moves to
`.abcd/development/releases/<version>.md`, then the page is written, then the
changelog heading; a failure rolls the earlier writes back. A fixes-only cut
writes no page. A refused payload returns every reason as data, and the host
recomposes until it is valid, reporting each attempt (itd-2609231013154443).

**The archive render is the release gate's half of the pin.** It renders the
plugin archive of the release the newest dated CHANGELOG heading names, from the
checked-out tree, into an existing directory. Bound to the tag being released,
it refuses (exit 1) unless the committed catalog pins exactly that archive's
address and digest, and unless that address lies under the releasing
repository's own release downloads for the tag — removing the archive on either
refusal, so nothing unpinned can be published. `auto-release.yml` runs it on the
pushed commit before the tag is made, and the release workflow runs it again on
the tagged commit, each run bound to the repository the workflow runs in.

`commands/launch.md` carries the emit, compose and ingest orchestration over the
`release-changelog-composer` agent, including the release page's retry loop. The
deterministic emit alone is `abcd changelog`, read-only and prose-free.

**Commit, tag and publish stay a design target** (itd-72's publishing). The
verb neither commits, tags, nor publishes, so every step past the changelog
heading and the release page is performed by a human and by CI. A cut that
renders a payload runs the pre-flight gate suite first (§ 1), and the
dirty-tree override is that suite's one override: it waives the dirty-tree gate
and nothing else. There is no documentation-warning override, because a warning
refuses nothing unless the repository configures the suite strict. There is no
version flag at all: the version is derived, never authored
([adr-31](../../decisions/adrs/0031-derived-versioning-from-intents.md)).

**The scaffold writes the release machinery into a managed repo that lacks
it**: the two release workflows and the adr-37 release runbook, wired to the
repo's own default branch and Go version, token-scoped and injection-safe. The
workflows ship from a single embedded template that abcd's own release workflows
are regenerated from, proved byte-exact by a test, so a scaffolded repo and this
one cannot drift. The scaffolded workflow carries a **rehearsal** that arms the
full gate against a simulated changelog roll and publishes nothing, so a green
rehearsal is the runbook's precondition for a first real release. A bare repo
with no semantic detector degrades cleanly to the deterministic gates and a
generic build.

It is idempotent and fail-safe: a re-run on current machinery is a no-op
(exit 0), a hand-edited file is refused (exit 1) rather than clobbered unless
the caller confirms, and a structural fault exits 2.

**The preview is spelled `dry-run`, and it is a flag, not a sub-verb.** The
binary registers no `dry-run` subcommand under launch, and `commands/launch.md`
names it as a flag. Its report is
preview-only and always exits 0. It is **not** "ship minus publish": it runs the
same gate suite the cut's render path runs and reports what that path would
refuse on, but refusing is the cut's.

## 1. Pre-flight gates

Every gate runs on the preview and on the cut's render path alike, over the one
resolved bundle, so the two refuse on the same findings. The suite runs **all**
of its gates and collects **all** of their findings before anything decides: a
cut that refused on its first finding would hide the rest until the next
attempt. The preview reports each gate as a row in its JSON — a name, a status
(`ran`, `not_armed` where the repository has not adopted what the gate reads,
`not_implemented`, or `host-run`), a one-line measurement, a tier, and the
located findings.

The **hard-fail** gates refuse the release:

- **The secret and PII scan** over the resolved bundle, fail-closed where a
  scanner is unavailable or a file could not be covered.
- **Marker-block sanity** over every shipped Markdown file: a
  `<!-- BEGIN ABCD -->` never closed, an `<!-- END ABCD -->` with no open block,
  or a nested pair, each named with its file and line. A marker quoted in code
  is not a marker.
- **Change narration** over the shipped doc bodies — Markdown under `docs/` and
  at the payload root, the changelog and the release page excepted: a sentence
  carrying "changed from … to" or "migrated from" is named with its file, line
  and text, and so is one carrying a construct that reads as narration only in
  one of its uses: "used to" as the past habit ("the tool used to print"), not
  as a passive or a participle ("the token used to authenticate the request is
  read"); "no longer" and "renamed … to" beside a subject naming abcd or its
  behaviour (a command, a flag, a hook, the default), not beside anything else
  ("files that are no longer present"); and "previously … now" beside a
  past-tense change verb, not "as previously noted". Bare "now" and bare
  "previously" are present-tense prose and pass, alone or together; a
  construct inside code, or on a line carrying the docs-lint escape, is exempt,
  and every finding names that escape.
  The changelog is derived from the records
  ([adr-37](../../decisions/adrs/0037-changelog-driven-releases.md)), so the
  remedy is to rephrase the doc, not to move the sentence into the changelog.
- **The dirty tree**: any uncommitted change against `HEAD`, tracked or
  untracked (the local tier excepted), refuses the cut unless the cut is
  given the dirty-tree override. The override is recorded in the pre-flight report with every path
  it carried. A tree whose state git cannot read refuses whatever the flag
  says. The preview has no override, so a dirty tree is on its would-refuse
  list. The cut runs this gate before it writes anything; the render after its
  writes skips it, because by then the cut's own output is on disk.
- **The installability smoke** at its light tier (below).

The **warn** gates surface a concern and refuse nothing, unless the
repository's `.abcd/config/launch-payload.json` sets `"strict_warnings": true`,
which makes each warning a refusal:

- **The documentation audit** is the docs-lint engine over the repository's
  configured doc roots. It reports `not_armed` where there is no
  `.abcd/docs-lint.json`.
- **Hook compliance**: every payload file a hook command invokes is executable
  in the payload, every command handler names a command, and every timeout is a
  positive number. A hooks config that cannot be read or parsed is a concern of
  the row, never a row with nothing to judge.

Two rows report without a tier. The **citation baseline** tallies cited claims
against their receipts and refuses on a broken, unreceipted or overdue one. The
**semantic-receipts** row reports presence only, never a verdict, because the
release workflow owns the required-gates list and judges receipt validity: the
row exists because a preview silent about the receipt gate once let a
one-commit release branch reach a tag and fail-close there
(iss-2608231226342272).

The plain-text preview prints the version, the file count bundled, scan
hard-fails, citations, receipts and whether it would publish, then a
would-refuse-on line per refusal, a warning line per warn-tier concern, and
where its report landed. The JSON carries the gate detail.

The **manifest lockstep check** also runs for real on the preview, at its `dev`
polarity over the working tree — the polarity adr-19 requires the committed
manifests to satisfy, which is that they carry no version key — and folds any
drift, or an unreadable version-location contract, into what the preview would
refuse on. The preview's retention plan refuses where the existing release tags
could not be listed or the checkout is shallow, rather than reading an unread or
partly fetched tag set as the whole release set.

The gate suite does not carry a deeper opt-in secret scan, deep credential
verification, or a vulnerability check. The scan layers that do ship enforce a
per-kind severity floor that a repo's config can raise and never lower.

The installability smoke is worth stating in full, because it is the gate that
answers "will this artefact actually install". It resolves **one** declared
surface and then makes two tiers of assertion over it, so a deeper tier upgrades
the assertions and never redefines the surface. The declared surface is the
**union** of two registers, because the plugin-manifest schema defines every
explicit key as declaring entries *in addition to* the convention directory:
the convention register (the command, agent, skill and hook paths a plugin is
expected to carry) and the manifest register (the optional keys in
`plugin.json`). A hook's plugin-root-rooted command adds the payload file it
invokes; the plugin executable itself is install-supplied, so its absence from
the payload proves nothing. Every entry records which register declared it. This
is not the compatibility surface, which records manifest keys and discards
values; installability is the mirror question, over the values.

The **light tier** ships: both manifests and every hooks config the payload
carries parse (a host registers no hook from a config it cannot parse), each
local marketplace source
resolves to a manifest whose name matches the listing (a pinned archive source
resolves to the payload root it is rendered from, and its pin must be an https
`.zip` URL with a 64-hex digest), and every declared path
the payload is responsible for is carried. Resolution reads the resolved bundle,
so a file present in the tree but excluded from the payload fails here.
`dry-run` reports it; the payload render refuses on it, because the render is
the only step that materialises an artefact. The **deep tier** (itd-66,
deferred) would, over the same resolved list, import every shipped entrypoint
and render each command's help in an isolated subprocess rooted at the rendered
payload.

Every preview, and every cut that renders a payload, writes its pre-flight
report (§ 4). The scan gate is side-effect-free with respect to the repo: its
only writes are to a private temporary tree it removes.

## 2. Curated release artefact (default-deny)

The payload is an **include list**, pinned in
`.abcd/config/launch-payload.json`, over one tree. Everything not named is
excluded, along with anything `.gitignore` matches. The list is the only
mechanism that can put a path *into* the release artefact.

The `.abcd/` exclusion is not an entry on a deny list: it is **structural**
(`internal/core/launch/bundle.go`, per
[adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)). No include
entry can promote a denied namespace, so nothing can re-include any part of the
design record or the knowledge store. The release artefact carries plugin code,
never the project's design record.

The lifeboat is not among the excluded tiers, because it is not an in-tree tier
at all: `/abcd:disembark` is read-only over the source repo and lands the
lifeboat at an operator-chosen destination outside it, with its voyage log at
the operator level (adr-35), so there is nothing for the payload filter to
exclude.

The restrictive-licence gate over kept originals is likewise **not this
artefact's gate**. Its real consumer is the lifeboat, the surface that publishes
curated project memory and provenance. At launch the gate is future and inert,
and the shipped `dry-run` renders no licence verdicts. That gate's own
evaluation-input allowlist belongs to its design and is distinct from packaging:
it re-includes files into the gate's own evaluation input, never into the
release artefact. Two mechanisms, never one name.

## 3. Versioning + marketplace

The curated release artefact is the only abcd artefact that carries a semantic
version. Versions are an *output* of cutting a release, never a sequencing input
on the design record: the repo organises work by **phase** (see
[adr-9](../../decisions/adrs/0009-phase-as-product-layer.md)), and a release
number is what falls out when a stretch of that work is published. The brief,
intents and roadmap carry no version label.

Versioning is **strict SemVer**: the version string is `MAJOR.MINOR.PATCH` with
no leading `v` at the selected version location, and the release's git tag is
`v<version>`. While the version is `0.y.z` the operator surface may still change
between minor versions; `1.0.0` marks the first stable `/abcd:*` surface. The
tag drives identification: what is installed against what is available is
compared through the tagged, released artefact, because the working tree carries
no version at all (adr-19, adr-28).

### Bump-tier rule

The version is derived from the impact the shipped records declare, and the
tier and its reason are recorded so every published version is traceable to
*why* it bumped.

| Tier | Trigger |
|---|---|
| **Major** | Any shipped intent since the last release carries `impact: breaking`. |
| **Minor** | No breaking, at least one `impact: additive`. |
| **Patch** | Only `impact: fix` intents, or a release with no intent-tied change. |

Every intent carries an impact, set when the intent is shaped and enforced by
the record lint (adr-31). At release, the cut gathers the intents shipped since
the previous release and takes the highest-severity impact. A change not tied to
any intent falls back to conventional-commit derivation.

### Refusal kinds

A cut that cannot proceed is **refused under a named kind**, and the kind is the
wire format both front doors emit (`internal/core/release/emit.go`). Every one
is fail-closed: the cut stops rather than deriving a number or a changelog that
would be wrong. There are eight, and an operator sees them as
`refused (<kind>)`.

| Kind | Raised when |
|---|---|
| `no-release-tag` | there is no immutable base to measure the cut from |
| `release-in-flight` | the newest changelog heading is ahead of the newest tag, so a release sits between its merge and its tag |
| `unlabelled-record` | a record the cut adds carries no valid impact |
| `stale-intent` | an intent in `planned/` has a spec that has closed |
| `surface-guard` | the surface guardrail failed, or could not compare |
| `unfixed-finding` | a consequential finding this cycle captured is still open, with no recorded decision to defer it |
| `deleted-finding` | a consequential record the anchor held in `open/` is in no status directory at HEAD: the cut removed the finding instead of answering it |
| `empty-cut` | nothing user-facing shipped, so there is no release |

`release-in-flight` is the one an operator meets most often outside a release
window, because it fires on any tree whose changelog has been rolled and not yet
tagged.

**Surface-diff guardrail.** The cut snapshots the command, flag and manifest
surface and compares it to the previous release. A removed or altered surface
with no breaking intent in the release fails the launch under `surface-guard`: a
mislabelled impact cannot ship a compatibility lie.

**Unfixed-findings guardrail.** The cut and the read-only `changelog` preview
both ask one further question of the cut: of the findings **this cycle**
produced, is any of them consequential, still open, and unanswered? A cut that
carries one is refused, and a refused cut carries no derived version at all.
The gate fired on this release's own first cut, so it is live behaviour rather
than a design target.

- **What counts as this cycle's** is a set difference of issue-ledger membership
  between the anchor tag's tree and `HEAD`, keyed on the record id across all
  three status directories. The id survives a re-slug and a move to a terminal
  folder, which a path does not, so a standing backlog record that merely moved
  is never reported as newly captured. The standing backlog is out of scope by
  construction: a gate that blocked on it would refuse every release until the
  whole ledger was drained, which is how a gate gets switched off rather than
  satisfied.
- **What blocks** is a severity of `major` or `critical`, and also a severity
  that is absent, misspelled, or outside the ledger's enum. An unreadable grade
  has not been judged, and "not judged" must not read as "not serious".
- **The waiver** is the frontmatter pair `deferred_after` plus
  `deferral_reason`, both schema-accepted keys. `deferred_after` names the cut's
  **anchor** tag, not the version being derived, which is what makes a waiver
  single-use: the anchor moves at the next release and every waiver written
  against the old one lapses, so a deferred finding is re-asked rather than
  forgotten. Half a waiver does not stand.
- **What the render shows.** Every render of `abcd changelog` and of the launch
  cut carries a findings line
 (the verdict, the count of unfixed findings and
  the anchor they were measured from, and the count deferred), then one line per
  waiver naming the record, its severity and its stated reason. A deferral an
  operator cannot see in the report they actually read is indistinguishable from
  having ignored the finding, which is the thing the waiver exists to be the
  opposite of. The whole verdict is on the cut's JSON.
- **The four routes out**, in the order the refusal states them: fix the defect
  and resolve its record inside the cut; record the decision not to fix it with
  the capture verb's wontfix, which clears the gate with no special case because a
  wontfix carries a reason and is the cited non-action the rule asks for; defer
  it out loud with the waiver pair; or re-grade the record honestly when the
  severity was wrong in the first place. The gate asks only whether the record is
  still open at `HEAD`, so deleting it outright clears the gate as well. That is
  a hole, not a fifth route.

### Where the version is written

The cut writes the version into **the selected version location**, never a
hard-coded manifest. The location is read from the decision artefact
`.abcd/config/version-location.json` as a manifest path plus a JSON pointer (see
[adr-19](../../decisions/adrs/0019-plugin-json-version-carve-out.md)). The
shipped lockstep checker fails closed when that artefact is missing or
malformed, and a blocked decision has no schema-valid location, so
version-writing refuses and the escalation stands. Concretely, the cut:

1. Stamps the bumped version into the **release artefact** at the selected
   location. The manifest renderer reads the decision artefact and never parses
   a location string.
2. Leaves the **working-tree** manifests unversioned. The version is
   single-sourced in the cut artefact, so the repo's committed manifests carry
   no version and there is nothing to keep in sync on the working-tree side.
3. Records the version and changelog entry in the marketplace metadata at the one
   canonical `.claude-plugin/marketplace.json`, never a root-level copy. **Later
   phase** ([adr-20](../../decisions/adrs/0020-manifest-version-lockstep.md)):
   that entry conforms to a schema, validated programmatically by this step.
4. Stamps nothing else. Those two locations are the whole of it, and the render
   names both, so there is no third place for a version to drift out of step.

### The pinned plugin archive

A release publishes its plugin as one zip, `<plugin>-plugin-v<version>.zip`,
and the committed catalog names it:
`{"source": "archive", "url": "<repository>/releases/download/v<version>/<name>", "sha256": "<digest>"}`
([adr-2609231048308186](../../decisions/adrs/2609231048308186-the-catalog-pins-the-latest-release-s-plugin-archive.md),
amending adr-19 and adr-20 on the 2026-09-23 ruling). The harness downloads the
zip and refuses it when the digest differs, so an install or update at the tip of
`main` receives the latest cut release, stamped with its version and
fingerprinted — not the unversioned working tree.

- **Pinned only on the declaration.** The cut pins only when the
  version-location contract declares `"publishes_plugin_archive": true`, the
  statement that the repository's release workflow uploads the archive; abcd
  declares it. The contract alone does not count: the workflows the scaffolder
  renders for a managed repository upload no archive, so a catalog pinned there
  would name an asset nothing publishes. Without the declaration the catalog is
  left untouched and the cut's report says so; a declaration that is not a
  boolean is refused before anything is written.
- **Rendered twice, identically.** The cut renders the archive from its tree to
  learn the digest it commits, and refuses a payload with uncommitted changes
  first. `auto-release.yml` renders it again from the pushed commit before the
  tag is made, and the release workflow from the tagged commit, in `verify`
  before anything is built and in the publish job on the bytes that ship; none
  proceeds unless the digests agree, and each binds the address to the
  repository it runs in, since `plugin.json`'s `repository` names another one
  after a rename, a transfer or a fork. The archive is
  reproducible by construction: sorted entries, stored uncompressed, one fixed
  timestamp, modes normalised to 0644 or 0755.
- **The catalog is left out of the zip.** It is the file that names the zip's
  digest. The stamped marketplace version and changelog entry (points 3 and 4
  above) therefore live in the staged payload, where the lockstep proves them,
  and not in the published artefact; the published version is the archived
  `plugin.json`'s.
- **Published with the binaries.** The archive is checksummed into
  `checksums.txt`, covered by the build-provenance attestation, uploaded, and
  after publication downloaded fresh, attestation-verified and byte-compared.
- **The window.** From the ship's merge, the catalog on `main` names an archive
  the publish job has not uploaded yet. An install or update in that window fails
  closed and leaves an installed plugin on its previous release; nothing else can
  install, because the digest refuses other bytes. It cannot be closed, since the
  pin must be in the tagged tree, so the release runbook keeps it short.
- **The harness floor.** An archive source needs Claude Code v2.1.224 or later;
  older harnesses fail to install it, and very old ones fail to load the
  marketplace. The install instructions and the release notes state the floor.
- **Contributors** load the plugin from their own checkout rather than through a
  second catalog entry (`CONTRIBUTING.md`).

**Anti-drift.** The two manifests in the artefact describe one release, so the
version at the selected location and the marketplace entry must agree. A
read-only lockstep checker proves this over the path list adr-20 records, and a
half-state is drift. The cut's bump step runs it against the staged artefact at
the **public** polarity and refuses to publish on drift; the **dev** polarity
runs in `dry-run` over the working tree, asserting the committed manifests carry
no version key. The checker has no bypass flag, and adr-20 records that
a dirty-tree override must not bypass manifest consistency.

The release commit message format — carrying the bump tier and its reason — is
a **full-cut design target** (itd-72). The shipped cut never
commits, and no shipped path produces that format; the release cuts made so far
use hand-written prose.

### Release cut and retention

Every cut is designed to make one released snapshot: the release
commit, the `v<version>` tag on it, the marketplace changelog entry, and, on the
tag, a published GitHub Release with SLSA provenance attached to the artefact
(adr-28). The version lives only on the tag and in the cut artefact.

Retention is **newest-per-line**: each `MAJOR.MINOR` line keeps only its newest
release, so shipping a patch removes the superseded patch and its Release
assets, while shipping a new minor keeps the previous line's last release as
that line's terminal snapshot. Three safety rules bound the prune: the release
just published is never pruned; pruning refuses outright if a release newer than
the one just published already exists, because that is an out-of-order ship a
human must resolve; and only release tags and Releases are pruned, never git
history. The launch report is the durable record of every launch including
pruned ones, so deleting a release tag never deletes the evidence a launch
happened.

A prune is a destructive, outward-visible act, so the design has the cut report
exactly which release it removed, or why it refused. **Removal itself is a
full-cut design target** (itd-70). What ships today computes the decision and
renders it — which releases a line keeps, which the plan would prune, and the
reason a refusal stands — and stops there: no shipped path deletes a tag, a
release or an asset, so the plan is a statement of intent a person still carries
out.

## 4. Reports

Every preview, and every cut that renders a payload, writes a pre-flight report:
`preflight.json` and `preflight.md` in a directory of its own under
`.abcd/.work.local/logs/launch/`, named for the run's instant, in the gitignored
local tier (per the iss-36 and iss-56 adjudication resolved as iss-73). The
report carries the mode (preview or cut), the verdict, every refusal, every
warning, every gate row, and the dirty-tree override with the paths it carried.
A refused cut writes its report too, and the refusal names where it landed. The
preview's JSON carries `report_path`, the cut's `preflight_report`. A detector
fails the build if any non-test Go source under `internal/` so much as names the
retired runtime location.

## 5. Bootstrap exception

The first release cut of abcd itself is a manual tag and GitHub Release: the
machinery a scaffolded repo inherits cannot cut the release that first publishes
it.

`commands/launch.md` documents the publishing chain rather than deferring it.
Its release-day runbook walks the two semantic passes, the local gate proof, the
release pull request and its merge, the automatic tag, the
build-checksum-attest-publish step behind an approval gate, the site deploy that
approval releases with it, and the post-release check. What stays a design
target is abcd's own publishing **automation** (itd-72): the shipped cut
neither commits, tags, nor publishes, so every step past the changelog heading is
performed by a human and by CI.

## 6. Acceptance

- **Given** any abcd-aware terminal, **when** the user runs bare `/abcd:launch`,
  **then** the dispatcher shows current launch readiness, the available
  sub-verbs, and suggested next actions, and mutates nothing. **Not built:** the
  shipped bare `abcd launch` refuses with exit 1 and a hint to ask for the preview,
  and the shipped plugin command runs the dry-run preview directly.
- **Given** a clean tree with a deliberate PII fixture inside the resolved
  artefact, **when** the preview runs, **then** the report-only gate prints that
  it *would* refuse on that finding, naming the offending file and line, still
  exits 0, and writes no artefact beyond its pre-flight report. **When** a cut
  that renders a payload runs on the same tree, **then** it refuses before it
  writes anything.
- **Given** a clean tree, **when** the preview runs, **then** the report lists
  exactly the include and exclude manifest of [§ 2](#2-curated-release-artefact-default-deny)
  with no surprises, and no artefact is written.
- **Given** only fix-impact intents shipped since the last release, **when**
  the cut runs, **then** the bump tier is patch and the next patch version is
  written into the selected version location in the **release artefact** only,
  the working-tree manifests staying unversioned, plus the canonical marketplace
  manifest.
- **Given** a repository whose version-location contract declares
  `"publishes_plugin_archive": true` and a clean payload, **when** the cut writes
  the dated heading, **then** the catalog's plugin source becomes the release's
  pinned archive — its download address and the digest of the archive rendered
  from that tree — the working-tree manifests stay version-free, and the archive
  gate on the resulting commit reproduces the digest and exits 0. **Given** a
  payload file changed after the pin, the archive gate exits 1, names both
  digests, and leaves no archive behind; **given** an uncommitted payload
  change, the cut refuses before writing anything. **Given** the contract
  without the declaration, the cut leaves the catalog byte-identical and reports
  it as not pinned. **Given** a pinned address under another repository than the
  one the gate is bound to, the archive gate exits 1 and leaves no archive
  behind.
- **Given** at least one additive intent and no breaking intent, **when** the cut
  runs, **then** the tier is minor and the launch report names the intents that
  drove it. **Given** any breaking intent, the tier is major and the report names
  it.
- **Given** a command, flag or manifest surface removed or altered since the
  previous release with no breaking intent in the release, **when** the cut runs,
  **then** the surface guardrail fails the launch, the mislabel is reported, and
  nothing is published.
- **Given** an issue record captured since the anchor tag, graded major or
  critical (or carrying no readable grade), still open and carrying no standing
  waiver, **when** the cut or `changelog` runs, **then** the cut is refused under
  `unfixed-finding`, the refusal names every such record with its grade and path,
  no version is derived, and the findings line says how many were counted and
  from which anchor.
- **Given** an issue record the anchor tag held in `open/`, graded major or
  critical (or carrying no readable grade), and present in no status directory at
  HEAD, **when** the cut or `changelog` runs, **then** the cut is refused under
  `deleted-finding`, the refusal names the record with the grade and path the
  anchor held and says it is in no status directory, and the three dispositions —
  a move to `resolved/`, a move to `wontfix/`, a re-slug inside `open/` — each go
  on clearing the gate, because each leaves the record in the ledger.
- **Given** the same record carrying a waiver anchored to that cut's anchor tag
  and a non-empty reason, **when** the cut is emitted, **then** the gate passes
  and the render carries a deferred line naming the record, its severity and its
  reason — and the same record blocks again at the next release, because the
  anchor has moved and the waiver has lapsed.
- **Given** a prior release of the same line exists, **when** the cut publishes
  the next one, **then** the superseded release's tag and Release assets are
  removed, the removal is named in the launch report, and the last release of
  every other line is untouched. **Given** a release newer than the
  just-published version already exists, the retention step refuses to prune
  anything and records the refusal reason. *(Not built: the shipped cut renders
  the retention decision and stops before any removal; removal is itd-70's.)*
- **Given** a documentation-auditor or hook-compliance warning, **when** the
  preview or the cut runs, **then** the warning is shown and refuses nothing;
  **given** `"strict_warnings": true` in the launch-payload config, the same
  warning refuses.
- **Given** findings in several gates at once, **when** the preview or the cut
  runs, **then** every one of them is reported in the one pass and in the
  pre-flight report.
- **Given** a dirty working tree, **when** a cut that renders a payload runs,
  **then** it refuses before writing anything and names the uncommitted paths;
  **with** the dirty-tree override it proceeds and its pre-flight report
  records the override and every path it carried.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd launch`

Sub-verbs: `abcd launch archive`, `abcd launch scaffold`, `abcd launch ship`.

| Flag | Type |
|---|---|
| `--dry-run` | bool |

### `abcd launch archive`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--out` | string |
| `--repository` | string |
| `--tag` | string |
| `--verify` | bool |

### `abcd launch scaffold`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--confirm` | bool |

### `abcd launch ship`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--allow-dirty` | bool |
| `--changelog-json` | string |
| `--payload-dir` | string |

<!-- surface-appendix:end -->
