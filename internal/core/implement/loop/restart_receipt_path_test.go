package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestARestartRefusesAReceiptOutsideTheLanesDirectory: an await's receipt path
// comes from the run's state file, which an archive of the checkout can carry
// hand-written, and a restart reads, copies aside and then removes the file it
// names. A state naming a file of the checkout outside the lane's directory
// had the restart move that file into the lane's aside; one naming a path
// inside the lane's asides could overwrite a saved patch. The restart refuses
// any path that is not a clean file of the lane's own directory outside its
// asides, before anything is read or changed.
func TestARestartRefusesAReceiptOutsideTheLanesDirectory(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	planted := filepath.Join(repo.Root(), "planted.txt")
	if err := os.WriteFile(planted, []byte("TOKEN=hunter2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	laneDir := RunRelDir + "/" + runID + "/lane-1"
	// A directory inside the lane's directory that is a symlink to the checkout
	// root, and a file whose name folds to the patch's on a case-insensitive
	// filesystem.
	if err := os.Symlink("../../../../..", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Changes.patch"), []byte("ATTACKER\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, receipt := range []string{
		"planted.txt",
		laneDir + "/../../../../../planted.txt",
		laneDir + "/" + AsideDirName + "/20261012T000000Z/" + AsidePatchName,
		laneDir + "/" + AsidePatchName,
		laneDir + "/link/planted.txt",
		laneDir + "/Changes.patch",
		laneDir + "/ASIDE/20261012T000000Z/receipt.json",
		laneDir + "/a\u017fide/20261012T000000Z/receipt.json",
		laneDir + "/ASIDE.JSON",
		laneDir + `/..\..\..\..\..\planted.txt`,
	} {
		t.Run(receipt, func(t *testing.T) {
			err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
				st.Lanes[0].Awaits[0].Receipt = receipt
				return true, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			before := stateBytes(t, repo.Root(), runID)
			_, err = Restart(repo.Root(), runID, "lane-1", "", Options{})
			if r := mustRefusal(t, err); !strings.Contains(r.Reason, "lane's own directory") {
				t.Fatalf("the restart refuses a receipt outside the lane's directory: %+v", r)
			}
			if got, err := os.ReadFile(planted); err != nil || string(got) != "TOKEN=hunter2\n" {
				t.Fatalf("the file the state named is where it was: %q %v", got, err)
			}
			noAside(t, dir)
			editsIntact(t, repo, l)
			if string(before) != string(stateBytes(t, repo.Root(), runID)) {
				t.Fatal("a refused restart writes nothing")
			}
		})
	}
}
