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
	t.Setenv(PersonHomeEnv, person)
	if got, err := MachineHome(); err != nil || got != person {
		t.Fatalf("MachineHome = %q %v, want %q", got, err, person)
	}
	t.Setenv(PersonHomeEnv, filepath.Base(person))
	if got, _ := MachineHome(); got != stage {
		t.Fatalf("a relative value is ignored: %q", got)
	}
	if err := os.Unsetenv(PersonHomeEnv); err != nil {
		t.Fatal(err)
	}
	if got, _ := MachineHome(); got != stage {
		t.Fatalf("with none named it is the process's own: %q", got)
	}
}
