package ahoy

import (
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// refusedScanDeepNote says that an answer to the deep-scan question was not one
// it accepts, so scan.deep was not saved, and how to give it without being
// asked: the documented `yes |` pipe answers "y", which is neither
// (iss-2610020700263496).
func refusedScanDeepNote(answer string) string {
	return "the answer " + termsafe.Sanitize(strconv.Quote(answer)) + " to scan_deep names none of " +
		strings.Join(scanDeepChoices, ", ") + ", so scan.deep was not saved; run abcd ahoy install again with --scan-deep true or --scan-deep false"
}
