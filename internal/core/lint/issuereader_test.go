// This file is an EXTERNAL test package on purpose: core/capture reaches core/lint
// through core/site (the reframe verb reads the framing chapter's sections with
// site's walk, and site's build imports this package), so an internal test file
// importing capture would be a cycle. The registration below still runs before
// every test in the binary, the internal ones included, because a test binary
// initialises all of its packages before it runs a test.
package lint_test

import (
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// The package's tests run with the ledger reader registered, as the front doors
// run the gate. core/site imports this package, so the body leg is registered as
// a stand-in that renders every body: the legs under test here are this
// package's, the site renderer's verdict is pinned by core/site's own tests, and
// TestRecordSchemaRefusesAnIssueBodyTheSiteCannotRender swaps in a refusing one.
// Registering both is also what keeps record_schema from naming an
// unregistered seam on every fixture that holds an issue record.
func init() {
	lint.SetIssueReader(capture.ReadRefusal)
	lint.SetRecordBodyCheck(func(rel, content string) error { return nil })
}
