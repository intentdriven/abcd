//go:build !race

package scanner

// raceEnabled reports whether the race detector is instrumenting this build.
// A duration assertion has to know: the detector multiplies wall clock by an
// order of magnitude, so a bound that holds is indistinguishable from one that
// does not unless the ceiling moves with it (iss-2609091215552981).
const raceEnabled = false
