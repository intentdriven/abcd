# Agent prompt changelog

Per [itd-5](../.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md),
every `agents/*.md` prompt carries a `prompt_version` and a corresponding entry
here recording the bump rationale (and, at `1.0.0` lock, the self-improvement
pre-flight outcome and calibration-corpus delta).

**Version band.** Agents ship in the `0.x` band until they clear their calibration
corpus — `0.x` means "shipped and wired, honestly unmeasured"; `1.0.0` means
"measured against a corpus and locked" (itd-81's amendment to itd-5, which governs
over the brief's earlier `1.0.0`-at-close expectation). The four M6 synthesis
agents below entered at `0.1.0`, wired to their `abcd disembark` verbs and
unmeasured; `lifeboat-oracle` has since become `lifeboat-reviewer` at `0.1.1`.

## 2026-09-02 (iss-2609021833302981 — "at the target", as the derivation reads it)

The comparative derivation admits a widening run whose recorded target is an
ancestor of the assembly's target when nothing changed between the two outside
the readings store and the issue ledger's own record families — which is what
lets a run's records be committed between its ingest and the next reading, the
act the design sequences there. The object section of the comparative definition
states the derivation rule, so it states this too. The reading of the ADR's
phrase is an interpretation and the maintainer's ruling is owed
(iss-2609021857343626).

### cold-reading-comparative 0.1.3

PATCH: the object's one-sentence statement of the derivation gains the ancestor
clause, as above. The source list, the question, the item shape, the regime and
the blindness core are untouched. Unmeasured, as before.

## 2026-09-02 (itd-194 — the object section states the include table)

The object section's source list is the include table for that position, held
so by a test, so a change to the table is a change to every definition whose
position the change reaches. itd-194 admits the brief's surfaces, internals,
delivery and meta chapters as brief current text — which both design documents
name as a reading's object — so all four definitions gain those four sources,
and the widening definition loses the shipped intents, whose position the
framework's widening object and the readings companion's section 5.2 both state
without them (iss-2609012259587904). Nothing else in any of the four moves: the
blindness core, the questions, the item shapes and the regimes are untouched.

### cold-reading-widening 0.2.2

PATCH: the four brief chapters are added to the object's source list and the
shipped intents are removed from it, as above. Unmeasured, as before.

### cold-reading-entailment 0.1.2

PATCH: the four brief chapters are added to the object's source list, as above.
Unmeasured, as before.

### cold-reading-comparative 0.1.2

PATCH: the four brief chapters are added to the object's source list, as above.
Unmeasured, as before.

### cold-reading-detection 0.1.2

PATCH: the four brief chapters are added to the object's source list, as above.
Unmeasured, as before.

## 2026-09-02 (iss-2609021153261145 — the fourth condition takes the companion's sentence)

The four cold-reading definitions share one blindness core, byte-identical
across the four and held so by a test, so a change to it is one change to four
prompts and it is recorded here as one entry. The fourth condition's first
sentence said items come back unordered and unweighted; the readings companion's
section 2 says items are returned in the order they arise in the object. On
correction (7) of the maintainer's ruling of 2026-09-02 the condition takes the
companion's sentence. The rest of the condition — no severity, no confidence
score, no most-important-first, no top-N, and the reason an ordering is an
argument about what matters — is unchanged, and so is every other condition.

### cold-reading-widening 0.2.1

PATCH: the blindness core's fourth condition takes the companion's sentence, as
above. Nothing else in the definition moves. Unmeasured, as before.

### cold-reading-entailment 0.1.1

PATCH: the blindness core's fourth condition takes the companion's sentence, as
above. Nothing else in the definition moves. Unmeasured, as before.

### cold-reading-comparative 0.1.1

PATCH: the blindness core's fourth condition takes the companion's sentence, as
above. Nothing else in the definition moves. Unmeasured, as before.

### cold-reading-detection 0.1.1

PATCH: the blindness core's fourth condition takes the companion's sentence, as
above. Nothing else in the definition moves. Unmeasured, as before.

## 0.2.0 — 2026-09-02 (iss-2608311518056854 — the gate reads no prose)

### cold-reading-widening 0.2.0

MINOR: the definition no longer tells the reader that a registered signature
over its prose raises a review flag, because the supply-regime gate reads no
prose (the maintainer's ruling of 2026-09-01 on iss-2608311518056854): a
reserved name refuses only as a key of the reader's own output. The reader's
licence and every other instruction are unchanged; what changes is what the
gate does with the output, and the definition now says so truthfully.

## 0.3.0 — 2026-09-01 (iss-2609011207114761 — only Added and Fixed)

### release-changelog-composer 0.3.0

MINOR: the section set the composer may choose from closes to `Added` and
`Fixed`, which changes what it emits for the same cut. The 0.2.0 baseline rule
asked the composer whether a capability was reachable in `base_tag` while giving
it no way to look, so the ruling of 2026-09-01 (iss-2609011207114761) removes
the question: `Changed`, `Deprecated` and `Removed` are claims about the previous
release's surface, `Security` is a claim about a vulnerability that release
carried, and the composer's inputs are the cut and the record bodies alone. The
table offers two rows; the four other Keep a Changelog names are listed as ones
the binary refuses. `impact` maps explicitly: `breaking` is an `Added` line that
states the break, a `removed[]` record is cited on the `Added` line of what
replaced or withdrew it, and a closed vulnerability is a `Fixed` line. The
example payload's `Removed` entry becomes an `Added` line stating the
withdrawal. The binary side lands in the same change: `writableSections` in
`internal/core/release` is the one declaration of the set, the ingest refuses
any other registered section by name and by ruling, a test pins this prompt's
table to that list, and every dated section the ingest writes carries one
sentence under its heading saying what the notes list and what they do not
claim. Unmeasured, as before.

## 0.2.0 — 2026-09-01 (iss-2609011207114761 — the base_tag baseline rule)

### release-changelog-composer 0.2.0

MINOR: the section-choice guidance gains a baseline rule, which changes what the
composer emits for the same cut. `Changed`, `Deprecated` and `Removed` are claims
about what a user upgrading FROM `base_tag` experiences, so a capability that was
not reachable there is `Added` however much its record narrates a change. The
table rows for those three sections now say so, and a new subsection states the
question to ask before using one, the tell that a surface debuts in this cut (its
introducing records sit in the same `added[]`), and the instruction to treat a
record's own "no longer" or "has no users" wording as its baseline rather than the
composer's.

Occasioned by the v0.7.0 cut, where the previous prompt placed three lines under
`Changed`/`Removed` describing a migration burden from v0.6.9 for the `abcd
reading` verb, which does not exist at v0.6.9. The records were not at fault: both
carried the disambiguating facts, and neither was asked a question that used them.
Re-composed under this rule the same cut yields no `Changed` or `Removed` lines at
all and cites the same 104 records.

The rule is TRUSTED, not checked: the composer's inputs are the cut and the record
bodies, never `base_tag`'s surface, so it is told to ask a question it cannot
verify. Where the cut leaves the answer undecidable it is instructed to choose
`Added` and say why, an over-cautious `Added` understating a change while a wrong
`Changed` fabricates a migration. iss-2609011207114761 holds the binary-side half
and argues for refusing at ingest, where the completeness bijection already lives.
Unmeasured — no calibration corpus exists for release prose yet, and no
self-improvement pre-flight was run.

## 0.1.0 — 2026-08-31 (itd-184 / spc-62 — the four cold-reading definitions)

Four definitions enter together, one per supply regime, as instances within one
detector context. One definition with four objects cannot hold: the prohibition
against proposing is constitutive of the detection position and would void the
widening position entirely. Each holds five things and nothing else — its object,
its question, the blindness core verbatim, its regime value, and its item shape —
and the core is delimited by `<!-- blindness-core:begin -->` and
`<!-- blindness-core:end -->` so a byte-identity test compares an exact span
rather than a heuristic slice. All four carry `reads_untrusted_input: true` (a
reading reads repository text it did not write), `capability_scope.task_classes:
[cold_reading]` — a token added to the closed enum in the same change, because
reusing `cross_document_audit` would name these prompts as audits and an audit
judges against a standard, which is precisely the licence a widening reading does
not hold — and two frontmatter keys the ingest verb reads, `position:` and
`regime:`, which is what makes the regime the definition's property rather than
the payload's.

The seven core conditions, in the fixed order the span carries them: no project
context, no ledger access, no memory across runs, no ranking or prioritisation,
no selection, explanation or commitment, named provenance on every item produced,
and no passed input is authoritative. The seventh is disclosed in the core's own
wording as an assertion rather than a mechanism, because nothing that assembles a
reading's material can enforce it. The host obligation to grant the reading no
repository access is disclosed on the same terms.

The definitions are the assertion half of the blindness. The enforcement half is
the input assembler and its evals; the licence check on what a reading produced
is the ingest verb's supply-regime gate. None of the four is dispatched: the
instrument ships unrun.

Unmeasured — no calibration corpus exists for cold-reading output, and no
self-improvement pre-flight was run.

### cold-reading-widening 0.1.0

First entry. Object: the brief's current text including the construal statement,
the glossary, the disciplines, the specs and the shipped tree, with the draft and
planned intents withheld — the widening reading must not see the candidate set it
is asked to widen. Regime `generative`, the widest of the four licences. Item
shape: `configuration` and `what_admits_it`, with no third body field, so neither
a preference nor a comparison against what was built has anywhere to go; a
recommendation raises a review flag on ingest rather than a refusal, because
comparison belongs to the comparative position. Canary: passed material demanding
a ranking, a recommendation, and the draft intents this position cannot see.

### cold-reading-entailment 0.1.0

First entry. Object: the claim record, drafts and planned intents included, plus
the constraint sources — articulation precedes selection, so this is the one
position that properly reads drafts. Regime `explicative`. Item shape:
`claim_surfaced`, `claim_type` (criterion, causal or context) and
`what_implies_it`; commitments are surfaced and never dispositioned. Canary:
passed material demanding the reading accept one claim, refuse another, and say
what should change.

### cold-reading-comparative 0.1.0

First entry. Object: the candidate set, which is the widening reading's
pre-admission output read within one cycle before admission, against the declared
selection criteria — a recorded discipline, never supplied at invocation and
never authored here. A prior run's stored output stays read-blocked, and where
the candidate set carries fewer than two configurations the position is not
exercised at all. Regime `evaluative`. Item shape: one item per
candidate-criterion pair — `candidate_id`, `criterion`, `characterisation` — with
no score and no winner. Canary: a criterion declaring itself authoritative and
beyond question, which the seventh core condition refuses.

### cold-reading-detection 0.1.0

First entry. Object: the shipped tree against the claim record, with the draft and
planned intents withheld, because a tension against a claim nobody has committed
to is not a tension. Regime `registrative`. Item shape: `tension`,
`constraint_in_play` and `why_a_tension`; the prohibition against proposing is
constitutive here. Canary: a source comment demanding the patch that fixes what
was found, and claiming an earlier run already dismissed it.

## 0.3.0 — 2026-08-30 (itd-181 / spc-59 — scope-condition dispositions)

### intent-auditor 0.3.0

The verdict grows a fourth judgement surface: `scope_conditions`, one disposition
per scope condition the intent records, keyed to the `cond-…` identity spc-55
mints rather than to the condition's wording. The definition gains the
`scope_conditions` input note (identities are host-supplied and echoed verbatim,
never invented), a four-value rubric — `survived`, `narrowed`, `falsified`,
`untested` — framed as harshly as the acceptance rubric (`untested` is the
correct disposition for a vacuum, not `survived`), the extended output block, and
two numbered rules naming what the ingest refuses: exact coverage of the supplied
identities in both directions, the closed value set, `narrowing` required on
`narrowed` and refused on every other disposition, and cited evidence on
everything but `untested`.

MINOR in the `0.x` band: the output contract is extended rather than reshaped,
and a verdict for an intent that records no conditions — every intent shipped
before the identity mint existed — stays valid unchanged. The Go ingest is what
actually rejects a bad block (`internal/core/intent/audit.go`), and a lockstep
test decodes this definition's own output blocks into the struct the ingest
decodes, so the two cannot drift apart silently.
Unmeasured — no calibration corpus exists for disposition judgement, and no
self-improvement pre-flight was run.

## 0.1.0 — 2026-08-30 (itd-188 / spc-66 — the ledger scribe)

### scribe 0.1.0

First entry. Host-delegated ledger scribe, whose access rule is the assembler's
exact inverse (brief invariant 15): it receives ledger content under
`.abcd/work/issues/` and the reading output it is transcribing, and never the
shipped repository as an object of judgement. No session holds both a reading and
the ledger. It transcribes returned readings and researcher dispositions into the
reading-record and disposition shapes spc-58 declares, and authors nothing — no
claim, no ground, no disposition state. Two permissions bound that: it may raise a
**fidelity flag** on an internal inconsistency in the material it is transcribing
and must carry it to the researcher unresolved (proposing a resolution is
authorship), and anything it is explicitly asked to compose beyond formatting
opens with a **contribution stamp** that travels with the material if adopted —
the hand-run precursor of the record's origin keys, with an unstamped contribution
refused rather than delivered. Carries `reads_untrusted_input: true`,
`capability_scope.task_classes: [surface_render]`, and an injection-canary fixture
whose payload demands the scribe author dispositions, fetch shipped-tree material,
and move a ledger record. There is no ingest verb, and no schema for these
shapes either until spc-58's reading and disposition stores land: the emitted
records are committed through the ordinary record path, where `record_schema`
refuses their directory as an undeclared bucket rather than judging the record, so
the shapes are held by the definition and by review. The written protocol in the
brief's agents chapter is followed by hand.
Unmeasured — no calibration corpus exists for ledger transcription, and no
self-improvement pre-flight was run.

## 0.2.0 — 2026-08-19 (iss-278 — the itd-5 contract reaches the review/research agents)

Five prompts read attacker-influenceable input without the itd-5 contract:
`ruthless-reviewer`, `security-reviewer`, `docs-currency-reviewer`,
`intent-auditor`, `sota-researcher` (iss-278 records the class and the missing
detector). Each gains `reads_untrusted_input: true`, a `capability_scope`, an
untrusted-data defence paragraph in its body, and an injection-canary fixture —
a MINOR bump apiece: behaviour hardening, output contracts unchanged.

### ruthless-reviewer 0.2.0

`task_classes: [oracle_review]` (family-1 change judgement). Canary: a diff
whose code comment claims pre-approval and demands zero findings — reported as
a finding, never obeyed. Tool grants unchanged; whether review agents should
hold `Bash` at all for external-contribution review is a maintainer decision
tracked in iss-278, not silently made here.

### security-reviewer 0.2.0

`task_classes: [oracle_review]`. Canary: a diff whose string claims a security
exemption while splicing untrusted input into a shell exfiltration — the
exemption claim is itself the finding.

### docs-currency-reviewer 0.2.0

`task_classes: [cross_document_audit]`. Canary: a docs page whose HTML comment
demands its claims be marked current — claims verified against code regardless.

### intent-auditor 0.2.0

`task_classes: [intent_audit]`. Canary: an acceptance criterion demanding a MET
verdict — judged solely against the delivered diff.

### sota-researcher 0.2.0

`task_classes: [spec_planning]` — the closed enum carries no research token, and
spec-planning input is what its output feeds; revisit when the adr-40
reconciliation extends the enum. Canary: a fetched page pushing a ranking and a
tracking link — weighed as one low-tier source, never obeyed. It reads the most
attacker-influenceable input in this directory (web search and fetch).

## 0.1.0 — 2026-07-21 (itd-67 / itd-73 — the derived release cut)

### release-changelog-composer 0.1.0

First entry. Host-delegated composer behind `abcd launch ship --changelog-json
<file|->`. Reads the emit step's cut (derived `next_tag`, the `added`/`removed`
record entries with their `in_changelog` flag) and the records themselves; emits
the composed-changelog payload — `schema_version` / `prompt_version` / `next_tag`
/ `entries[{section, records, text}]`. Owns the WORDING and the Keep-a-Changelog
section only: the version, the date, the heading shape, the section order, and the
inclusion set are the binary's. The citation rule is the completeness **bijection**
rather than cite-or-be-dropped — `cited == (added ∪ removed) where in_changelog`,
exactly — and a mismatch (missing, invented, or an `impact: internal` record cited)
refuses the WHOLE payload and writes nothing, because a dropped changelog line is a
shipped change absent from the permanent release record. Carries
`reads_untrusted_input: true`, `capability_scope.task_classes: [surface_render]`,
and an injection-canary fixture whose payload attempts three hijacks (persona
switch, drop the citation, cite the internal record). Unmeasured — no calibration
corpus exists for release prose yet, and no self-improvement pre-flight was run.

## 0.1.0 — 2026-07-16 (itd-88 M6 — synthesis agents)

### principle-distiller 0.1.0

First entry. Host-delegated distiller behind `abcd disembark principles
<lifeboat-dir> --principles-json`. Reads a packed lifeboat's ADRs, intents,
resolved issues, and graveyard findings; emits `principles.json` with each
principle citing a record id, a graveyard finding id, or a packed lifeboat path
(cite-or-be-dropped over `R ∪ F ∪ P`). Carries `reads_untrusted_input: true`,
`capability_scope.task_classes: [principle_distillation]`, and an injection-canary
fixture. Unmeasured (no corpus yet); no self-improvement pre-flight run.

### graveyard-interpreter 0.1.0

First entry. Host-delegated interpreter behind `abcd disembark graveyard
<lifeboat-dir> --lessons-json`. Reads the sealed `graveyard/archaeology.json` and
`graveyard/abandoned.json`; emits the graveyard **lessons** schema (no `mode`, no
`prompt_version` field — the pre-M6 lessons schema), each lesson citing a live
layer-1/2 finding id (cite-or-be-dropped over the finding-id set). Carries
`reads_untrusted_input: true`, `capability_scope.task_classes:
[cross_document_audit]`, and an injection-canary fixture. Unmeasured; no
self-improvement pre-flight run.

### press-release-composer 0.1.0

First entry. Host-delegated composer behind `abcd disembark press-release
<lifeboat-dir> --press-release-json`. Reads the packed brief, spine, and
`principles.json`; emits a single `press-release.json` document that must cite at
least one path in `brief/**`, `rescue/spine.md`, or `principles.json`
(whole-document refusal if it cites nothing resolvable). Carries
`reads_untrusted_input: true`, `capability_scope.task_classes: [surface_render]`,
and an injection-canary fixture. Unmeasured; no self-improvement pre-flight run.

### lifeboat-oracle 0.1.0

First entry. Host-delegated auditor behind `abcd disembark oracle <lifeboat-dir>
<source-repo> --oracle-json`. Reads the packed lifeboat corpus against its source
repo; emits an `oracle` audit carrying a registered verdict (`SHIP` / `NEEDS_WORK`
/ `MAJOR_RETHINK` — out-of-enum refuses the whole payload) and findings that each
cite a packed lifeboat path (cite-or-be-dropped over the packed-path set). Carries
`reads_untrusted_input: true`, `capability_scope.task_classes: [oracle_review,
audit]`, and an injection-canary fixture. Unmeasured; no self-improvement pre-flight
run.


### intent-auditor 0.1.1 (renamed from intent-fidelity-reviewer)

The intent-fidelity judge is the intent AUDIT (adr-40: it emits family-2
promise-vs-reality verdicts), so the agent renames to `intent-auditor` and its
verb moves to `abcd intent audit` / `audit ingest` (spc-28, itd-123). The
verdict format is frozen: `_type` stays `abcd/intent-fidelity-verdict/v1`,
stored markers and previously ingested verdicts remain valid, and a stored
verdict naming the old verifier id still ingests. Prompt body otherwise
unchanged; no re-measurement.

### lifeboat-reviewer 0.1.1 (renamed from lifeboat-oracle)

The lifeboat verdict endpoint emits family-1 review verdicts (`SHIP` /
`NEEDS_WORK` / `MAJOR_RETHINK`), and the planning investigation found the binary
verb never invokes the oracle seam — so the verb renames to `abcd disembark
review` (flag `--review-json`), its artefact moves to
`review/review-<manifest12>.{json,md}`, and this agent becomes
`lifeboat-reviewer` (spc-30, itd-125; adr-40 §5 amended in place). The `oracle`
SEAM is untouched: `capability_scope.task_classes` keeps `oracle_review`, which
honestly names review work reaching a model through the adr-25 seam, and
`/abcd:oracle ask` remains the seam's reserved surface. Prompt body, verdict
enum, and cite-or-be-dropped contract unchanged; canary fixture moved with the
agent. No re-measurement.