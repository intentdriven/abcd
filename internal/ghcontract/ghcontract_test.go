package ghcontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/ghcontract"
)

// The pinned lists hold the fields abcd asks for, and not the one gh refused.
func TestTheListsAreGhsOwn(t *testing.T) {
	for _, c := range []struct {
		command, has, hasNot string
	}{
		{"pr view", "headRefOid", "isInMergeQueue"},
		{"pr list", "baseRefName", "isInMergeQueue"},
		{"repo view", "nameWithOwner", "headRefOid"},
	} {
		got := " " + strings.Join(ghcontract.Fields(c.command), " ") + " "
		if !strings.Contains(got, " "+c.has+" ") || strings.Contains(got, " "+c.hasNot+" ") {
			t.Fatalf("%s fields %q: want %s and not %s", c.command, got, c.has, c.hasNot)
		}
	}
}

func TestCheckRefusesAFieldGhRefuses(t *testing.T) {
	for _, c := range []struct {
		args []string
		ok   bool
	}{
		{[]string{"pr", "view", "7", "--json", "mergeable,headRefOid"}, true},
		{[]string{"pr", "list", "--json=number,url,baseRefName"}, true},
		{[]string{"repo", "view", "--json", "nameWithOwner,sshUrl,defaultBranchRef"}, true},
		{[]string{"pr", "merge", "7", "--auto"}, true},
		{[]string{"pr", "view", "7", "--json", "mergeable,isInMergeQueue"}, false},
		{[]string{"pr", "list", "--json=number,isInMergeQueue"}, false},
		{[]string{"repo", "view", "--json", "headRefOid"}, false},
		{[]string{"issue", "view", "1", "--json", "title"}, false},
	} {
		if err := ghcontract.Check(c.args); (err == nil) != c.ok {
			t.Fatalf("Check(%v) = %v, want ok %v", c.args, err, c.ok)
		}
	}
}

// The shell form refuses what Check refuses, for a stub written as a script.
func TestShellRefusesWhatCheckRefuses(t *testing.T) {
	script := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"+ghcontract.Shell()+"ghcontract_check \"$@\"\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"pr", "view", "7", "--json", "mergeable,headRefOid"},
		{"pr", "list", "--json=number,url"},
		{"pr", "view", "7", "--json", "state", "--jq", ".state"},
		{"pr", "view", "7", "--json", "mergeable,isInMergeQueue"},
		{"pr", "list", "--json=number,isInMergeQueue"},
		{"repo", "view", "--json", "headRefOid"},
		{"issue", "view", "1", "--json", "title"},
		{"pr", "merge", "7", "--auto"},
	} {
		out, err := exec.Command(script, args...).CombinedOutput()
		want := ghcontract.Check(args) == nil
		if (err == nil) != want || want != (strings.TrimSpace(string(out)) == "ok") {
			t.Fatalf("stub %v = %q, %v; want ok %v", args, out, err, want)
		}
	}
}
