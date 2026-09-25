---
name: launch
description: Preview the public launch — the file bundle, the secret/PII scan, and the release gates — in dry-run mode, cut a release by deriving its version and composing its changelog and release page, render and verify the release's pinned plugin archive, and scaffold the changelog-driven release gate into a managed repo. The preview performs zero writes; `ship` writes the dated CHANGELOG heading, the RELEASE.md page and the archive pin and never publishes; `archive` writes one zip where it is told and never publishes; `scaffold` writes the release workflows and never publishes.
argument-hint: "[--dry-run] | ship [--changelog-json <path>] | archive --out <dir> [--tag <vX.Y.Z>] [--verify] [--repository <owner/name>] | scaffold"
---

# `/abcd:launch` release preview and release cut

Two flows over the abcd binary, kept apart on purpose:

- **preview** (`dry-run`) — the bundle, the scan, and the gates. **Zero writes.**
- **ship** — the release cut: derive the version from what shipped, compose the
  changelog prose and the release page, write them. It writes the dated section
  of `CHANGELOG.md`, the release page `RELEASE.md`, and the outgoing page's copy
  under `.abcd/development/releases/`; in a repository that publishes a versioned
  plugin it also pins the release's plugin archive in
  `.claude-plugin/marketplace.json` (refreshing the surface snapshot beside it).
  It **never publishes**.

Neither flow publishes. Tagging is `.github/workflows/auto-release.yml`'s job, and
it reads the dated heading `ship` writes. A third verb, **archive**, is the release
workflow's half of the pin: it renders the plugin archive from a commit and proves
the committed pin names exactly that archive.

## Release day: what a human actually does

The rest of this page describes the verbs. This section describes the **day** —
what you click, what runs on its own, and where it stops and waits for you. Read
it if you are cutting a release and are not the person who built the machinery.

Nothing here publishes by accident. The release stops and asks for a human twice:
once when you merge, and once at a deployment gate that no merge can bypass.

### The shape of it

| # | Step | Who |
| --- | --- | --- |
| 1 | Run the two semantic passes; record their receipts | you, with the agent |
| 2 | Prove the release gate locally | you, one command |
| 3 | Open the release PR, wait for green, merge | you, in GitHub |
| 4 | Tag the release | automatic |
| 5 | Build, checksum, attest, publish | **stops and waits for your approval** |
| 6 | Deploy the website from the tag | automatic, after step 5 |
| 7 | Update your plugin and check what you got | you, locally |

Steps 4-6 are one GitHub Actions run. You do not visit Cloudflare: the site
deploy is invoked by the release workflow, so approving step 5 releases step 6
with it.

### Step 3 — merging, and the one hazard

Merge the release PR the normal way, from the PR page's **Merge pull request**
button (or the merge queue if the repository uses one).

**Then let nothing else merge until the tag exists.** This is the only part of
release day with a trap in it. The release workflow works out which commit the
reviewers actually read by walking back from the merge, so another PR landing in
the gap makes it read the wrong commit, and the receipts you just recorded no
longer match. The window is a minute or two. Tell anyone else working in the
repository to hold.

### Step 5 — the approval gate, in detail

A minute or two after the merge, the release pauses. **It will not continue
until you approve it**, and nothing tells you unless you look.

Where to look, easiest first:

1. **Your GitHub notifications.** You are a named reviewer on the `release`
   environment, so GitHub emails you and shows a notification when a deployment
   needs review.
2. **The repository's Actions tab** — `https://github.com/<owner>/<repo>/actions`.
   The top run shows a yellow **Waiting** badge.
3. **The run page itself**, `https://github.com/<owner>/<repo>/actions/runs/<run-id>`.

On the run page a banner appears at the top: **"Review pending deployments"**.
Click it, tick the **release** environment, optionally leave a comment, and click
**Approve and deploy**.

What you are approving: building four platform binaries stamped with the tag,
rendering the plugin archive (`abcd-plugin-vX.Y.Z.zip`) and proving it is the one
the catalog pins, checksumming exactly those bytes, signing a provenance
attestation, publishing a GitHub Release with the binaries and the archive
attached, and deploying the website from the tag. That is why it is gated — it is the step that puts bytes in front of the
public. The reviewer requirement itself lives in the repository's environment
settings rather than in a workflow file, and the scaffold-parity test fails any
edit that drops the `environment:` binding from the job.

**If the banner does not appear**, the run has not reached that job yet. The
checks before it (`verify`) take a few minutes: they build, test and lint the
tagged commit. Wait for `verify` to go green.

**Approve promptly.** From the moment the release PR merges, the catalog on `main`
names this release's archive, and the archive exists only once this job uploads
it. A plugin install or update in between fails closed — the download is not
there yet, so the harness refuses and an installed plugin stays on its previous
release — and nothing else can be installed in its place, because the pinned
digest refuses any other bytes. The window cannot be closed (the pin has to be in
the tagged tree, so it lands before the tag), so it is kept short.

### Step 7 — afterwards, check what you actually got

Publishing does not change the copy of abcd on your machine. Take a plugin
update in your agent harness (in Claude Code: `/plugin`, then update the abcd
plugin — the update downloads the pinned archive of the release you just cut,
which needs Claude Code v2.1.224 or later), then start a new session and check:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" version
```

It should report the version you just released. If it still reports `dev` or the
previous version, the update has not taken effect — the plugin's binary tracks
the newest **published release**, and its provisioning skips the check when a
binary is already in place, so it needs the plugin update to fetch a new one.

### When it goes wrong

Six failures are worth recognising, because each looks like something else.

- **The run says `Waiting` for a long time and nothing happens.** That is the
  approval gate, not a hang. Approve it.
- **The release succeeded but `site / deploy` failed.** Known, recorded as
  iss-2608231912566984, and not a release failure: the binaries, checksums and
  attestation are published and correct, and only the website is behind. Deploy
  it with one command, which is the dispatch path `site.yml` documents as its
  emergency redeploy and treats as production by definition:

  ```bash
  gh workflow run site.yml
  ```

  With no tag input it resolves the newest published release. Reach for it when
  the chain's own deploy fails; a workflow reached by `workflow_call` resolves
  its environment secrets perfectly well, provided every caller above it passes
  `secrets: inherit` — measured on a canary secret, and pinned by
  `TestReleaseChainPassesSecretsAtEveryLevel`.
- **The release job fails on `Semantic-gate receipts`.** The receipts do not
  match the commit the workflow derived. The tag exists by then and the workflow
  never moves a tag, so the version is consumed: it needs the tag deleted and the
  release re-cut. Step 2 exists to catch this before the merge — run it.
- **`auto-release` fails in `detect`, on `Plugin archive reproduces the committed
  pin, before the tag`, and no tag appears.** The merged commit renders a
  different archive from the one the ship pinned — a payload file (`commands/`,
  `agents/`, `hooks/`, `scripts/`, `docs/`, the README or the plugin manifest)
  changed between the ship and the merge, most often because the merge queue
  batched the release pull request with another one — or the pinned address is
  not this repository's release, because `plugin.json`'s `repository` names
  another one. Nothing was tagged, so the version is still free. Land a
  follow-up pull request that fixes `main`: set the pin's `sha256` in
  `.claude-plugin/marketplace.json` to the rendered digest the refusal names (or
  revert the payload change), or correct `repository`. Its merge re-runs
  `detect`, which tags once the proof passes. Until then the catalog on `main`
  names an archive that does not exist, so installs and updates fail closed, as
  in the approval window.
- **`verify` fails on `Plugin archive reproduces the committed pin`.** The same
  proof, made again on the tagged commit. On the `auto-release` path it passed
  before the tag, so this is rare there; a hand-pushed tag has no earlier proof.
  The tag exists, so the version is consumed. Catch it before the merge instead:
  in a source checkout of the release branch,
  `go run ./cmd/abcd launch archive --out "$(mktemp -d)" --tag vX.Y.Z --verify --repository <owner/name>`,
  naming the repository the tag will be pushed to, exits 0 when the release
  will pass. Without `--repository` a pin whose address names another
  repository passes locally and is refused after the tag, consuming the version.
- **A new release never starts, and an older run sits `Waiting` forever.**
  Release runs are serialised, so one parked run blocks every later one. Cancel
  the stale run from its page (**Cancel workflow**), and the queued one starts.
  Cancel it *after* your release PR has merged, never before.

## Preview (`dry-run`)

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" launch --dry-run --json
```

Then summarise the JSON for the user:

- `version` — the version the release would carry.
- `bundle.files` — the files the bundle would include (an array; report its length as the count).
- `scan.hard_fails` — secret/PII findings that would block the release.
- `smoke.ok` — whether the payload would install: both plugin manifests parse,
  the marketplace source resolves, and every declared command, agent, skill and
  hook path is carried. `smoke.findings` names any path that is not.
- `gates` — every release gate and its disposition. Report the whole array,
  not a summary: `ran` gates carry their measurement, `not_implemented` ones name
  what is deferred, and `semantic-receipts` (`host-run`) reports which semantic
  receipts are recorded for the candidate commit. That row is the one a release
  fails on most expensively, so never omit it.
- `would_publish` — **always `false`** in a dry-run: this command previews and
  never publishes, and two gates are Phase-5 deferred, so it is not a verdict on
  the release. Read `gates` and `would_refuse_on` for that.
- `lockstep` and `retention` — the manifest-lockstep result and the release
  retention plan. Both feed `would_refuse_on`, so a lockstep drift or a
  retention refusal is invisible to anyone who reads only the gate list.
- `would_refuse_on` — if non-empty, the gates that would refuse, so the user
  knows what to fix before a real launch.

This is preview-only: publishing is not driven from this command.

## Ship — the release cut

A release cut is **three steps over two Go entry points**, with a host-run agent in
the middle. It is the `disembark` synthesis shape: a deterministic step, a
delegated composition, a validating ingest.

Those three steps write the CHANGELOG heading. They do **not** finish the
release. Two host-run semantic passes must also run and record receipts, and the
release branch has to carry them in a second commit — see *Semantic receipts*
below. A branch that skips them merges and tags cleanly and then fails at
`release.yml`'s fail-closed receipt gate, which is the most expensive place to
find out: the tag is already created by then, and the workflow never moves a tag.

### 1. Emit the cut (deterministic, writes nothing)

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" launch ship --json
```

The binary derives everything the release is allowed to be: the base tag, the
`next_tag`, the deciding `impact`, the record set (`added` and `removed`), the
surface guardrail's verdict (`guard`), and the unfixed-findings guardrail's
verdict (`findings`). Each entry carries `in_changelog` and `in_press_release`:
the second marks the intents the release page must cite (the user-facing intents
that entered `shipped/` since the base tag; never an issue, an `impact: internal`
intent, a removed intent or anything still planned). The human render lists them
under `release page:`, or says `release page: none` for a cut that ships fixes
alone. Read-only preview of the same thing: `abcd changelog --json`.

Exit codes gate the flow:

- **0** — the cut is ready. Continue to step 2.
- **1** — the cut **REFUSES**. Render the whole report to the user and **stop**.
  Every refusal names the specific record, version, or surface that blocks it — a
  release in flight, a merged feature whose intent still sits in `planned/`, a
  missing surface baseline, a surface break with no `breaking` record, or a
  consequential finding this cycle captured and never answered (see *The
  findings gate* below). A refusal is a result to relay, not a crash, and not
  something to work around.
- **2** — a structural fault (the repository could not be read). Relay it and stop.

### The findings gate

Both renders — `abcd changelog` and `abcd launch ship` — carry two lines about
the issue ledger, and they are the two most easily skipped lines in the report:

```
  findings:   failed (2 unfixed finding(s) captured since v0.7.0) (1 record(s) deleted from the ledger since v0.7.0)
    deferred: iss-2609012313465609 [major] — the CI split lands next cycle
```

**What refuses.** An issue record that entered the ledger *since the anchor tag*,
is graded `major` or `critical`, and is still in `open/`, refuses the cut under
the refusal kind `unfixed-finding`. A refused cut carries **no derived version**,
so nothing downstream has a release to make. A record whose `severity` is
missing, misspelled, or outside the ledger's enum refuses too: it has not been
judged, and "not judged" must not read as "not serious".

**A deletion refuses too**, under its own kind `deleted-finding`. A record the
anchor tag held in `open/`, graded the same way, that sits in no status directory
at HEAD has been removed from the ledger rather than answered — and the ledger's
status signal *is* folder membership, so a record in no folder has no status left
to read. Every other route leaves a trace the next reader can follow; this one
leaves nothing to audit, which is why it is named separately: the record has to
come back before it can be resolved, wontfixed or deferred.

The anchor is what bounds it. Records that already existed at the last tag are
the standing backlog and are never this cut's to answer; only what this cycle
itself captured is in scope. "It was already there when I started" is therefore
not available as a defence for anything the gate names.

**What the render shows.** The `findings:` line gives the verdict, the count of
unfixed findings and the anchor they were counted from, and the count deferred.
One `deferred:` line follows per waiver, naming the record, its severity and the
reason recorded on it. Report both to the user verbatim — a deferral nobody sees
in the report they actually read is indistinguishable from a finding that was
ignored. The whole verdict is on the cut's `findings` JSON key.

**The four routes out.** Relay them; do not pick one for the user.

1. **Fix it and resolve the record in this cut** — the intended answer. The
   change and its `Resolves:` trailer land in the release branch, the record
   reaches `.abcd/work/issues/resolved/`, and the gate stops seeing it.
2. **`abcd capture wontfix`** — the recorded decision not to fix. It clears the
   gate with no special case, because the gate looks only at `open/`, and a
   wontfix carries a stated reason. That is the conscious, cited non-action the
   rule asks for, not a loophole in it.
3. **The waiver pair.** Write it with
   `"${CLAUDE_PLUGIN_ROOT}/abcd" capture defer <iss-N> --after <anchor tag> --reason "<why>"`,
   which sets `deferred_after` and `deferral_reason` in the record's frontmatter
   and appends a dated `## Deferral` section, refusing a tag that is not the
   current anchor, an empty reason, and a record that is not open or not
   `major` or `critical`.
   `deferred_after` names the **anchor** — the tag the cut is measured from, not
   the version being derived — which is what makes the waiver single-use: at the
   next release the anchor moves and every waiver written against the old one
   lapses, so the finding is re-asked rather than forgotten. Half a waiver does
   not stand. One field without the other, or an anchor that is not this cut's,
   leaves the record blocking and the report says which of those it was.
4. **Re-grade the record honestly**, if and only if the severity was wrong when
   it was written. Downgrading a finding to get past the gate is the failure the
   gate exists to catch, and the record's history shows the edit.

Never delete the record to clear the gate — the cut refuses under
`deleted-finding` when you do — and never hand-edit `CHANGELOG.md` to route
around a refusal.

### 2. Compose the prose (host-delegated)

Run the **`release-changelog-composer`** agent
(`agents/release-changelog-composer.md`) over the emitted cut and the records it
names. It returns one payload (`schema_version` 2) carrying both documents:
`prompt_version`, `next_tag` echoed verbatim, `entries[{section, records, text}]`
for the changelog, and `press_release` for the release page —
`headlines[{records, text}]` (the intents told as prose), `listed[]` (every other
intent in the set, by id; the binary renders each as its title) and
`quotes[{record, text, attribution}]` (persona quotes carried word for word from
the press release of a told intent). `press_release` is `null` exactly when no
intent is marked `in_press_release`. The page looks back only: no date, no
future release, nothing planned. The agent owns
the **wording** and the choice between the two writable **Keep a Changelog
sections**, `Added` and `Fixed`; the version, the date, the heading, the section
order, the inclusion set and the writable set stay the binary's. `Changed`,
`Deprecated`, `Removed` and `Security` are claims about the previous release's
surface, which the composer cannot see, so the ingest refuses a payload carrying
one and the dated section says so under its heading (iss-2609011207114761). A
`breaking` record is an `Added` line that states the break.

**LOUD STAGE — if the composer cannot run in this context, the flow STOPS here.**
No fallback exists and none may be improvised:

- Do **not** hand-write the changelog lines yourself. Hand-written prose is the
  exact thing this flow abolishes, and prose written outside the composer carries a
  `prompt_version` that traces to no prompt — a provenance lie the payload has no
  way to express.
- Do **not** write a partial section, and do **not** invoke the ingest step with a
  payload covering some of the records "for now". The bijection would refuse it
  anyway; a partial cut is not a smaller release, it is a false one.
- Do **not** edit `CHANGELOG.md` or `RELEASE.md` by hand to unblock the release.

Say plainly that the composer is unreachable, that **nothing was written**, and
that the cut from step 1 is still valid and can be shipped once it is reachable.

### 3. Ingest the payload and write the release

Write the agent's payload to a file and hand it back to the binary:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" launch ship --changelog-json <path>   # or - for stdin
"${CLAUDE_PLUGIN_ROOT}/abcd" launch ship --changelog-json <path> --payload-dir <dir>   # also stage the payload
```

In a repository whose version-location contract
(`.abcd/config/version-location.json`) declares `"publishes_plugin_archive": true`
— the statement that its release workflow uploads the plugin archive — a written
heading is followed by the **archive pin**: the binary renders the release's plugin archive from the working tree, then rewrites
the plugin's `source` in `.claude-plugin/marketplace.json` to
`{"source": "archive", "url": ".../releases/download/vX.Y.Z/abcd-plugin-vX.Y.Z.zip",
"sha256": "<digest>"}` and refreshes the committed surface snapshot beside it.
The download address derives from `plugin.json`'s `repository`. Before anything
is written it refuses a payload with uncommitted changes, because the release
renders the archive again from the tagged commit and publishes nothing unless the
digests agree. These three files — `CHANGELOG.md`, the catalog and the snapshot —
are the release-content commit.

Without that declaration the catalog is left untouched, and the report says so
(`archive: not pinned — …`, or `archive_unpinned` in `--json`). The contract alone
is not enough: it says where the version lives, not that a release uploads an
archive, and the workflows `scaffold` writes for a managed repository upload
none — a catalog pinned there would name an asset every install fails to fetch.
A declaration that is not `true` or `false` is refused before anything is
written.

With `--payload-dir` the binary additionally stages the release payload in that
directory — an empty directory outside the repository — with the derived version
stamped into the payload's copies of `plugin.json` and `marketplace.json`. The
repository's own manifests are never touched: they carry no version, and the
version belongs to the artefact. The staged payload is proved consistent before
the command returns, so a stamp that missed a pinned location is a refusal rather
than a published half-state. Every refusal the staging step can make is checked
BEFORE the dated heading is written, and a refusal that slips past that check
rolls the heading back — so a ship that exits non-zero leaves no release record
behind for the next attempt to trip over. Without the flag nothing is staged;
`--payload-dir` on its own (no `--changelog-json`) is an operand error, because
only a completed cut has a version to stamp.

The binary re-derives the cut, then proves the prose describes it — the
**completeness bijection**: the set of record ids the payload cites must equal
`(added ∪ removed)` minus the records marked `in_changelog: false`. On any
mismatch it writes **nothing** and names three groups apart, because the fix
differs for each:

- **MISSING** — a shipped record no line cites; the release record would lie by
  omission.
- **INVENTED** — a cited id that is not in this cut at all.
- **INTERNAL** — a cited id that *is* in the cut but declares `impact: internal`;
  those records earn no changelog line.

The release page is held to the same rule: every intent marked
`in_press_release` is told in a headline or listed, once, and nothing else is
cited. Every quote must match, word for word and with its attribution after
`said` or `says`, a whole quoted sentence in the `## Press Release` section of an intent a
headline tells. The rendered page and the rendered changelog section are both
checked against the outbound policy (no session URL, no tool attribution
footer), and the page against the repository's persona registry
(`persona_registry`, which record-lint cannot reach at the root).

On success it writes, in this order, only after every check has passed:

1. **the archive** — the outgoing `RELEASE.md` moves to
   `.abcd/development/releases/<its version>.md`, the version read from its own
   heading. It never overwrites: an archive page already standing there stops
   the cut.
2. **the page** — the new `RELEASE.md`, whose heading `# Release X.Y.Z
   (YYYY-MM-DD)` names the release it describes.
3. **the changelog** — the dated section, spliced directly beneath
   `## [Unreleased]`, the insertion anchor, which must **exist** and be **empty**
   (a derived cut never folds hand-written prose into a generated section). It is
   written last, so the file the tagging workflow reads lands only after the page
   did.

If a later write fails, the earlier ones are rolled back, and the report says so
— or says `THE ROLLBACK FAILED — recover by hand` and names each file. On the
**first** cut no `RELEASE.md` exists, so nothing is archived and a rollback
removes the new page. A cut that ships **fixes alone** writes the changelog
section only: `RELEASE.md` and the archive stay as they are, and the report says
`No release page written: no user-facing intent shipped in this cut; RELEASE.md
stays on <version>`.

Exit codes, same shape as step 1:

- **0** — written. Report `heading`, `path`, `lines`, `cited` (the proof set) and
  `page` (the page's path, heading and counts, and `archived` when a page moved;
  or its `reason` when none was written).
- **1** — the **cut** refuses. Nothing was written, whatever the payload said.
- **2 with `payload_refusal`** — the **payload** is refused: nothing was written,
  and the composer must rewrite it. See *The retry loop* below.
- **2 without `payload_refusal`** — a **stop**: the repository cannot take the
  cut (unreadable, a degraded scanner config, an outgoing `RELEASE.md` without a
  release heading, an archive collision, a missing or non-empty
  `## [Unreleased]`, or a failed write that was rolled back). No rewrite of the
  payload can fix it. Relay the report and stop.

### The retry loop: a refused payload is recomposed

A refused payload is rewritten automatically, with **no attempt limit**. With
`--json` the refusal reads:

```json
{"cut": {}, "written": false, "payload_refusal": {"reasons": [{"code": "quote-not-verbatim", "at": "press_release.quotes[0]", "detail": "..."}]}}
```

Loop until the ingest exits 0 or stops:

1. **Tell the user at once** that attempt N was refused, and list every reason
   (`code`, `at`, `detail`). The loop runs in the open, so an autonomous run shows
   it as it happens.
2. On `stale-cut`, re-run step 1 first: the record set moved under the composer.
3. Re-invoke the composer with the cut, its previous payload and the reasons, and
   ingest the new payload.

In the final cut report, **report every refused attempt** and its reasons before
the written result. The loop stops only on success, on a stop (exit 2 without
`payload_refusal`, or exit 1), or when a person stops it — a composer that keeps
making the same fault loops until someone does, and the per-attempt report is
what makes that visible. If the composer cannot run at all, the LOUD STAGE above
applies: nothing is hand-written.

The reason codes:

| Code | The payload |
| --- | --- |
| `payload-oversize` | is over the 1 MiB cap |
| `malformed-json` | is not one JSON document |
| `unknown-field` | carries a key the contract does not have |
| `trailing-data` | carries data after the document |
| `schema-version` | is not `schema_version` 2 |
| `prompt-version` | carries no semver `prompt_version` |
| `stale-cut` | echoes a `next_tag` this cut no longer derives |
| `text-oversize` | carries a text over 4096 bytes, or too many entries, headlines, quotes or records on one line |
| `malformed-id` | cites something that is not `itd-N` or `iss-N` |
| `no-citation` | has a line or headline citing nothing |
| `empty-prose` | has a line or headline with no prose |
| `no-entries` | carries no changelog lines |
| `section` | names a section Keep a Changelog does not register |
| `section-not-writable` | names `Changed`, `Deprecated`, `Removed` or `Security` |
| `changelog-missing` | leaves a required record uncited in the changelog |
| `changelog-invented` | cites a record not in this cut in the changelog |
| `changelog-internal` | cites an `impact: internal` record in the changelog |
| `missing` | leaves an intent in the page's set neither told nor listed |
| `outside-set` | cites on the page an id outside its set (the detail says why: not in this cut, an issue, internal, or removed) |
| `duplicate-citation` | cites one intent, or carries one quote, twice on the page |
| `no-headline` | tells no intent on a page that has a set |
| `page-for-empty-set` | carries a page for a cut that ships fixes alone |
| `heading` | has a page text opening with `#` |
| `fence` | has a page text carrying a code fence |
| `blockquote` | has a page text opening with `>`, or a headline attributing words with `said <Name>,` |
| `quote-source` | quotes an intent no headline tells |
| `quote-not-verbatim` | carries a quote that is not word for word from its intent's press release, with its attribution |
| `outbound-policy` | would put a session URL or a tool attribution footer in the page or the changelog |
| `persona-registry` | would put on the page words attributed to a persona the registry does not hold |

Then show the user the written heading and the diff, so a human reviews the release
record before it is committed. This command never commits, tags, or publishes.

## Semantic receipts — the second half of the cut

`release.yml` arms `receipt_gate` fail-closed against the release **content**
commit. It refuses unless every required gate has a PROMOTE receipt naming that
exact commit, pinning a judge model, and declaring the matching detector. CI
cannot produce these: they spawn LLM agents, so they are host-run, and an un-run
pass is never a silent pass.

**The release branch is exactly two commits.** A receipt names the commit its
reviewer read and must live in a LATER commit — it can never sit in the tree of
the commit it names, because adding it would change that commit's sha. So:

1. **The CHANGELOG roll** — the release-content commit, written by the three
   steps above: the dated section, and on a feature cut the release page and its
   archive move. This is what the reviewers read.
2. **The receipts** — a commit recording the semantic verdicts that name commit 1.

On merge, `release.yml` derives the content commit as `<merge>^2^` and finds its
receipts in the released tree. A one-commit branch breaks this: the single commit
is taken as the receipts commit, the gate arms against whatever preceded it, and
no receipt names that commit.

### Running the passes

Run both in the agent harness against commit 1, then commit their receipts:

- **`docs-currency-reviewer`** — verifies every user-facing claim still matches
  the code. The agent is `agents/docs-currency-reviewer.md`.
- **`iss35-brief-surface-crosscheck`** — the brief's surface prose against the
  shipped binary's actual behaviour. Its scope, depth and prompt are pinned by
  [`.abcd/development/release-gate/manifest.json`](../.abcd/development/release-gate/manifest.json),
  and a receipt echoes that file's sha256 as `manifestHash`.

Receipts live at `.abcd/work/reviews/<content-sha>/<gate>.json`. The shape is
[`.abcd/development/release-gate/receipt.example.json`](../.abcd/development/release-gate/receipt.example.json);
the full procedure, including the tiered depth a release's impact class requires,
is the adr-37 runbook at
[`.abcd/development/release-gate/README.md`](../.abcd/development/release-gate/README.md).

A receipt is bound to its gate by its `policy.detector` value, not its filename,
and a mismatched, malformed, HOLD, model-less or wrong-detector receipt blocks.
Report a HOLD to the user and stop: a HOLD is a result, not an obstacle to route
around, and the receipts cannot be hand-written to unblock a release.

### Prove the gate before you merge

`receipt_gate` runs inside the release job, which is **after** the tag is
created. A refusal there does not block the release, it consumes the version: the
workflow never moves a tag, and its recovery path rebuilds from the tagged
commit, whose tree can never gain the missing receipts. Recovering means deleting
a tag the machinery treats as immutable (recorded as `adr-52`, undecided).

So reproduce the gate's verdict locally, on the release branch, while nothing is
tagged. From the repository root:

```bash
go run ./cmd/record-lint --release-gate <content-commit-sha> \
  --require-gate docs-currency-reviewer \
  --require-gate iss35-brief-surface-crosscheck
```

- `<content-commit-sha>` is the **full 40-character** sha of the commit the
  receipts name, which on a correctly shaped release branch is the receipts
  commit's parent (`git rev-parse HEAD^`). Use the full sha: an abbreviated one
  is well-formed, finds no receipt, and makes the gate refuse as though the
  semantic pass had never run.
- `record-lint` is a repository-local program, not an installed binary. `go run
  ./cmd/record-lint` is the invocation; there is no `record-lint` on `PATH`.
- The required-gate names come from `release.yml`, which owns that list on
  purpose. If they diverge, the workflow is right and this command is stale.

**Exit 0 means the release will pass the gate.** A non-zero exit names what is
missing, and costs nothing to fix, because no tag exists yet.

## Archive — the release's pinned plugin archive

`archive` renders the plugin archive of the release the newest dated CHANGELOG
heading names, from the checked-out tree, and writes
`<plugin>-plugin-vX.Y.Z.zip` into an existing directory. The archive is
reproducible — sorted entries, stored uncompressed, one fixed timestamp, modes
normalised — so the same commit renders the same bytes on any machine. The
catalog is left out of it, because the catalog is what names its digest.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" launch archive --out <dir> [--tag vX.Y.Z] [--verify] [--repository <owner/name>] --json
```

- `--tag` refuses unless the newest dated CHANGELOG version is that tag.
- `--verify` refuses unless the committed catalog pins exactly this archive's
  download address and digest. `auto-release.yml` runs it on the pushed commit
  in `detect`, before the tag is made; the release workflow runs it again on the
  tagged commit in `verify`, before anything is built, and once more in the
  publish job, where the verified archive is the file it checksums, attests and
  uploads.
- `--repository <owner/name>` refuses unless the archive's download address lies
  under that repository's
  `https://github.com/<owner>/<name>/releases/download/<tag>/`, compared
  case-insensitively. The address derives from `plugin.json`'s `repository`,
  which a rename, a transfer or a fork leaves naming another repository —
  `--verify` passes on it and every install fails to fetch. Every run in the
  workflows passes `--repository "${GITHUB_REPOSITORY}"`.

Exit codes: **0** the archive was written (and every gate asked for passed);
**1** `--verify` or `--repository` refused — the report names both digests or
both addresses, and the archive is removed so no later step can publish it;
**2** a structural fault (no dated release, a `--tag` naming another release, a
`--repository` that is not `owner/name`, an unusable `--out`, a render refusal),
with nothing left behind.

Relay `archive.name`, `archive.sha256`, `url`, `pin` and `repository`. Between releases, `main`
pins the last release's archive, which its moved-on tree no longer reproduces, so
`--verify` there is expected to refuse: it proves a release commit, not a branch
tip.

## Scaffold — the release-gate scaffolder

`scaffold` writes the changelog-driven release machinery into a **managed repo that
lacks it** — a different job from `ship` (which cuts a release in a repo that
already has the machinery). It **never publishes**.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" launch scaffold --json
```

It writes three files, wired to the repo's own default branch and Go version:

- `.github/workflows/release.yml` — verify → build → publish, the verify gate
  armed against the reviewed **content** commit (`HEAD^2^` on the auto-release
  merge path, `HEAD^` on a direct tag), so the first public release cannot hit the
  receipt-vs-tag self-reference.
- `.github/workflows/auto-release.yml` — newest dated CHANGELOG heading → tag that
  commit → call `release.yml`. `GITHUB_TOKEN`-only, no personal access token.
- `.abcd/development/release-gate/README.md` — the adr-37 runbook.

The workflows come from one embedded template that abcd-cli's own release
workflows are regenerated from (self-scaffold parity), so every abcd release
exercises the exact machinery a managed repo receives. The scaffolded `release.yml`
carries a `workflow_dispatch` **rehearsal**: run it green once before the first
real release — it arms the full gate against a simulated changelog roll and
reviewed-content commit, proves the gate admits, and publishes nothing (no tag,
Release, or attestation).

It is idempotent and fail-safe. Exit codes gate the flow:

- **0** — every file written, or already current (a no-op re-run). Report the
  per-file disposition.
- **1** — a file exists and **differs** (hand-edited or stale); the report names
  it and **nothing was written**. Relay it; re-run with `--confirm` to overwrite.
- **2** — a structural fault (the repository or a template could not be read).

A refusal is a result to relay, not a crash. Never hand-edit the workflows to work
around it: re-run with `--confirm` when the operator intends to replace the drift.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
