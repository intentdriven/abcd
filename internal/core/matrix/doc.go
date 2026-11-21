// Package matrix owns a model-matrix run (itd-2610161733316518,
// spc-2610161949559050): one task done many times across a declared matrix of
// models from one frozen commit, judged the way the task allows, merged
// nowhere, and recorded exactly enough to be started again and its drift shown.
//
// This first step holds the run's records and the seam a later move to another
// harness would replace only the scheduler behind:
//
//   - the declaration (declaration.go): the person's matrix, decoded strictly,
//     so a key nobody reads is refused rather than ignored;
//   - the run record (run.go): the frozen commit, the binary's digest and
//     version, the prompt and tool-definition hashes, the render template, the
//     machine, the concurrency, the load cap and the spending cap, with every
//     field the run could not pin listed as unpinned with oracle's reasons,
//     never left out;
//   - the trial record (trial.go): one copy, keyed by task, cell and repeat,
//     carrying the runners' role receipts and the routes' receipts unchanged
//     (itd-166's asked and served sides), never a parallel shape;
//   - the export (export.go): the trial record mapped field by field onto a
//     Harbor trial result and an Inspect eval sample, and back;
//   - the copies (clone.go, cell.go, localserver.go; step 3): each copy made
//     in its own clone of the frozen commit alone in the machine-scoped store,
//     outside every working tree, with no local tier and no other copy's
//     refs; dispatched in the runners' cell mode, so a runner's failure is
//     the cell's and nothing falls back; an outage before the copy began
//     retried on the same pin and never scored; a clone bound for a provider
//     passed through the launch bundle's scan first; and one local inference
//     server at a time, as an owned process group. The build loop's trial
//     mode (loop.Options.Trial) runs a copy's stages taking no reservation
//     and landing nothing.
//   - the code task's judging (hidden.go, compliance.go; step 4): the hidden
//     tests kept read-only beside the copies and never in one, a copy refused
//     whose clone holds one; and the compliance check every copy passes
//     before it is ranked, reading the copy as data: a copy that edited the
//     tests it was given or read another copy is excluded and named, one the
//     check could not examine is unexamined and reported apart, and what no
//     check can read (the network policy, a host copy's isolation) is
//     recorded as asserted, not proven.
//
// A run is a lab procedure (decision 4): its records live under the lab's
// state/matrix/ and reach it only through lab.Procedure, so a run obeys the
// lab's halts and never writes outside its own directory.
//
// A reviewer model is blinded (review.go, udiff.go, reviewscore.go; step 5,
// adr-2610161733400551): it is handed one copy's normalised diff from the
// frozen tree, assembled through the reading surface's include table under an
// opaque label, and nothing else, never a commit, trailer, author, branch or
// record of the run; a hashed manifest the run keeps says what travelled and
// what was withheld. Each reviewer is scored on its own, never by vote, on
// copies whose truth is known: recall on the failing and the seeded-defect
// copies, the true-negative rate on the passing ones, its guess at each
// copy's author as the measured leak, and its verdicts on its own developer's
// copies reported apart.
//
// A check task's copies are compared in two phases (checkcopy.go,
// checkmatch.go, checkjudge.go; step 7, decisions 22, 27 and 30). Each copy's
// findings are read as untrusted input into the run's record before its clone
// is deleted, a doc-fidelity verdict through the recorder's own decoder and
// never saved where the release gates read, every position computed from the
// frozen material; and the run proves it left the doc-fidelity receipts, the
// review flags and the issue ledger as they were. Findings are matched into
// rows by exact key first (the consistency finding key, or the document,
// chapter and quoted sentence), then by overlapping lines, and a matcher model
// is shown only what neither joins, its agreement with a labelled sample
// reported. The person judges each row first, shuffled, with no model named
// and no proposed verdict; only once their verdicts are locked does the run
// show the outside judge's proposals and the overlap table, with each model's
// precision, unique valid findings, and recall on the seeded falsehoods and
// against the findings judged valid.
//
// The package never writes to stdout: it returns results, and the front doors
// under internal/surface render them. Its entry points (parseDeclaration,
// newRunRecord, writeRun) are unexported until the front door that calls them
// lands (spec step 2): a core function arrives with its caller
// (internal/reachaudit). The exported types and their methods are the records
// those entry points take and return.
package matrix
