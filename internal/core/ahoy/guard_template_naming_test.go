package ahoy

import (
	"regexp"
	"testing"
)

// TestScaffoldedGuardHooksCiteNoAbcdRecord pins the product thinker's ruling F
// (2026-09-23, on iss-2609231103413459): the name-guard hooks keep their marker
// and their naming as the one sanctioned mention of abcd in an adopted
// repository, because they run the binary — but the ruling covers the markers,
// not abcd's own record ids. An adopter's repository holds none of abcd's
// records, so an id or a path into abcd's design record is abcd-internal content
// that resolves to nothing there, and prepare-this-repo's acceptance keeps it out
// of every committed artefact.
func TestScaffoldedGuardHooksCiteNoAbcdRecord(t *testing.T) {
	recordID := regexp.MustCompile(`\b(itd|spc|iss|adr|rcp|rfc)-[0-9]+\b`)
	designRecord := regexp.MustCompile(`\.abcd/development/`)
	for name, tmpl := range map[string][]byte{
		"pre-commit":       guardHookTemplate,
		"pre-merge-commit": guardMergeHookTemplate,
	} {
		for _, m := range recordID.FindAll(tmpl, -1) {
			t.Errorf("%s template cites abcd record id %q", name, m)
		}
		for _, m := range designRecord.FindAll(tmpl, -1) {
			t.Errorf("%s template points into abcd's design record: %q", name, m)
		}
		if !guardHookMarkerRe.Match(tmpl) {
			t.Errorf("%s template lost its name-guard marker, which the ruling keeps", name)
		}
	}
}
