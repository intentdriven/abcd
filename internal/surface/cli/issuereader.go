package cli

import (
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/site"
)

// init registers the issue ledger's reader and the site renderer's body check
// with the lint for every lint the CLI runs (`abcd lint docs`, `abcd lint`), so a config arming record_schema over an
// issue store gets the reader-parity and body legs the record-lint gate runs.
func init() {
	lint.SetIssueReader(capture.ReadRefusal)
	lint.SetRecordBodyCheck(site.CheckRecordBody)
}
