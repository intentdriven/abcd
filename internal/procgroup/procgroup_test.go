package procgroup

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// r6: Kill on a command that never started has no group to address, and says
// so as an exec.Cmd.Cancel does, rather than dereferencing a nil process.
func TestKillOnACommandThatNeverStartedIsProcessDone(t *testing.T) {
	cmd := exec.Command("true")
	Lead(cmd)
	if err := Kill(cmd); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("Kill before Start = %v, want os.ErrProcessDone", err)
	}
}
