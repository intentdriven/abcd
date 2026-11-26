package landing

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The one resolver (itd-2611252049031692 criterion 16): every reader of
// facilitator.mode answers through resolveMode, so one reading of the default
// branch gives one answer wherever it is asked. The audit is structural: a
// second reader beside the resolver fails it.

// resolverFile is where the mode's one decoder and its one resolver live.
const resolverFile = "internal/core/landing/config.go"

// modeLayersCallers are the functions that may decode the mode's layers: the
// resolver's readers, the past commit's reader of the repository layer alone
// (CommittedMode), and the kept reading's writer and reader.
var modeLayersCallers = []string{"readModeContext", "CommittedMode", "AutoForSessions", "keepMode", "KeptMerging"}

// auditModeReaders walks the Go files under mod's internal and cmd folders,
// tests aside, and names each place that reads facilitator.mode beside the
// resolver: a claim of the facilitator namespace, a lookup or decode of the
// key, or a Facilitator built with a mode, outside the functions that are its
// one decoder and its one resolver; and a call of the decoder from any
// function not in modeLayersCallers. seen counts what the walk recognised, so
// a scan that looks at nothing is told from a clean one.
func auditModeReaders(t *testing.T, mod string) (findings []string, seen int) {
	t.Helper()
	for _, top := range []string{"internal", "cmd"} {
		if _, err := os.Stat(filepath.Join(mod, top)); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(filepath.Join(mod, top), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, err := filepath.Rel(mod, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			for _, decl := range f.Decls {
				fn, _ := decl.(*ast.FuncDecl)
				name := ""
				if fn != nil {
					name = fn.Name.Name
				}
				at := rel + " " + name
				ast.Inspect(decl, func(n ast.Node) bool {
					switch n := n.(type) {
					case *ast.CallExpr:
						callee := calleeName(n.Fun)
						switch {
						case callee == "Claim" && len(n.Args) > 0 && isString(n.Args[0], "facilitator"):
							seen++
							if rel != resolverFile || name != "modeLayers" {
								findings = append(findings, at+": claims the facilitator namespace")
							}
						case (callee == "Lookup" || callee == "decodeLayers" || callee == "Decode") && slices.ContainsFunc(n.Args, namesTheKey):
							seen++
							if rel != resolverFile || name != "modeLayers" {
								findings = append(findings, at+": decodes "+KeyFacilitatorMode)
							}
						case callee == "modeLayers":
							seen++
							if !strings.HasPrefix(rel, "internal/core/landing/") || !slices.Contains(modeLayersCallers, name) {
								findings = append(findings, at+": decodes the mode's layers")
							}
						}
					case *ast.CompositeLit:
						if calleeName(n.Type) == "Facilitator" && setsMode(n) {
							seen++
							if rel != resolverFile || name != "resolveMode" {
								findings = append(findings, at+": builds a Facilitator with a mode")
							}
						}
					}
					return true
				})
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return findings, seen
}

// calleeName is the last name of an identifier or a selector, "" otherwise.
func calleeName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.IndexExpr:
		return calleeName(e.X)
	case *ast.IndexListExpr:
		return calleeName(e.X)
	}
	return ""
}

// isString says whether e is the string literal want.
func isString(e ast.Expr, want string) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	v, err := strconv.Unquote(lit.Value)
	return err == nil && v == want
}

// namesTheKey says whether e is the mode's key, by its constant or spelt out.
func namesTheKey(e ast.Expr) bool {
	return calleeName(e) == "KeyFacilitatorMode" || isString(e, KeyFacilitatorMode)
}

// setsMode says whether a composite literal sets a Mode field.
func setsMode(lit *ast.CompositeLit) bool {
	for _, el := range lit.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok && calleeName(kv.Key) == "Mode" {
			return true
		}
	}
	return false
}

func TestEveryReaderOfTheModeAnswersThroughTheResolver(t *testing.T) {
	mod, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	findings, seen := auditModeReaders(t, mod)
	for _, f := range findings {
		t.Errorf("%s: a reader of %s beside the resolver (resolveMode)", f, KeyFacilitatorMode)
	}
	if seen == 0 {
		t.Error("the audit recognised nothing: it is not looking at the mode's readers")
	}
}

// TestTheResolverAuditFindsAPlantedSecondReader: each shape of a second reader,
// planted in a module of its own, is named.
func TestTheResolverAuditFindsAPlantedSecondReader(t *testing.T) {
	mod := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(mod, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(resolverFile, `package landing

func modeLayers() { s.Claim("facilitator", "mode"); decodeLayers(nil, KeyFacilitatorMode, nil, "") }
func resolveMode() Facilitator { return Facilitator{Mode: "human"} }
func readModeContext() { modeLayers() }
`)
	write("internal/surface/cli/second.go", `package cli

func secondReader() bool {
	s.Claim("facilitator", "mode")
	found, _ := s.Lookup(landing.KeyFacilitatorMode)
	v, _ := layered.Decode[string](found[0].Raw, "facilitator.mode")
	f := landing.Facilitator{Mode: v}
	return f.Mode == "auto"
}
`)
	write("internal/core/landing/other.go", `package landing

func shortcut() Facilitator { modeLayers(); return Facilitator{Key: KeyFacilitatorMode, Mode: ModeAuto} }
`)
	findings, _ := auditModeReaders(t, mod)
	for _, want := range []string{
		"internal/surface/cli/second.go secondReader: claims the facilitator namespace",
		"internal/surface/cli/second.go secondReader: decodes facilitator.mode",
		"internal/surface/cli/second.go secondReader: builds a Facilitator with a mode",
		"internal/core/landing/other.go shortcut: decodes the mode's layers",
		"internal/core/landing/other.go shortcut: builds a Facilitator with a mode",
	} {
		if !slices.Contains(findings, want) {
			t.Errorf("the audit did not name %q; it named %q", want, findings)
		}
	}
	if len(findings) != 6 {
		t.Errorf("the audit named %d findings, want 6 (the planted decode is named once per call): %q", len(findings), findings)
	}
}
