package reachaudit

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// configIndex is the page whose "Root machine records" table indexes the
// tracked files under .abcd/ the shipped binary reads.
const configIndex = ".abcd/README.md"

// configPathLiteral matches a record under .abcd/config/ as non-test code
// spells it: in one string or comment, or as the arguments of a filepath.Join.
var (
	configPathLiteral = regexp.MustCompile(`\.abcd/config/([A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)+)`)
	configPathJoined  = regexp.MustCompile(`"\.abcd",\s*"config",\s*"([A-Za-z0-9_.-]+)"`)
)

// TestEveryConfigRecordNamedInInternalHasAnIndexRow holds the root machine
// records table of .abcd/README.md to the directory it indexes
// (iss-2610241549107888): every .abcd/config/ path non-test code under
// internal/ names has a row. A record has a row when a table row whose first
// cell names `config/...` carries the record's file name in backticks, so the
// `config/` directory row covers the records it lists and each optional
// record has a row of its own.
func TestEveryConfigRecordNamedInInternalHasAnIndexRow(t *testing.T) {
	root := filepath.Join("..", "..")
	named, err := configRecordsNamed(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(named) == 0 {
		t.Fatal("found no .abcd/config/ path in non-test code under internal/; the check is reading the wrong directory")
	}
	page, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(configIndex)))
	if err != nil {
		t.Fatal(err)
	}
	rows := configIndexRows(string(page))
	if len(rows) == 0 {
		t.Fatalf("%s: found no `config/` row under \"## Root machine records\"; the check is reading the wrong table", configIndex)
	}
	var names []string
	for name := range named {
		names = append(names, name)
	}
	sort.Strings(names)
	var missing []string
	for _, name := range names {
		if !configRecordIndexed(rows, name) {
			missing = append(missing, ".abcd/config/"+name+" (named in "+named[name]+")")
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%s: the root machine records table has no row for %d record(s) the binary's code names. Add a row for each, naming its reader and its owning record:\n%s",
			configIndex, len(missing), strings.Join(missing, "\n"))
	}
}

// configRecordsNamed maps each file name under .abcd/config/ that a non-test
// Go file under dir names to the first file, in walk order, naming it.
func configRecordsNamed(dir string) (map[string]string, error) {
	named := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, re := range []*regexp.Regexp{configPathLiteral, configPathJoined} {
			for _, m := range re.FindAllSubmatch(src, -1) {
				name := string(m[1])
				if _, seen := named[name]; !seen {
					named[name] = filepath.ToSlash(p)
				}
			}
		}
		return nil
	})
	return named, err
}

// configIndexRows returns the rows of the "Root machine records" table whose
// first cell names a path under config/.
func configIndexRows(page string) []string {
	var rows []string
	in := false
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(line, "## ") {
			in = strings.TrimSpace(line) == "## Root machine records"
			continue
		}
		if in && strings.HasPrefix(line, "| `config/") {
			rows = append(rows, line)
		}
	}
	return rows
}

// configRecordIndexed reports whether a row names the record: as
// `config/<name>` or, in the directory row's list, as `<name>`.
func configRecordIndexed(rows []string, name string) bool {
	for _, row := range rows {
		if strings.Contains(row, "`config/"+name+"`") || strings.Contains(row, "`"+name+"`") {
			return true
		}
	}
	return false
}
