package embargo

import (
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The plan fixtures stand origin in for GitHub with a bare repository named
// fixture-origin.git, since a test binary's git reaches no network: such an
// origin is read as naming the store's example repository. Any other origin
// is read as the plan reads it in production (originRepo), so a test that
// repoints origin at another github.com repository meets the real check.
func init() {
	SwapOriginRepoForTest(func(top string) (string, error) {
		url, err := gitutil.Run(top, "remote", "get-url", "origin")
		if err == nil && filepath.Base(strings.TrimSpace(url)) == "fixture-origin.git" {
			return "example/example", nil
		}
		return originRepo(top)
	})
}
