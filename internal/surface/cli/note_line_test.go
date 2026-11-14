package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// layerNotesFunc is the function that returns the layers' resolution notes,
// and the index of the notes among its results. The audit starts from it: a
// value is a layer note because this function returned it, whatever the
// variable holding it is called.
const (
	layerNotesFunc   = "layered.RootsFor"
	layerNotesResult = 1
)

// layeredImportPath is the package the resolvers live in. The audit names
// them as layered.<Func>, so every file must import it under that name.
const layeredImportPath = "github.com/intentdriven/abcd/internal/core/layered"

// layerNoteFuncs are the resolvers that return one note as a string rather
// than a list. Their callers are pinned with the rest, and each spells its own
// line: the audit does not judge how.
var layerNoteFuncs = []string{"layered.HomeRoots", "layered.MachineRoots"}

// noteAudit is what one reading of the package's source found.
type noteAudit struct {
	// offenders are the positions where a layer note is printed, or used,
	// another way than fmt.Fprintln(w, noteLine(n)) in a loop over the notes.
	offenders []string
	// sites is one line for each call that resolves layers: the function it
	// is written in, the function called, and what became of the notes.
	sites []string
	// prints counts the fmt.Fprintln(w, noteLine(n)) prints of a layer note.
	prints int
}

// auditLayerNotes follows the layer notes from the function that returns them.
// A function of the package that returns the notes it was handed (drawnTerminal)
// is found and followed the same way, to a fixed point. The notes may be
// discarded (assigned to _), returned from a named function, or ranged over
// with every use of the loop's value being fmt.Fprintln(w, noteLine(n)).
// Anything else is an offender: a loop that prints another way, whatever its
// variable is called, a note passed to another function, a note appended to a
// list, a call whose results are not assigned.
func auditLayerNotes(fset *token.FileSet, files []*ast.File) noteAudit {
	sources := map[string]int{layerNotesFunc: layerNotesResult}
	for {
		grew := false
		for _, u := range noteUnits(files) {
			for _, idx := range auditNoteUnit(fset, u, sources, nil) {
				if _, ok := sources[u.name]; !ok && u.named {
					sources[u.name] = idx
					grew = true
				}
			}
		}
		if !grew {
			break
		}
	}
	var a noteAudit
	for _, u := range noteUnits(files) {
		auditNoteUnit(fset, u, sources, &a)
	}
	// The audit knows the resolvers by the package's own name, so a file that
	// imports it under another would hide every call it makes.
	for _, f := range files {
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == layeredImportPath && imp.Name != nil && imp.Name.Name != "layered" {
				a.offenders = append(a.offenders, fset.Position(imp.Pos()).String()+": the layered package imported under another name, which hides its calls from the audit")
			}
		}
	}
	sort.Strings(a.sites)
	sort.Strings(a.offenders)
	return a
}

// noteUnit is one top-level declaration: a function, or a package-level
// variable or constant, whose initialiser may hold a function literal.
type noteUnit struct {
	name  string
	named bool // a plain function, callable by its name within the package
	node  ast.Node
}

func noteUnits(files []*ast.File) []noteUnit {
	var out []noteUnit
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Body == nil {
					continue
				}
				u := noteUnit{name: d.Name.Name, named: d.Recv == nil, node: d.Body}
				if d.Recv != nil && len(d.Recv.List) == 1 {
					u.name = noteExprName(d.Recv.List[0].Type) + "." + d.Name.Name
				}
				out = append(out, u)
			case *ast.GenDecl:
				for _, s := range d.Specs {
					if vs, ok := s.(*ast.ValueSpec); ok && len(vs.Names) > 0 {
						out = append(out, noteUnit{name: "var " + vs.Names[0].Name, node: vs})
					}
				}
			}
		}
	}
	return out
}

// noteExprName is a call's function, or a receiver's type, as written: an
// identifier, pkg.Name, or the type under a pointer; "" for anything else.
func noteExprName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return noteExprName(e.X)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
	}
	return ""
}

// auditNoteUnit reads one declaration. It returns the result indexes at which
// the declaration's own function returns layer notes, and, when a is not nil,
// records the offenders, sites and prints it found.
func auditNoteUnit(fset *token.FileSet, u noteUnit, sources map[string]int, a *noteAudit) (returns []int) {
	sourceOf := func(e ast.Expr) (string, int, bool) {
		call, ok := e.(*ast.CallExpr)
		if !ok {
			return "", 0, false
		}
		name := noteExprName(call.Fun)
		idx, ok := sources[name]
		return name, idx, ok && name != ""
	}
	offend := func(p token.Pos, what string) {
		if a != nil {
			a.offenders = append(a.offenders, fset.Position(p).String()+": "+what)
		}
	}

	// Pass 1: where the notes are bound. vars are the names holding layer
	// notes in this declaration; bound are the calls and identifiers already
	// accounted for.
	vars := map[string]bool{}
	bound := map[ast.Node]bool{}
	type site struct{ fn, name string }
	var kept, dropped []site
	bind := func(lhs []ast.Expr, rhs []ast.Expr) {
		if len(rhs) != 1 {
			return
		}
		fn, idx, ok := sourceOf(rhs[0])
		if !ok || idx >= len(lhs) {
			return
		}
		id, ok := lhs[idx].(*ast.Ident)
		if !ok {
			return // left unbound: the call is reported below
		}
		bound[rhs[0]] = true
		bound[id] = true
		if id.Name == "_" {
			dropped = append(dropped, site{fn, ""})
			return
		}
		vars[id.Name] = true
		kept = append(kept, site{fn, id.Name})
	}
	ast.Inspect(u.node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			bind(n.Lhs, n.Rhs)
		case *ast.ValueSpec:
			lhs := make([]ast.Expr, len(n.Names))
			for i, id := range n.Names {
				lhs[i] = id
			}
			bind(lhs, n.Values)
		}
		return true
	})

	// Pass 2: what becomes of them. ranged and returned are by variable name.
	// called holds each function expression seen in call position, so a
	// function that returns layer notes named anywhere else is a value the
	// audit cannot follow (var rootsFor = layered.RootsFor).
	ranged := map[string]bool{}
	returned := map[string]bool{}
	called := map[ast.Expr]bool{}
	asValue := func(e ast.Expr) {
		if _, ok := sources[noteExprName(e)]; ok && !called[e] {
			offend(e.Pos(), "a function that returns layer notes taken as a value, which the audit cannot follow")
		}
	}
	var walk func(n ast.Node, lits int)
	walk = func(n ast.Node, lits int) {
		ast.Inspect(n, func(m ast.Node) bool {
			switch m := m.(type) {
			case *ast.FuncLit:
				walk(m.Body, lits+1)
				return false
			case *ast.SelectorExpr:
				asValue(m)
				walk(m.X, lits) // the selected name is a field or method, never the variable
				return false
			case *ast.KeyValueExpr:
				if _, ok := m.Key.(*ast.Ident); ok {
					walk(m.Value, lits) // a struct literal's field name
					return false
				}
			case *ast.ReturnStmt:
				if len(m.Results) == 1 {
					if fn, idx, ok := sourceOf(m.Results[0]); ok && !bound[m.Results[0]] {
						bound[m.Results[0]] = true
						if lits > 0 || !u.named {
							offend(m.Pos(), "layer notes returned from a function the audit cannot follow by name")
						} else {
							returns = append(returns, idx)
							if a != nil {
								a.sites = append(a.sites, u.name+": "+fn+": notes returned")
							}
						}
					}
				}
				for i, r := range m.Results {
					id, ok := r.(*ast.Ident)
					if !ok || !vars[id.Name] {
						continue
					}
					bound[id] = true
					returned[id.Name] = true
					if lits > 0 || !u.named {
						offend(id.Pos(), "layer notes returned from a function the audit cannot follow by name")
						continue
					}
					returns = append(returns, i)
				}
			case *ast.RangeStmt:
				id, ok := m.X.(*ast.Ident)
				if !ok || !vars[id.Name] {
					return true
				}
				bound[id] = true
				ranged[id.Name] = true
				prints, bad := noteLoopPrints(m)
				if a != nil {
					a.prints += prints
				}
				for _, p := range bad {
					offend(p, "a layer note used another way than fmt.Fprintln(w, noteLine(n))")
				}
				if prints == 0 && len(bad) == 0 {
					offend(m.Pos(), "a loop over layer notes that prints none through noteLine")
				}
			case *ast.CallExpr:
				called[m.Fun] = true // Inspect visits the call before its function
				if _, _, ok := sourceOf(m); ok && !bound[m] {
					// A return of the call alone is taken by the ReturnStmt
					// case first, since Inspect visits the statement before it.
					offend(m.Pos(), "layer notes not assigned: the call's results are used in place")
				}
			case *ast.Ident:
				if vars[m.Name] && !bound[m] {
					offend(m.Pos(), "layer notes used other than by ranging over them or returning them")
				}
				asValue(m)
			}
			return true
		})
	}
	walk(u.node, 0)

	if a == nil {
		return returns
	}
	for _, s := range dropped {
		a.sites = append(a.sites, u.name+": "+s.fn+": notes discarded")
	}
	for _, s := range kept {
		var did []string
		if ranged[s.name] {
			did = append(did, "printed")
		}
		if returned[s.name] {
			did = append(did, "returned")
		}
		if len(did) == 0 {
			did = []string{"unused"}
			offend(u.node.Pos(), "layer notes held in "+s.name+" in "+u.name+" are neither printed nor returned")
		}
		a.sites = append(a.sites, u.name+": "+s.fn+": notes "+strings.Join(did, " and "))
	}
	ast.Inspect(u.node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			name := noteExprName(call.Fun)
			for _, f := range layerNoteFuncs {
				if name == f {
					a.sites = append(a.sites, u.name+": "+name+": one note, spelled by the caller")
				}
			}
		}
		return true
	})
	return returns
}

// noteLoopPrints reads a loop over layer notes: how many times its value is
// printed as fmt.Fprintln(w, noteLine(n)), and the position of every other use
// of the value (or of the loop itself, when it names no value to print).
func noteLoopPrints(rs *ast.RangeStmt) (prints int, bad []token.Pos) {
	val, ok := rs.Value.(*ast.Ident)
	if !ok || val.Name == "_" {
		return 0, []token.Pos{rs.Pos()}
	}
	good := map[*ast.Ident]bool{}
	ast.Inspect(rs.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || noteExprName(call.Fun) != "fmt.Fprintln" || len(call.Args) != 2 {
			return true
		}
		inner, ok := call.Args[1].(*ast.CallExpr)
		if !ok || noteExprName(inner.Fun) != "noteLine" || len(inner.Args) != 1 {
			return true
		}
		if id, ok := inner.Args[0].(*ast.Ident); ok && id.Name == val.Name {
			good[id] = true
			prints++
		}
		return true
	})
	ast.Inspect(rs.Body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == val.Name && !good[id] {
			bad = append(bad, id.Pos())
		}
		return true
	})
	return prints, bad
}

// namedNoteLoops is the audit by name, kept for the notes the audit by source
// cannot reach: a list another package composes and hands over (matrix.PlanForLab,
// statusline.Load) is not the result of the function that returns layer notes,
// and where its caller ranges over a variable whose name ends in notes, each
// fmt.Fprint* in the loop is fmt.Fprintln(w, noteLine(n)).
func namedNoteLoops(fset *token.FileSet, files []*ast.File) (offenders []string, seen int) {
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			rs, ok := n.(*ast.RangeStmt)
			if !ok {
				return true
			}
			if id, ok := rs.X.(*ast.Ident); !ok || !strings.HasSuffix(strings.ToLower(id.Name), "notes") {
				return true
			}
			ast.Inspect(rs.Body, func(m ast.Node) bool {
				call, ok := m.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := noteExprName(call.Fun)
				if !strings.HasPrefix(name, "fmt.Fprint") {
					return true
				}
				seen++
				through := false
				if name == "fmt.Fprintln" && len(call.Args) == 2 {
					if c, ok := call.Args[1].(*ast.CallExpr); ok && noteExprName(c.Fun) == "noteLine" {
						through = true
					}
				}
				if !through {
					offenders = append(offenders, fset.Position(call.Pos()).String())
				}
				return true
			})
			return true
		})
	}
	return offenders, seen
}

// surfaceSource parses the package's own source, tests left out.
func surfaceSource(t *testing.T) (*token.FileSet, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the surface package: %v", err)
	}
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse the surface package: %v", err)
		}
		files = append(files, f)
	}
	return fset, files
}

// TestEveryLayerNoteIsPrintedThroughNoteLine pins the one spelling of a
// resolution note's stderr line. RootsFor hands the HOME refusal over already
// prefixed "abcd: ", so a site that adds a prefix of its own prints it twice
// ("abcd abcd: HOME …", "abcd land: abcd: HOME …"), and a site that adds none
// prints the rules root's refusal unattributed. The prints are found by the
// function that returns the notes (auditLayerNotes), not by what the variable
// holding them is called (iss-2611121942072836): every use of a note
// layered.RootsFor returned is fmt.Fprintln(w, noteLine(n)), the note's line
// alone, and a site written another way fails here, naming its position. The
// loops over a list named notes that another package composed are held to the
// same spelling by name (namedNoteLoops).
func TestEveryLayerNoteIsPrintedThroughNoteLine(t *testing.T) {
	fset, files := surfaceSource(t)
	a := auditLayerNotes(fset, files)
	if a.prints < 15 {
		t.Fatalf("found %d prints of a layer note, want at least 15: the scan no longer sees the print sites", a.prints)
	}
	if len(a.offenders) > 0 {
		t.Fatalf("a resolution note is not printed through noteLine at:\n%s", strings.Join(a.offenders, "\n"))
	}
	offenders, seen := namedNoteLoops(fset, files)
	if seen < 15 {
		t.Fatalf("found %d note prints by name, want at least 15: the scan no longer sees the print sites", seen)
	}
	if len(offenders) > 0 {
		t.Fatalf("a note in a loop over a list named notes is printed without noteLine at:\n%s", strings.Join(offenders, "\n"))
	}
}

// TestTheFunctionsThatResolveLayersArePinned is the list of the functions in
// this package that resolve the configuration layers, by the function each is
// written in (a verb's constructor, or the helper the verbs share), and what
// each does with the notes it is handed. A new caller, a caller that stops
// printing its notes, and one that starts discarding them each change the list
// and fail here, so the change is made on purpose: add the line, having checked
// that a caller discarding the notes runs beside one that prints them.
func TestTheFunctionsThatResolveLayersArePinned(t *testing.T) {
	want := []string{
		"askLibraryClass: drawnTerminal: notes printed",
		"boardOracle: layered.RootsFor: notes printed",
		"drawnTerminal: layered.RootsFor: notes returned",
		"newAgentsCommand: layered.HomeRoots: one note, spelled by the caller",
		"newAhoyConnectCommand: layered.RootsFor: notes printed",
		"newAhoyCredentialCommand: layered.RootsFor: notes printed",
		"newAhoyWorktreePruneCommand: layered.RootsFor: notes printed",
		"newBuildCommand: layered.RootsFor: notes printed",
		"newBuildNextCommand: layered.RootsFor: notes printed",
		"newImplementStepCommand: layered.RootsFor: notes discarded",
		"newImplementStepCommand: layered.RootsFor: notes printed",
		"newImplementStepCommand: layered.RootsFor: notes printed",
		"newIntentConsistencyCommand: layered.RootsFor: notes discarded",
		"newLandCommand: layered.RootsFor: notes printed",
		"newPrompter: drawnTerminal: notes printed",
		"noteMachineHome: layered.MachineRoots: one note, spelled by the caller",
		"questionLimitsAt: layered.RootsFor: notes discarded",
		"resolveMatch: layered.RootsFor: notes printed",
		"routeFlag.resolve: layered.RootsFor: notes printed",
		"runAhoyProviders: layered.RootsFor: notes printed",
		"runConnectGuide: layered.RootsFor: notes printed",
		"runDrain: layered.RootsFor: notes printed",
		"runDrive: layered.RootsFor: notes discarded",
		"runWrittenInterview: drawnTerminal: notes discarded",
		"runWrittenInterview: layered.RootsFor: notes printed",
		"startDrive: layered.RootsFor: notes printed",
		"var loadLessonsRouter: layered.RootsFor: notes printed",
	}
	fset, files := surfaceSource(t)
	got := auditLayerNotes(fset, files).sites
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("the functions that resolve layers changed; the list is now:\n%s\n\nwas:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestLayerNoteAuditFollowsTheValue runs the audit over source written to
// escape the audit by name: the notes under another name, printed another way,
// handed on, or returned by a helper.
func TestLayerNoteAuditFollowsTheValue(t *testing.T) {
	cases := []struct {
		name, body string
		offenders  int
		prints     int
		sites      []string
	}{
		{
			name: "a loop under another name that prints its own prefix",
			body: `func run(w io.Writer, cwd string) {
	roots, diag := layered.RootsFor(cwd)
	for _, d := range diag {
		fmt.Fprintf(w, "abcd: %s\n", d)
	}
	_ = roots
}`,
			offenders: 1,
			sites:     []string{"run: layered.RootsFor: notes printed"},
		},
		{
			name: "a loop under another name that prints through noteLine",
			body: `func run(w io.Writer, cwd string) {
	_, diag := layered.RootsFor(cwd)
	for _, d := range diag {
		fmt.Fprintln(w, noteLine(d))
	}
}`,
			prints: 1,
			sites:  []string{"run: layered.RootsFor: notes printed"},
		},
		{
			name: "a note written without fmt",
			body: `func run(w io.Writer, cwd string) {
	_, notes := layered.RootsFor(cwd)
	for _, n := range notes {
		io.WriteString(w, n+"\n")
	}
}`,
			offenders: 1,
			sites:     []string{"run: layered.RootsFor: notes printed"},
		},
		{
			name: "the notes handed to another function",
			body: `func run(w io.Writer, cwd string) {
	_, notes := layered.RootsFor(cwd)
	report(w, notes)
}`,
			offenders: 2, // the use, and notes neither printed nor returned
			sites:     []string{"run: layered.RootsFor: notes unused"},
		},
		{
			name: "the notes discarded",
			body: `func run(cwd string) {
	roots, _ := layered.RootsFor(cwd)
	_ = roots
}`,
			sites: []string{"run: layered.RootsFor: notes discarded"},
		},
		{
			name: "a helper that returns the notes, and its caller's loop",
			body: `func resolve(cwd string) (layered.Roots, []string, bool) {
	r, found := layered.RootsFor(cwd)
	return r, found, true
}
func run(w io.Writer, cwd string) {
	_, lines, _ := resolve(cwd)
	for _, l := range lines {
		fmt.Fprintln(w, "abcd run:", l)
	}
}`,
			offenders: 1,
			sites: []string{
				"resolve: layered.RootsFor: notes returned",
				"run: resolve: notes printed",
			},
		},
		{
			name: "a helper that returns the call itself",
			body: `func resolve(cwd string) (layered.Roots, []string) {
	return layered.RootsFor(cwd)
}
func run(w io.Writer, cwd string) {
	_, lines := resolve(cwd)
	for _, l := range lines {
		fmt.Fprintln(w, noteLine(l))
	}
}`,
			prints: 1,
			sites: []string{
				"resolve: layered.RootsFor: notes returned",
				"run: resolve: notes printed",
			},
		},
		{
			name: "a closure that returns the notes",
			body: `func run(cwd string) {
	f := func() []string {
		_, notes := layered.RootsFor(cwd)
		return notes
	}
	_ = f
}`,
			offenders: 1,
			sites:     []string{"run: layered.RootsFor: notes returned"},
		},
		{
			name: "the one-note resolvers are listed and not judged",
			body: `func run(w io.Writer) {
	if _, note := layered.HomeRoots(); note != "" {
		fmt.Fprintln(w, "abcd: "+note)
	}
}`,
			sites: []string{"run: layered.HomeRoots: one note, spelled by the caller"},
		},
		{
			name: "the function bound to a variable and called under its name",
			body: `var rootsFor = layered.RootsFor

func run(w io.Writer, cwd string) {
	_, diag := rootsFor(cwd)
	for _, d := range diag {
		fmt.Fprintf(w, "abcd: %s\n", d)
	}
}`,
			offenders: 1,
		},
		{
			name: "a helper of the package taken as a value",
			body: `func resolve(cwd string) (layered.Roots, []string) {
	return layered.RootsFor(cwd)
}

func run(w io.Writer, cwd string) {
	f := resolve
	_, diag := f(cwd)
	for _, d := range diag {
		fmt.Fprintf(w, "abcd: %s\n", d)
	}
}`,
			offenders: 1,
			sites:     []string{"resolve: layered.RootsFor: notes returned"},
		},
		{
			name: "the layered package imported under another name",
			body: `import l "github.com/intentdriven/abcd/internal/core/layered"

func run(cwd string) {
	l.RootsFor(cwd)
}`,
			offenders: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "fixture.go", "package cli\n"+c.body, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			a := auditLayerNotes(fset, []*ast.File{f})
			if len(a.offenders) != c.offenders {
				t.Errorf("offenders = %d, want %d:\n%s", len(a.offenders), c.offenders, strings.Join(a.offenders, "\n"))
			}
			if a.prints != c.prints {
				t.Errorf("prints = %d, want %d", a.prints, c.prints)
			}
			if strings.Join(a.sites, "\n") != strings.Join(c.sites, "\n") {
				t.Errorf("sites:\n%s\nwant:\n%s", strings.Join(a.sites, "\n"), strings.Join(c.sites, "\n"))
			}
		})
	}
}
