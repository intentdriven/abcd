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
//     in its own clone of the frozen commit in the machine-scoped store,
//     outside every working tree, with no local tier and no other copy's
//     refs; dispatched in the runners' cell mode, so a runner's failure is
//     the cell's and nothing falls back; an outage before the copy began
//     retried on the same pin and never scored; a clone bound for a provider
//     passed through the launch bundle's scan first; and one local inference
//     server at a time, as an owned process group. The build loop's trial
//     mode (loop.Options.Trial) runs a copy's stages taking no reservation
//     and landing nothing.
//
// A run is a lab procedure (decision 4): its records live under the lab's
// state/matrix/ and reach it only through lab.Procedure, so a run obeys the
// lab's halts and never writes outside its own directory.
//
// The package never writes to stdout: it returns results, and the front doors
// under internal/surface render them. Its entry points (parseDeclaration,
// newRunRecord, writeRun) are unexported until the front door that calls them
// lands (spec step 2): a core function arrives with its caller
// (internal/reachaudit). The exported types and their methods are the records
// those entry points take and return.
package matrix
