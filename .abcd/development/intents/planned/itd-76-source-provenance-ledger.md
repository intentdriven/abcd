---
id: itd-76
slug: source-provenance-ledger
spec_id: spc-31
kind: standalone
suggested_kind: null
reclassification_history: []
severity: major
impact: additive
blocked_by: [itd-74]
builds_on: [itd-77]
---

# abcd Lets You Consult Sources You Cannot Cite — and Remembers Every Debt

## Press Release

> **Consult any source freely; cite only by deliberate human choice — abcd keeps the ledger in between.** A developer-researcher often works from material they are not free to name in public: working papers under review, a collaborator's private repository, notes shared under an NDA. The agent should be able to *read* that material when it bears on a decision — and must never *cite* it in anything published. abcd manages this with a personal corpus and a provenance ledger, guarded mechanically. A **local-only corpus** (abcd's user-level home, `~/.abcd/sources/` by default, itself a no-remote git repository) holds the documents and a machine-readable bibliography (CSL-JSON; a `custom` block carries `confidential`, `permission_status`, retrieval keywords, and banned aliases). An **append-only provenance ledger** (JSONL, one file per consuming repo) records every meaningful influence — which source, which decision, what claim, what kind of influence — gated twice: the source's `permission_status` grants the *right* to cite, and each ledger line's `cited_publicly` flag *exercises* it, flipped only by the human. And the **guardrails are mechanical for what mechanism can catch**: the confidential entries *generate* the pattern block in each repo's untracked private-names banlist, the pre-commit guard refreshes and enforces it on every commit, and a `cite-check` scan clears any text before it is shared.
>
> "I could never let an agent near my working papers before, because one helpful footnote could burn a collaborator's trust," said Alice, a researcher-developer. "Now it reads everything, records what influenced what, and cites nothing. When the paper behind a decision is finally published, I flip one flag — and the whole influence trail is already written."

## Why This Matters

Automatic citation is a virtue that becomes a breach the moment a source is confidential: an agent that helpfully names "the working paper this design follows" in a commit message has leaked something no history rewrite fully recalls. The naive fix — keep the material away from the agent — throws away exactly the context that makes its design work good. The resolution is to split *consultation* from *citation* and put a durable, machine-readable record between them: influence is captured eagerly and automatically (cheap, local, append-only), citation happens lazily and manually (when permission exists). The ledger is also the seed of something bigger — a team bibliography and a reconstructable paper — but those travel as their own intents ([itd-126](../drafts/itd-126-a-team-shares-one-bibliography-without-sharing-anyone-s-corp.md), [itd-127](../drafts/itd-127-a-paper-is-reconstructed-from-the-provenance-ledger-claims-g.md), each `refines` this one); this intent is the personal core they stand on.

This composes existing abcd designs rather than inventing new machinery: the two-layer name banlist ([itd-74](../shipped/itd-74-name-banlist.md)) supplies the leak guard; the append-only audit chain ([itd-16](../drafts/itd-16-hash-chain-merkle-audit.md)) is a possible later integrity backend for the ledger — the corpus repo's git history carries tamper-evidence until then, and nothing here depends on itd-16 shipping; and the provenance substrate ([`09-provenance-substrate.md`](../../brief/05-internals/09-provenance-substrate.md)) already defines citation blocks, a source registry, and an NDA-aware publish gate for *ingested* content — this intent extends the same stance to *consulted* content. The trust boundary itself — documents and ledgers never leave the user tier; a public citation requires both gates — is recorded as [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md) and brief invariant 9, which this intent cites rather than declares; the standing stance is the [consult-freely-cite-deliberately](../../principles/consult-freely-cite-deliberately.md) principle.

## What It Looks Like

- **`abcd source add`** registers a document: CSL-JSON entry (permission status, keywords, aliases), original and extracted text stored in a per-source folder whose *location* — `confidential/<key>/` or `public/<key>/` — is the classification, declared once at ingestion. Derived artifacts (summaries, notes) live in the same folder and inherit the class; declassification is a visible move, never a silent flag edit. Consultation is plain search over the folders — no index, no service. The corpus lives in abcd's **user-level home** (`~/.abcd/`, path configurable; relocation is [itd-77](../drafts/itd-77-relocatable-user-home.md)) — the first user-tier surface alongside the repo's `.abcd/` tiers.
- **`abcd source ledger`** appends an influence record — `{decision_ref, claim, source_key, locator, influence, cited_publicly}` — and commits it; corrections are new lines, never edits. A public citation requires **both** gates: the source permits (`permission_status`) *and* the line is flipped (`cited_publicly: true`).
- **`abcd source sync-banlist`** projects every confidential entry's identifying strings — title and aliases always; author names only by per-source opt-in (`ban_authors: true`, for the rare collaboration that is itself secret — the common confidential types, one's own submitted work, purchased reports, and private repos, are protected by title and aliases, and banning their authors would mostly ban legitimate names, including one's own) — into the repo's untracked private-names banlist (the itd-74 private layer). The pre-commit guard **auto-refreshes** this block when the corpus is present, and when the corpus is absent every corpus-dependent step no-ops *and says so* — never a silent skip, never a failure.
- **`abcd source cite-check`** scans any text and reports offending entries by key only — its output is safe to relay.

## What It Cannot Enforce

The mechanical layer blocks **literal identifying strings**. It cannot detect a paraphrase that identifies a source without naming it — "a forthcoming paper shows X beats Y", or a description of a private repository's distinctive architecture. That residual risk is handled behaviourally (the consultation skill forbids identifying description, not just naming) and by the human review that gates every publish; abcd states this boundary plainly rather than implying coverage it does not have. Durability is likewise bounded: a no-remote corpus survives disk loss only via machine backup and offline `git bundle` snapshots — abcd documents that discipline; it cannot perform it.

## Dogfood (target)

A convention-first scaffold (corpus layout, ledger, guard scripts, and an agent-side consultation skill) was prototyped for this repo's development and is recorded in the plan ([`2026-07-08-confidential-sources-scaffold.md`](../../plans/2026-07-08-confidential-sources-scaffold.md)) and the SOTA survey ([`2026-07-08-confidential-sources-provenance-sota.md`](../../research/notes/2026-07-08-confidential-sources-provenance-sota.md)); no live corpus currently exists on the development machine. The feature is to deliver these as abcd verbs so any managed repo inherits them — and re-establishing this repo's own corpus through those verbs is the first validation.

## Scope Conditions

None stated.

## Acceptance Criteria

- Given a document and its metadata, when Alice runs `abcd source add` declaring it confidential, then the corpus gains a CSL-JSON entry (with the `custom` block) and the document plus extracted text land under `confidential/<key>/` — folder location *is* the classification.
- Given a consulted source influencing a decision, when Alice records it, then `abcd source ledger` appends `{decision_ref, claim, source_key, locator, influence, cited_publicly: false}` as a new line; corrections are new lines, never edits.
- Given confidential entries in the corpus, when a commit runs in a managed repo, then the pre-commit guard's generated block is refreshed (titles and aliases always; author names only under `ban_authors: true`) and a commit containing a banned string is refused.
- Given any text about to leave the machine, when Alice runs `abcd source cite-check`, then offending sources are reported by key only, so the output itself is safe to relay.
- Given a ledger line whose source lacks citation permission, when Alice attempts to set `cited_publicly: true`, then the flip is refused naming the failing gate; with permission present, the flip succeeds and is itself a new ledger line.
- Given a machine with no corpus, when abcd runs in a managed repo, then every corpus-dependent step no-ops and says so — never a silent skip, never a failure.
- Given a confidential source that gets published, when Alice moves its folder `confidential/ → public/`, then the next banlist refresh drops its strings and its ledger lines become eligible for the citation flip.

## Open Questions

- Ledger ownership once work spans machines: **explicitly deferred** (maintainer ruling, 2026-08-16) — per-repo files in the user-level corpus serve one machine; revisit when a second machine actually exists.
- The share/ingest questions that previously lived here (conflict shape between teammates, provenance marks on ingested entries) travel with [itd-126](../drafts/itd-126-a-team-shares-one-bibliography-without-sharing-anyone-s-corp.md).
