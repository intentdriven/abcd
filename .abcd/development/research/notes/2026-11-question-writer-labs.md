# The question writer's recorded labs (spc-2610301917245442 step 5)

Dated 2026-11-15. This note holds the evidence for the criteria of
itd-2610230756259215 that rest on a live model: does abcd's question writer
(`agents/question-drafter.md`) return what `abcd guard question` passes, and
does the hook refuse a question the writer did not write? The spec's
approach 11 sets the three labs. Each lab ran on one pinned binary, in a
managed repository holding the plugin binary and no abcd checkout. Each
criterion was judged over three recorded runs and passed only when all three
did. The labs live in the machine's lab store
(`~/.abcd.noindex/lab/<root-sha>/<lab-id>/`), each with its `INTENTION.md`
written before any run, its probe records, `findings.md` and
`harvest/harvest.md`.

**Bottom line.** At page 0.2.3 (pin a7cecf11d) all nine runs pass. The page
took four rounds to reach that: three edits, each forced by a run that failed
the round before. The live-session halves of Labs B and C need a host session
whose own hook runs the lab binary. They are queued for the person
(dcn-2611151605470998), and this note does not claim them.

## Method

- **The world.** `git init`, then `abcd ahoy install --adopt` from the lab
  binary with HOME set to the lab's own `home/`. Its `.abcd/config.json` sets
  `{"question": {"rows": 24}}`: the 25-row case is over the limit only at 24,
  and itd-2610301914350541 raised the default to 32.
- **The writer.** A sub-agent of the operator's session, on the model the
  person set for question drafting (sonnet), with the pinned page as its
  instructions. The installed plugin carries an older page, so the pinned one
  stands in for the plugin root's.
- **Before each run**, the operator set the world's mode to the addressee, as
  the brief says the caller did, and set it back to `managed` afterwards.
- **Each run's pass**, judged by the operator rather than taken from the
  writer:
  - the lab binary's verb, rerun from the world on every returned part,
    exits 0;
  - every tab is within 24 rows by the verb's own count;
  - headers are in number form;
  - `checked_by: abcd`, and the version is the lab binary's;
  - the transcript's `guard question` calls are within the bound (three for
    Lab A, the page's five otherwise);
  - nothing was written: no write tool in the transcript, the world's
    `git status` is empty, and there is no new file under the lab home
    outside `state/`.
- **The write check excludes four things the operator touches itself:** the
  world's git index, the clone's `.git/`, the lab's own documents and the
  world's mode store. Each lab's `state/refinements.md` records the
  exclusions, and each transcript is read for writes instead.
- **Lab B's hook-level half** feeds the lab binary's `guard hook` the
  PreToolUse payload that a session's question call would carry. The 21-fault
  call (`testdata/questions/corpus/four-tabs-21-faults.json`) is expected to
  be denied and the writer's redraft admitted on the second ask.
- **Lab C's briefs** are the three the 2026-10-30 planning interview's
  drafter refused for length (iss-2610301855200679 counts three of ten),
  kept verbatim. Only the older chip changed to the number header, with the
  mode set to match.

## Rounds

| Round (pin, page) | Lab A: 25-row case | Lab B: hook, hook-level half | Lab C: refused-for-length briefs | Page change it forced |
|---|---|---|---|---|
| a7b0131, 0.2.0 | 1 of 3: two runs spent four checks | not run (pending the live-session decision) | not run | 0.2.1: cut a tab's whole row excess in one edit (dcn-2611151612261749) |
| 905f543, 0.2.1 | 3 of 3 | 3 of 3 | 2 of 3: one run kept a 29-row tab because the brief asked for clarity over brevity; one dropped a fact an answer turned on | 0.2.2: a rows finding is never kept, whatever the brief says; never drop a fact an answer turns on; "Now:" states what is in force (dcn-2611151635157888) |
| a0a4658, 0.2.2 | 2 of 3: one run cut 30 → 27 → 25 → 24 over four checks | 3 of 3 | 1 of 3: one run refused after five checks stuck at 25 rows; one spent six checks | 0.2.3: work out what an edit saves before checking again; split where no fact-keeping cut covers the excess; stop at the fifth check (dcn-2611151643599731) |
| a7cecf1, 0.2.3 | **3 of 3**, two checks each | **3 of 3**, checks 2, 2, 1 | **3 of 3**, checks 5, 2, 4 | none |

Labs at the last round:

- **lab-261115164428-a7cecf1 (A).** Every run returns one question at 24 rows
  with two checks, and nothing written. F-1.
- **lab-261115164431-a7cecf1 (B).** Every first ask is denied, naming 21
  parts. Every writer redraft passes the verb, and every second ask is
  admitted (exit 0, no output). F-1.
- **lab-261115164434-a7cecf1 (C).** No run refuses, and every run returns a
  question the verb passes at 24 rows (F-1). No run returned parts. Each
  quoted its material shorter, which approach 9 allows for one decision whose
  material alone is too long, and every answer's gain and cost survived in
  its option. F-2 records what the shortening dropped:
  - run 3 dropped that the beacon is read when the window closes;
  - run 1 wrote "Now:" for a draft plan rather than something in force.

  Neither is a fact an answer turns on. The person's review of the writer's
  prose is the judge of F-2, not the rows check.

## What the rounds taught

- **The writer trims by trial unless the page tells it to count.** At 0.2.0
  and 0.2.2 the failing runs made edits whose saving the writer did not know.
  It cut one or two rows at a time, or reworded lines that kept their row
  count, and used the check to find out. 0.2.3 has it work out the saving by
  the wrapping rules before it re-checks. Every run since has finished within
  bound.
- **A brief's request for length outweighs a page that does not answer it.**
  The 2026-10-30 brief says clarity beats brevity. At 0.2.1 one run kept a
  29-row tab for that reason, which is how the old drafter came to refuse
  that same brief. The page now says a rows finding is never kept, and that
  such a brief is answered within the limit.
- **Shortening is chosen over splitting.** In nine Lab C runs across three
  rounds, no writer returned parts. The page allows either for one decision
  whose material is too long. Whether the person prefers a two-part question
  to a compressed one is a design question these labs do not settle.

## Not covered here, queued for the person

Two halves need a host session whose own hook runs the lab binary, and an
operator's sub-agent cannot be that session:

- Lab B's live half: a session asks the 21-fault question without the
  writer, is refused, and its second ask is admitted.
- Lab C's live half: three interviews of at least ten questions each, with
  the hook's log read for any deny.

Starting such a session means starting an agent outside this one on the
person's account. The 14 November drive lab made that a stop, so both halves
are queued (dcn-2611151605470998).
