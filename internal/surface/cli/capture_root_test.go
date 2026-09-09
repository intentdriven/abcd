package cli

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// The ledger a capture verb addresses is the CHECKOUT's ledger, wherever in the
// checkout the caller happens to stand. These are iss-2609090951291524's
// detectors: the front doors handed the working directory to the core as the
// repo root, verbatim, so a verb run from a subdirectory read an empty ledger
// and a write minted a second one under that subdirectory — silently in both
// directions, and out of reach of every gate that reads the real ledger.

// captureLedgerFixture builds a git working tree with a seeded ledger and a
// package-shaped subdirectory two levels down, and returns both plus the ids it
// seeded. HOME is redirected so nothing consults the developer's own home.
func captureLedgerFixture(t *testing.T, n int) (repo, sub string, ids []string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	sub = filepath.Join(repo, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	for i := 0; i < n; i++ {
		var res struct {
			ID string `json:"id"`
		}
		out := runCLI(t, "capture", "a seeded observation number "+string(rune('a'+i))+" for the root ledger", "--json")
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("seed capture %d: not JSON: %v\n%s", i, err, out)
		}
		ids = append(ids, res.ID)
	}
	return repo, sub, ids
}

// openCount reads the bare board's open count from wherever the process stands.
func openCount(t *testing.T) int {
	t.Helper()
	var board struct {
		OpenCount int `json:"open_count"`
	}
	out := runCLI(t, "capture", "--json")
	if err := json.Unmarshal(out, &board); err != nil {
		t.Fatalf("capture --json: not JSON: %v\n%s", err, out)
	}
	return board.OpenCount
}

// noStrayLedger fails when a ledger skeleton exists anywhere under dir.
func noStrayLedger(t *testing.T, verb, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".abcd")); err == nil {
		t.Errorf("`capture %s` minted a second ledger at %s/.abcd — a record filed there is invisible to every gate that reads the checkout's ledger", verb, dir)
	}
}

// TestCaptureStatusFromSubdirectoryReadsTheCheckoutLedger is the headline
// reproduction, asserted on a real record count rather than on a bare
// non-error: the same tree, read from two directories, must report one ledger.
func TestCaptureStatusFromSubdirectoryReadsTheCheckoutLedger(t *testing.T) {
	repo, sub, ids := captureLedgerFixture(t, 3)

	fromRoot := openCount(t)
	if fromRoot != len(ids) {
		t.Fatalf("from the checkout root the board reports open %d, want %d (the fixture is wrong, not the subject)", fromRoot, len(ids))
	}
	t.Chdir(sub)
	fromSub := openCount(t)
	if fromSub != fromRoot {
		t.Fatalf("`capture` reports open %d from %s and open %d from the checkout root: the verb addressed a ledger that is not the checkout's",
			fromSub, strings.TrimPrefix(sub, repo+string(filepath.Separator)), fromRoot)
	}
}

// TestCaptureWriteFromSubdirectoryLandsInTheCheckoutLedger is the write half:
// the record must appear in the checkout's ledger, and no second ledger may
// appear under the subdirectory the caller happened to stand in.
func TestCaptureWriteFromSubdirectoryLandsInTheCheckoutLedger(t *testing.T) {
	repo, sub, ids := captureLedgerFixture(t, 2)

	t.Chdir(sub)
	var res struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	out := runCLI(t, "capture", "an observation filed from a package directory", "--json")
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("capture --json: not JSON: %v\n%s", err, out)
	}
	landed := filepath.Join(repo, filepath.FromSlash(res.Path))
	if _, err := os.Stat(landed); err != nil {
		t.Errorf("the record %s reported at %q is not in the checkout ledger: %v", res.ID, res.Path, err)
	}
	noStrayLedger(t, "<text>", sub)
	if n := len(ids) + 1; openCount(t) != n {
		t.Errorf("after a capture from the subdirectory the board reports open %d, want %d", openCount(t), n)
	}
}

// captureVerbCase is one registered capture verb plus the assertion that proves
// the ledger it addressed was the checkout's.
type captureVerbCase struct {
	args  func(ids []string, item string) []string
	check func(t *testing.T, repo string, ids []string, item string, out []byte, err error)
}

// TestEveryCaptureVerbAddressesTheCheckoutLedger table-drives the whole verb
// surface, and DERIVES the roster from the command tree rather than from a
// hand-kept list: a verb added to `capture` with no case here fails this test
// rather than slipping past it.
func TestEveryCaptureVerbAddressesTheCheckoutLedger(t *testing.T) {
	cases := map[string]captureVerbCase{
		// The bare form: the read-only board, which must count the checkout's
		// records and not the subdirectory's absence of any.
		"": {
			args: func([]string, string) []string { return []string{"capture", "--json"} },
			check: func(t *testing.T, _ string, ids []string, _ string, out []byte, err error) {
				if err != nil {
					t.Fatalf("capture: %v\n%s", err, out)
				}
				var board struct {
					OpenCount   int `json:"open_count"`
					Outstanding struct {
						Undispositioned []struct {
							ID string `json:"id"`
						} `json:"undispositioned"`
					} `json:"reading_outstanding"`
				}
				if jerr := json.Unmarshal(out, &board); jerr != nil {
					t.Fatalf("capture --json: not JSON: %v\n%s", jerr, out)
				}
				if board.OpenCount != len(ids) {
					t.Errorf("the board reports open %d from the subdirectory, want the checkout's %d", board.OpenCount, len(ids))
				}
				// The board's other half reads the ledger through its own path,
				// so it is asserted separately: one resolved root, both halves.
				if len(board.Outstanding.Undispositioned) != 1 {
					t.Errorf("the board's outstanding-readings roster names %d item(s) from the subdirectory, want the checkout's 1",
						len(board.Outstanding.Undispositioned))
				}
			},
		},
		"list": {
			args: func([]string, string) []string { return []string{"capture", "list", "--open", "--json"} },
			check: func(t *testing.T, _ string, ids []string, _ string, out []byte, err error) {
				if err != nil {
					t.Fatalf("capture list: %v\n%s", err, out)
				}
				var res struct {
					Issues []struct {
						ID string `json:"id"`
					} `json:"issues"`
				}
				if jerr := json.Unmarshal(out, &res); jerr != nil {
					t.Fatalf("capture list --json: not JSON: %v\n%s", jerr, out)
				}
				if len(res.Issues) != len(ids) {
					t.Errorf("list returns %d open issue(s) from the subdirectory, want the checkout's %d", len(res.Issues), len(ids))
				}
			},
		},
		"resolve": {
			args: func(ids []string, _ string) []string {
				return []string{"capture", "resolve", ids[0], "closed by the same change",
					"--impact", "internal", "--grounds", "pursued: the trail must stay in one ledger", "--json"}
			},
			check: func(t *testing.T, repo string, ids []string, _ string, out []byte, err error) {
				if err != nil {
					t.Fatalf("capture resolve %s from the subdirectory: %v\n%s", ids[0], err, out)
				}
				assertRecordIn(t, repo, "resolved", ids[0])
			},
		},
		"wontfix": {
			args: func(ids []string, _ string) []string {
				return []string{"capture", "wontfix", ids[1], "the behaviour is intended", "--json"}
			},
			check: func(t *testing.T, repo string, ids []string, _ string, out []byte, err error) {
				if err != nil {
					t.Fatalf("capture wontfix %s from the subdirectory: %v\n%s", ids[1], err, out)
				}
				assertRecordIn(t, repo, "wontfix", ids[1])
			},
		},
		"promote": {
			args: func(ids []string, _ string) []string {
				return []string{"capture", "promote", ids[2], "--grounds", "pursued: this observation is worth an intent of its own", "--json"}
			},
			check: func(t *testing.T, repo string, ids []string, _ string, out []byte, err error) {
				if err != nil {
					t.Fatalf("capture promote %s from the subdirectory: %v\n%s", ids[2], err, out)
				}
				body := recordBody(t, repo, "open", ids[2])
				if !strings.Contains(body, "promoted_to") {
					t.Errorf("promote from the subdirectory left the checkout's record %s unstamped:\n%s", ids[2], body)
				}
			},
		},
		"disposition": {
			args: func(_ []string, item string) []string {
				return []string{"capture", "disposition", item, "--state", "accepted",
					"--grounds", "the tension is real and worth acting on", "--json"}
			},
			check: func(t *testing.T, repo string, _ []string, item string, out []byte, err error) {
				if err != nil {
					t.Fatalf("capture disposition %s from the subdirectory: %v\n%s", item, err, out)
				}
				var res struct {
					Path string `json:"path"`
				}
				if jerr := json.Unmarshal(out, &res); jerr != nil {
					t.Fatalf("capture disposition --json: not JSON: %v\n%s", jerr, out)
				}
				if _, serr := os.Stat(filepath.Join(repo, filepath.FromSlash(res.Path))); serr != nil {
					t.Errorf("the disposition reported at %q is not in the checkout ledger: %v", res.Path, serr)
				}
			},
		},
	}

	// The roster is the command tree's own. A verb registered under `capture`
	// with no case above is a verb nothing proves resolves the checkout root.
	for _, sub := range captureSubcommandNames(t) {
		if _, ok := cases[sub]; !ok {
			t.Fatalf("`capture %s` is registered but has no ledger-identity case in this table: "+
				"every verb that reaches the ledger must be proven to address the CHECKOUT's ledger from a subdirectory", sub)
		}
	}

	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		tc := cases[name]
		label := name
		if label == "" {
			label = "(bare)"
		}
		t.Run(label, func(t *testing.T) {
			repo, sub, ids := captureLedgerFixture(t, 3)
			item := seedReadingItem(t, repo)
			t.Chdir(sub)
			out, err := runCLIErr(t, tc.args(ids, item)...)
			tc.check(t, repo, ids, item, out, err)
			noStrayLedger(t, name, sub)
		})
	}
}

// captureSubcommandNames reads the registered `capture` sub-verbs off the
// command tree, so the table above is checked against the code rather than
// against a list somebody has to remember to update.
func captureSubcommandNames(t *testing.T) []string {
	t.Helper()
	for _, c := range NewRootCommand().Commands() {
		if c.Name() != "capture" {
			continue
		}
		var out []string
		for _, sub := range c.Commands() {
			if sub.Name() == "help" || sub.Hidden {
				continue
			}
			out = append(out, sub.Name())
		}
		return out
	}
	t.Fatal("no `capture` command is registered on the root")
	return nil
}

// seedReadingItem ingests one detection run into the checkout's ledger and
// returns the minted item id, so the disposition verb has a real record to
// answer rather than an id that would be unknown from either directory.
func seedReadingItem(t *testing.T, repo string) string {
	t.Helper()
	body := map[string]string{}
	for _, f := range issueschema.ReadingBodyFields["detection"] {
		body[f] = "text for " + f
	}
	res, err := capture.IngestReading(capture.IngestReadingRequest{
		RepoRoot: repo,
		Run:      "rdg-2609090000000001",
		Manifest: "sha256:" + strings.Repeat("a", 64),
		Position: "detection",
		Regime:   issueschema.ReadingRegime("detection"),
		Items:    []capture.ReadingItem{{Pattern: "a stated constraint", Body: body}},
	})
	if err != nil {
		t.Fatalf("seed reading item: %v", err)
	}
	return res.Records[0].ID
}

// assertRecordIn fails unless the record for id sits in the named status folder
// of the CHECKOUT's ledger.
func assertRecordIn(t *testing.T, repo, status, id string) {
	t.Helper()
	dir := filepath.Join(repo, filepath.FromSlash(capture.LedgerRelPath), status)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id+"-") {
			return
		}
	}
	t.Errorf("%s is not in the checkout ledger's %s/ after the transition (the verb moved a record in some other ledger)", id, status)
}

// recordBody returns the bytes of one record in the checkout's ledger.
func recordBody(t *testing.T, repo, status, id string) string {
	t.Helper()
	dir := filepath.Join(repo, filepath.FromSlash(capture.LedgerRelPath), status)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id+"-") {
			b, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
			if rerr != nil {
				t.Fatalf("read %s: %v", e.Name(), rerr)
			}
			return string(b)
		}
	}
	t.Fatalf("%s is not in the checkout ledger's %s/", id, status)
	return ""
}

// TestCaptureOutsideAnyRepositoryRefuses pins the no-repository ruling: with no
// checkout anywhere above, a capture verb REFUSES and writes nothing. Minting a
// ledger in whatever directory the caller stood in is the failure this whole
// record is about, one directory further out — the records would be committed
// by nothing and read by nothing.
func TestCaptureOutsideAnyRepositoryRefuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := realPath(t, t.TempDir())
	t.Chdir(plain)

	for _, args := range [][]string{
		{"capture", "an observation filed outside any repository"},
		{"capture"},
		{"capture", "list", "--open"},
	} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Errorf("`abcd %s` outside a repository succeeded; it must refuse:\n%s", strings.Join(args, " "), out)
			continue
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("`abcd %s` refused without naming the repository as the reason: %v", strings.Join(args, " "), err)
		}
	}
	if _, err := os.Stat(filepath.Join(plain, ".abcd")); err == nil {
		t.Errorf("a refused capture still minted a ledger at %s/.abcd — the refusal must write nothing", plain)
	}
}

// TestCaptureRefusesARepoShapedTreeGitWillNotAnswerFor pins the third state a
// repo root can be in. A directory carrying a .git that git will not answer for
// is repo-SHAPED, and the marker walk behind the core's discovery would hand it
// back as a root without checking either its shape or its owner
// (iss-2609090947359464). Resolving the front doors must not make that walk
// live: the verb refuses here, and says which of the two refusals it is.
func TestCaptureRefusesARepoShapedTreeGitWillNotAnswerFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	planted := realPath(t, t.TempDir())
	if err := os.Mkdir(filepath.Join(planted, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(planted)

	out, err := runCLIErr(t, "capture", "an observation beneath a planted marker")
	if err == nil {
		t.Fatalf("a repo-shaped tree git will not answer for was accepted as a root:\n%s", out)
	}
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Errorf("the refusal does not say that git could not answer: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(planted, filepath.FromSlash(capture.LedgerRelPath))); serr == nil {
		t.Errorf("the refusal still minted a ledger under the planted marker")
	}
}

// TestCaptureFromTheCheckoutRootIsUnchanged is the anti-vacuity control: the
// resolution must be invisible where the caller already stood at the root.
// A fix that refused everywhere, or that resolved to some other tree, would
// pass the detectors above and fail here.
func TestCaptureFromTheCheckoutRootIsUnchanged(t *testing.T) {
	repo, _, ids := captureLedgerFixture(t, 2)

	if got := openCount(t); got != len(ids) {
		t.Fatalf("from the checkout root the board reports open %d, want %d", got, len(ids))
	}
	var res struct {
		Path string `json:"path"`
	}
	out := runCLI(t, "capture", "an observation filed from the checkout root", "--json")
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("capture --json: not JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(res.Path, capture.LedgerRelPath) {
		t.Errorf("the record path %q is not repo-relative to the ledger", res.Path)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(res.Path))); err != nil {
		t.Errorf("the record reported at %q is not where it says: %v", res.Path, err)
	}
	if got := openCount(t); got != len(ids)+1 {
		t.Errorf("after one more capture the board reports open %d, want %d", got, len(ids)+1)
	}
}

// TestCaptureReportsAStrayLedgerBelowTheCheckoutRoot: a ledger this defect
// already minted under a subdirectory holds records nobody will ever see again,
// because every verb now correctly addresses the checkout's. The resolution
// therefore SAYS the stray store is there, on stderr, rather than stepping over
// it in silence. It reports; it moves nothing.
func TestCaptureReportsAStrayLedgerBelowTheCheckoutRoot(t *testing.T) {
	repo, sub, _ := captureLedgerFixture(t, 1)
	stray := filepath.Join(sub, filepath.FromSlash(capture.LedgerRelPath), "open")
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(stray, "iss-2609090000000001-an-orphaned-record.md")
	if err := os.WriteFile(orphan, []byte("---\nid: \"iss-2609090000000001\"\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub)
	out, err := runCLIErr(t, "capture")
	if err != nil {
		t.Fatalf("capture: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), filepath.Join("internal", "core", ".abcd")) {
		t.Errorf("the bare board did not report the stray ledger under %s:\n%s",
			strings.TrimPrefix(sub, repo+string(filepath.Separator)), out)
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Errorf("the report moved or removed the orphaned record: %v", err)
	}
}

// TestCaptureRequestsNeverCarryTheRawWorkingDirectory is the static half, and
// the one that covers a verb this package has not grown yet: EVERY
// capture.*Request built in this package must take its RepoRoot from the
// resolved checkout root, never from the caller's working directory. It reads
// the call sites out of the source, so an eighth front door written the old way
// fails here rather than shipping.
func TestCaptureRequestsNeverCarryTheRawWorkingDirectory(t *testing.T) {
	const resolved = "repoRoot" // the one spelling: captureLedgerRoot's answer.

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse the surface package: %v", err)
	}
	var offenders []string
	seen := 0
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				sel, ok := lit.Type.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || pkgIdent.Name != "capture" || !strings.HasSuffix(sel.Sel.Name, "Request") {
					return true
				}
				seen++
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if !ok || key.Name != "RepoRoot" {
						continue
					}
					value, ok := kv.Value.(*ast.Ident)
					if ok && value.Name == resolved {
						continue
					}
					pos := fset.Position(kv.Pos())
					offenders = append(offenders, fmt.Sprintf("%s:%d: capture.%s{RepoRoot: %s}",
						filepath.Base(pos.Filename), pos.Line, sel.Sel.Name, exprText(kv.Value)))
				}
				return true
			})
		}
	}
	if seen == 0 {
		t.Fatal("no capture.*Request literal found in the surface package: this detector is inspecting the wrong tree")
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("%d capture request(s) take a repo root that is not the resolved checkout root (%q):\n  %s\n"+
			"every capture front door resolves the checkout root first — see captureLedgerRoot",
			len(offenders), resolved, strings.Join(offenders, "\n  "))
	}
}

// exprText renders an expression for a diagnostic, so the failure above names
// what the call site actually wrote.
func exprText(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprText(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		return exprText(v.Fun) + "(…)"
	default:
		return "<expression>"
	}
}
