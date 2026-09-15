package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/provenance"
)

// TestSpecNumIgnoresOverflow proves an over-int64 spec number carries no usable
// number (0), not the clamped MaxInt64: with the clamp, every such spelling
// would compare equal under SameNum and Lookup would hand back a spec that no
// reference names.
func TestSpecNumIgnoresOverflow(t *testing.T) {
	if n := specNum("spc-99999999999999999999999"); n != 0 {
		t.Errorf("specNum(over-int64) = %d, want 0 (an unreal number names nothing)", n)
	}
	if SameNum("spc-99999999999999999999999", "spc-99999999999999999999998") {
		t.Error("two over-int64 spellings must not compare equal through the clamp")
	}
	if n := specNum("spc-7-a-slug"); n != 7 {
		t.Errorf("specNum(spc-7-a-slug) = %d, want 7", n)
	}
}

// writeFile writes content to root/rel, creating parent directories. Shared by
// both test files in this package.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestBodyIsStubLockstep proves the stub detector and the minted template can
// never drift: a freshly rendered spec body is a stub, and a body whose
// placeholder was replaced with real content is not.
func TestBodyIsStubLockstep(t *testing.T) {
	minted := renderSpec("spc-1", "a-slug", "itd-9", mustStamp(t))
	if !BodyIsStub(minted) {
		t.Errorf("BodyIsStub(renderSpec(...)) = false, want true (template and detector drifted)")
	}
	written := "---\nid: spc-1\nslug: a-slug\nintent: itd-9\n---\n# a-slug\n\n## Summary\n\nA real design record: scope, approach, AC mapping.\n"
	if BodyIsStub(written) {
		t.Errorf("BodyIsStub(written body) = true, want false")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		wantErr bool
	}{
		{"good", Spec{ID: "spc-1", Slug: "thing", Intent: "itd-9"}, false},
		{"bad id", Spec{ID: "spec-1", Intent: "itd-9"}, true},
		{"empty id", Spec{ID: "", Intent: "itd-9"}, true},
		{"bad intent", Spec{ID: "spc-1", Intent: "itd-x"}, true},
		{"empty intent", Spec{ID: "spc-1", Intent: ""}, true},
		{"traversal id", Spec{ID: "spc-../../etc", Intent: "itd-9"}, true},
		{"traversal intent", Spec{ID: "spc-1", Intent: "itd-../../etc"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(%+v) err = %v, wantErr %v", tt.spec, err, tt.wantErr)
			}
		})
	}
}

func TestStoreLookupAndByIntent(t *testing.T) {
	store := Store{Specs: []Spec{
		{ID: "spc-1", Slug: "a", Intent: "itd-9", Status: StatusOpen},
		{ID: "spc-2", Slug: "b", Intent: "itd-12", Status: StatusClosed},
	}}

	if s, ok := store.Lookup("spc-2"); !ok || s.Intent != "itd-12" {
		t.Fatalf("Lookup(spc-2) = %+v, %v", s, ok)
	}
	if _, ok := store.Lookup("spc-99"); ok {
		t.Fatal("Lookup(spc-99) unexpectedly found")
	}
	if s, ok := store.ByIntent("itd-9"); !ok || s.ID != "spc-1" {
		t.Fatalf("ByIntent(itd-9) = %+v, %v", s, ok)
	}
	if _, ok := store.ByIntent("itd-77"); ok {
		t.Fatal("ByIntent(itd-77) unexpectedly found")
	}
}

// TestLookupResolvesBySpecNumber proves the store yields to the record lint: a
// spec_id is written bare, zero-padded, and with its slug across the corpus, and
// record-lint compares the NUMBER, so a literal-only Lookup would refuse a
// lint-green record. An exact string match still wins when the store holds one.
func TestLookupResolvesBySpecNumber(t *testing.T) {
	store := Store{Specs: []Spec{
		{ID: "spc-9", Intent: "itd-1"},
		{ID: "spc-10", Intent: "itd-2"},
	}}
	for _, ref := range []string{"spc-9", "spc-9-widget", "spc-009"} {
		sp, ok := store.Lookup(ref)
		if !ok || sp.ID != "spc-9" {
			t.Errorf("Lookup(%q) = %+v, %v; want spc-9, true", ref, sp, ok)
		}
	}
	for _, ref := range []string{"spc-11", "spc-", "null", ""} {
		if sp, ok := store.Lookup(ref); ok {
			t.Errorf("Lookup(%q) = %+v, true; want no match", ref, sp)
		}
	}
	// A store carrying both spellings resolves to the record the caller named.
	both := Store{Specs: []Spec{{ID: "spc-009", Intent: "itd-1"}, {ID: "spc-9", Intent: "itd-2"}}}
	if sp, ok := both.Lookup("spc-9"); !ok || sp.ID != "spc-9" {
		t.Errorf("Lookup(spc-9) = %+v, %v; want the exact-match record", sp, ok)
	}
}

// mustStamp is the default disclosure pair, built through the one constructor.
func mustStamp(t *testing.T) provenance.Stamp {
	t.Helper()
	s, err := provenance.NewStamp(provenance.KindResearcherAuthored, "")
	if err != nil {
		t.Fatalf("NewStamp: %v", err)
	}
	return s
}
