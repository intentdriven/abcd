package abcdhome

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMachineHomeIsThePersonsWhereOneIsNamed: under a stage's home the shared
// records stay the person's; a relative value names nothing.
func TestMachineHomeIsThePersonsWhereOneIsNamed(t *testing.T) {
	stage, person := t.TempDir(), t.TempDir()
	t.Setenv("HOME", stage)
	t.Setenv(StageHomeEnv, stage)
	t.Setenv(PersonHomeEnv, person)
	if got, err := MachineHome(); err != nil || got != person {
		t.Fatalf("MachineHome = %q %v, want %q", got, err, person)
	}
	t.Setenv(PersonHomeEnv, filepath.Base(person))
	if got, _ := MachineHome(); got != stage {
		t.Fatalf("a relative value is ignored: %q", got)
	}
	// A process that has since moved its home, as a test's fixture does, is no
	// longer under the stage's, and keeps to its own.
	t.Setenv(PersonHomeEnv, person)
	other := t.TempDir()
	t.Setenv("HOME", other)
	if got, _ := MachineHome(); got != other {
		t.Fatalf("a home moved off the stage's is its own: %q", got)
	}
	t.Setenv("HOME", stage)
	if err := os.Unsetenv(PersonHomeEnv); err != nil {
		t.Fatal(err)
	}
	if got, _ := MachineHome(); got != stage {
		t.Fatalf("with none named it is the process's own: %q", got)
	}
}
