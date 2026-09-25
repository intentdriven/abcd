package scanner

import "testing"

// device_slug_test.go — iss-2609240646532741: net_device_hostname matched any
// hyphenated token ending in a device word, so a prose slug the author named a
// page with ("migrating-to-the-nas") was stored as [redacted-hostname] and the
// record's reproduction step could no longer be followed. A slug that carries
// an English function word between its hyphens is prose; a machine is named
// <owner>-<device> and carries none.
func TestDeviceHostnameSparesAProseSlug(t *testing.T) {
	pats, sev := DefaultPatterns(), DefaultIdentitySeverities()
	for _, line := range []string{
		"ingest the page topic_home_migrating-to-the-nas and lint",
		"slug `backup-photos-onto-a-nas` reproduces it",
		"see notes/moving-off-my-laptop.md",
	} {
		if f := ScanText(line, Identity{}, pats, sev, "f"); hasKind(f, kindNetDeviceHost) {
			t.Errorf("a prose slug was reported as a device hostname: %q %+v", line, f)
		}
	}
	for _, line := range []string{
		"ssh bobs-macbook",
		"mounted office-nas over smb",
		"the lab machine lab3-thinkpad",
	} {
		if f := ScanText(line, Identity{}, pats, sev, "f"); !hasKind(f, kindNetDeviceHost) {
			t.Errorf("a device hostname was not reported: %q", line)
		}
	}
}
