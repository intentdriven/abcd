package loop

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// mutate is saveRun as a test calls it, judged as a session with production's options
// judges a save. Production code has no frame of this shape, so every save it makes
// names the options it is judged by.
func mutate(repoRoot, runID string, fn func(root *os.Root, st *State) (bool, error)) error {
	return saveRun(repoRoot, runID, Options{}, fn)
}

// writeState is storeState as a test seeds a run's file with: a new file, or one at
// this abcd's version, which takes no right (wholeSave.covers).
func writeState(root *os.Root, st State) error {
	return storeState(root, st, wholeSave{})
}

// TestTheJudgeOfAWholeSave: a save of a run's whole state over a file that states
// another version is made only by the live session its control record names as its
// driver, when that session is this one. Every other reading leaves the file's bytes
// as they were, runs nothing of the save, and says why, with a way forward.
func TestTheJudgeOfAWholeSave(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version int
		set     func(t *testing.T, repoRoot, runID string, o *Options)
		saves   bool
		why     string
		remedy  string
	}{
		{name: "this abcd's version, by a session that is not its driver", version: SchemaVersion,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostTwo, hostOne, hostOne, hostTwo)
			}, saves: true},
		{name: "this session drives it", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostOne, hostOne, hostOne, hostTwo)
			}, saves: true},
		{name: "no driver is recorded", version: schemaVersionPrevious,
			set: func(t *testing.T, _, _ string, _ *Options) { procsOf(t, hostOne, hostOne) },
			why: "its control record names no driver", remedy: "`abcd implement resume --take "},
		{name: "the control record cannot be read", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, _ *Options) {
				procsOf(t, hostOne, hostOne)
				if err := os.WriteFile(filepath.Join(repoRoot, filepath.FromSlash(controlRel(runID))), []byte("{not json"), 0o600); err != nil {
					t.Fatal(err)
				}
			}, why: "its control record cannot be read", remedy: "restore " + RunRelDir},
		{name: "the process table cannot be read", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostOne, hostOne, hostOne)
				readProcs = func() (sessionname.Table, error) { return sessionname.Table{}, errors.New("no table") }
			}, why: "the process table cannot be read", remedy: "once the process table can be read"},
		{name: "the session this call runs under cannot be found", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostOne, hostOne, hostOne)
				hereFrom = func() int { return 77777 }
			}, why: "the session this call runs under cannot be found", remedy: "run it from the session that is to drive the run"},
		{name: "its driver's process id now names another process", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostOne, hostOne, hostOne)
				root, err := os.OpenRoot(repoRoot)
				if err != nil {
					t.Fatal(err)
				}
				defer root.Close()
				c := controlOf(t, repoRoot, runID)
				c.Driver.StartUnix -= 3600
				if err := writeControl(root, runID, c); err != nil {
					t.Fatal(err)
				}
			}, why: "is gone, and this session has not claimed it", remedy: "`abcd implement step --run "},
		{name: "another live session drives it", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostTwo, hostOne, hostOne, hostTwo)
			}, why: "which is alive", remedy: "back off: that session steps it on"},
		{name: "its driver is gone", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostTwo, hostOne, hostOne)
			}, why: "is gone, and this session has not claimed it", remedy: "`abcd implement resume --take "},
		{name: "a renewal, under its driver's own session", version: schemaVersionPrevious,
			set: func(t *testing.T, repoRoot, runID string, o *Options) {
				claimedBy(t, repoRoot, runID, *o, hostOne, hostOne, hostOne)
				o.afar = true
			}, why: "a renewal may run a lane's own abcd", remedy: "the run's driver saves it"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			o := Options{stopHome: home}
			tc.set(t, repo.Root(), runID, &o)
			var before []byte
			if tc.version == SchemaVersion {
				before = readStateBytes(t, repo.Root(), runID)
			} else {
				before = stateAtVersion(t, repo.Root(), runID, tc.version)
			}
			ran := false

			err := saveRun(repo.Root(), runID, o, func(_ *os.Root, st *State) (bool, error) {
				ran = true
				st.Record = append(st.Record, Entry{Stage: StageDriver, Note: "a whole save"})
				return true, nil
			})

			if tc.saves {
				if err != nil {
					t.Fatalf("the save was refused: %v", err)
				}
				if v := stateVersionOf(t, repo.Root(), runID); v != strconv.Itoa(SchemaVersion) {
					t.Fatalf("the save wrote version %s, not %d", v, SchemaVersion)
				}
				return
			}
			r, ok := leftAtVersion(err)
			if !ok {
				t.Fatalf("want the refusal that leaves the run at its version, got %v", err)
			}
			if ran {
				t.Fatal("the save ran over a run it left")
			}
			if !r.Contention || r.Stage != StageDriver {
				t.Fatalf("the refusal is not the driver's, at exit 3: %+v", r)
			}
			wantLeft(t, repo.Root(), runID, tc.version, before, r.Error())
			if !strings.Contains(r.Reason, tc.why) || !strings.Contains(r.Remedy, tc.remedy) {
				t.Fatalf("the refusal does not say %q with %q:\n%s", tc.why, tc.remedy, r.Error())
			}
		})
	}
}

// readStateBytes is the run's state file as it stands.
func readStateBytes(t *testing.T, repoRoot, runID string) []byte {
	t.Helper()
	data, err := os.ReadFile(runStatePath(repoRoot, runID))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestTheStateFilesWriterRefusesAnotherVersionWithoutItsRight: storeState itself
// refuses to replace a file that states another version unless it is handed the right
// mayWholeSave minted for that run at that version, so a save that skipped the judge
// writes nothing.
func TestTheStateFilesWriterRefusesAnotherVersionWithoutItsRight(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, repo.Root(), runID, o, hostOne, hostOne, hostOne)
	before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	granted, err := mayWholeSave(root, runID, schemaVersionPrevious, o)
	if err != nil {
		t.Fatalf("premise: this session drives the run: %v", err)
	}
	other, err := mayWholeSave(root, runID, SchemaVersion, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		right wholeSave
	}{
		{"no right", wholeSave{}},
		{"a right for this abcd's version", other},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := storeState(root, st, tc.right)
			r, ok := leftAtVersion(err)
			if !ok {
				t.Fatalf("want the refusal that leaves the run at its version, got %v", err)
			}
			wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, r.Error())
		})
	}
	if err := storeState(root, st, granted); err != nil {
		t.Fatalf("the driver's right was refused: %v", err)
	}
	if v := stateVersionOf(t, repo.Root(), runID); v != strconv.Itoa(SchemaVersion) {
		t.Fatalf("the driver's save wrote version %s, not %d", v, SchemaVersion)
	}
}

// stateWriters are the functions of this package allowed to write a run's state file:
// storeState, the one whole writer, and saveRecord, which keeps the version the file
// states.
var stateWriters = []string{"storeState", "saveRecord"}

// writeCalls are the calls that write or replace a file.
var writeCalls = []string{"WriteFileAtomicInRoot", "WriteFileAtomic", "WriteFileAtomicPreserveMode", "WriteFile", "Rename", "Create", "OpenFile"}

// TestEveryWholeSaveGoesThroughTheStateFilesWriter: no function in this package but
// the state file's two writers both names the state file's path and writes a file, and
// no function but mayWholeSave mints the right storeState asks for, so a whole save
// added later cannot replace another version's file without the judge.
func TestEveryWholeSaveGoesThroughTheStateFilesWriter(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var found []string
	for _, p := range files {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			names, writes := false, false
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.Ident:
					if n.Name == "StateRelPath" || n.Name == "StateFileName" {
						names = true
					}
				case *ast.BasicLit:
					if n.Kind == token.STRING && strings.Contains(n.Value, StateFileName) {
						names = true
					}
				case *ast.SelectorExpr:
					if slices.Contains(writeCalls, n.Sel.Name) {
						writes = true
					}
				case *ast.CompositeLit:
					if id, ok := n.Type.(*ast.Ident); ok && id.Name == "wholeSave" && len(n.Elts) > 0 && fd.Name.Name != "mayWholeSave" {
						found = append(found, fmt.Sprintf("%s:%d: %s mints the right to a whole save, which only mayWholeSave may", p, fset.Position(n.Pos()).Line, fd.Name.Name))
					}
				}
				return true
			})
			if names && writes && !slices.Contains(stateWriters, fd.Name.Name) {
				found = append(found, fmt.Sprintf("%s:%d: %s writes beside the run's state path; save through saveRun (or saveRecord for a line in the record)", p, fset.Position(fd.Pos()).Line, fd.Name.Name))
			}
		}
	}
	if len(found) > 0 {
		t.Fatalf("a run's state file is written outside its writers:\n%s", strings.Join(found, "\n"))
	}
}

// claimAsDriver claims the run for this test's process, as a step's front door claims
// before it steps (ClaimDriver), so the test's own saves of a run whose file states
// another version are its driver's, as a driver resumed after an upgrade saves its run.
func claimAsDriver(t *testing.T, repoRoot, runID string, o Options) {
	t.Helper()
	if err := ClaimDriver(repoRoot, runID, o); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repoRoot, runID); c.Driver == nil {
		t.Fatal("premise: the claim names this test's process as the run's driver")
	}
}
