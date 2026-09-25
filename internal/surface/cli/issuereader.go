package cli

import (
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// init registers the issue ledger's reader with the lint for every lint the CLI
// runs (`abcd docs lint`, `abcd lint`), so a config arming record_schema over an
// issue store gets the reader-parity leg the record-lint gate runs.
func init() { lint.SetIssueReader(capture.ReadRefusal) }
