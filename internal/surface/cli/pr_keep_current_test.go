package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// scripts/pr-keep-current.sh updates a pull request that fell behind its base,
// taking its arming off first and pinning the update to the head it read. An
// update that fails leaves the head where it was, so the arming goes back on at
// that head; one that moves the head leaves it unarmed for abcd land; and each
// line says what happened.

const keepCurrentHead = "1111111111111111111111111111111111111111"

const keepCurrentForge = `#!/bin/sh
d="$(cd "$(dirname "$0")" && pwd)"
printf '%s\n' "$*" >> "$d/gh.log"
case "$1 $2" in
  "repo view") echo acme/widgets ;;
  "pr list") echo 7 ;;
  "api graphql") echo "BEHIND none ` + keepCurrentHead + ` SQUASH" ;;
  "api -X")
    if [ -f "$d/refuse-update" ]; then echo "HTTP 502: Bad Gateway" >&2; exit 1; fi
    echo "Updating pull request branch." ;;
  "pr view") cat "$d/head" ;;
  "pr merge")
    case "$*" in
      *--disable-auto*) if [ -f "$d/refuse-disarm" ]; then echo "stand-in: refused" >&2; exit 1; fi ;;
      *) if [ -f "$d/refuse-rearm" ]; then echo "stand-in: refused" >&2; exit 1; fi ;;
    esac ;;
  *) echo "stand-in: unexpected call: $*" >&2; exit 1 ;;
esac
`

func runKeepCurrent(t *testing.T, head string, flags ...string) (out, log string) {
	t.Helper()
	gh := t.TempDir()
	if err := os.WriteFile(filepath.Join(gh, "gh"), []byte(keepCurrentForge), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gh, "head"), []byte(head+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range flags {
		if err := os.WriteFile(filepath.Join(gh, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("bash", filepath.Join(prePushSourceRoot(t), "scripts", "pr-keep-current.sh"))
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "PATH="+gh+string(os.PathListSeparator)+os.Getenv("PATH"), "PR_KEEP_CURRENT_SETTLE=0")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the script failed: %v\n%s", err, raw)
	}
	logged, err := os.ReadFile(filepath.Join(gh, "gh.log"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw), string(logged)
}

func TestKeepCurrentLeavesAnUpdatedPullRequestUnarmedAtItsNewHead(t *testing.T) {
	newHead := strings.Repeat("2", 40)
	out, log := runKeepCurrent(t, newHead)
	off, update := strings.Index(log, "pr merge 7 --disable-auto"), strings.Index(log, "update-branch")
	if off < 0 || update < off {
		t.Fatalf("the arming comes off before the update:\n%s", log)
	}
	if !strings.Contains(log, "expected_head_sha="+keepCurrentHead) {
		t.Fatalf("the update is not pinned to the head read:\n%s", log)
	}
	if strings.Contains(log, "pr merge 7 --auto") {
		t.Fatalf("the update commit, which no review receipt names, was armed:\n%s", log)
	}
	if !strings.Contains(out, newHead[:12]) || !strings.Contains(out, "abcd land") {
		t.Fatalf("the script does not name the new head and what arms it:\n%s", out)
	}
}

func TestKeepCurrentPutsTheArmingBackWhenTheUpdateFails(t *testing.T) {
	out, log := runKeepCurrent(t, keepCurrentHead, "refuse-update")
	if !strings.Contains(log, "pr merge 7 --auto --squash --match-head-commit="+keepCurrentHead) {
		t.Fatalf("the arming is not back, by its method, at the unmoved head:\n%s", log)
	}
	if strings.Contains(out, "new head") || !strings.Contains(out, "HTTP 502") || !strings.Contains(out, "re-armed") {
		t.Fatalf("the script does not say the update failed and the arming is back:\n%s", out)
	}
}

func TestKeepCurrentSaysWhenItCannotPutTheArmingBack(t *testing.T) {
	out, _ := runKeepCurrent(t, keepCurrentHead, "refuse-update", "refuse-rearm")
	if !strings.Contains(out, "could NOT") || !strings.Contains(out, "gh pr merge 7 --auto --squash --match-head-commit="+keepCurrentHead) {
		t.Fatalf("the script does not name the command that arms it:\n%s", out)
	}
}

// An update the forge accepted but has not made is not armed under: it may still
// land. The line says the head has not moved yet.
func TestKeepCurrentDoesNotArmAnUpdateStillPending(t *testing.T) {
	out, log := runKeepCurrent(t, keepCurrentHead)
	if strings.Contains(log, "pr merge 7 --auto") {
		t.Fatalf("an arming was made under a pending update:\n%s", log)
	}
	if !strings.Contains(out, "not moved") || strings.Contains(out, "new head") {
		t.Fatalf("the script does not say the head has not moved:\n%s", out)
	}
}

func TestKeepCurrentDoesNotUpdateAPullRequestItCouldNotWithdraw(t *testing.T) {
	out, log := runKeepCurrent(t, keepCurrentHead, "refuse-disarm")
	if strings.Contains(log, "update-branch") {
		t.Fatalf("the update went ahead over an arming still in force:\n%s", log)
	}
	if !strings.Contains(out, "could NOT") {
		t.Fatalf("the script does not say so:\n%s", out)
	}
}
