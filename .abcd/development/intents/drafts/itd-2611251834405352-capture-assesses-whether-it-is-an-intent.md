---
id: itd-2611251834405352
slug: capture-assesses-whether-it-is-an-intent
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
related_issues: [iss-2611251832355077]
origin: extracted-from-record
production_mode: hand-written
---

# A capture says whether it files an issue, an intent or a ruling

## Press Release

> Every capture now says what kind of thing it files: a fault against what abcd promises, new behaviour a person would see, or a decision a person owes. Iris, a product thinker, comes back to a ledger that agents filled while they were away. Nothing in it is a capability filed as a defect: what read as new behaviour was promoted to a draft intent in the act of filing it, or carries the one line that says why it was not, and every record states its kind and the grounds for it. Before a release, abcd lists each issue assessed as an intent that is still not promoted, so none is cut past unseen.

## Why This Matters

Graduated from iss-2611251832355077, which holds the observation. The capture page says where a user-facing change goes and leaves the choice to whoever files; the verb checks the text, the remedy and likely doubles, and asks nothing about kind. On 2026-11-25 a request for new behaviour was filed as an issue, and the product thinker had to ask whether it was an intent before it was promoted. Two more filed the same evening read as capabilities too (iss-2611251742484774, iss-2611251828573245). Of the seventeen captures this ledger holds from that day, four were promoted afterwards, each once the product thinker had said it was an intent. The decomposition table (itd-84) routes a proposal by kind only when an intent is being filed, so the route most findings take, a capture, is the one with no routing. A drain (itd-82) does hand an issue back by kind, but only an eligible one, and only when a drain reaches it.

## Mechanism

We expect a required answer at the filing to catch what a pointer in the page did not, because the question is put at the one moment the filer holds the text and cannot file without answering it, and its three answers are the routing the four-piece table already uses when an intent is filed. Shown wrong if, among the captures filed under the rule over one release cycle, the share a filer assessed as an issue and a person then promoted all the same is no smaller than the four in seventeen of 2026-11-25, counted by a reader of the ledger; or if the same issue stands on the release list of unpromoted intents at two cuts running.

## Scope Conditions

- Holds for a capture a person or an agent files through the verb or its page. A filer of abcd's own answers no question, and its records carry no kind.
- Holds for records filed once the rule ships. Nothing assesses the records already in the ledger.
- Holds where the text says enough to tell the three kinds apart. The verb checks that an answer and its grounds were given, never that the answer is right; that is read from the record by a person.
- Assumes a text of one kind. A text that is partly a fault and partly new behaviour is assessed by the part its filer names, and the grounds say which.

## Acceptance Criteria

- **Given** the capture page a session is handed, **when** an agent reads it before filing, **then** it finds the question to answer first, with the three kinds and the one-line test of each: new behaviour a person would see is an intent, a fault against what is promised is an issue, a decision a person owes is a ruling. The paragraph that left the choice to whoever files is replaced by it, and `abcd explain capture` prints the same three.
- **Given** a person or an agent filing through the verb, **when** the capture names no kind, a kind outside the three, or a kind with no grounds, **then** it is refused with exit 2 and nothing is written, and the line they read in the terminal (the error object under `--json`) names the flag, the three values, and the grounds it wants.
- **Given** a capture assessed as an issue or as a ruling, with its one line of grounds, **when** it is filed, **then** a reader of the record under `open/` finds the kind and the grounds on it, the result the filer reads reports both, nothing is promoted, and for a ruling the result names `abcd decide` as where the answer goes once a person gives it.
- **Given** a capture assessed as an intent with no reason given against promoting it, **when** it is filed, **then** the one call writes the issue and promotes it: a draft in `drafts/` seeded from it, the issue naming the draft in `related_intents` and the draft naming the issue in `related_issues`, and the result the filer reads names both ids and both paths. If the promotion fails after the issue is written, the issue stands, and the result says the promotion is owed and names `abcd capture promote <iss-N>`.
- **Given** a capture assessed as an intent with a reason not to promote it now, **when** it is filed, **then** no draft is made and a reader of the record finds the reason beside the kind; a blank reason is refused with exit 2 and nothing is written.
- **Given** the consistency pass, the inbox's promote, the owed-audit filer, or any other filer of abcd's own that is no person's command, **when** it files, **then** the record is written as it is today, with the remedy it writes today, no kind and no promotion, and the record gates pass on it; a test files through each one and reads the record.
- **Given** a capture that names a kind and uses any flag the verb has today (the remedy, severity, category, source, found-at, found-during, lapsed-at, slug, blocked-by, production mode), **when** it is filed, **then** each flag writes and refuses what it did before and the filing-time match still links a likely double; the verb's existing tests pass with the kind added to their calls and no other change.
- **Given** any capture command abcd prints as a next step, or carries in a command page, an agent's brief, or the published docs, **when** this ships, **then** it runs as printed: a test collects each one and none is refused for a missing kind.
- **Given** an open issue with no kind, filed before the rule or by a filer of abcd's own, **when** the board, `abcd capture list`, the record gates, the release cut, or a drain reads the ledger, **then** it is read as it is today and nothing names it as at fault. A drain's dispositions are what the same fields gave before, an issue promoted in the act of capture reads there as `promoted`, and `abcd capture promote` still promotes an issue of any kind or of none, leaving the first assessment on the record.
- **Given** open issues assessed as an intent and not promoted, **when** the person runs the release preview or the cut (`abcd launch --dry-run`, `abcd launch ship`), **then** they read each one listed by id with its reason and the promote command, in the plain report and under `--json`; nothing is refused for them, and with none the report adds no line.

## Open Questions

- **Deferred** Whether a drain should read the assessed kind, handing back an issue assessed as a ruling before any lane opens on it, is left for the product thinker. Nothing here needs it: a drain's dispositions are unchanged, and its own judgement of each eligible issue is asked as before.

## Decisions

1. The product thinker's, on 2026-11-25: this is an intent, it matters, and it is built in their six-day unattended run. They gave no wording for a criterion. The criteria, the Mechanism, and the scope conditions are the writing session's, written from the remedy on iss-2611251832355077, for their review on return; a change then is made through the record.
2. The writing session's, no question put: the requirement stands at the verb a person or an agent runs, and the filers of abcd's own stay as they are, as the remedy rule already keeps the two apart. A machine has no answer to give, and a default written for it would read as a judgement nobody made.
3. The writing session's, no question put: the release check lists and refuses nothing. The remedy says "lists", and a refusal would hold a cut on a judgement of kind. To reverse: make it a refusal with a recorded deferral, as the unfixed-findings guard has.
4. The writing session's, no question put: the names of the flags and of the record's fields are the spec's, since a name is a technical choice. The three values are `issue`, `intent`, and `ruling`.
5. The writing session's, for review: the impact is stamped `additive`, as the requirement of a remedy was (itd-82). The new flag refuses a capture that was accepted before, so the product thinker may judge it `breaking`; the stamp is theirs to change before the close.

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent: a capture is assessed by kind as it is filed. The three-way routing is a stance that already has a home, the decomposition discipline (itd-84), whose routing this carries to the capture; no new principle. No trust rule: nothing here changes what an agent may do or who gives a consent. The flags, the record's fields, the page's wording, the printed commands, and the release list are plumbing, for the spec and for the brief's capture and launch chapters (`04-surfaces/06-capture.md`, `04-surfaces/04-launch.md`). Typed links: `refines` itd-84 and `refines` itd-4, the ledger whose promotion the act of filing now reaches; `related_issues` iss-2611251832355077, which the change that ships this resolves. No reversal is flagged. Verdict: FILE-AS-IS, the routing for a person to confirm.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
