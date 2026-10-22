package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestApproveLetsAZshAlwaysBlockFailAsBash: zsh's `{ … } always { … }` runs
// the second group with the pipe's input still open to it, so under zsh
// `echo 'touch ran' | { true; } always { bash; }` hands bash the text
// `touch ran` as its script, though the guard reads the bash in a group no
// pipe reaches. The guard reads bash's grammar, and the approver runs every
// approved command under bash, which reads `always` after a group as a
// syntax error. So the line is approved, and handed back it fails
// harmlessly: under either outer shell it exits with bash's syntax error and
// nothing of it runs.
func TestApproveLetsAZshAlwaysBlockFailAsBash(t *testing.T) {
	const line = "echo 'touch ran' | { true; } always { bash; }"
	if zsh, err := exec.LookPath("zsh"); err == nil {
		work := t.TempDir()
		runText(t, zsh, work, line)
		if _, err := os.Stat(filepath.Join(work, "ran")); err != nil {
			t.Fatalf("%q under zsh did not run the piped text (%v), so this case tests nothing", line, err)
		}
	}
	shells := outerShells(t)
	r := plainRig(t)
	text := approvedText(t, r, line)
	for _, sh := range shells {
		out, code := runText(t, sh, r.home, text)
		if code != 2 || !strings.Contains(out, "syntax error") {
			t.Errorf("handed back and run under %s: exited %d, want bash's syntax error (2): %s", sh, code, out)
		}
		for _, d := range []string{r.dir, r.home} {
			if _, err := os.Stat(filepath.Join(d, "ran")); err == nil {
				t.Errorf("handed back and run under %s: the piped text ran in %s", sh, d)
			}
		}
	}
}
