package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/releasedigest"
	"github.com/intentdriven/abcd/internal/gittest"
)

// digestRel is where the fixtures declare their digest.
const digestRel = "internal/rd/digest.json"

// existingDigest is a digest already holding the base release.
const existingDigest = `{
  "schema_version": 1,
  "releases": [
    {
      "tag": "v0.4.0",
      "base_tag": "v0.3.9",
      "entries": [
        {
          "line": "the base.",
          "records": [
            {
              "id": "iss-1",
              "impact": "fix"
            }
          ]
        }
      ]
    }
  ]
}
`

// digestRepo is shippableRepo declaring a release digest at digestRel, with
// the digest already holding the base release unless digest is "".
func digestRepo(t *testing.T, digest string) *gittest.Repo {
	t.Helper()
	r := shippableRepo(t)
	r.Write(".abcd/config.json", `{"release":{"digest":"`+digestRel+`"}}`+"\n")
	if digest != "" {
		r.Write(digestRel, digest)
	} else {
		r.Write("internal/rd/README.md", "the digest's directory\n")
	}
	r.Commit("declare a release digest")
	return r
}

func readDigest(t *testing.T, root string) *releasedigest.Digest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(digestRel)))
	if err != nil {
		t.Fatalf("reading the digest: %v", err)
	}
	d, err := releasedigest.Parse(data)
	if err != nil {
		t.Fatalf("the written digest does not parse: %v", err)
	}
	return d
}

// TestCutWritesTheDigestBeforeTheHeading (spc-2610170635097509 step 3): a
// repository that declares release.digest gets the cut's release appended to
// it, after the page and before the CHANGELOG heading, which stays last. The
// release is the cut's next tag, its base tag and each validated line in the
// CHANGELOG's order, with every cited record's impact.
func TestCutWritesTheDigestBeforeTheHeading(t *testing.T) {
	r := digestRepo(t, existingDigest)
	ops := newRecordingOps(r.Root())
	res, err := ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt, ops)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if !res.Written {
		t.Fatalf("nothing was written; cut refusals = %v", refusalKinds(res.Cut))
	}
	log := strings.Join(ops.log, "|")
	want := "replace " + PageFile + "|write " + digestRel + "|replace " + changelogFile
	if log != want {
		t.Fatalf("writes = %s, want %s", log, want)
	}
	d := readDigest(t, r.Root())
	if len(d.Releases) != 2 || d.Releases[1].Tag != "v0.4.0" {
		t.Fatalf("the base release was not kept: %+v", d.Releases)
	}
	got := d.Releases[0]
	if got.Tag != "v0.4.1" || got.BaseTag != "v0.4.0" {
		t.Fatalf("release = %s from %s, want v0.4.1 from v0.4.0", got.Tag, got.BaseTag)
	}
	wantEntries := []releasedigest.Entry{
		{Line: "**A version is a fact.** Derived, not typed.", Records: []releasedigest.Record{{ID: "itd-73", Impact: releasedigest.ImpactAdditive}}},
		{Line: "**The crash is gone.** It no longer crashes.", Records: []releasedigest.Record{{ID: "iss-51", Impact: releasedigest.ImpactFix}}},
	}
	if len(got.Entries) != len(wantEntries) {
		t.Fatalf("entries = %+v", got.Entries)
	}
	for i, e := range got.Entries {
		if e.Line != wantEntries[i].Line || len(e.Records) != 1 || e.Records[0] != wantEntries[i].Records[0] {
			t.Errorf("entry %d = %+v, want %+v", i, e, wantEntries[i])
		}
		if line := "- " + e.Line + " (" + e.Records[0].ID + ")"; !strings.Contains(readChangelog(t, r.Root()), line+"\n") {
			t.Errorf("digest line %q is not a CHANGELOG line", line)
		}
	}
}

// TestCutCreatesTheFirstDigest: a declared digest not yet written is created
// with the cut's release alone.
func TestCutCreatesTheFirstDigest(t *testing.T) {
	r := digestRepo(t, "")
	if _, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if d := readDigest(t, r.Root()); len(d.Releases) != 1 || d.Releases[0].Tag != "v0.4.1" {
		t.Fatalf("releases = %+v", d.Releases)
	}
}

// TestNoDeclaredDigestWritesNone: a repository that declares no digest gets
// none, and the cut's writes are what they were.
func TestNoDeclaredDigestWritesNone(t *testing.T) {
	r := shippableRepo(t)
	ops := newRecordingOps(r.Root())
	if _, err := ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt, ops); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if strings.Contains(strings.Join(ops.log, "|"), "write ") {
		t.Fatalf("a digest was written with none declared: %v", ops.log)
	}
}

// TestDigestRollsBack: a failure at the digest or after it puts the digest
// back as it was (removed when the cut created it), and the undo the ship
// verb applies restores it too.
func TestDigestRollsBack(t *testing.T) {
	for _, tc := range []struct{ name, digest, fail string }{
		{"the heading fails after an append", existingDigest, "replace " + changelogFile},
		{"the heading fails after a first digest", "", "replace " + changelogFile},
		{"the digest write fails", existingDigest, "write " + digestRel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := digestRepo(t, tc.digest)
			before := treeDigest(t, r.Root())
			_, err := ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt, newRecordingOps(r.Root(), tc.fail))
			if err == nil || !strings.Contains(err.Error(), "rolled back") {
				t.Fatalf("err = %v, want a rolled-back failure", err)
			}
			if after := treeDigest(t, r.Root()); after != before {
				t.Error("a failed cut left the tree changed")
			}
		})
	}
	t.Run("the ship verb's undo", func(t *testing.T) {
		r := digestRepo(t, existingDigest)
		before := treeDigest(t, r.Root())
		res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
		if err != nil {
			t.Fatalf("Ingest: %v", err)
		}
		if failures := res.Undo.Apply(r.Root()); len(failures) > 0 {
			t.Fatalf("undo failed: %v", failures)
		}
		if after := treeDigest(t, r.Root()); after != before {
			t.Error("the undo did not restore the digest")
		}
	})
}

// TestDigestPathIsRefusedBeforeAnyWrite: a declared path outside the tree,
// under .abcd/ or .git/, not ending in .json, or whose directory is not in
// the tree, a key the release namespace does not have, and a digest that
// does not parse each stop the cut before anything is written.
func TestDigestPathIsRefusedBeforeAnyWrite(t *testing.T) {
	outside := t.TempDir()
	for _, tc := range []struct{ name, config, setup string }{
		{"a parent step", `{"release":{"digest":"../digest.json"}}`, ""},
		{"an absolute path", `{"release":{"digest":"/tmp/digest.json"}}`, ""},
		{"a dot segment", `{"release":{"digest":"internal/./rd/digest.json"}}`, ""},
		{"a backslash", `{"release":{"digest":"internal\\rd\\digest.json"}}`, ""},
		{"under .abcd", `{"release":{"digest":".abcd/digest.json"}}`, ""},
		{"under .abcd in another case", `{"release":{"digest":".ABCD/digest.json"}}`, ""},
		{"under .git", `{"release":{"digest":".git/digest.json"}}`, ""},
		{"not JSON", `{"release":{"digest":"internal/rd/digest.txt"}}`, ""},
		{"empty", `{"release":{"digest":""}}`, ""},
		{"not a string", `{"release":{"digest":true}}`, ""},
		{"an unknown key", `{"release":{"digets":"internal/rd/digest.json"}}`, ""},
		{"a directory not in the tree", `{"release":{"digest":"nowhere/digest.json"}}`, ""},
		{"a directory symlinked out of the tree", `{"release":{"digest":"out/digest.json"}}`, "symlink"},
		{"a digest that does not parse", `{"release":{"digest":"` + digestRel + `"}}`, "garbage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := shippableRepo(t)
			r.Write(".abcd/config.json", tc.config+"\n")
			r.Write("internal/rd/README.md", "the digest's directory\n")
			switch tc.setup {
			case "symlink":
				if err := os.Symlink(outside, filepath.Join(r.Root(), "out")); err != nil {
					t.Fatal(err)
				}
			case "garbage":
				r.Write(digestRel, `{"schema_version":1,"releases":[],"extra":true}`)
			}
			r.Commit("a bad digest declaration")
			// The tree digest cannot walk a symlinked directory; that case is
			// held by the writes logged and the outside directory left empty.
			var before string
			if tc.setup != "symlink" {
				before = treeDigest(t, r.Root())
			}
			ops := newRecordingOps(r.Root())
			res, err := ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt, ops)
			if err == nil {
				t.Fatalf("the cut went ahead (written=%v, writes %v)", res.Written, ops.log)
			}
			if len(ops.log) > 0 {
				t.Errorf("writes were made before the refusal: %v", ops.log)
			}
			if tc.setup != "symlink" {
				if after := treeDigest(t, r.Root()); after != before {
					t.Error("the refused cut changed the tree")
				}
			}
			if entries, _ := os.ReadDir(outside); len(entries) > 0 {
				t.Error("a file was written outside the tree")
			}
		})
	}
}

// TestDigestIsReadFromTheRepositoryLayerOnly: a machine's own
// ~/.abcd.noindex/config.json declaring release.digest is never read, so a
// repository that declares none gets none.
func TestDigestIsReadFromTheRepositoryLayerOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".abcd.noindex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".abcd.noindex", "config.json"),
		[]byte(`{"release":{"digest":"`+digestRel+`"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := shippableRepo(t)
	r.Write("internal/rd/README.md", "the digest's directory\n")
	r.Commit("a directory where a machine-declared digest would go")
	if got, err := declaredDigest(r.Root()); err != nil || got != "" {
		t.Fatalf("declaredDigest = %q, %v; want none: the machine layer is never read", got, err)
	}
}

// TestAbcdDeclaresItsDigest: abcd's own configuration declares the digest
// its binary embeds.
func TestAbcdDeclaresItsDigest(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	got, err := declaredDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "internal/core/releasedigest/digest.json" {
		t.Fatalf("abcd declares release.digest = %q, want internal/core/releasedigest/digest.json", got)
	}
}
