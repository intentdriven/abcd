# Release gate — the pre-tag procedure

Every `v*` release must clear this gate before the tag is pushed. The gate spans
**two enforcement planes**, by design: deterministic checks run in CI (they need
no model); semantic checks run **host-side in the agent harness** (they need a
model, which CI does not have). This runbook is the single human enumeration of
both planes; it owns the semantic half and defers to
[`release.yml`](../../../.github/workflows/release.yml) as the source of truth
for the deterministic half.

Design of record: [`../plans/2026-07-11-iss35-semantic-release-gate.md`](../plans/2026-07-11-iss35-semantic-release-gate.md).

## Deterministic gates (CI-enforced)

The [`release.yml`](../../../.github/workflows/release.yml) `verify` job runs
these, in order, on Linux (`ubuntu-latest`); the macOS + Linux matrix is
`ci.yml`'s `check` job, which gates the merge. `release.yml` is authoritative;
this list is the human-readable mirror.

1. Format (gofmt)
2. Build
3. Vet
4. Test
5. Test (race, internal)
6. Record-lint (design-record drift gate)
7. Docs-lint (docs-currency gate)
8. Reviews-charter discipline (RD001-RD003)
9. Smoke every command (self-discovering harness)

This list is machine-checked: the `gate_lockstep` `record-lint` rule blocks if it
diverges from `release.yml`'s `verify` job steps (setup steps excepted). Edit both
together — the mirror cannot silently drift.

## Enforced at merge only

`verify` is a subset of what CI enforces at merge, not a substitute for it: the
merge gate additionally runs the issue-resolution gate (RS001-RS003 plus its
cases), the site-render gate, the macOS leg, secret scanning, the workflow
audit, dependency review, `govulncheck`, the attribution gate and the external
review. The normal release path (`auto-release` tagging the main tip) re-checks
nothing those gates have not already passed on the exact tagged commit; a
hand-pushed tag on a commit outside `main` forgoes them, which is one more
reason the runbook's branch → PR → merge path is the supported one.

## Semantic gates (host-run, before the tag)

CI cannot run these — they spawn LLM agents. Run them in the agent harness
against the exact commit to be tagged:

1. **`docs-currency-reviewer`** — verifies every user-facing claim still matches
   the code (the semantic complement of `docs lint`; see
   [`../brief/04-surfaces/10-docs.md`](../brief/04-surfaces/10-docs.md)).
2. **Brief↔surface cross-check** — [`brief-surface-crosscheck.js`](brief-surface-crosscheck.js),
   the Direction-A semantic half of the iss-35 graduation: the brief's surface
   *prose* (flags, sub-verbs, exit codes, schema fields, counts) vs. the shipped
   binary's actual behaviour. The deterministic Direction-B half is the
   `surface_coverage` `record-lint` rule and already runs in CI. Its scope and
   depth are pinned by [`manifest.json`](manifest.json) — see *Pinned inputs and
   tiered depth* below.

## Recording the semantic verdict

**A receipt names the commit its reviewer READ, and lives in a LATER commit** —
it can never sit in the tree of the commit it names, because adding it would
change that commit's sha. Under changelog-driven auto-release (adr-37) the
release branch is therefore exactly two commits: the CHANGELOG roll (the
release-content commit the reviewers read), then the semantic receipts naming
it. On merge, the released tree carries the receipts, and `release.yml` derives
the *content* commit from the **receipts directory** it carries — the receipts
name the commit they gate, so `record-lint --derive-content-sha` reads the
`.abcd/work/reviews/<sha>/` entry on the released lineage and returns that `<sha>`
— not the merge commit, and not `<merge>^2^` ancestry, which a batched
merge-queue push can point at an unrelated PR's commit (`github.sha` is the batch
tip, iss-355). `subject.digest.gitCommit` therefore still matches the armed
commit exactly and the gate stays strict. (Before this, the gate armed with the tagged merge commit, whose
tree can never hold a receipt naming itself — an unsatisfiable self-reference.
Dormant while the repo was private, it surfaced at the first public release and
fail-closed it, v0.3.0, iss-108.)

Each semantic pass records its outcome as a **commit-sha-keyed receipt** — a
Verification Summary Attestation (VSA) shape carrying `verificationResult`
(PROMOTE / HOLD), the pinned judge-model snapshot, the detector version, and the
failing categories. Receipts live at `.abcd/work/reviews/<commit-sha>/<gate>.json`;
[`receipt.example.json`](receipt.example.json) is the concrete shape. The
`receipt_gate` `record-lint` rule verifies, before a tag, that each required gate
has a PROMOTE receipt whose subject digest is the target commit, whose
`policy.detector` names that gate, and which pins a judge model; a missing,
mismatched, malformed, HOLD, model-less, or wrong-detector receipt **blocks** the
release (fail-closed — an un-run semantic pass is never a silent pass). Pinned
is checked by shape: `judgeModel` must carry a version or date component
(`claude-opus-4-8`, a dated snapshot id) and must not be a rolling alias — a
bare family name (`opus`) or anything naming `latest` is refused, because a
receipt that cannot say which judge produced it cannot be re-run against that
judge.

A receipt is bound to its gate by its `policy.detector` value, not by its
filename: the `<gate>.json` at `.abcd/work/reviews/<sha>/` must carry
`policy.detector` equal to `<gate>`. This stops one genuine PROMOTE receipt from
being copied across every gate's path to satisfy them all — each gate needs its
own receipt from its own detector.

## Pinned inputs and tiered depth

The cross-check runs against a **committed input manifest**,
[`manifest.json`](manifest.json), which pins the reproducibility inputs: the
brief-doc list (every `04-surfaces/` chapter plus the constraints, internals
and delivery chapters that count or enumerate the shipped surface), the two
directions (A: brief→surface, B: surface→brief), the checker count (one per
brief doc plus one per surface), and the prompt (context plus both direction
templates, with the prompt's own `sha256` over the three parts joined by blank
lines). The context names every `commands/` page and every `agents/` prompt,
and says that a surface claim found outside the pinned chapters is in scope.
Two honest runs of the same tier therefore mean the same thing — the
maintainer no longer chooses the scope per run. The
[`brief-surface-crosscheck.js`](brief-surface-crosscheck.js) detector consumes
this manifest as its input rather than composing an ad-hoc list.

Pinning makes runs comparable to each other, not to the tree: a surface that
ships without joining the pin is invisible to a run that obeys it, and the
receipt attests coverage it never had. `TestReleaseGateManifestIsCurrent`
(`internal/core/lint`) holds the manifest to the tree — every `04-surfaces/`
chapter pinned, the pinned command and agent rosters equal to `commands/` and
`agents/` on disk, `checkerCount` equal to the sum of the lists, `promptHash`
equal to the prompt text — so a new verb, agent or chapter fails the unit
suite until the manifest names it, and the `manifestHash` a receipt echoes
changes with it.

Depth is **tiered by the release's impact class**:

- **`full`** — both directions over the whole brief-doc list — is required for a
  **feature (additive) or breaking** release.
- **`shallow`** — Direction B only — is sufficient for a **patch (fix/internal)**
  release.

The impact class is derived at gate time from the shipped records — the same
signal the version itself derives from (changelog-driven versioning, adr-37).
The version *number* alone cannot carry it while abcd is pre-1.0: an additive
feature and a fix both bump the patch component at 0.x, so only the records'
`impact` frontmatter tells a feature release apart from a patch one. The gate
reads the strongest impact in the cut between the newest release tag older than
the current CHANGELOG version and `HEAD`.

A manifest-era receipt therefore carries two further fields alongside the ones
above:

- **`manifestHash`** — the `sha256` of `manifest.json` the run pinned its inputs
  against.
- **`tier`** — `full` or `shallow`, the depth the run used.

`receipt_gate` adds three **procedural** refusals once the manifest exists in the
release's content tree:

1. `manifestHash` does not equal `manifest.json`'s actual hash — the run's pinned
   inputs are not this release's.
2. `tier` is insufficient for the release's impact class — a shallow receipt on a
   feature or breaking release.
3. A `failing` entry carries no `disposition` — an un-triaged finding.

None of these judges a finding's **content or severity**. Confirmed findings
route to the maintainer, whose PROMOTE with recorded dispositions is the gate
(verifier-selects-gates-decide); the gate does not hard-block on a confirmed
major and keeps no never-worse ratchet across tiers.

**Era gating.** The manifest's presence in the armed content tree is the era
marker. The three refusals above arm only when `manifest.json` is present at the
gate's armed commit, so receipts written before the manifest existed — the
receipts committed under `.abcd/work/reviews/` for earlier releases — stay valid
for their own commits, judged by the checks that predate the manifest.

The `receipt_gate` rule is **disabled by default** — it must never fire on
ordinary PRs/pushes, only at release time — and is armed by `release.yml`'s
**`verify` job, before the tag** (adr-52): the semantic gate joins the
deterministic gates that refuse before anything is built or published, so a
semantic refusal blocks the release without the version-consuming wedge of a gate
that sat in the publish path (iss-2608231226347380). `verify` supplies the
content commit and the required-gate list from the workflow (the trust root), not
the in-tree config: `record-lint --release-gate <sha> --require-gate <name>…`,
where `<sha>` is `record-lint --derive-content-sha`. The rule is skipped on the
rehearsal path (`workflow_dispatch`), where no real receipts exist. Once `verify`
has admitted the release, `release.yml`'s publish job signs the receipts with
`actions/attest` (predicate `.../semantic-release-gate/v1`) and verifies the
attestation with `gh attestation verify` — no new dependency (the same attest
family + `gh` the binary provenance already uses).

> **Live, and visibility-gated.** Artifact attestation is a public-repo
> feature, so the whole gate is gated `if: !github.event.repository.private` —
> exactly like the binary attestation — and does nothing on a private repo. The
> repo is public, so the gate is armed on every release, and it has fail-closed
> two of them: v0.3.0 on the self-reference defect (iss-108) and v0.6.0 on a
> missing receipts commit (iss-326). And the signature is **auditable release
> provenance, not committer-forgery-proof**: a receipt forged and committed
> before the tag would be signed too; that residual is bounded by the iss-62
> identity gate + branch protection. Stated per
> [`../principles/loud-staging.md`](../principles/loud-staging.md), not implied
> away. Full forgery-prevention would need host-side signing at receipt
> production — a later step if the threat model warrants it.

## Procedure

1. Land all work on the release commit; open the release the normal way (branch
   → PR → merge). `ci.yml`'s required checks gate the merge; `verify` re-runs
   the deterministic gates against the tagged commit before anything is
   published.
2. On the merged commit, run the two semantic gates above in the harness.
3. Record each verdict as a receipt keyed to that commit's sha.
4. Tag `vX.Y.Z` on the commit. `receipt_gate` is armed fail-closed in
   `release.yml`'s **`verify` job, before the binaries are built or published**
   (adr-52): the release is refused unless every semantic receipt is present and
   PROMOTE, alongside the deterministic gates. The tag itself is never moved
   (anti-tag-move); moving the gate to the safe side of the tag is what stops a
   semantic refusal from wedging a version the way a gate in the publish path did
   (iss-2608231226347380, iss-326). On the auto-release path the `tag` job still
   mints the tag before it invokes `release.yml`, so closing that residual for the
   automated path is tracked separately.
