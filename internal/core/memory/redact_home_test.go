package memory

import (
	"strings"
	"testing"
)

// TestStoreRedactorSweepsHomeOnAPathBoundary drives the memory store's literal
// $HOME backstop under a short home: the home is collapsed to "~" where it
// stands as a path and a body that merely shares its prefix survives intact.
// The unanchored sweep rewrote "/rootfs/etc/hosts" to "~fs/etc/hosts" before
// the page was committed, and the fail-closed check that followed it could
// never fire.
func TestStoreRedactorSweepsHomeOnAPathBoundary(t *testing.T) {
	t.Setenv("HOME", "/root")
	r, err := newStoreRedactor(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := "Mount table copied from /rootfs/etc/hosts on this box.\n" +
		"Scratch lives at /root/scratch on this box; notes in /root_2/x.\n"
	got, _, err := r.redactText(body, "page")
	if err != nil {
		t.Fatalf("redactText: %v", err)
	}
	if !strings.Contains(got, "/rootfs/etc/hosts") {
		t.Errorf("a body sharing the home's prefix was corrupted:\n%s", got)
	}
	if strings.Contains(got, "/root/scratch") || !strings.Contains(got, "~/scratch") {
		t.Errorf("the home standing as a path was not swept to ~:\n%s", got)
	}
	if strings.Contains(got, "/root_2") {
		t.Errorf("the home followed by '_' survived:\n%s", got)
	}
}
