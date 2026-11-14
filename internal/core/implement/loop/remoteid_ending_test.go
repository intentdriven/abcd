package loop

// remoteid_ending_test.go holds ruling 3 of fix round 4 of
// iss-2611131308266006: what a drive leaves when a held change ends it is
// said.

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestADriveEndedByAHeldChangeSaysWhatItLeftAndHowToReleaseIt: a
// core.sshCommand in the person's global settings changed during a lane ends
// the drive at the config stage. Its release of what it holds is a push, so
// none is tried: the person is told nothing was released and why, each
// reservation left by its record id, when its lease lapses, and that once the
// setting is put back they release it with `abcd reserve <id> --release` or
// start the drive again. Watched failing with the let-go's comparison
// before the drain lock taken out: the release was tried, its fetch of the
// reservations was not run, and the person was not told what was left.
func TestADriveEndedByAHeldChangeSaysWhatItLeftAndHowToReleaseIt(t *testing.T) {
	w := newTerminalWorld(t)
	stagedGit(t)
	global := filepath.Join(t.TempDir(), "gitconfig")
	writeMode(t, global, "[core]\n\tsshCommand = ssh\n", 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	landed, before := w.changedDuringTheLane(t, func() {
		writeMode(t, global, "[core]\n\tsshCommand = /bin/true\n", 0o644)
	})
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.bounded(t, 3*time.Hour))
	w.heldStop(t, res, err, *landed, *before, "core.sshcommand")
	said := w.said()
	if !strings.Contains(said, "nothing this drive holds was released: releasing is a push, and the drive runs none while what the drive holds changed since it started (core.sshcommand") {
		t.Fatalf("the person is told nothing was released, and why:\n%s", said)
	}
	left := regexp.MustCompile("(?m)^(\\S+): its reservation was not released and lapses at (\\S+); once what the drive holds is as it should be, release it with `abcd reserve (\\S+) --release`, or start the drive again, which takes the run up$").FindStringSubmatch(said)
	if left == nil || left[1] != res.Issue || left[3] != res.Issue {
		t.Fatalf("the reservation left is named with when it lapses and how to release it (issue %q):\n%s", res.Issue, said)
	}
	if at, err := time.Parse(time.RFC3339, left[2]); err != nil || !at.After(time.Now()) {
		t.Fatalf("the lease's end is a time to come: %q %v", left[2], err)
	}
	if strings.Contains(said, "git push was not run") {
		t.Fatalf("no release push is tried under the change:\n%s", said)
	}
}
