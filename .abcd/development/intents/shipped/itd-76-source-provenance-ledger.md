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

Automatic citation is a virtue that becomes a breach the moment a source is confidential: an agent that helpfully names "the working paper this design follows" in a commit message has leaked something no history rewrite fully recalls. The naive fix — keep the material away from the agent — throws away exactly the context that makes its design work good. The resolution is to split *consultation* from *citation* and put a durable, machine-readable record between them: influence is captured eagerly and automatically (cheap, local, append-only), citation happens lazily and manually (when permission exists). The ledger is also the seed of something bigger — a team bibliography and a reconstructable paper — but those travel as their own intents ([itd-126](../drafts/itd-126-a-team-shares-one-bibliography-without.md), [itd-127](../drafts/itd-127-a-paper-is-reconstructed-from-the.md), each `refines` this one); this intent is the personal core they stand on.

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
- The share/ingest questions that previously lived here (conflict shape between teammates, provenance marks on ingested entries) travel with [itd-126](../drafts/itd-126-a-team-shares-one-bibliography-without.md).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-595934bbc552 -->
Fidelity review — receipt rcp-595934bbc552 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:9c702d8c1070e2b1d1a46d9c4e1b52d251dc706e2e889b0e0f24d95adb3998a0
Input attestations: commit:5bc0ebdc2b235a1221fc43934547906fe4b02bae (spc-31 close, itd-76 ships); tree audited at 52c2236a55830421c2af5fa58f5a196c4eb7fdd5@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Add refuses without a declared class, writes the CSL-JSON entry with its custom block and lays original plus text.md under < class>/< key>/, committed in the corpus; TestAddConfidentialLandsUnderItsClassFolder passes at BASE
  evidence: internal/core/source/add.go:154 — "declare the class — confidential or public; it is decided once, here, and never defaulted"
  evidence: internal/core/source/add.go:98 — "folder := filepath.Join(c.Dir, req.Class, entry.ID)"
  evidence: internal/core/source/add.go:129 — "writeSources(c.Dir, append(fresh.raw, raw))"
  evidence: internal/core/source/source_test.go:119 — "func TestAddConfidentialLandsUnderItsClassFolder"
- ac-2 — MET: Record carries decision_ref, claim, source_key, locator, influence and cited_publicly; Append never sets cited_publicly and writes through AppendLineIn only, a correction is a new line naming the earlier one; TestLedgerAppendsAndNeverEdits proves the prefix is preserved
  evidence: internal/core/source/ledger.go:25 — "type Record struct"
  evidence: internal/core/source/ledger.go:118 — "cited_publicly is always false here: exercising the right to cite is Flip's, and only Flip's"
  evidence: internal/core/source/ledger.go:237 — "fsutil.AppendLineIn(root, rel, b, 0o600)"
  evidence: internal/core/source/source_test.go:229 — "a correction rewrote an earlier line"
- ac-3 — MET_WITH_CONCERNS: Projection bans titles and aliases always and authors only under ban_authors; SyncBanlist writes the owned block; this repository's guard rebuilds ./cmd/abcd and refreshes on every commit (TestTheRepositoryGuardRefreshesWithItsOwnBuild) and a banned string refuses the commit (TestSyncBanlistFeedsTheGuard). Concern: the hook `ahoy install` scaffolds into a MANAGED repository refreshes only after a repo-local opt-in (`git config --local abcd.sourcesBinary`), otherwise one line and no refresh — a signed-off narrowing (DECISIONS.md 2026-09-25) pending the ruling in open iss-2609250834251447
  evidence: internal/core/source/guard.go:17 — "func (c *Corpus) Projection() ([]banlist.KeyedPattern, error)"
  evidence: internal/core/source/add.go:441 — "if e.Custom.BanAuthors {"
  evidence: .githooks/pre-commit:599 — "if declares_format "$sources_first"; then sources_refresh"
  evidence: internal/core/source/source_test.go:572 — "func TestTheRepositoryGuardRefreshesWithItsOwnBuild"
  evidence: internal/core/source/source_test.go:401 — "the guard let a confidential title through"
  evidence: internal/core/ahoy/defaults/pre-commit:481 — "elif [ -z "$sources_bin" ]; then"
  evidence: .abcd/work/DECISIONS.md:2562 — "The copy `abcd ahoy` scaffolds refreshes only when the clone opts in with \`git config --local abcd.sourcesBinary"
- ac-4 — MET: CiteCheck scans through the private layer's matcher and a Finding carries source key, field, line and offset only; TestCiteCheckReportsByKeyOnly asserts the JSON report holds no title, alias or author
  evidence: internal/core/source/guard.go:91 — "type Finding struct"
  evidence: internal/core/source/guard.go:135 — "Finding{Source: key, Field: field, Line: h.Line, Offset: h.Offset}"
  evidence: internal/core/source/source_test.go:415 — "func TestCiteCheckReportsByKeyOnly"
- ac-5 — MET: Flip refuses naming gate 1 when permission_status is not citable or the folder is not public/, appends nothing on refusal, and a successful flip is a new line with flips naming the original; TestFlipNeedsBothGates covers both outcomes
  evidence: internal/core/source/ledger.go:194 — "gate 1 — source %q has permission_status %q, and only %q grants the right to cite"
  evidence: internal/core/source/ledger.go:202 — "flip := orig flip.TS = stamp(now) flip.CitedPublicly = true"
  evidence: internal/core/source/source_test.go:270 — "func TestFlipNeedsBothGates"
- ac-6 — MET_WITH_CONCERNS: Every core step returns ErrNoCorpus and creates nothing (TestNoCorpusIsNamedByEveryStep); the guard prints one line and proceeds; `sync-banlist --refresh` is one line and exit 0; every other verb says so on one line. Concern: the verbs exit 3, a non-zero code, where the criterion says 'never a failure' — the spec's design (item 6) chose a distinct no-corpus exit for the verbs and 0 for the guard, so the narrowing is signed off
  evidence: internal/core/source/source.go:92 — "ErrNoCorpus = errors.New("sources corpus is absent")"
  evidence: internal/surface/cli/source.go:54 — "if errors.Is(err, source.ErrNoCorpus) { return &exitError{Code: 3"
  evidence: internal/surface/cli/source.go:451 — "if refresh && errors.Is(err, source.ErrNoCorpus) {"
  evidence: .githooks/pre-commit:593 — "no sources corpus at ~/.abcd/sources — generated banlist block not refreshed (skipped)"
  evidence: internal/core/source/source_test.go:518 — "func TestNoCorpusIsNamedByEveryStep"
  evidence: internal/surface/cli/source_surface_test.go:42 — "func TestSourceNoCorpusSaysSoOnEveryVerb"
  evidence: .abcd/development/specs/closed/spc-31-source-provenance-ledger.md:73 — "exit 0 (guard) / a distinct no-corpus exit (verbs)"
- ac-7 — MET: Declassify is a `git mv confidential/<key> public/<key>` plus the entry update in one commit; the next SyncBanlist drops the key's block lines and Flip then succeeds; TestDeclassifyDropsTheBanAndOpensTheFlip proves all three
  evidence: internal/core/source/add.go:405 — "corpusGit(c.Dir, "mv", "--", from, to)"
  evidence: internal/core/source/guard.go:20 — "if c.Class(e.ID) != ClassConfidential { continue"
  evidence: internal/core/source/source_test.go:449 — "func TestDeclassifyDropsTheBanAndOpensTheFlip"

Gap audit:
- honoured:
  - folder location is the classification and a corpus whose folders and entries disagree is refused by every step that derives a ban
    evidence: internal/core/source/source.go:297 — "func (c *Corpus) requireConsistent() error"
    evidence: internal/core/source/source_test.go:488 — "func TestAMismatchedClassRefusesTheSync"
  - the two-gate citation boundary of adr-41: permission_status grants and a human-flipped ledger line exercises
    evidence: internal/core/source/ledger.go:161 — "Flip exercises the right to cite for one ledger line (adr-41 gate 2), and only after gate 1 grants it"
  - one matcher for guard and cite-check: the projection runs through banlist.PhrasePattern and banlist.ScanText
    evidence: internal/core/source/add.go:426 — "p, err := banlist.PhrasePattern(phrases...)"
    evidence: internal/core/source/guard.go:127 — "hits, err := banlist.ScanText(pats, text)"
  - the corpus is a no-remote git repository in the user-level home and every write is committed
    evidence: internal/core/source/source.go:411 — "corpusGit(dir, "init", "-q")"
    evidence: internal/core/source/source.go:405 — "the location is inside another git working tree"
  - the verbs are wired on the CLI and the plugin surface
    evidence: commands/source.md:4 — "add [document] --key K --confidential|--public [...] | declassify < key> | ledger"
    evidence: internal/surface/cli/source.go:62 — "func newSourceCommand(asJSON *bool) *cobra.Command"
- diverged:
  - the pre-commit guard auto-refreshes the block on every commit in a managed repo — delivered as opt-in per clone (abcd.sourcesBinary) for the scaffolded hook, automatic only in abcd's own checkout, pending the ruling in iss-2609250834251447
    evidence: internal/core/ahoy/defaults/pre-commit:482 — "this hook refreshes its banlist block only on opt-in: git config --local abcd.sourcesBinary"
    evidence: .abcd/work/issues/resolved/iss-2609252007419997-the-scaffolded-pre-commit-template.md:14 — "iss-2609250834251447's ruling stays open and can widen it"
  - every corpus-dependent step no-ops and says so, never a failure — the verbs answer with exit 3 (the spec's distinct no-corpus code), the guard with exit 0
    evidence: internal/surface/cli/source.go:8 — "3 there is no corpus at the configured location — the distinct no-corpus code"
- missing:
  - the dogfood target — re-establishing this repository's own corpus through the shipped verbs as the first validation — is not evidenced; the ship commit records it as left to the person whose corpus it is
    evidence: .abcd/development/specs/closed/spc-31-source-provenance-ledger.md:79 — "## First validation"
    evidence: .abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:41 — "re-establishing this repo's own corpus through those verbs is the first validation"
<!-- abcd-review-end receipt=rcp-595934bbc552 -->
