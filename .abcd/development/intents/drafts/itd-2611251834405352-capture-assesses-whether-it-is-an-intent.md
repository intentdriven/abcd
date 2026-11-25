---
id: itd-2611251834405352
slug: capture-assesses-whether-it-is-an-intent
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
related_issues: [iss-2611251832355077]
origin: extracted-from-record
production_mode: hand-written
---

# A capture says whether it files an issue, an intent, a ruling or an observation

## Press Release

> Every capture now says what kind of thing it files: a fault against what the project promises, new behaviour a person would see, a decision a person owes, or an observation that is not yet any of those. Iris, a product thinker, comes back to a ledger that agents filled while they were away. Nothing in it is a capability filed as a defect: what read as new behaviour was promoted to a draft intent in the act of filing it, or carries the one line that says why it was not, and every record states its kind and the reason for it. Before a release, abcd lists each issue assessed as an intent that is still not promoted, so none is cut past unseen.

## Why This Matters

Graduated from iss-2611251832355077, which holds the observation. The capture page says where a user-facing change goes and leaves the choice to whoever files; the verb checks the text, the remedy and likely doubles, and asks nothing about kind. On 2026-11-25 a request for new behaviour was filed as an issue, and the product thinker had to ask whether it was an intent before it was promoted. Two more filed the same evening read as capabilities too (iss-2611251742484774, iss-2611251828573245). Of the sixteen captures this ledger holds from that day up to 18:34, when the last of them was promoted, four were promoted, each once the product thinker had said it was an intent. The decomposition table (itd-84) routes a proposal to its homes only when an intent is being filed, so the route most findings take, a capture, is the one with no routing. A drain (itd-82) does hand an issue back by kind, but only an eligible one, and only when a drain reaches it.

## Mechanism

We expect a required answer at the filing to catch what a pointer in the page did not, because the question is put at the one moment the filer holds the text and cannot file without answering it, and its answers are the kinds a drain's hand-back already routes by (itd-82): a user moment to an intent draft, a decision to a person, the rest to the ledger. Shown wrong if, among the captures filed under the rule over one release cycle, the share a filer assessed as an issue or an observation and a person then promoted all the same is no smaller than the four in sixteen of 2026-11-25, counted by a reader of the ledger; or if the same issue stands on the release list of unpromoted intents at two cuts running.

## Scope Conditions

- Holds for a capture a person or an agent files through the verb or its page. A filer of abcd's own answers no question, and its records carry no kind.
- Holds for records filed once the rule ships. Nothing assesses the records already in the ledger.
- Holds where the text says enough to tell an intent, an issue and a ruling apart; where it does not, the kind is an observation and its reason says what would settle it. The verb checks that an answer and its reason were given, never that the answer is right; that is read from the record by a person.
- Assumes a text of one kind. A text that is partly a fault and partly new behaviour is assessed by the part its filer names, and the reason says which.

## Acceptance Criteria

- **Given** the capture page a session is handed, **when** an agent reads it before filing, **then** it finds the question to answer first, with the four kinds and the one-line test of each: new behaviour a person would see is an intent, a fault against what the project promises is an issue, a decision a person owes is a ruling, and what is not yet any of the three is an observation, and its line says what would settle which it is. The paragraph that left the choice to whoever files is replaced by it, and `abcd explain capture` prints the same four.
- **Given** a person or an agent filing through the verb, **when** the capture names no kind, a kind outside the four, or a kind with no reason, **then** it is refused with exit 2 and nothing is written, and the line they read in the terminal (the error object under `--json`) names the flag, the four values, and the reason it wants.
- **Given** a capture assessed as an issue, a ruling or an observation, with its one line of reason, **when** it is filed, **then** a reader of the record under `open/` finds the kind and the reason on it, the result the filer reads reports both, nothing is promoted, and for a ruling the result names `abcd decide` as where the answer goes once a person gives it.
- **Given** a capture assessed as an intent that gives no line against promoting it, **when** it is filed, **then** the one call writes the issue and promotes it: a draft in `drafts/` seeded from it, the issue naming the draft in `related_intents` and the draft naming the issue in `related_issues`, and the result the filer reads names both ids and both paths. If the promotion fails after the issue is written, the issue stands, the call exits 1 (never the 2 of a refusal, which writes nothing), and the result says the promotion is owed and names `abcd capture promote <iss-N>`.
- **Given** a capture assessed as an intent that says why it is not promoted now, **when** it is filed, **then** no draft is made and a reader of the record finds that line beside the kind and its reason; given blank, it is refused with exit 2 and nothing is written.
- **Given** the consistency pass, the inbox's promote, the owed-audit filer, or any other filer of abcd's own that is no person's command, **when** it files, **then** the record is written as it is today, with the remedy it writes today, no kind and no promotion, and the record gates pass on it; a test files through each one and reads the record.
- **Given** a capture that names a kind and uses any flag the verb has today (the remedy, severity, category, source, found-at, found-during, lapsed-at, slug, blocked-by, production mode), **when** it is filed, **then** each flag writes and refuses what it did before and the filing-time match still links a likely double; the verb's existing tests pass with the kind added to their calls and no other change. A capture that files a fixed finding straight to `resolved/`, where itd-2611251831138387 has shipped it, takes the kind and its reason as any capture does.
- **Given** any capture command abcd prints as a next step, or carries in a command page, an agent's brief, the bundled rules domains or this repository's own, the published docs, or `evals/`, **when** this ships, **then** it runs as printed: a test collects each one and none is refused for a missing kind.
- **Given** an open issue with no kind, filed before the rule or by a filer of abcd's own, **when** the board, `abcd capture list`, the record gates, the release cut, or a drain reads the ledger, **then** it is read as it is today and nothing names it as at fault. A drain's dispositions are what the same fields gave before, an issue promoted in the act of capture reads there as `promoted`, and `abcd capture promote` still promotes an issue of any kind or of none, leaving the first assessment on the record.
- **Given** an issue filed under the rule, **when** a copy of abcd at the last release reads the ledger, **then** it counts the record as it does one filed today: the kind and its lines are written where that release's closed schema does not refuse them, so the record is in that copy's lists and totals and not under `skipped`, and that copy's record-lint does not block on it. A test holds a record the new verb wrote to the keys the schema at the last release tag accepts.
- **Given** open issues assessed as an intent and not promoted, **when** the person runs the release preview or the cut (`abcd launch --dry-run`, `abcd launch ship`), **then** they read each one listed by id with the line that says why it was not promoted and the promote command, in the plain report and under `--json`; nothing is refused for them, and with none the report adds no line.

## SOTA

Not searched for this draft: no source was opened, and what follows is what the writing session knows of the tools, not re-fetched. The independent fit-challenge is owed before the build.

- **A required type on a tracker's new-issue form** (a form with a required field; a tracker's own issue types): mature. It types the record at filing, which is this intent's first half; none known to the writing session makes the other record in the same act.
- **Triage after filing** (a label a person or a bot puts on once the issue exists): mature, and it is what abcd has today in a drain's hand-back (itd-82), which reaches only an eligible issue and only when a drain runs.

Path 2: a native field on abcd's own record and no dependency, since the ledger is files in the working tier (adr-32) and no tracker holds it. The seam is thin and is said so: the kind is one field with a closed set of values, which is what a tracker's issue type is, so a ledger mirrored to one could map it; nothing is built for that.

## Open Questions

- **Deferred** Whether a drain should read the assessed kind, handing back an issue assessed as a ruling before any lane opens on it, is left for the product thinker. Nothing here needs it: a drain's dispositions are unchanged, and its own judgement of each eligible issue is asked as before.

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611251921171761): this was filed as an issue and is an intent, it matters, and it is built in their six-day unattended run with nobody to ask. They chose it from a sentence or two and gave no wording for a criterion; the criteria were not listed to them. The criteria, the Mechanism, and the scope conditions are the writing session's, written from the remedy on iss-2611251832355077, for their review on return; a change then is made through the record. To reverse: leave it a draft until they have read the criteria.
2. The writing session's, no question put: the requirement stands at the verb a person or an agent runs, and the filers of abcd's own stay as they are, as the remedy rule already keeps the two apart. A machine has no answer to give, and a default written for it would read as a judgement nobody made.
3. The writing session's, no question put: the release check lists and refuses nothing. The remedy says "lists", and a refusal would hold a cut on a judgement of kind. To reverse: make it a refusal with a recorded deferral, as the unfixed-findings guard has.
4. The writing session's, no question put: the names of the flags and of the record's fields are the spec's, since a name is a technical choice. The four values are `issue`, `intent`, `ruling`, and `observation`. The fourth was added after a second reader's review on 2026-11-25: the ledger exists for what is half-formed (the capture page's "a half-formed observation, question, or nitpick"; itd-4), and three kinds refused exactly that.
5. The writing session's, for review: the impact is `breaking`. A flag required on an existing command refuses a call accepted before, which the release's surface guard names as a break (`flag_required`) and itd-2610141646454347 stamped so. The remedy's requirement (itd-82) is no precedent: it is enforced in the command's run, and the guard never saw it. At 0.x the next release is a minor one.
6. The writing session's, no question put, after a test on 2026-11-25: a copy of abcd at the last release lists a record that carries a field its closed schema does not know under `skipped`, with the reason, outside its totals, and its record-lint blocks on it. So the kind and its lines are written where that schema does not refuse them; where that is, is the spec's. The other course, a scope condition that admits older copies set such records aside, was not taken; it is the product thinker's to choose instead.

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent: a capture is assessed by kind as it is filed. Routing by kind is a stance that already has a home, a drain's hand-back (itd-82), whose kinds this asks for at the filing; no new principle. No new trust rule is written; what the promotion in the act of filing sets aside is flagged below. The flags, the record's fields, the page's wording, the printed commands, and the release list are plumbing, for the spec and for the brief's capture and launch chapters (`04-surfaces/06-capture.md`, `04-surfaces/04-launch.md`). Typed links: `refines` itd-119, whose promotion without retyping the act of filing now reaches; `refines` itd-82, whose hand-back kinds and whose required remedy stand at the same verb; `refines` itd-4, the ledger; `related_issues` iss-2611251832355077, which the change that ships this resolves. Flagged for the product thinker, never auto-classified: for a capture assessed as an intent it sets aside itd-84's first criterion (nothing is filed until a human confirms the routing) and its "no blocking gate or auto-filing", as the drain's promotion already does (itd-82); and it departs from "never refuse, never drop" (itd-2609212137116617, decision 1). Verdict: FILE-AS-IS, the routing for a person to confirm.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
