package guard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// wordReaders is every function in this package that reads a segment word's
// dash or a command's name, with how it goes through the unknown-word rule
// (unknown.go). TestEveryWordReaderGoesThroughUnknown finds the readers in the
// source and fails on one this table does not list, so a new reader is a
// decision someone writes down, never a bypass nobody saw (review3-guard). An
// entry that is not "exempt:" must also call something unknown.go defines.
var wordReaders = map[string]string{
	// unknown.go: the rule and the walks that apply it.
	"unknownFlagCouldBe": "the rule itself",
	"flagCouldBe":        "the rule itself",
	"clusterCouldCarry":  "the rule itself",
	"readWord":           "the rule itself",
	"nameCouldBe":        "the rule itself",
	"nameCouldBeAny":     "the rule itself",
	"anyProgram":         "the rule itself",
	"walkToCommand":      "the rule itself",
	"arrivalsOf":         "the rule itself",
	"commandNamed":       "the rule itself",
	"sitesNamed":         "the rule itself",
	"operandAcceptance":  "the rule itself",
	"operandReadings":    "the rule itself",
	"firstOperands":      "the rule itself",
	"firstOperandLimit":  "the rule itself",

	// match.go
	"matchSegmentNamed":    "sitesNamed: every place the entry's command can sit; anyProgram marks a name nothing fixes",
	"newEntryMatcher":      "operandAcceptance over readWord",
	"precededByCD":         "commandSites and nameCouldBeAny",
	"steppedBeforeCommand": "vanishable (Tier 2's start filter)",
	"flagGroupHit":         "unknownFlagCouldBe and knownText on every argument",
	"flagValueHit":         "unknownFlagCouldBe and knownText on every argument",
	"flagMatches":          "exempt: compares one alternative with the known text flagGroupHit and flagValueHit hand it",
	"flagShaped":           "exempt: reads the known text flagMatches is handed",
	"isShortFlag":          "exempt: reads a registry alternative, never a command word",
	"isShortCluster":       "exempt: a known word's shape; clusterCouldCarry reads the unknown word",
	"pathOf":               "exempt: reads a URL scheme's characters in an operand pathArgMatches reads as the rule says",

	// payload.go
	"payloadsOf":           "arrivalsOf and nameCouldBe: every env on the walk",
	"splitStringValue":     "commandArrivals and nameCouldBe",
	"scanEnvSplits":        "readWord, flagCouldBe and clusterCouldCarry on every unknown word",
	"launcherPayloads":     "readWord on every word",
	"guessedEvalPayload":   "exempt: reads eval's literal `--`; vanishable drops a word that may print nothing",
	"evalPayload":          "exempt: reads eval's literal `--`, which no substitution spells (the rule's terminator clause)",
	"shellCPayloads":       "clusterCouldCarry on every word",
	"shellOperands":        "readWord on every unknown word",
	"pipesIntoInterpreter": "commandSites and nameCouldBeAny",
	"readsScriptStream":    "commandSites and nameCouldBeAny",
	"shellReadsStream":     "readWord and clusterCouldCarry on every unknown word",
	"sourceReadsStream":    "exempt: reads source's literal `--`; its operand goes through scriptIsStream, which reads wordCouldBe",
	"isPlainCommand":       "refuses unknownMark outright",

	// execstring.go
	"execStringPayloads": "commandArrivals and nameCouldBe",
	"scanExecString":     "readWord, flagCouldBe and clusterCouldCarry on every word",
	"clusteredPayload":   "exempt: reads the known word scanExecString hands it",

	// gitconfig.go, hookspath.go, stash.go, gitabbrev.go
	"expandGitAliasesAt":     "sitesNamed",
	"anyAliasRewrite":        "sitesNamed",
	"rewriteGitAliases":      "firstOperands",
	"readGitConfig":          "firstOperandLimit, and flagCouldBe on every unknown word",
	"hooksPathRewrite":       "sitesNamed and firstOperands",
	"bareStash":              "sitesNamed and operandReadings",
	"stashHasMessage":        "exempt: fail-closed by construction — an unknown word is never read as the message, so the stash stays bare",
	"abbreviatesAlternative": "exempt: reads the known text flagGroupHit hands it",

	// speculate.go
	"eligibleStart": "steppedBeforeCommand and anyProgram: no start where no program name is fixed",
	"allNoglob":     "commandSites",

	// Grammar and registry readers, not command words.
	"seqWidth":         "exempt: a brace sequence's number sign, before any word exists",
	"allReserved":      "exempt: reserved words are grammar, which no substitution prints",
	"keywordAt":        "exempt: reserved words are grammar, which no substitution prints",
	"readHeredocDelim": "exempt: the `<<-` operator is grammar",
	"Validate":         "exempt: reads registry entries, not command words",
	"validEntryID":     "exempt: reads a registry id, not a command word",
}

// TestEveryWordReaderGoesThroughUnknown is the grep the rule promises, run as a
// test: every non-test function in this package that tests a word for a
// leading dash, reads a command's basename, looks a name up in the wrapper
// table or the interpreter set, or reads a value-flag table is listed in
// wordReaders, and each listed reader that is not exempt calls into unknown.go.
func TestEveryWordReaderGoesThroughUnknown(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	ruleNames := map[string]bool{}
	type fn struct {
		name, file string
		body       *ast.BlockStmt
	}
	var fns []fn
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if f == "unknown.go" {
					ruleNames[d.Name.Name] = true
				}
				if d.Body != nil {
					fns = append(fns, fn{d.Name.Name, f, d.Body})
				}
			case *ast.GenDecl:
				if f != "unknown.go" {
					continue
				}
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, n := range vs.Names {
							ruleNames[n.Name] = true
						}
					}
				}
			}
		}
	}

	var unlisted []string
	listed := map[string]bool{}
	for _, f := range fns {
		if !readsWords(f.body) {
			continue
		}
		listed[f.name] = true
		how, ok := wordReaders[f.name]
		if !ok {
			unlisted = append(unlisted, f.file+": "+f.name)
			continue
		}
		if strings.HasPrefix(how, "exempt:") || how == "the rule itself" {
			continue
		}
		if !callsAny(f.body, ruleNames) {
			t.Errorf("%s (%s) reads words but calls nothing unknown.go defines; wordReaders says %q", f.name, f.file, how)
		}
	}
	sort.Strings(unlisted)
	if os.Getenv("LIST_READERS") != "" {
		for _, f := range fns {
			if readsWords(f.body) {
				t.Logf("READER %s %s", f.file, f.name)
			}
		}
	}
	for _, u := range unlisted {
		t.Errorf("%s reads a word's dash or a command's name and is not in wordReaders: route it through unknown.go, or list it with the reason it need not", u)
	}
	for name := range wordReaders {
		if !listed[name] {
			t.Errorf("wordReaders lists %s, which no longer reads words (or no longer exists); drop the row", name)
		}
	}
}

// readsWords reports whether a function body reads a word as an option parser
// or a command lookup does: it tests a string for a leading dash or compares it
// with a dash-led literal, compares a byte with '-', reads a basename, tests a
// short-option shape, looks a name up in the wrapper, verb, launcher or
// interpreter tables, reads a value-flag table, or asks unknown.go itself.
func readsWords(body *ast.BlockStmt) bool {
	dashLit := func(e ast.Expr) bool {
		lit, ok := e.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return false
		}
		v, err := strconv.Unquote(lit.Value)
		return err == nil && strings.HasPrefix(v, "-")
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			switch name := callName(n); name {
			case "strings.HasPrefix", "strings.CutPrefix", "strings.TrimPrefix":
				if len(n.Args) == 2 && dashLit(n.Args[1]) {
					found = true
				}
			case "path.Base", "isShellFamily", "shellFamilyGlob", "isShortCluster", "isShortFlag",
				"readWord", "flagCouldBe", "clusterCouldCarry", "unknownFlagCouldBe", "nameCouldBe",
				"nameCouldBeAny", "commandNamed", "commandArrivals", "commandSites", "sitesNamed",
				"operandReadings", "firstOperands", "firstOperandLimit", "operandAcceptance":
				found = true
			case "containsString":
				if len(n.Args) == 2 {
					if id, ok := n.Args[0].(*ast.Ident); ok && strings.HasSuffix(strings.ToLower(id.Name), "flags") {
						found = true
					}
				}
			}
		case *ast.IndexExpr:
			if id, ok := n.X.(*ast.Ident); ok {
				switch id.Name {
				case "wrappers", "wrapperValueFlags", "wrapperOperands", "execStringVerbs", "singleStringLaunchers", "reserved":
					found = true
				}
			}
		case *ast.BinaryExpr:
			if n.Op == token.EQL || n.Op == token.NEQ {
				if dashLit(n.X) || dashLit(n.Y) {
					found = true
				}
				if lit, ok := n.Y.(*ast.BasicLit); ok && lit.Kind == token.CHAR && lit.Value == "'-'" {
					found = true
				}
			}
		}
		return !found
	})
	return found
}

// callsAny reports whether a body calls a function or reads a name in names.
func callsAny(body *ast.BlockStmt, names map[string]bool) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && names[id.Name] {
			found = true
		}
		return !found
	})
	return found
}

// callName spells a call's function as `pkg.Name` or `Name`.
func callName(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + f.Sel.Name
		}
	}
	return ""
}
