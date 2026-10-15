package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The brief's first step (scripts/cloud/begin.sh, C3) and the environment
// check it ends on (scripts/cloud/env-check.sh, C2), spc-2610120941441282 step
// 3. The fixture is newCloudCase's clone; `id` and `go` are fakes, so a test
// decides the uid and the toolchain the scripts see.

const (
	beginScript    = "scripts/cloud/begin.sh"
	envCheckScript = "scripts/cloud/env-check.sh"
	remoteMarker   = "CLAUDE_CODE_REMOTE=true"
	absentGateTool = "abcd-test-absent-gate-tool"
	absentSideTool = "abcd-test-absent-side-tool"
)

func (c *cloudCase) declaredGo() string {
	c.t.Helper()
	b, err := os.ReadFile(filepath.Join(c.dir, "go.mod"))
	if err != nil {
		c.t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^go ([0-9][0-9.]*)$`).FindSubmatch(b)
	if m == nil {
		c.t.Fatal("the fixture go.mod declares no go line")
	}
	return "go" + string(m[1])
}

// gateTools replaces the fixture's gate-tool list, so a test names exactly the
// tools in play.
func (c *cloudCase) gateTools(lines ...string) {
	c.t.Helper()
	c.write("scripts/gate-tools.tsv", "# fixture\n"+strings.Join(lines, "\n")+"\n", 0o644)
}

// goReady is the environment in which the fake go is the declared toolchain.
func (c *cloudCase) goReady() string { return "FAKE_GOVERSION=" + c.declaredGo() }

func TestBeginRefusesOutsideCloud(t *testing.T) {
	c := newCloudCase(t)
	for _, env := range [][]string{nil, {"CLAUDE_CODE_REMOTE=false"}} {
		c.resetCalls()
		out, code := c.script(env, beginScript)
		if code != 1 {
			t.Errorf("begin outside the cloud (%v) exited %d, want 1\n%s", env, code, out)
		}
		if !strings.Contains(out, "not running in the environment the brief was written for") {
			t.Errorf("begin outside the cloud does not say where it is not\n%s", out)
		}
		if strings.Contains(out, "env-check") {
			t.Errorf("begin outside the cloud ran the environment check\n%s", out)
		}
		if calls := c.callLog(); calls != "" {
			t.Errorf("begin outside the cloud ran something after the place check:\n%s", calls)
		}
	}
}

func TestBeginRefusesRoot(t *testing.T) {
	c := newCloudCase(t)
	c.git("config", "user.name", "Alice Example")
	c.git("config", "user.email", "12345+alice@users.noreply.github.com")
	c.resetCalls()
	out, code := c.script([]string{remoteMarker, "FAKE_UID=0", c.goReady()}, beginScript)
	if code != 1 {
		t.Fatalf("begin as uid 0 exited %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "root") || !strings.Contains(out, "uid 0") {
		t.Errorf("the refusal does not say it is running as root\n%s", out)
	}
	if calls := c.callLog(); strings.Contains(calls, "git ") || strings.Contains(calls, "go ") {
		t.Errorf("begin went on past the root refusal:\n%s", calls)
	}
	if strings.Contains(out, "env-check") {
		t.Errorf("begin ran the environment check as root\n%s", out)
	}
}

func TestEnvCheckRefusesRoot(t *testing.T) {
	c := newCloudCase(t)
	c.gateTools("git\tpreflight\tthe image")
	out, code := c.script([]string{"FAKE_UID=0", c.goReady()}, envCheckScript)
	if code == 0 || !strings.Contains(out, "uid 0") {
		t.Errorf("env-check as uid 0 exited %d, want a refusal naming uid 0\n%s", code, out)
	}
}

func TestBeginStopsOnVendorOrUnsetIdentity(t *testing.T) {
	type ident struct{ name, email string }
	for _, tc := range []struct {
		label  string
		config *ident   // the checkout's user.name / user.email; nil sets neither
		env    []string // identity from the environment, as git reads it
		hidden []string // what must appear nowhere in the output
		cause  string   // the cause the message names
	}{
		{"an AI vendor's name and mail", &ident{"Claude", "noreply@anthropic.com"}, nil,
			[]string{"Claude", "noreply@anthropic.com"}, "not a person's"},
		{"an AI vendor's mail domain", &ident{"Dev Box", "devbox@anthropic.com"}, nil,
			[]string{"Dev Box", "devbox@anthropic.com"}, "not a person's"},
		{"a forge bot", &ident{"dependabot[bot]", "49699333+dependabot[bot]@users.noreply.github.com"}, nil,
			[]string{"dependabot", "49699333"}, "not a person's"},
		{"a noreply author mailbox", &ident{"Container User", "noreply@example.com"}, nil,
			[]string{"Container User", "noreply@example.com"}, "not a person's"},
		{"a machine committer from the environment", &ident{"Alice Example", "12345+alice@users.noreply.github.com"},
			[]string{"GIT_COMMITTER_NAME=release-bot", "GIT_COMMITTER_EMAIL=release-bot@example.com"},
			[]string{"release-bot"}, "not a person's"},
		{"no identity at all", nil, nil, nil, "not set"},
		{"a name with no email", &ident{"Alice Example", ""}, nil, []string{"Alice Example"}, "not set"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			c := newCloudCase(t)
			c.gateTools("git\tpreflight\tthe image")
			if tc.config != nil {
				if tc.config.name != "" {
					c.git("config", "user.name", tc.config.name)
				}
				if tc.config.email != "" {
					c.git("config", "user.email", tc.config.email)
				}
			}
			out, code := c.script(append([]string{remoteMarker, c.goReady()}, tc.env...), beginScript)
			if code != 1 {
				t.Fatalf("begin with %s exited %d, want 1\n%s", tc.label, code, out)
			}
			if !strings.Contains(out, "identity") || !strings.Contains(out, tc.cause) {
				t.Errorf("the stop does not name its cause (%q)\n%s", tc.cause, out)
			}
			for _, h := range tc.hidden {
				if strings.Contains(strings.ToLower(out), strings.ToLower(h)) {
					t.Errorf("the output repeats the refused identity (%q)\n%s", h, out)
				}
			}
			if strings.Contains(out, "env-check") {
				t.Errorf("begin ran the environment check with no person's identity\n%s", out)
			}
		})
	}

	// The control: a person's identity passes, and begin ends on the
	// environment check, still never printing the identity.
	c := newCloudCase(t)
	c.gateTools("git\tpreflight\tthe image")
	c.git("config", "user.name", "Alice Example")
	c.git("config", "user.email", "12345+alice@users.noreply.github.com")
	out, code := c.script([]string{remoteMarker, c.goReady()}, beginScript)
	if code != 0 || !strings.Contains(out, "env-check") {
		t.Fatalf("begin with a person's identity exited %d without reaching the environment check\n%s", code, out)
	}
	if strings.Contains(out, "Alice") || strings.Contains(out, "12345") {
		t.Errorf("begin printed the identity it accepted\n%s", out)
	}
}

func TestEnvCheckNamesMissingToolSetupAndGates(t *testing.T) {
	c := newCloudCase(t)
	c.gateTools(
		"git\tpreflight\tthe image",
		absentGateTool+"\tsmoke,docs-lint\tthe setup's example step",
		absentSideTool+"\t-\tthe image's extras",
	)
	out, _ := c.script([]string{c.goReady()}, envCheckScript)
	var gateLine, sideLine string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, absentGateTool) {
			gateLine = l
		}
		if strings.Contains(l, absentSideTool) {
			sideLine = l
		}
	}
	if gateLine == "" || !strings.Contains(gateLine, "the setup's example step") ||
		!strings.Contains(gateLine, "smoke") || !strings.Contains(gateLine, "docs-lint") {
		t.Errorf("no one line names the missing gate tool, its setup step and its gates\n%s", out)
	}
	if sideLine == "" || !strings.Contains(sideLine, "the image's extras") {
		t.Errorf("no one line names the missing side tool and its setup step\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, " git ") {
			t.Errorf("a tool that is present was reported: %q", l)
		}
	}
}

func TestEnvCheckStopsForAGateTool(t *testing.T) {
	c := newCloudCase(t)
	c.gateTools("git\tpreflight\tthe image", absentGateTool+"\tsmoke\tthe setup's example step")
	out, code := c.script([]string{c.goReady()}, envCheckScript)
	if code != 1 {
		t.Fatalf("a missing gate tool exited %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "before any work") {
		t.Errorf("the stop does not say the session stops before any work\n%s", out)
	}

	// The toolchain is a gate tool too: a go that is not the declared release,
	// with nothing cached, stops the session and names the release.
	c.gateTools("go\tpreflight\tthe setup's toolchain step")
	out, code = c.script([]string{"FAKE_GOVERSION=go1.0.0", "FAKE_GOMODCACHE=" + t.TempDir()}, envCheckScript)
	if code != 1 || !strings.Contains(out, c.declaredGo()) {
		t.Errorf("a go that is not the declared toolchain exited %d without naming %s\n%s", code, c.declaredGo(), out)
	}
	// The declared release already in go's toolchain cache is ready without a fetch.
	cache := t.TempDir()
	goBin := filepath.Join(cache, "golang.org", "toolchain@v0.0.1-"+c.declaredGo()+".testos-testarch", "bin")
	if err := os.MkdirAll(goBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goBin, "go"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, code := c.script([]string{"FAKE_GOVERSION=go1.0.0", "FAKE_GOMODCACHE=" + cache}, envCheckScript); code != 0 {
		t.Errorf("the declared toolchain in the cache was not taken as ready (exit %d)\n%s", code, out)
	}
	if out, code := c.script([]string{c.goReady()}, envCheckScript); code != 0 {
		t.Errorf("the declared toolchain on PATH was not taken as ready (exit %d)\n%s", code, out)
	}
}

func TestEnvCheckWarnsForANonGateTool(t *testing.T) {
	c := newCloudCase(t)
	c.gateTools("git\tpreflight\tthe image", absentSideTool+"\t-\tthe image's extras")
	out, code := c.script([]string{c.goReady()}, envCheckScript)
	if code != 0 {
		t.Fatalf("a missing tool no gate needs exited %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "warning") || !strings.Contains(out, absentSideTool) {
		t.Errorf("the missing side tool is not warned about\n%s", out)
	}
}

type fileStamp struct {
	size int64
	mode fs.FileMode
	mod  time.Time
}

func treeStamps(t *testing.T, root string) map[string]fileStamp {
	t.Helper()
	stamps := map[string]fileStamp{}
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		stamps[p] = fileStamp{info.Size(), info.Mode(), info.ModTime()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return stamps
}

func TestEnvCheckWritesNothing(t *testing.T) {
	c := newCloudCase(t)
	statusBefore := c.git("status", "--porcelain", "--ignored")
	for _, tools := range [][]string{
		{"git\tpreflight\tthe image", absentGateTool + "\tsmoke\tthe setup's example step"},
		{"go\tpreflight\tthe setup's toolchain step", absentSideTool + "\t-\tthe image"},
	} {
		c.gateTools(tools...)
		before := treeStamps(t, c.dir)
		out, _ := c.script([]string{"FAKE_GOVERSION=go1.0.0", "FAKE_GOMODCACHE=" + t.TempDir()}, envCheckScript)
		after := treeStamps(t, c.dir)
		for p, s := range after {
			if b, ok := before[p]; !ok {
				t.Errorf("env-check created %s\n%s", p, out)
			} else if b != s {
				t.Errorf("env-check changed %s\n%s", p, out)
			}
		}
		for p := range before {
			if _, ok := after[p]; !ok {
				t.Errorf("env-check removed %s", p)
			}
		}
	}
	// The fixture's own list edits are the only change git sees.
	c.git("checkout", "--", "scripts/gate-tools.tsv")
	if got := c.git("status", "--porcelain", "--ignored"); got != statusBefore {
		t.Errorf("git status changed across the checks:\nbefore %q\nafter  %q", statusBefore, got)
	}
}
