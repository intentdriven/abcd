package guard

import (
	"os"
	"path/filepath"
	"testing"
)

// TestApproveRefusesWiderCriticalRemovals: a removal, a move or a recursive
// change of a directory whose loss is not the working tree's to decide is
// refused: home's direct children, the repository's other checkouts, a find
// that deletes from home or the root, a move of home or another checkout, an
// rsync that deletes into one, and a recursive chmod or chown of one.
func TestApproveRefusesWiderCriticalRemovals(t *testing.T) {
	main, repo := gitRig(t)
	lanes := t.TempDir()
	wt, other := filepath.Join(lanes, "lane"), filepath.Join(lanes, "other")
	repo.Git("worktree", "add", "-q", "-b", "lane", wt)
	repo.Git("worktree", "add", "-q", "-b", "other", other)
	if err := os.MkdirAll(filepath.Join(main.home, "Documents"), 0o700); err != nil {
		t.Fatal(err)
	}
	r := approveRig{dir: wt, home: main.home}
	critical := refusedAs(criticalRemovalEntryID)
	r.run(t, []approveCase{
		{`rm -rf ~/Documents`, critical},
		{`rm -rf ` + filepath.Join(main.home, "Documents"), critical},
		{`rmdir ~/Documents`, critical},
		{`rm -rf ` + main.dir, critical},
		{`rm -rf ` + filepath.Join(main.dir, "internal"), critical},
		{`rm -f ` + filepath.Join(main.dir, "README"), critical},
		{`rm -rf ` + other, critical},
		{`find ~ -name '*.tmp' -delete`, critical},
		{`find / -name core -delete`, critical},
		{`find "$HOME" -type f -delete`, critical},
		{`find ~ -type f -exec rm {} +`, critical},
		{`find -L /Users -exec rm -f {} \;`, critical},
		{`find ~/ -execdir rm -rf {} +`, critical},
		{`mv ~ /tmp/elsewhere`, critical},
		{`mv ` + main.dir + ` /tmp/elsewhere`, critical},
		{`mv -t /tmp ` + other, critical},
		{`rsync -a --delete src/ ~/`, critical},
		{`rsync -a --delete-after src/ ` + main.home, critical},
		{`rsync -a --del src/ ~`, critical},
		{`chmod -R 777 ~`, critical},
		{`chown -R bob ` + main.home, critical},
		{`chmod --recursive u+w ~`, critical},
		{`chgrp -Rh staff /`, critical},
		// The working tree's own.
		{`rm -rf build`, nil},
		{`find . -name '*.o' -delete`, nil},
		{`find build -type f -delete`, nil},
		{`find ~ -name '*.tmp'`, nil},
		{`mv a.txt b.txt`, nil},
		{`rsync -a --delete src/ build/`, nil},
		{`rsync -a src/ ~/backup`, nil},
		{`rsync -a --delete src/ host:backup/`, nil},
		{`chmod -R u+w build`, nil},
		{`chmod 644 README`, nil},
	})
}
