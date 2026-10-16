package decisions

import (
	"reflect"
	"testing"
)

// TestFrozenEntriesSplitsTheLogByTopLevelBullet pins the frozen log's reader
// (spc-2610110931381316, step 4): the header belongs to no entry, an entry
// keeps its indented continuation lines and the line its bullet opens on, and
// a carriage return is not part of the text.
func TestFrozenEntriesSplitsTheLogByTopLevelBullet(t *testing.T) {
	log := "# DECISIONS\n\nHeader prose naming itd-10.\n\n" +
		"- 2026-09-20 — first (itd-10),\r\n  continued.\n" +
		"- 2026-09-21 — second.\n" +
		"not indented, so no entry's\n" +
		"- 2026-09-22 — third.\n"
	got := FrozenEntries(log)
	want := []FrozenEntry{
		{Line: 5, Text: "- 2026-09-20 — first (itd-10),\n  continued."},
		{Line: 7, Text: "- 2026-09-21 — second."},
		{Line: 9, Text: "- 2026-09-22 — third."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FrozenEntries = %+v\nwant %+v", got, want)
	}
}

// TestNamesMatchesWholeIDs: an id that only starts a longer one is not named,
// and an empty id names nothing.
func TestNamesMatchesWholeIDs(t *testing.T) {
	for text, want := range map[string]bool{
		"the loop (itd-10).":     true,
		"itd-10":                 true,
		"itd-100 only":           false,
		"itd-10-alpha is a slug": false,
		"xitd-10":                false,
	} {
		if got := Names(text, "", "itd-10"); got != want {
			t.Errorf("Names(%q) = %v, want %v", text, got, want)
		}
	}
	if Names("anything", "") {
		t.Error("an empty id names nothing")
	}
}
