package lint

import "github.com/intentdriven/abcd/internal/core/capture"

// The package's tests run with the ledger reader registered, as the front doors
// run the gate. core/capture does not import this package, so the test binary
// has no cycle. core/site does import it, so the body leg is registered as a
// stand-in that renders every body: the legs under test here are this package's,
// the site renderer's verdict is pinned by core/site's own tests, and
// TestRecordSchemaRefusesAnIssueBodyTheSiteCannotRender swaps in a refusing one.
// Registering both is also what keeps record_schema from naming an
// unregistered seam on every fixture that holds an issue record.
func init() {
	SetIssueReader(capture.ReadRefusal)
	SetRecordBodyCheck(func(rel, content string) error { return nil })
}
