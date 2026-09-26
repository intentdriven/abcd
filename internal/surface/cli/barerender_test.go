package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// barerender_test.go — every top-level verb renders state on a bare call, or
// is a recorded exception with its reason (iss-2609091642508271).

func topLevelVerbs(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, c := range NewRootCommand().Commands() {
		if c.Hidden || c.Deprecated != "" || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		out = append(out, c.Name())
	}
	return out
}

func TestEveryTopLevelVerbRendersStateBareOrIsAnException(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	for _, verb := range topLevelVerbs(t) {
		if _, ok := bareRenderExceptions[verb]; ok {
			continue // never executed: an exception may write bare (update does)
		}
		t.Run(verb, func(t *testing.T) {
			repo, _ := sessionEndRepo(t) // a checkout with one commit, HOME isolated
			t.Chdir(repo)
			t.Setenv("PATH", "/usr/bin:/bin")
			root := NewRootCommand()
			var so, se bytes.Buffer
			root.SetOut(&so)
			root.SetErr(&se)
			root.SetIn(strings.NewReader(""))
			root.SetArgs([]string{verb})
			_ = root.Execute() // a render may exit non-zero (lint's findings); the render is what is judged
			out := so.String()
			if strings.TrimSpace(out) == "" || strings.Contains(out, "\nUsage:") || strings.HasPrefix(out, "Usage:") {
				t.Errorf("bare `abcd %s` renders no state (stdout %q, stderr %q): give it a state render, "+
					"or record why it has none in bareRenderExceptions (barerender.go)", verb, out, se.String())
			}
		})
	}
}

func TestBareRenderExceptionsNameRealVerbsWithAReason(t *testing.T) {
	verbs := map[string]bool{}
	for _, v := range topLevelVerbs(t) {
		verbs[v] = true
	}
	for verb, reason := range bareRenderExceptions {
		if !verbs[verb] {
			t.Errorf("bareRenderExceptions names %q, which is not a visible top-level verb", verb)
		}
		if len(strings.Fields(reason)) < 6 {
			t.Errorf("the exception for %q carries no reason worth the name: %q", verb, reason)
		}
	}
}

// The brief's one enumeration of the exceptions names every one the table
// records, so the prose cannot claim conformance the tree does not have.
func TestTheBriefEnumeratesEveryBareRenderException(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", ".abcd", "development", "brief", "04-surfaces", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, section, ok := strings.Cut(string(data), "## Bare invocation\n")
	if !ok {
		t.Fatal("04-surfaces/README.md has no Bare invocation section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	for verb := range bareRenderExceptions {
		if !strings.Contains(section, "`"+verb+"`") && !strings.Contains(section, "`abcd "+verb+"`") &&
			!strings.Contains(section, "abcd\n"+verb+"`") && !strings.Contains(section, "`abcd\n"+verb+"`") {
			t.Errorf("the brief's Bare invocation section does not name the exception %q", verb)
		}
	}
}
