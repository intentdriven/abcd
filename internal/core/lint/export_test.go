package lint

// RecordStorePrefixesForTest is the code-side store list, for the coverage check
// in the external test package (duplicatekeyreaders_test.go). It lives in an
// _test.go file, so it is linked into the test binary alone and enlarges no
// production API — the store list stays code rather than becoming a surface.
//
// It exists because the two halves of that check sit on opposite sides of an
// import edge that has only one direction: core/lint cannot import the packages
// whose readers the table exercises (core/record imports core/capture, and
// core/capture's own tests import core/lint), while an external test package can
// import everything. So the rows live over there and the store list is exported
// to them, rather than the check scraping this package's source.
func RecordStorePrefixesForTest() []string {
	out := make([]string, 0, len(recordStores))
	for _, s := range recordStores {
		out = append(out, s.prefix)
	}
	return out
}
