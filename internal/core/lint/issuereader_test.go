package lint

import "github.com/intentdriven/abcd/internal/core/capture"

// The package's tests run with the ledger reader registered, as the front doors
// run the gate. core/capture does not import this package, so the test binary
// has no cycle.
func init() { SetIssueReader(capture.ReadRefusal) }
