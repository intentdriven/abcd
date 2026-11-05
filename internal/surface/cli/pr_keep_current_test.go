package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// scripts/pr-keep-current.sh updates a pull request that fell behind its base,
// taking its arming off first and pinning the update to the head it read. An
// update that fails leaves the head where it was, so the arming goes back on at
// that head; one that moves the head leaves it unarmed for abcd land; and each
// line says what happened.

// keepCurrentDeadline bounds one run of the script on a loaded machine.
const keepCurrentDeadline = 2 * time.Minute

const keepCurrentHead = "1111111111111111111111111111111111111111"

const keepCurrentForge = `#!/bin/sh
d="$(cd "$(dirname "$0")" && pwd)"
printf '%s\n' "$*" >> "$d/gh.log"
case "$1 $2" in
  "repo view") echo acme/widgets ;;
  "pr list") echo 7 ;;
  "api graphql")
    if [ -f "$d/no-arming" ]; then echo "BEHIND none ` + keepCurrentHead + ` none"
    else echo "BEHIND none ` + keepCurrentHead + ` SQUASH"; fi ;;
  "api -X")
    if [ -f "$d/refuse-update" ]; then cat "$d/refuse-update" >&2; exit 1; fi
    echo "Updating pull request branch." ;;
  "pr view")
    case "$*" in
      *autoMergeRequest*)
        if [ -f "$d/refuse-readback" ]; then echo "stand-in: no read" >&2; exit 1; fi
        if [ -f "$d/still-armed" ]; then echo true; else echo false; fi ;;
      *) cat "$d/head" ;;
    esac ;;
  "pr merge")
    case "$*" in
      *--disable-auto*) if [ -f "$d/refuse-disarm" ]; then echo "stand-in: refused" >&2; exit 1; fi ;;
      *) if [ -f "$d/refuse-rearm" ]; then echo "stand-in: refused" >&2; exit 1; fi ;;
    esac ;;
  *) echo "stand-in: unexpected call: $*" >&2; exit 1 ;;
esac
`

// keepCurrent runs the script with the stand-in holding head and the files given
// (name to content), with args.
func keepCurrent(t *testing.T, head string, files map[string]string, args ...string) (out, log string) {
	t.Helper()
	gh := t.TempDir()
	if err := os.WriteFile(filepath.Join(gh, "gh"), []byte(keepCurrentForge), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gh, "head"), []byte(head+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for f, body := range files {
		if err := os.WriteFile(filepath.Join(gh, f), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A deadline of its own, so a --watch that does not stop fails here.
	ctx, cancel := context.WithTimeout(context.Background(), keepCurrentDeadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", append([]string{filepath.Join(prePushSourceRoot(t), "scripts", "pr-keep-current.sh")}, args...)...)
	cmd.WaitDelay = 5 * time.Second
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "PATH="+gh+string(os.PathListSeparator)+os.Getenv("PATH"), "PR_KEEP_CURRENT_SETTLE=0", "PR_KEEP_CURRENT_INTERVAL=0")
	raw, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("the script did not finish within %v\n%s", keepCurrentDeadline, raw)
	}
	if err != nil {
		t.Fatalf("the script failed: %v\n%s", err, raw)
	}
	logged, err := os.ReadFile(filepath.Join(gh, "gh.log"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw), string(logged)
}

func runKeepCurrent(t *testing.T, head string, flags ...string) (out, log string) {
	t.Helper()
	files := map[string]string{}
	for _, f := range flags {
		files[f] = ""
	}
	return keepCurrent(t, head, files)
}

const (
	refused422 = "gh: Validation Failed (HTTP 422)\n"
	failed502  = "gh: Bad Gateway (HTTP 502)\n"
)

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

func TestKeepCurrentPutsTheArmingBackWhenTheForgeRefusesTheUpdate(t *testing.T) {
	out, log := keepCurrent(t, keepCurrentHead, map[string]string{"refuse-update": refused422})
	if !strings.Contains(log, "pr merge 7 --auto --squash --match-head-commit="+keepCurrentHead) {
		t.Fatalf("the arming is not back, by its method, at the unmoved head:\n%s", log)
	}
	if strings.Contains(out, "is updated to") || !strings.Contains(out, "HTTP 422") || !strings.Contains(out, "re-armed") {
		t.Fatalf("the script does not say the update was refused and the arming is back:\n%s", out)
	}
}

// A 5xx or a timeout says nothing of whether the update landed, so nothing is
// armed and the line gives the command that arms it.
func TestKeepCurrentLeavesItUnarmedWhenTheUpdateMayHaveLanded(t *testing.T) {
	for _, msg := range []string{failed502, "gh: i/o timeout\n"} {
		out, log := keepCurrent(t, keepCurrentHead, map[string]string{"refuse-update": msg})
		if strings.Contains(log, "pr merge 7 --auto") {
			t.Fatalf("armed on %q:\n%s", msg, log)
		}
		if !strings.Contains(out, "gh pr merge 7 --auto --squash --match-head-commit="+keepCurrentHead) || !strings.Contains(out, "unarmed") {
			t.Fatalf("the script does not name the command that arms it:\n%s", out)
		}
	}
}

func TestKeepCurrentSaysWhenItCannotPutTheArmingBack(t *testing.T) {
	out, _ := keepCurrent(t, keepCurrentHead, map[string]string{"refuse-update": refused422, "refuse-rearm": ""})
	if !strings.Contains(out, "could NOT") || !strings.Contains(out, "gh pr merge 7 --auto --squash --match-head-commit="+keepCurrentHead) {
		t.Fatalf("the script does not name the command that arms it:\n%s", out)
	}
}

// An arming the second read does not find is not withdrawn, updated under or
// made again.
func TestKeepCurrentSkipsAPullRequestWhoseArmingIsGone(t *testing.T) {
	_, log := keepCurrent(t, keepCurrentHead, map[string]string{"no-arming": ""})
	if strings.Contains(log, "pr merge") || strings.Contains(log, "update-branch") {
		t.Fatalf("an arming that is gone was acted on:\n%s", log)
	}
}

// gh can exit 0 without withdrawing, so the arming is read back before the update.
func TestKeepCurrentReadsTheArmingBackBeforeItUpdates(t *testing.T) {
	for _, flag := range []string{"still-armed", "refuse-readback"} {
		out, log := keepCurrent(t, keepCurrentHead, map[string]string{flag: ""})
		if strings.Contains(log, "update-branch") {
			t.Fatalf("%s: the update went ahead:\n%s", flag, log)
		}
		if !strings.Contains(out, "could NOT") {
			t.Fatalf("%s: the script does not say so:\n%s", flag, out)
		}
	}
}

// Under --watch an update the forge refuses is not asked for again at that head,
// so the arming is not withdrawn and made again on every pass.
func TestKeepCurrentWatchStopsOnAnUpdateTheForgeRefuses(t *testing.T) {
	out, log := keepCurrent(t, keepCurrentHead, map[string]string{"refuse-update": refused422}, "--watch")
	if got := strings.Count(log, "update-branch"); got != 1 {
		t.Fatalf("asked %d times, want once:\n%s", got, log)
	}
	if !strings.Contains(out, "#7") || strings.Contains(out, "no open pull request has auto-merge armed") {
		t.Fatalf("the watch does not say what it left:\n%s", out)
	}
}

// The closing line says "stays armed" only of an arming the script put back.
func TestKeepCurrentWatchDoesNotCallAFailedReArmArmed(t *testing.T) {
	out, _ := keepCurrent(t, keepCurrentHead, map[string]string{"refuse-update": refused422, "refuse-rearm": ""}, "--watch")
	if strings.Contains(out, "stays armed") || !strings.Contains(out, "unarmed") || !strings.Contains(out, "#7") {
		t.Fatalf("the closing line does not say #7 is unarmed:\n%s", out)
	}
}

// An update the forge accepted but has not made is not armed under: it may still
// land. The line says the head has not moved yet.
func TestKeepCurrentDoesNotArmAnUpdateStillPending(t *testing.T) {
	out, log := runKeepCurrent(t, keepCurrentHead)
	if strings.Contains(log, "pr merge 7 --auto") {
		t.Fatalf("an arming was made under a pending update:\n%s", log)
	}
	if !strings.Contains(out, "not moved") || strings.Contains(out, "is updated to") {
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
