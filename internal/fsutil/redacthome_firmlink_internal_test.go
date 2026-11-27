package fsutil

import "testing"

// On macOS /Users is a firmlink into the data volume, and a path under it can
// be reported as /System/Volumes/Data/Users/<name>/…, so a home under /Users
// is redacted in that spelling too; off macOS the spelling names nothing and is
// not added.
func TestRedactHomeAtRedactsTheDataVolumeSpellingOfAUsersHome(t *testing.T) {
	restore := dataVolumeFirmlink
	t.Cleanup(func() { dataVolumeFirmlink = restore })
	in := "open /System/Volumes/Data/Users/alice/x: denied"

	dataVolumeFirmlink = true
	if got, want := RedactHomeAt(in, "/Users/alice"), "open ~/x: denied"; got != want {
		t.Errorf("RedactHomeAt(%q) = %q, want %q", in, got, want)
	}
	if in := "/System/Volumes/Data/srv/alice/x"; RedactHomeAt(in, "/srv/alice") != in {
		t.Errorf("a home outside /Users gained a data-volume spelling: %q", RedactHomeAt(in, "/srv/alice"))
	}

	dataVolumeFirmlink = false
	if got := RedactHomeAt(in, "/Users/alice"); got != in {
		t.Errorf("off macOS RedactHomeAt(%q) = %q, want it unchanged", in, got)
	}
}
