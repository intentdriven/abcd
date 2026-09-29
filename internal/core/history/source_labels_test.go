package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImportedTranscriptNamesItsToolAndItsRoute is ruling J13 on
// iss-2608230752354928: a stored transcript carries two separate labels, the
// tool that produced it and the route it reached the store by. A transcript
// exported by another tool and imported is stored naming that tool AND the
// import route, and both survive a read back — in the listing and in the
// record read by session.
func TestImportedTranscriptNamesItsToolAndItsRoute(t *testing.T) {
	repoRoot, _ := setupStore(t)

	res, err := Capture(repoRoot, testRootSHA, []byte("user: exported elsewhere\n"),
		CaptureMeta{SessionID: "sess-imported", Kind: RouteImport, Tool: "other-harness"})
	if err != nil {
		t.Fatalf("Capture of an imported transcript: %v", err)
	}
	if res.Record.SourceKind != RouteImport || res.Record.SourceTool != "other-harness" {
		t.Fatalf("stored labels = route %q tool %q, want import / other-harness",
			res.Record.SourceKind, res.Record.SourceTool)
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"\nsource_kind: import\n", "\nsource_tool: other-harness\n"} {
		if !strings.Contains(string(onDisk), line) {
			t.Errorf("record frontmatter lacks %q:\n%s", strings.TrimSpace(line), onDisk)
		}
	}

	rec, _, err := Read(repoRoot, testRootSHA, "sess-imported")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if rec.SourceKind != RouteImport || rec.SourceTool != "other-harness" {
		t.Errorf("read back route %q tool %q, want import / other-harness", rec.SourceKind, rec.SourceTool)
	}
	recs, err := List(repoRoot, testRootSHA)
	if err != nil || len(recs) != 1 {
		t.Fatalf("List = %d records, err %v", len(recs), err)
	}
	if recs[0].SourceKind != RouteImport || recs[0].SourceTool != "other-harness" {
		t.Errorf("listed route %q tool %q, want import / other-harness", recs[0].SourceKind, recs[0].SourceTool)
	}
}

// TestNativeCaptureNamesTheHost pins the native route: abcd's own capture of the
// transcript the host harness wrote. A caller that names no tool is recording
// the host, and the record says so rather than leaving the tool blank.
func TestNativeCaptureNamesTheHost(t *testing.T) {
	repoRoot, _ := setupStore(t)

	res, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"),
		CaptureMeta{SessionID: "sess-native", Kind: RouteNative})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if res.Record.SourceKind != RouteNative || res.Record.SourceTool != ToolHost {
		t.Errorf("native capture labels = route %q tool %q, want native / host",
			res.Record.SourceKind, res.Record.SourceTool)
	}
}

// TestSourceLabelsCannotBeForgedIntoEachOther holds the two labels apart. The
// route vocabulary is closed, so a tool name cannot stand in the route; a route
// word (or the pre-J13 composite that fused the two) cannot stand in the tool;
// and an import cannot claim the host as its tool, because the host is what the
// native route records. Every refusal writes nothing.
func TestSourceLabelsCannotBeForgedIntoEachOther(t *testing.T) {
	repoRoot, home := setupStore(t)

	for _, tc := range []struct {
		name, route, tool string
	}{
		{"a tool name in the route", "other-harness", ""},
		{"the host in the route", ToolHost, ""},
		{"a route word in the tool (native)", RouteNative, RouteNative},
		{"a route word in the tool (import)", RouteImport, RouteImport},
		{"the fused pre-J13 value in the tool", RouteImport, "specstory-import"},
		{"any -import composite in the tool", RouteImport, "other-import"},
		{"an import claiming the host", RouteImport, ToolHost},
		{"an import naming no tool", RouteImport, ""},
		{"a tool outside the slug charset", RouteImport, "Other Harness"},
		{"a tool carrying a line break", RouteImport, "other\nsource_kind: native"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Capture(repoRoot, testRootSHA, []byte("user: forged\n"),
				CaptureMeta{SessionID: "sess-forged", Kind: tc.route, Tool: tc.tool})
			if err == nil {
				t.Fatalf("route %q tool %q was accepted", tc.route, tc.tool)
			}
		})
	}
	entries, err := os.ReadDir(filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("a refused capture wrote %s", e.Name())
		}
	}
}

// TestPreJ13RecordsReadBothLabels is the old-vintage read. A record written
// before the split carries source_kind alone, and it keeps reading: the two
// labels are derived from it on read, without rewriting the record. `native`
// was always abcd's own capture of the host's transcript; `specstory-import`
// fused the import route with the tool that exported it.
func TestPreJ13RecordsReadBothLabels(t *testing.T) {
	repoRoot, home := setupStore(t)

	old := func(session, kind string) string {
		return strings.Join([]string{
			"---",
			"schema: 3",
			"session_id: " + session,
			"root_commit: " + testRootSHA,
			"captured_at: 2026-01-01T00:00:00Z",
			"source_kind: " + kind,
			"source_sha256: " + strings.Repeat("0", 64),
			"redacted_secrets: 0",
			"redacted_home_paths: 0",
			"---",
			"user: an older binary wrote this",
			"",
		}, "\n")
	}
	nativePath := planted(t, home, "20260101T000000.000000000Z-sess-oldnative.md", old("sess-oldnative", "native"))
	planted(t, home, "20260101T000001.000000000Z-sess-oldimport.md", old("sess-oldimport", "specstory-import"))
	before, err := os.ReadFile(nativePath)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ session, route, tool string }{
		{"sess-oldnative", RouteNative, ToolHost},
		{"sess-oldimport", RouteImport, "specstory"},
	} {
		rec, _, err := Read(repoRoot, testRootSHA, tc.session)
		if err != nil {
			t.Fatalf("Read %s: %v", tc.session, err)
		}
		if rec.SourceKind != tc.route || rec.SourceTool != tc.tool {
			t.Errorf("%s reads as route %q tool %q, want %q / %q",
				tc.session, rec.SourceKind, rec.SourceTool, tc.route, tc.tool)
		}
	}
	after, err := os.ReadFile(nativePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("reading an old record rewrote it; only migrate writes the store's existing records")
	}
}

// TestLegacyFusedKindStillCaptures keeps the pre-J13 spelling working on the
// write side: a caller still passing specstory-import is recorded under the two
// labels it always meant, never under the fused value.
func TestLegacyFusedKindStillCaptures(t *testing.T) {
	repoRoot, _ := setupStore(t)

	res, err := Capture(repoRoot, testRootSHA, []byte("user: legacy spelling\n"),
		CaptureMeta{SessionID: "sess-legacy", Kind: "specstory-import"})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if res.Record.SourceKind != RouteImport || res.Record.SourceTool != "specstory" {
		t.Errorf("legacy kind stored as route %q tool %q, want import / specstory",
			res.Record.SourceKind, res.Record.SourceTool)
	}
}

// TestSourceToolIsScannedWithTheBody extends the frontmatter redaction guard to
// the tool label: it is caller-supplied and open-ended, so it passes the same
// scan as every other scalar. A tool name redaction would change is not a tool
// name, and the capture is refused rather than stored under a masked label.
func TestSourceToolIsScannedWithTheBody(t *testing.T) {
	repoRoot, home := setupStore(t)
	cfgDir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"patterns":{"leaky":{"regex":"zzleaktool[0-9]+","kind":"token","label":"leaky tool","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"),
		CaptureMeta{SessionID: "sess-leaky", Kind: RouteImport, Tool: "zzleaktool42"})
	if err == nil {
		t.Fatal("a tool label the scanner redacts was stored")
	}
	entries, _ := os.ReadDir(filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("the refused capture wrote %s", e.Name())
		}
	}
}

// TestStagingHoldsOnlyTheHostsOwnTranscripts: the staging area is the native
// hook's half of capture, and its sidecar and its drain carry no source labels
// — every staged transcript drains as native from the host. A stage naming any
// other route or tool would be relabelled on the way through, so it is refused
// at the door rather than stored under labels its caller did not give.
func TestStagingHoldsOnlyTheHostsOwnTranscripts(t *testing.T) {
	repoRoot, _ := setupStore(t)

	for _, m := range []CaptureMeta{
		{SessionID: "sess-stage-imp", Kind: RouteImport, Tool: "other-harness"},
		{SessionID: "sess-stage-tool", Kind: RouteNative, Tool: "other-harness"},
	} {
		if _, err := Stage(repoRoot, testRootSHA, StageMeta{Lineage: m}, []byte("user: hi\n")); err == nil {
			t.Errorf("staging accepted route %q tool %q, which the drain would store as native from host", m.Kind, m.Tool)
		}
	}
	if _, err := Stage(repoRoot, testRootSHA, StageMeta{Lineage: CaptureMeta{SessionID: "sess-stage-host", Tool: ToolHost}}, []byte("user: hi\n")); err != nil {
		t.Errorf("staging refused the host's own transcript: %v", err)
	}
}
