package capture

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/site"
)

// refuseUnrenderableBody holds a record body to the check record_schema runs
// at the gate (the front doors register site.CheckRecordBody with
// lint.SetRecordBodyCheck), so text the site renderer refuses is refused at the
// write, naming the construct, instead of by the next gate run
// (iss-2610261616580903).
func refuseUnrenderableBody(rel, content string) error {
	if err := site.CheckRecordBody(rel, content); err != nil {
		return refused(fmt.Errorf("capture: the issue text carries markdown the site renderer refuses (%v); "+
			"nothing written: put a placeholder, a path or code in a code span, or write the construct "+
			"inside the renderer's subset", err))
	}
	return nil
}
