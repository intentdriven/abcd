package embargo

import (
	"slices"
	"testing"
)

// TestReleaseTagsGoThroughTheSharedParser: embargo's release tag is the one
// launch.ParseReleaseTag accepts (spc-2610170635097509 step 1,
// one-canonical-primitive), so a tag whose component is wider than the shared
// parser bounds is no release tag here either, and --release refuses it before
// it is looked up.
func TestReleaseTagsGoThroughTheSharedParser(t *testing.T) {
	for _, tag := range []string{"v1000000000.0.0", "v0.0.99999999999999999999"} {
		if isReleaseTag(tag) {
			t.Errorf("isReleaseTag(%q) = true, want false: the shared parser bounds each component", tag)
		}
	}
	for _, tag := range []string{"v0.9.0", "v10.20.30"} {
		if !isReleaseTag(tag) {
			t.Errorf("isReleaseTag(%q) = false, want true", tag)
		}
	}
	got := sortReleaseTags([]string{"v0.10.0", "v1000000000.0.0", "v0.9.1", "v0.9.1-rc.1", "dev", " v0.2.0 "})
	want := []string{"v0.2.0", "v0.9.1", "v0.10.0"}
	if !slices.Equal(got, want) {
		t.Errorf("sortReleaseTags = %q, want %q", got, want)
	}
}
