package reachaudit

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// coreBaseline is the committed list of exported core functions that no
// production code outside their package reaches today (iss-2609252211487887).
const coreBaseline = "testdata/core-unreached.txt"

// TestEveryExportedCoreFunctionIsReachedOrBaselined is the caller audit iss-33
// asked for, over every package under internal/core, as a ratchet: the number
// of exported functions no production code outside their package reaches may
// only fall. A name that is unreached and not in the baseline is new silent
// scaffolding and fails; a baseline name that is now reached, or no longer
// exists, is a stale line and fails too, so the baseline only shrinks.
func TestEveryExportedCoreFunctionIsReachedOrBaselined(t *testing.T) {
	reach, err := Scan(filepath.Join("..", ".."), "internal/core")
	if err != nil {
		t.Fatal(err)
	}
	if len(reach) == 0 {
		t.Fatal("found no exported core functions; the audit is reading the wrong directory")
	}
	baseline, err := readBaseline(coreBaseline)
	if err != nil {
		t.Fatal(err)
	}
	var problems []string
	for _, name := range reach.Unreached() {
		if !baseline[name] {
			problems = append(problems, fmt.Sprintf("%s: new exported function no production code outside its package reaches. Wire the front door that calls it, unexport it if only its own package uses it, delete it if nothing does, or name it ...ForTest if it is a cross-package test seam. Do not add it to %s: the baseline only shrinks.", name, coreBaseline))
		}
	}
	for name := range baseline {
		reached, exists := reach[name]
		switch {
		case !exists:
			problems = append(problems, fmt.Sprintf("%s: in %s but no longer an exported function; delete its line.", name, coreBaseline))
		case reached:
			problems = append(problems, fmt.Sprintf("%s: in %s but production code outside its package now reaches it; delete its line.", name, coreBaseline))
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("exported-reach audit over internal/core (%d unreached, %d baselined):\n%s", len(reach.Unreached()), len(baseline), strings.Join(problems, "\n"))
	}
}

// readBaseline reads a baseline file: one "<package dir>.<Name>" per line,
// sorted and unique, with blank lines and #-comments ignored. An unsorted or
// repeated line is an error, so the file stays a diffable list.
func readBaseline(p string) (map[string]bool, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	prev := ""
	sc := bufio.NewScanner(bytes.NewReader(data))
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line <= prev {
			return nil, fmt.Errorf("%s:%d: %q is out of order or repeated; keep the list sorted and unique", p, n, line)
		}
		prev = line
		out[line] = true
	}
	return out, sc.Err()
}
