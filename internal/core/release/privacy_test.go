package release

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// TestCutRefusesAHardFailFindingInReleaseText — iss-2609290405381338. The
// changelog lines and the page's headlines are host-composed prose written to
// CHANGELOG.md and RELEASE.md, public release text. They were held to the
// outbound policy alone, so a well-formed token or the caller's own home path
// was written as it stood. The rendered documents are now held to the bar the
// launch scan applies to the same files: any hard_fail finding refuses the cut,
// nothing is written, and the reason names the kind and the line, never the
// matched text.
func TestCutRefusesAHardFailFindingInReleaseText(t *testing.T) {
	token := "ghp_" + strings.Repeat("A", 40) // FAKE GitHub PAT shape
	plainKey := "sk-" + testsecret.Synthetic(62, 40)
	home := filepath.Join(t.TempDir(), "zzcallerhome")
	t.Setenv("HOME", home)
	cases := []struct {
		name, at, leak string
		raw            func(t *testing.T) []byte
	}{
		{"a token in a changelog line", "entries", token, func(t *testing.T) []byte {
			e := pageEntries()
			e[2].Text = "Fixed the fetch that sent " + token + " upstream."
			return marshalPage(t, "v0.4.1", e, goodPage())
		}},
		{"a token in a headline", "press_release", token, func(t *testing.T) []byte {
			p := goodPage()
			p.Headlines[0].Text = "Releases carry " + token + " now."
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}},
		// A plain sk- key only warns, so a hash-like string already committed
		// cannot fail the launch scan, but it is a token: composed public text
		// must not carry it.
		{"a plain sk- key in a changelog line", "entries", plainKey, func(t *testing.T) []byte {
			e := pageEntries()
			e[2].Text = "Fixed the client that logged " + plainKey + " at start."
			return marshalPage(t, "v0.4.1", e, goodPage())
		}},
		{"the caller's home in a changelog line", "entries", "zzcallerhome", func(t *testing.T) []byte {
			e := pageEntries()
			e[2].Text = "Fixed the cache under " + home + "/cache."
			return marshalPage(t, "v0.4.1", e, goodPage())
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := pageRepo(t)
			before := treeDigest(t, r.Root())
			res, err := Ingest(r.Root(), liveSurface(), tc.raw(t), cutAt)
			refusal := refusalOf(t, err)
			reason, ok := reasonWith(refusal, ReasonPrivacy)
			if !ok {
				t.Fatalf("codes = %v, want %s", codesOf(refusal), ReasonPrivacy)
			}
			if reason.At != tc.at || !strings.Contains(reason.Detail, "line ") {
				t.Errorf("reason = %+v, want it at %s naming the line", reason, tc.at)
			}
			if strings.Contains(refusal.Error(), tc.leak) {
				t.Errorf("the refusal echoes the finding: %v", refusal)
			}
			if res.Written {
				t.Error("a refused payload reported a write")
			}
			if after := treeDigest(t, r.Root()); after != before {
				t.Error("a refused payload changed the working tree")
			}
		})
	}
}

// TestCutRefusesAnotherAccountsHomePathAndAdmitsAUsername — iss-2609290405451613,
// ruled by the product thinker: public release text is held to a middle bar.
// Another account's home path (home_path_other, a warn the launch scan lets
// pass) is refused in a changelog line and a headline, naming the kind and the
// line and never the path; a GitHub username (github_username, also a warn)
// stays allowed, because a release note may legitimately name the repository's
// owner.
func TestCutRefusesAnotherAccountsHomePathAndAdmitsAUsername(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), "zzcallerhome"))
	other := "/Users/zzotherperson/notes" // abcd-lint:allow — a planted third-party home path the refusal must not echo
	cases := []struct {
		name, at string
		raw      func(t *testing.T) []byte
	}{
		{"in a changelog line", "entries", func(t *testing.T) []byte {
			e := pageEntries()
			e[2].Text = "Fixed the cache that read " + other + " at start."
			return marshalPage(t, "v0.4.1", e, goodPage())
		}},
		{"in a headline", "press_release", func(t *testing.T) []byte {
			p := goodPage()
			p.Headlines[0].Text = "Releases no longer read " + other + " now."
			return marshalPage(t, "v0.4.1", pageEntries(), p)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := pageRepo(t)
			before := treeDigest(t, r.Root())
			res, err := Ingest(r.Root(), liveSurface(), tc.raw(t), cutAt)
			refusal := refusalOf(t, err)
			reason, ok := reasonWith(refusal, ReasonPrivacy)
			if !ok {
				t.Fatalf("codes = %v, want %s", codesOf(refusal), ReasonPrivacy)
			}
			if reason.At != tc.at || !strings.Contains(reason.Detail, "home_path_other") ||
				!strings.Contains(reason.Detail, "line ") {
				t.Errorf("reason = %+v, want it at %s naming home_path_other and the line", reason, tc.at)
			}
			if strings.Contains(refusal.Error(), "zzotherperson") {
				t.Errorf("the refusal echoes the path: %v", refusal)
			}
			if res.Written {
				t.Error("a refused payload reported a write")
			}
			if after := treeDigest(t, r.Root()); after != before {
				t.Error("a refused payload changed the working tree")
			}
		})
	}

	t.Run("a username is admitted", func(t *testing.T) {
		r := pageRepo(t)
		r.Git("remote", "add", "origin", "https://github.com/zzremoteowner/widgets.git")
		e := pageEntries()
		e[2].Text = "Thanks to zzremoteowner for the crash report."
		// The line must really carry a github_username finding, or admitting it
		// proves nothing.
		sc, err := scanner.New(r.Root())
		if err != nil {
			t.Fatalf("scanner.New: %v", err)
		}
		var sawUser bool
		for _, f := range sc.ScanText(e[2].Text, "probe") {
			sawUser = sawUser || f.Kind == "github_username"
		}
		if !sawUser {
			t.Fatal("the probe line carries no github_username finding; the case would be vacuous")
		}
		res, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", e, goodPage()), cutAt)
		if err != nil {
			t.Fatalf("a line naming a username was refused: %v", err)
		}
		if !res.Written {
			t.Error("a line naming a username was not written")
		}
	})
}
