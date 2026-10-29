---
id: itd-2610271130017437
slug: a-finding-with-a-safe-fix-is-fixed-where
spec_id: null
kind: discipline
kind_notes: "A rule every finder inherits, in every repository abcd manages: a confirmed finding with a safe fix is fixed in the change that found it, and only a finding that needs a ruling is filed. Staged: taught first, then held at capture and at the automated filers."
suggested_kind: discipline
reclassification_history: []
builds_on: []
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A finding with a safe fix is fixed where it is found

## The rule

**A confirmed finding with a safe fix is fixed in the change that found it, and
only a finding that needs a ruling is filed.** The rule holds in every repository
abcd manages and for every finder: an automated pass (a lint, the consistency pass,
a review), an agent at work, and a person.

1. **Confirm first.** A finding is checked against the current record before
   anything is done with it. A duplicate merges into the finding it repeats and
   keeps its location. A finding a record already settles is dropped, naming that
   record. A difference in level of detail, a summary against its fuller statement,
   is not a conflict. A finding that is not real is dropped, and the pass keeps the
   reason so it is not raised again.
2. **Fix what is safe.** A fix is safe when it changes no reader's understanding of
   what the project requires, or when a record the change cites (an ADR, a
   decision, the passage that is the rule's one home) already decides it. A safe fix
   is made in the change that found the finding, or, when it is too large to review
   beside that work, in the next change of its own. It is not filed: the commit that
   makes it is its record.
3. **Fix a shared cause once.** Findings with one cause are one finding with many
   locations. Its detector lands first, then every instance is cleared in one
   mechanical change, and the detector stops new ones.
4. **Propose across a boundary.** A safe fix to something the finder may not change
   (another person's area, a peer's uncommitted work, a checkout it must not write
   to) is proposed to its owner as a change, not filed.
5. **File only a ruling.** A finding whose fix would change what a passage requires,
   with no record saying which version wins, is filed, and its remedy names the
   decision and its defensible answers. A finding about a security topic follows its
   own private route and is never filed here.

## Why

The product thinker ruled it on 2026-10-27, in these words: "minor editing
inconsistencies should only be an issue if they cannot be fixed in place; otherwise,
it's much easier to simply fix them in place while they are being discovered.
probably requires SOTA first." Then: "should be a discipline that can be enforced?
It should apply to all abcd-managed repos".

The occasion was the 2026-10-26 consistency pass over the brief and every intent
(`.abcd/work/reviews/2026-10-26-consistency/`). It filed 1,301 records from 1,307
findings. Its triage found 584 to be one-passage edits with a known fix, 648 to need
no change at all, and 69 to need a person's ruling. Every one was filed, read by a
triage reader and approved one batch at a time (iss-2610271115000748 records it),
and the edits are still to be made: work a fix at discovery does once, without
the noise.

The state-of-the-art check the ruling asked for is
`.abcd/development/research/notes/2026-10-27-fix-in-place-or-file-sota.md`, the one
home of the evidence. In short: Facebook's Infer, run as a batch that filed findings,
saw "almost none" acted on, and the same analysis at code review was fixed "over
70%" of the time; Google's FindBugs Fixit filed 1,746 reports and saw 640 warnings
fixed. Linters draw the safe line the same way (Ruff: a safe fix keeps "the meaning
and intent"; ESLint: a fix that could change behaviour is offered as a suggestion).
The check also corrected the first statement of the rule in three places, which
points 1, 3 and 4 carry: confirm before fixing, fix a shared cause behind its
detector, and propose rather than file across a boundary.

## The gate

- **Given** a capture of category `inconsistency` or `documentation` at severity
  `minor` or `nitpick`, **when** it is filed, **then** it names why it cannot be
  fixed in place, as one of `needs-ruling`, `outside-boundary` or `too-large`, and
  a capture naming none is refused with the fix-in-place route as its remedy.
- **Given** a finding an automated filer meets (the consistency ingest, a reading's
  ingest, a review's receipt), **when** the finding carries a fix the filer marks
  safe, **then** the filer applies the fix or proposes it as a change and files
  nothing; only findings marked as needing a ruling become records.
- **Given** a pass whose triage shows more than one finding in ten not real,
  **when** the pass is run again, **then** its output names the rate, and the pass's
  prompt or rule is fixed before its findings are acted on.

## Fit

- **It decides when a finding is deferred at all.** itd-2610141956319695 says a
  deferred finding is its own captured record; this rule says which findings are
  deferred in the first place: a ruling, or a safe fix that cannot be made now.
- **It keeps fix-the-detector's order** (`.abcd/development/principles/fix-the-detector.md`):
  a shared cause gets its detector before its instances are cleared.
- **It sharpens pre-existing-is-not-a-defence**
  (`.abcd/development/principles/pre-existing-is-not-a-defence.md`): a pre-existing
  finding with a safe fix is fixed in place like any other, not filed and stepped
  over.
- **It is the default the consistency pass's automatic mode plans**
  (itd-2609291923559186): fix the obvious, file the rest.
- **It ships to every managed repository**, as a principle that shapes how abcd helps
  a product thinker build their app (the product thinker's ruling of 2026-10-27,
  dcn-2610271112339101): the bundled rules teach it, and the gates are the binary's.

## Staging

Nothing is armed yet. The rungs, in order:

1. **Taught.** The bundled `ISSUES` rules domain, which every managed repository
   loads, states the rule, so an agent meets it before it files.
2. **Held at capture.** The first gate: the capture verb refuses a minor
   inconsistency or documentation finding that names no reason it cannot be fixed in
   place.
3. **Held at the automated filers.** The second gate, starting with the consistency
   ingest: its findings carry a fix and a safe-or-ruling mark, safe ones are applied
   or proposed, and rulings alone are filed.
4. **Measured.** The third gate: each pass reports its not-real rate from its
   triage.

The build of all four is iss-2610271115000748.
