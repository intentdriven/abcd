package dogfood

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// dogfood_remote_test.go — which declared remotes may vouch for the reviewed
// tip (iss-2610070547057816, security review round 2). A remote this account
// can write — a filesystem path or a file:// URL — is one an agent in the
// checkout can move with `git -C <mirror> fetch <checkout> +HEAD:refs/heads/main`,
// and a plaintext transport (git://, http://) is one anyone on the network
// path can answer for. Either would let the builder's ls-remote record the
// agent's commit as the verified tip and build it.

// TestWritableRemoteNeverSelectsTheAgentsCommit: a line declaring a local
// path or file:// remote, which the agent moved to its own commit, builds
// nothing; the installed release runs and a note says the remote is refused.
// The real transport runs here, so before the fix the builder read the
// agent's commit from the mirror and built it.
func TestWritableRemoteNeverSelectsTheAgentsCommit(t *testing.T) {
	for _, shape := range []struct {
		name string
		url  func(f fixture) string
	}{
		{"absolute path", func(f fixture) string { return f.remote }},
		{"file URL", func(f fixture) string { return "file://" + f.remote }},
	} {
		t.Run(shape.name, func(t *testing.T) {
			f := newFixture(t, "github.com/intentdriven/abcd")
			prev := lsRemote
			t.Cleanup(func() { lsRemote = prev })
			lsRemote = lsRemoteGit
			evil := f.agentCommit(t)
			// The agent, as this account, writes its commit into the mirror the
			// line declares.
			git(t, f.remote, "fetch", "-q", f.checkout, "+HEAD:refs/heads/main")
			f.list(t, f.checkout+"\t"+shape.url(f)+"\tmain")
			p := settle(t, f.home, f.checkout)
			if matches, _ := filepath.Glob(abcdhome.Path(f.home, "dogfood", "*", evil)); len(matches) != 0 {
				t.Fatalf("the agent's commit %s was built from a remote it can write: %v", short(evil), matches)
			}
			if p.Binary != "" || p.StartBuild || p.Checkout != nil {
				t.Fatalf("Resolve with a writable remote declared = %+v, want the installed release and no check", p)
			}
			if n := notes(p); !strings.Contains(n, "REFUSED") || !strings.Contains(n, "installed release runs") {
				t.Fatalf("the note does not refuse the remote out loud and say what runs: %q", n)
			}
			if _, err := os.Lstat(abcdhome.Path(f.home, "dogfood")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("a refused remote touched the build store: %v", err)
			}
		})
	}
}

// TestPlaintextRemotesAreRefused: git:// and http:// let anyone on the
// network path choose the tip, so a line declaring either builds nothing and
// says why, and Build refuses it without asking the remote.
func TestPlaintextRemotesAreRefused(t *testing.T) {
	for _, url := range []string{
		"git://forge.example.invalid/intentdriven/abcd.git",
		"http://forge.example.invalid/intentdriven/abcd.git",
	} {
		t.Run(url, func(t *testing.T) {
			f := newFixture(t, "github.com/intentdriven/abcd")
			calls := 0
			prev := lsRemote
			t.Cleanup(func() { lsRemote = prev })
			lsRemote = func(context.Context, string, string, string) (string, error) {
				calls++
				return "", errors.New("asked")
			}
			f.list(t, f.checkout+"\t"+url+"\tmain")
			p := settle(t, f.home, f.checkout)
			if p.Binary != "" || p.StartBuild || p.Checkout != nil {
				t.Fatalf("Resolve with %s declared = %+v, want the installed release and no check", url, p)
			}
			if n := notes(p); !strings.Contains(n, "REFUSED") || !strings.Contains(n, "plaintext") {
				t.Fatalf("the note does not refuse the plaintext remote out loud: %q", n)
			}
			root := git(t, f.checkout, "rev-list", "--max-parents=0", "HEAD")
			if _, err := Build(f.home, Checkout{Toplevel: f.checkout, Repo: f.checkout, RootSHA: root, URL: url, Branch: "main"}); err == nil {
				t.Fatalf("Build accepted %s", url)
			}
			if calls != 0 {
				t.Fatalf("the remote was asked %d times", calls)
			}
		})
	}
}

// TestBuilderTransportRefusesLocalRepositories: the builder's ls-remote no
// longer allows git's file transport, so even a path that reached it is not
// read.
func TestBuilderTransportRefusesLocalRepositories(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	for _, url := range []string{f.remote, "file://" + f.remote} {
		if tip, err := lsRemoteGit(context.Background(), realDir(t), url, "main"); err == nil {
			t.Errorf("the builder's ls-remote read %s and answered %s", url, tip)
		}
	}
}

// TestDeclaredRemoteShapes: the forge transports are accepted, and every
// shape this account or the network path can move is refused with a reason.
func TestDeclaredRemoteShapes(t *testing.T) {
	accepted := []string{
		"https://github.com/intentdriven/abcd.git",
		"https://user:token@github.com/intentdriven/abcd.git",
		"ssh://git@github.com/intentdriven/abcd.git",
		"ssh://git@github.com:22/intentdriven/abcd.git",
		"git@github.com:intentdriven/abcd.git",
		"github.com:intentdriven/abcd.git",
	}
	for _, url := range accepted {
		if _, _, problem := parseRemote([]string{url, "main"}); problem != "" {
			t.Errorf("parseRemote(%q) refused it: %s", url, problem)
		}
	}
	refused := map[string]string{
		"/srv/git/abcd.git":                        "filesystem path",
		"./abcd.git":                               "filesystem path",
		"../abcd.git":                              "filesystem path",
		"abcd.git":                                 "filesystem path",
		"mirror/abcd.git":                          "filesystem path",
		"~/mirror/abcd.git":                        "filesystem path",
		"mirror/x:y.git":                           "filesystem path",
		"file:///srv/git/abcd.git":                 "file://",
		"file://localhost/srv/git/abcd.git":        "file://",
		"git://github.com/intentdriven/abcd.git":   "plaintext",
		"http://github.com/intentdriven/abcd.git":  "plaintext",
		"ext::sh -c touch% /tmp/pwned":             "transport",
		"fd::17":                                   "transport",
		"ftp://example.com/abcd.git":               "transport",
		"HTTPS://github.com/intentdriven/abcd.git": "transport",
		"https:///abcd.git":                        "host",
		"ssh://-oProxyCommand=touch/abcd.git":      "host",
		"-oProxyCommand=touch:abcd.git":            "safely",
		"git@:abcd.git":                            "host",
	}
	for url, why := range refused {
		_, _, problem := parseRemote([]string{url, "main"})
		if problem == "" {
			t.Errorf("parseRemote(%q) accepted it", url)
			continue
		}
		if !strings.Contains(problem, why) {
			t.Errorf("parseRemote(%q) = %q, want it to say %q", url, problem, why)
		}
	}
}
