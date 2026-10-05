package dashboard

import (
	"os"
	"testing"
)

// TestProcessIdentityIgnoresTheTimeZone holds a process's recorded identity
// to the process, not to the environment that read it: a dashboard started
// under one time zone or language and stopped under another is still the
// same process, so stop signals it rather than reading its run file as stale
// and leaving it listening.
func TestProcessIdentityIgnoresTheTimeZone(t *testing.T) {
	var ids []processID
	for _, env := range [][2]string{{"UTC", "C"}, {"Asia/Tokyo", "C"}, {"America/Los_Angeles", "fr_FR.UTF-8"}} {
		t.Setenv("TZ", env[0])
		t.Setenv("LC_ALL", env[1])
		id, err := readProcess(os.Getpid())
		if err != nil {
			t.Fatalf("TZ=%s LC_ALL=%s: %v", env[0], env[1], err)
		}
		ids = append(ids, id)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] != ids[0] {
			t.Errorf("one process read as %+v under one time zone and %+v under another", ids[0], ids[i])
		}
	}
}
