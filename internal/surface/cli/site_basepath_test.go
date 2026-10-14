package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestSiteVerbsTakeABasePath: `site build` and `lint site` both take the path
// the site is served below (itd-2610091010348892), and both refuse one that is
// not plain path segments before reading anything else.
func TestSiteVerbsTakeABasePath(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	t.Chdir(repo.Root())
	for _, verb := range [][]string{{"site", "build"}, {"lint", "site"}} {
		out := filepath.Join(t.TempDir(), "site")
		args := append(append([]string{}, verb...), "--out", out, "--base-path", "/../elsewhere/")
		got, err := runCLIErr(t, args...)
		if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "base path") {
			t.Fatalf("`abcd %s --base-path /../elsewhere/` = exit %d, %v:\n%s", strings.Join(verb, " "), code, err, got)
		}
		args = append(append([]string{}, verb...), "--out", out, "--base-path", "/widgets/")
		_, err = runCLIErr(t, args...)
		if err == nil || strings.Contains(err.Error(), "base path") || strings.Contains(err.Error(), "unknown flag") {
			t.Fatalf("`abcd %s --base-path /widgets/` was refused for its base path: %v", strings.Join(verb, " "), err)
		}
	}
}
