package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// A bundle's shared spec names every member: `intent:` keeps the first, the
// `intents:` list carries all of them, and `bundle:` names the bundle the close
// ships together (itd-34, spec scope 1).
func TestCreateBundleNamesEveryMember(t *testing.T) {
	root := t.TempDir()
	sp, err := CreateBundle(root, []string{"itd-10", "itd-11"}, "pair", "")
	if err != nil {
		t.Fatal(err)
	}
	if sp.Intent != "itd-10" || sp.Bundle != "pair" || strings.Join(sp.Intents, ",") != "itd-10,itd-11" {
		t.Fatalf("CreateBundle = %+v", sp)
	}
	body, err := os.ReadFile(filepath.Join(root, sp.Path))
	if err != nil {
		t.Fatal(err)
	}
	f := frontmatter.Fields(strings.Split(string(body), "\n"))
	if f["intent"].Value != "itd-10" || f["intents"].Value != "[itd-10, itd-11]" || f["bundle"].Value != "pair" {
		t.Fatalf("bundle spec frontmatter wrong:\n%s", body)
	}

	store, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := store.Lookup(sp.ID)
	if !ok || got.Bundle != "pair" || len(got.Intents) != 2 {
		t.Fatalf("Load did not read the bundle back: %+v", got)
	}
	// The second member is realised by the shared spec as much as the first is.
	if specs := store.SpecsForIntent("itd-11"); len(specs) != 1 || specs[0].ID != sp.ID {
		t.Fatalf("SpecsForIntent(itd-11) = %+v, want the shared spec", specs)
	}
	if claimer, ok := store.ByIntent("itd-11"); !ok || claimer.ID != sp.ID {
		t.Fatalf("ByIntent(itd-11) = %+v, %v", claimer, ok)
	}
	if !got.Names("itd-011") || got.Names("itd-12") {
		t.Fatalf("Names must match every member canonically and nothing else")
	}
}

// A bundle has at least two members, each named once, and a kebab-case name.
func TestCreateBundleRefusesMalformedRequests(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name    string
		members []string
		bundle  string
	}{
		{"one member", []string{"itd-10"}, "pair"},
		{"repeated member", []string{"itd-10", "itd-010"}, "pair"},
		{"bad member id", []string{"itd-10", "../x"}, "pair"},
		{"bad bundle name", []string{"itd-10", "itd-11"}, "Pair Two"},
	} {
		if _, err := CreateBundle(root, tc.members, tc.bundle, ""); err == nil {
			t.Errorf("%s: CreateBundle must refuse", tc.name)
		}
	}
	if entries, _ := os.ReadDir(filepath.Join(root, specsOpen)); len(entries) != 0 {
		t.Fatalf("a refused bundle must mint nothing, found %d file(s)", len(entries))
	}
}

// Members lists every intent a spec realises, `intent:` first, each once: an
// `intents:` list that omits the back-link still has it counted, and a repeat
// in another spelling counts once.
func TestSpecMembersListsEachMemberOnceBackLinkFirst(t *testing.T) {
	for _, c := range []struct {
		sp   Spec
		want string
	}{
		{Spec{Intent: "itd-10"}, "itd-10"},
		{Spec{Intent: "itd-10", Intents: []string{"itd-10", "itd-11"}}, "itd-10,itd-11"},
		{Spec{Intent: "itd-10", Intents: []string{"itd-11", "itd-010", "itd-11"}}, "itd-10,itd-11"},
	} {
		if got := strings.Join(c.sp.Members(), ","); got != c.want {
			t.Errorf("Members(%+v) = %s, want %s", c.sp, got, c.want)
		}
	}
}
