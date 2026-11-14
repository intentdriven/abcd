package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAWriteIsNotApprovedWhereTheHooksPathCannotBeRead: where git cannot
// answer the read of core.hooksPath, the approver cannot check a path against
// the hooks directories git runs, so it approves no write or removal there,
// naming the read; a line that writes nothing is still approved, and a drive's
// stage's write check says the same (iss-2611140432160343). Watched failing
// before: the write was approved and the stage's check let it through.
func TestAWriteIsNotApprovedWhereTheHooksPathCannotBeRead(t *testing.T) {
	r, _ := gitRig(t)
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in *--get-all\\ core.hooksPath*) echo 'fatal: bad config line 1 in file /x' >&2; exit 128 ;; esac\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, line := range []string{"echo x > notes.txt", "rm -f notes.txt"} {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != protectedPathEntryID || !strings.Contains(a.Reason, "git config --get-all core.hooksPath") {
			t.Errorf("%q: a write where the hooks path cannot be read is not approved, naming the read: %+v", line, a)
		}
	}
	if a := approveShell(clean(), r.files(), "ls"); !a.Allow {
		t.Errorf("a line that writes nothing is approved: %+v", a)
	}
	if why := StageWriteRefusal([]string{r.dir}, nil, filepath.Join(r.dir, "notes.txt")); !strings.Contains(why, "core.hooksPath") {
		t.Errorf("the stage's write check refuses it too, naming the read: %q", why)
	}
}
