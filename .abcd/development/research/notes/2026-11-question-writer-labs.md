# The question writer's recorded labs (spc-2610301917245442 step 5)

Dated 2026-11-15. This note holds the evidence for the criteria of
itd-2610230756259215 that rest on a live model. Does abcd's question writer
(`agents/question-drafter.md`) return what `abcd guard question` passes? Does
the hook refuse a question the writer did not write?

The spec's approach 11 sets the three labs:
- each lab runs on one pinned binary, in a managed repository holding the
  plugin binary and no abcd checkout;
- each criterion is judged over three recorded runs on one fixed material,
  and passes only when all three do.

The labs live in the machine's lab store
(`~/.abcd.noindex/lab/<root-sha>/<lab-id>/`). Each holds:
- its `INTENTION.md`, written before any run;
- its probe records;
- `findings.md`;
- `harvest/harvest.md`.

**Bottom line.** At page 0.2.4 (pin 727df1898) all fifteen runs pass:

| Lab | Runs | Result |
|---|---|---|
| A | 3 | pass |
| B | 3 | pass |
| C | 9 (each of its three briefs three times) | pass |

The page took five rounds to reach that. Each edit was forced by a failing run
or by a review of the round before. The live-session halves of Labs B and C
need a host session whose own hook runs the lab binary. They are queued for
the person (dcn-2611151605470998), and this note does not claim them.

## Method

**The world.** `git init`, then `abcd ahoy install --adopt` from the lab binary
with HOME set to the lab's own `home/`. Its `.abcd/config.json` sets
`{"question": {"rows": 24}}`, because:
- the 25-row case is over the limit only at 24;
- itd-2610301914350541 raised the default to 32.

**The writer.**
- It runs as a sub-agent of the operator's session.
- It uses the model the person set for question drafting (sonnet).
- Its instructions are the pinned page. The installed plugin carries an older
  page, so the pinned one stands in for the plugin root's.
- In round five it read its brief from its probe's `input` file rather than
  from its prompt, as each lab's `state/refinements.md` records. The brief
  text did not change.

**Before each run**, the operator set the world's mode to the addressee, as the
brief says the caller did. After the run, the operator set it back to
`managed`.

**Each run's pass** was judged by the operator, never taken from the writer. A
run passes when:
- the lab binary's verb, rerun from the world on every returned part, exits 0;
- every tab is within 24 rows by the verb's own count;
- every header is in number form;
- `checked_by` is `abcd`, and the version is the lab binary's;
- the transcript's `guard question` calls are within the bound: three for Lab
  A, the page's five otherwise;
- nothing was written:
  - no write tool appears in the transcript;
  - the world's `git status` is empty;
  - no new file appears under the lab home outside `state/`.

The write check excludes four things the operator touches itself:
- the world's git index;
- the clone's `.git/`;
- the lab's own documents;
- the world's mode store.

Each transcript is read for writes instead.

The pass rules judge only these mechanical properties. The writer's prose, such
as whether a "Now:" line is accurate or whether a shortened quotation drops
detail, is recorded as an observation and left to the person's review.

**Lab B's hook-level half** feeds the lab binary's `guard hook` the PreToolUse
payload a session's question call would carry. The 21-fault call
(`internal/surface/cli/testdata/questions/corpus/four-tabs-21-faults.json`)
must be denied, and the writer's redraft admitted on the second ask.

**Lab C's briefs** are the three that the 2026-10-30 planning interview's
drafter refused for length. iss-2610301855200679 counts three of ten. The
briefs are kept verbatim, except that the older chip becomes the number
header, with the mode set to match.

Each brief holds one decision. dcn-2611151700578698 reads the criterion's
"returns parts" through the intent's decision 33:
- a single question quoted shorter that keeps every answer's gain and cost
  passes;
- parts are expected only where the material holds several decisions or too
  many answers.

Up to round four, Lab C ran each brief once. Round five runs each brief three
times, as decision 22's three runs on one fixed material asks.

## Rounds

| Round (pin, page) | Lab A: 25-row case | Lab B: hook-level half | Lab C: refused-for-length briefs | What changed next |
|---|---|---|---|---|
| a7b0131, 0.2.0 | 1 of 3: two runs spent four checks | not run: its home was left unused when Lab A forced the page change, and every lab was minted again on the next pin (dcn-2611151612261749) | not run, for the same reason | 0.2.1: cut a tab's whole row excess in one edit (dcn-2611151612261749) |
| 905f543, 0.2.1 | 3 of 3 | 3 of 3 | 2 of 3 by the pass rules: one run kept a 29-row tab because the brief asked for clarity over brevity. Two prose defects outside the pass rules: a fact an answer turned on was dropped, and a "Now:" line stated a proposal | 0.2.2: a rows finding is never kept, whatever the brief says; a fact an answer turns on is never dropped; "Now:" states what is in force (dcn-2611151635157888) |
| a0a4658, 0.2.2 | 2 of 3: one run cut 30 → 27 → 25 → 24 over four checks | 3 of 3 | 1 of 3: one run refused after five checks stuck at 25 rows; one spent six checks | 0.2.3: work out what an edit saves before checking again; stop at the fifth check (dcn-2611151643599731) |
| a7cecf1, 0.2.3 | 3 of 3 | 3 of 3 | 3 of 3, one run per brief | 0.2.4: the step's reviews found that 0.2.3 told the writer to spread one decision over parts, which decision 33 rules out (dcn-2611151700578698) |
| 727df18, 0.2.4 | **3 of 3**, two checks each | **3 of 3**, checks 1, 2, 2 | **9 of 9**, three per brief, checks 4, 4, 3 / 3, 2, 3 / 5, 2, 2 | 0.2.5 and 0.2.6: wording only, from the step's reviews, which no run of this round reached (dcn-2611151710456359) |

Labs at the last round:

- **lab-261115170128-727df18 (A).** Every run returns one question at 24 rows
  after two checks, with nothing written. See F-1.
- **lab-261115170133-727df18 (B).** In every run:
  - the first ask is denied, naming 21 parts;
  - the writer's redraft passes the verb;
  - the second ask is admitted (exit 0, no output).

  See F-1.
- **lab-261115170139-727df18 (C).** No run refuses. Every run returns one
  question the verb passes at 24 rows (F-1). No run spreads a decision over
  parts, and every answer's gain and cost survive in its option. F-2 records
  what the prose check found:
  - runs 4 to 9 write a "Now:" that says nothing is set, as 0.2.4 asks;
  - one run of the first brief writes "Now: all three groups are read". That
    describes the draft plan as if it were already in force, so the 0.2.2
    "Now:" rule held in eight of nine runs, not all nine.

## What the rounds taught

- **The writer trims by trial unless the page tells it to count.** At 0.2.0
  and 0.2.2 the failing runs made edits whose saving the writer did not know.
  They cut one or two rows at a time, or reworded lines that kept their row
  count, and used the check to find out. Since 0.2.3 the page has the writer
  work out the saving by the wrapping rules before it checks again, and every
  run since has finished within the bound.
- **A brief's request for length outweighs a page that does not answer it.**
  The 2026-10-30 brief says clarity beats brevity. At 0.2.1 one run kept a
  29-row tab for that reason, which is how the old drafter came to refuse that
  same brief. The page now says a rows finding is never kept, and that such a
  brief is answered within the limit.
- **Shortening is chosen over splitting, and for one decision that is the
  rule.** In eighteen Lab C runs across four rounds, no writer returned parts.
  0.2.4 makes that the rule for one decision: drop first what no answer turns
  on, and never ask one choice twice. Whether the person prefers a compressed
  question to a longer explanation that cannot fit is a design question these
  labs do not settle.

## At a configured limit: the rows-40 lab (spc-2610301950037469 step 4)

itd-2610301914350541 lets a repository set the question limits. Its spec
(approach 10) sets one lab, `question-writer-rows-40`, run as the labs above,
with one change: the world's `.abcd/config.json` sets `question.rows` to 40.
The writer gets Lab A's 25-row material, verbatim, three times.

A run passes when:
- the writer returns one question and no later part;
- the lab binary's verb passes it from the world at 40 rows;
- the `limits` the writer reports are the verb's;
- the transcript holds no read of either configuration file;
- the other rules of Lab A hold, with the page's five checks as the bound.

**Result.** lab-261115224033-30a79f9, pinned at 30a79f966 with page 0.2.7:

| Run | Rows (limit 40) | Checks | Limits reported | Configuration read |
|---|---|---|---|---|
| 1 | 29 | 1 | rows 40, label words 5, sentences 2 | none |
| 2 | 30 | 1 | the same | none |
| 3 | 27 | 1 | the same | none |

All three pass. The first probe shows the verb reporting rows 40 from
`.abcd/config.json`, and the other two keys at abcd's defaults. No run cut
anything: each first draft was within 40 rows. At 24 rows the same material
needed cutting (Lab A above). The writer took the limits only from the check's
`limits`, which page 0.2.7 adds to its return.

## Not covered here, queued for the person

Two halves need a host session whose own hook runs the lab binary, and an
operator's sub-agent cannot be that session:
- **Lab B's live half:** a session asks the 21-fault question without the
  writer, is refused, and its second ask is admitted.
- **Lab C's live half:** three interviews of at least ten questions each, with
  the hook's log read for any deny.

Starting such a session means starting an agent outside this one on the
person's account. The 14 November drive lab made that a stop, so both halves
are queued (dcn-2611151605470998).
