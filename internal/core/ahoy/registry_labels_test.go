package ahoy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestRegistryLabelsMapsRootCommitsToNames: each registered root commit maps
// to its name and path label; an entry without a root commit or a name is left
// out, and the first named entry for a root commit wins.
func TestRegistryLabelsMapsRootCommitsToNames(t *testing.T) {
	home, _ := setupHermetic(t)
	root := abcdhome.Path(home, "history")
	if err := os.MkdirAll(root, abcdhome.DirMode); err != nil {
		t.Fatal(err)
	}
	const a, b, c = "0000000000000000000000000000000000000001", "0000000000000000000000000000000000000002", "0000000000000000000000000000000000000003"
	idx := historyIndex{Schema: 1, Description: historyIndexDescription, Repos: []historyRepo{
		{RootCommit: a, Name: "alpha", Path: "/gone/alpha", Status: "active"},
		{RootCommit: a, Name: "alpha-again", Status: "superseded"},
		{RootCommit: b, Name: "", Status: "active"},
		{RootCommit: "", Name: "keyless", Status: "active"},
		{RootCommit: c, Name: "gamma", Status: "active"},
	}}
	if err := writeJSON(filepath.Join(root, "index.json"), idx); err != nil {
		t.Fatal(err)
	}
	got, err := RegistryLabels()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]RegistryLabel{a: {Name: "alpha", Path: "/gone/alpha"}, c: {Name: "gamma"}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s: got %+v, want %+v", k, got[k], v)
		}
	}
}

// TestRegistryLabelsAbsentOrUnreadable: an absent registry is an empty map and
// no error; a malformed one is an empty map and the error, never a panic or a
// nil map a caller would index.
func TestRegistryLabelsAbsentOrUnreadable(t *testing.T) {
	home, _ := setupHermetic(t)
	got, err := RegistryLabels()
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("absent: got %v, %v; want an empty map and no error", got, err)
	}
	root := abcdhome.Path(home, "history")
	if err := os.MkdirAll(root, abcdhome.DirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.json"), []byte(`{"repos": [`), abcdhome.FileMode); err != nil {
		t.Fatal(err)
	}
	got, err = RegistryLabels()
	if err == nil || got == nil || len(got) != 0 {
		t.Fatalf("malformed: got %v, %v; want an empty map and the error", got, err)
	}
}
