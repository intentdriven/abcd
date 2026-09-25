# Intent-lifecycle automation — SOTA-researched design plan

**Status:** design plan recorded 2026-07-11 for **later execution** — not built.
Informed by a `sota-researcher` pass on how to build it (this doc distils that
verdict). The specification already lives across planned intents
([itd-34](../intents/planned/itd-34-three-intent-kinds.md),
[itd-46](../intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md),
[itd-48](../intents/planned/itd-48-intent-fidelity-reviewer-roles-2-3.md),
[itd-50](../intents/planned/itd-50-loop-toward-acceptance.md),
[itd-53](../intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md)) plus
`brief/04-surfaces/05-intent.md` and `intents/README.md`; this plan is the
build-shaped synthesis, not a new spec.

## The capability

An intent flows `drafts/ → planned/ → shipped/`, where the directory IS the state
(adr-26 directory-as-truth; no `status:` field, no `move` verb):
1. `abcd intent plan <itd-N>` links the intent to a native spec and moves it to
   `planned/` (bidirectional link: intent `spec_id: spc-N`, spec `intent: itd-N`).
2. When the linked spec is marked done, a deterministic **reconcile** step detects
   it via the link and moves the intent to `shipped/`.
3. The move emits a host-delegated **fidelity review** whose verdict — per-criterion
   (MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE) + a three-bucket audit
   (honoured / diverged / missing) — is ingested into `## Audit Notes`.

itd-3 was just driven through this by hand (PR #19) as the manual precedent — that
is the acceptance test the automation must reproduce.

## SOTA verdict (fit-aware) — the load-bearing findings

The verdict's headline: abcd's existing ADRs (25/26/27 + the VSA release-gate
shape) already sit on or ahead of SOTA. The build is mostly **naming what we have
after its mature counterpart** and closing two small gaps.

- **Directory-as-truth is right; make moves `os.Rename` (atomic, same-filesystem,
  stdlib).** The git-native trackers (git-bug CRDT, Fossil) that avoided
  directory-state did so for multi-writer merge — a pressure a single-maintainer
  config tool does not have. Do **not** make the compound plan/ship operation
  transactional; make each fact single-authoritative and let a **reconcile pass**
  repair a torn state (the control-loop shape adr-27 receipt drain already has).
- **Bidirectional link = ID-anchored, spec-side authoritative.** Anchor on the
  immutable `itd-N`/`spc-N` ids, never paths (paths move — that is the point). The
  spec's `intent: itd-N` is load-bearing; the intent's `spec_id` is the derived
  cache the reconcile rewrites on drift. **The link carries identity only** —
  never cache lifecycle state into it (that recreates the dual-source-of-truth
  adr-26 forbids). Lint detects drift (side-effect-free); reconcile repairs it.
- **LLM-as-judge fidelity review:** keep per-criterion + the 4-value ordinal +
  the explicit INCONCLUSIVE bucket + report-not-block — the 2025-26 calibration
  literature independently converged on exactly these (judges are volatile and
  poorly calibrated where humans disagree; abstention beats a forced wrong
  verdict; never gate a merge on a volatile signal). **Two gaps to close:**
  (1) **require a cited evidence pointer** (`file:line` / diff hunk) per criterion
  verdict and per gap-audit entry — kills the documented "no traceability
  artifact" failure of generative judges; (2) **pin `judge_id` + `judge_version`
  + `prompt_hash` + `rubric_hash`** into the receipt — host-delegation (adr-25)
  means we cannot force temperature 0, but we can record the conditions, and the
  async prompt-as-artifact gives us that pinning for free. Phrase failing
  conditions explicitly ("NOT_MET if …") to counter leniency bias.
- **Async ingestion = the transactional outbox/inbox pattern.** Emit = outbox
  slot; the parked receipt = correlation record carrying an idempotency key;
  ingest = inbox: schema-gate (syntactic) → semantic-check (every `criterion_id`
  exists in the intent, every verdict in-enum) → **idempotent append keyed on the
  receipt id** (re-ingest is a no-op). A malformed/absent verdict after retry →
  receipt `DEAD_LETTER`, affected criteria recorded `INCONCLUSIVE`, raw payload
  retained. All stdlib (`encoding/json` + hand-rolled validation, no new dep).
- **Shape the verdict like the existing VSA release-gate attestation** — the judge
  is a `verifier`, the intent is the `resourceUri`, the spec+diff are
  `inputAttestations`, the rubric/prompt is `policy`, per-criterion verdicts
  extend `verificationResult` to an array. One attestation idiom in the repo; do
  not invent a bespoke envelope.

## Native floor + easy opt-in to a better external (adr-22) — repo-wide framing

abcd's pattern is a **basic native default that always works, plus an easy path to
onboard a superior external tool** — the same seam shape as
`oracle`/`spec`/`run`/`history`/`scanner`. This plan honours it on both sides:

- **The native minimal spec store is the floor**, not the ceiling. adr-26 already
  frames the richer spec seam with the companion harness as an opt-in backend; the
  lifecycle needs only the minimum (link, status-by-directory, reconcile), and a
  better external spec/lifecycle engine onboards behind the seam later.
- **The fidelity-review is host-delegated (adr-25)** — abcd emits the prompt +
  receipt; the *better external judge* is whatever the host runs. The native floor
  is the deterministic emit/ingest/reconcile plumbing, which never depends on any
  one model.
- **Sibling application — the rules loader (itd-3).** Same principle: itd-3
  shipped the native Go loader as the floor; the opt-in "does it better" backend
  is plausibly **CARL**, onboarded behind a `rules` seam (`rules.backend: native |
  carl`). Captured as `iss-64`; not part of this plan, but it is the same shape and
  worth building in lockstep with the seam habit.

## Recommended first slice (the steel thread) — one real intent, end-to-end

Vertical, not horizontal: drive one intent `planned → shipped` with a real audit.
1. `abcd intent plan <itd-N>` writes the ID-anchored bidirectional link + `os.Rename`
   to `planned/`, minting the `spc-N` id (itd-3 reserved `spc-1`).
2. Reconcile: linked spec marked done → `os.Rename` to `shipped/`, rewriting any
   drifted derived side; keep the `intent_lifecycle` record-lint contract green.
3. The move emits one VSA-shaped review request + parks one `OWED` receipt.
4. Host runs the markdown `intent-fidelity-reviewer` agent (a new `agents/` dir);
   the verdict JSON is schema-gated + semantic-checked on ingest.
5. Per-criterion verdicts + honoured/diverged/missing (each with a **cited**
   evidence pointer) appended idempotently to `## Audit Notes`.

**Explicit deferrals (not in slice 1):** the inter-intent dependency graph;
multi-role or multi-judge reviewers (the verdict is clear that correlated-error
juries are pure cost at this scale); loop-to-acceptance (itd-50 — slice 1 is
one-shot, report-only); auto-generation/auto-planning of the spec; a
calibration/human-agreement harness (add only once a real verdict corpus exists);
cross-filesystem move safety (we are always same-fs).

## Verdict / receipt schema sketch (VSA-shaped — adopt directly)

```jsonc
// review-receipt — parked at the ship-move (outbox + correlation record)
{
  "receipt_id": "rcp-<content-hash>",              // idempotency / dedup key
  "intent_id": "itd-N", "spec_id": "spc-N",
  "resource_digest": "sha256:<intent+spec+diff>",  // what was reviewed
  "rubric_hash": "sha256:<acceptance-criteria>",
  "prompt_hash": "sha256:<emitted-prompt>",
  "status": "OWED"                                 // OWED -> INGESTED | DEAD_LETTER
}

// review-verdict — host returns this; schema-gated at ingest; VSA-shaped
{
  "_type": "abcd/intent-fidelity-verdict/v1",
  "receipt_id": "rcp-<content-hash>",              // must match a parked receipt
  "verifier": { "id": "<judge-agent>", "version": "<model-id>" },   // pinning
  "policy": { "rubric_hash": "sha256:...", "prompt_hash": "sha256:..." },
  "input_attestations": [ { "kind": "diff", "ref": "<commit-range>", "digest": "sha256:..." } ],
  "criteria": [
    { "criterion_id": "ac-1",
      "verdict": "MET|MET_WITH_CONCERNS|NOT_MET|INCONCLUSIVE",
      "rationale": "reasoning before verdict",
      "evidence": [ { "ref": "internal/core/x.go:42", "quote": "..." } ] }   // REQUIRED, cited
  ],
  "gap_audit": {
    "honoured": [ { "claim": "...", "evidence": [ ] } ],
    "diverged": [ { "claim": "...", "evidence": [ ] } ],
    "missing":  [ { "claim": "...", "evidence": [ ] } ]
  }
}
```

## Not worth adopting (investigated, rejected)

Multi-judge juries (correlated errors → far less than N votes at N× cost); a
`status:` field beside the directory (adr-26 forbids; the drift generator); a
synchronous/embedded judge (violates adr-25 and loses the pinned-prompt-as-artifact
async gives for free); holistic numeric scores or fine confidence (halo effect +
scale-sensitivity; the 4-value ordinal is in the reliable band); a CRDT op-log for
lifecycle state (solves a multi-writer merge we don't have); spec-kit
branch-per-spec (fragments the durable `shipped/` record we specifically want).

## Execution

A full autonomous-run brief (design-gate-first, prefer-sota, phased TDD, the
itd-3-style cadence and gates) is prepared for a `/loop` session. The design gate
in that run must still decompose and stop for maintainer sign-off before building
— this plan is the design input, not the authorisation.
