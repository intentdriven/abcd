package guard

// workTally, when non-nil, accumulates a count of the work the guard does on the
// paths whose cost its bounds exist to hold down: bytes tokenized, bytes the
// brace look-ahead scans, and tokens a pattern match walks. It is nil in
// production and set only by the cost guards (work_test.go), which assert the
// count's growth instead of a wall-clock ceiling — a count is the same on an
// idle machine and a loaded one (iss-2609240046582859). Tests in this package
// do not run in parallel, so one package-level counter is enough.
var workTally *int

// tally adds n units of work to workTally when a cost guard is counting.
func tally(n int) {
	if workTally != nil {
		*workTally += n
	}
}
