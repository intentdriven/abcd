package lint_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestSurfaceAdditionsMergeCleanly is the property the release-gate manifest's
// shape exists for (iss-2610150526192138): two branches cut from one commit,
// each adding a different surface, merge without a conflict in the manifest or
// its example receipt. Before that record, every such pair conflicted, because
// each branch rewrote the same lines: the prompt context that carried both
// rosters, promptHash over it, checkerCount, the line after the last numbered
// surface chapter in briefDocs, and the example receipt's literal copy of the
// manifest's sha256. Every conflict sent the second branch back to merge main
// and run a fresh preflight before it could queue again.
//
// The fixture is a throwaway repository holding copies of the two committed
// files, so the real checkout is never touched. Each branch makes the edits a
// surface-adding branch has to make for TestReleaseGateManifestIsCurrent to
// pass, then refreshes every derived value the files store, as a branch must.
// So a roster put back inline, or a hash stored again, makes both branches
// rewrite one line and fails this test with that file named. A stored count is
// different: both branches bump it to the same value, git merges the identical
// edits cleanly, and the result is silently wrong, which the manifest shape
// check in TestReleaseGateManifestIsCurrent refuses instead.
func TestSurfaceAdditionsMergeCleanly(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const manifestRel = ".abcd/development/release-gate/manifest.json"
	const exampleRel = ".abcd/development/release-gate/receipt.example.json"

	r := gittest.NewRepo(t)
	r.Write(manifestRel, readRepoFile(t, root, manifestRel))
	r.Write(exampleRel, readRepoFile(t, root, exampleRel))
	r.Commit("base: the release-gate manifest and its example receipt")

	// Both branches take the next chapter number, as two concurrent branches
	// do; the names land in different gaps of every sorted list.
	type surface struct{ branch, command, agent, chapter string }
	added := []surface{
		{"add-atlas", "atlas", "", ".abcd/development/brief/04-surfaces/38-atlas.md"},
		{"add-mint", "mint", "mint-checker", ".abcd/development/brief/04-surfaces/38-mint.md"},
	}
	for _, s := range added {
		r.Git("checkout", "-q", "-b", s.branch, "main")
		manifest := addSurface(t, readFixture(t, r, manifestRel), s.command, s.agent, s.chapter)
		r.Write(manifestRel, manifest)
		r.Write(exampleRel, refreshExampleManifestHash(readFixture(t, r, exampleRel), manifest))
		r.Commit("add the " + s.command + " surface")
	}

	r.Git("checkout", "-q", added[0].branch)
	merge := exec.Command("git", "-C", r.Root(),
		"-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false",
		"merge", "--no-edit", added[1].branch)
	merge.Env = r.Env()
	if out, err := merge.CombinedOutput(); err != nil {
		t.Fatalf("two branches each adding a surface conflict on merge (%v); conflicted: %s\n%s",
			err, strings.Join(strings.Fields(r.Git("diff", "--name-only", "--diff-filter=U")), ", "), out)
	}

	merged := readFixture(t, r, manifestRel)
	for _, p := range manifestShapeProblems([]byte(merged)) {
		t.Errorf("the merged manifest: %s", p)
	}
	m, err := parseReleaseGateManifest([]byte(merged))
	if err != nil {
		t.Fatalf("the merged manifest does not parse: %v", err)
	}
	for _, s := range added {
		if !slices.Contains(m.Commands, s.command) {
			t.Errorf("the merged manifest lost the %s command", s.command)
		}
		if s.agent != "" && !slices.Contains(m.Agents, s.agent) {
			t.Errorf("the merged manifest lost the %s agent", s.agent)
		}
		if !slices.Contains(m.BriefDocs, s.chapter) {
			t.Errorf("the merged manifest lost the %s chapter", s.chapter)
		}
	}
	if got, base := readFixture(t, r, exampleRel), readRepoFile(t, root, exampleRel); got != base {
		t.Errorf("the merged example receipt differs from the committed one; a surface addition changes nothing in it")
	}
}

func readFixture(t *testing.T, r *gittest.Repo, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.Root(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// addSurface makes, as text edits, what a branch adding one surface does to
// the manifest: the command (and agent, when one is given) joins its roster,
// the chapter joins briefDocs, and every derived value the manifest stores is
// recomputed. Each list is extended in the order the list is already in.
func addSurface(t *testing.T, manifest, command, agent, chapter string) string {
	t.Helper()
	lines := strings.Split(manifest, "\n")
	lines = addToRoster(t, lines, "commands", command)
	if agent != "" {
		lines = addToRoster(t, lines, "agents", agent)
	}
	m, err := parseReleaseGateManifest([]byte(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	if slices.IsSortedFunc(m.BriefDocs, briefDocOrder) {
		lines = insertInArray(t, lines, "briefDocs", chapter, briefDocOrder)
	} else {
		// Numeric chapter order, the shape before iss-2610150526192138: a new
		// chapter goes after the highest-numbered one.
		after := -1
		for i, l := range lines {
			if numberedChapter.MatchString(l) {
				after = i
			}
		}
		q, _ := json.Marshal(chapter)
		lines = slices.Insert(lines, after+1, leadingSpace(lines[after])+string(q)+",")
	}
	return refreshManifestDerived(t, strings.Join(lines, "\n"))
}

// addToRoster adds name to the manifest's roster: the sorted array when the
// manifest has one, else the inline list in the pinned context (the shape
// before iss-2610150526192138, "commands/ (a, b, ...)").
func addToRoster(t *testing.T, lines []string, roster, name string) []string {
	t.Helper()
	if arrayStart(lines, roster) >= 0 {
		return insertInArray(t, lines, roster, name, strings.Compare)
	}
	for i, l := range lines {
		value, ok := strings.CutPrefix(strings.TrimSpace(l), `"context": `)
		if !ok {
			continue
		}
		comma := strings.HasSuffix(value, ",")
		var context string
		if err := json.Unmarshal([]byte(strings.TrimSuffix(value, ",")), &context); err != nil {
			t.Fatal(err)
		}
		marker := roster + "/ ("
		before, rest, ok := strings.Cut(context, marker)
		if !ok {
			t.Fatalf("the context has no %q list", marker)
		}
		list, after, _ := strings.Cut(rest, ")")
		names := []string{name}
		for _, n := range strings.Split(list, ",") {
			names = append(names, strings.TrimSpace(n))
		}
		sort.Strings(names)
		lines[i] = leadingSpace(l) + `"context": ` + jsonString(t, before+marker+strings.Join(names, ", ")+")"+after)
		if comma {
			lines[i] += ","
		}
		return lines
	}
	t.Fatal("the manifest has neither a " + roster + " array nor a context")
	return nil
}

// insertInArray inserts value into the one-entry-per-line JSON string array
// under key, at the place cmp orders it, keeping the commas valid.
func insertInArray(t *testing.T, lines []string, key, value string, cmp func(a, b string) int) []string {
	t.Helper()
	start := arrayStart(lines, key)
	if start < 0 {
		t.Fatalf("the manifest has no %q array", key)
	}
	end := start + 1
	for !strings.HasPrefix(strings.TrimSpace(lines[end]), "]") {
		end++
	}
	pos := end
	for i := start + 1; i < end; i++ {
		var entry string
		if err := json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimSpace(lines[i]), ",")), &entry); err != nil {
			t.Fatalf("%s entry %q: %v", key, lines[i], err)
		}
		if cmp(value, entry) < 0 {
			pos = i
			break
		}
	}
	entry := leadingSpace(lines[start+1]) + jsonString(t, value)
	if pos == end {
		lines[end-1] += ","
		return slices.Insert(lines, end, entry)
	}
	return slices.Insert(lines, pos, entry+",")
}

func arrayStart(lines []string, key string) int {
	return slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == `"`+key+`": [` })
}

// refreshManifestDerived recomputes every derived value the manifest stores:
// promptHash always, and checkerCount when the manifest stores one.
func refreshManifestDerived(t *testing.T, manifest string) string {
	t.Helper()
	m, err := parseReleaseGateManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("an added surface left the manifest unparseable: %v", err)
	}
	lines := strings.Split(manifest, "\n")
	for i, l := range lines {
		switch {
		case strings.HasPrefix(strings.TrimSpace(l), `"checkerCount": `):
			lines[i] = leadingSpace(l) + fmt.Sprintf(`"checkerCount": %d,`, len(m.BriefDocs)+len(m.Surfaces))
		case strings.HasPrefix(strings.TrimSpace(l), `"promptHash": `):
			lines[i] = leadingSpace(l) + `"promptHash": "` + releaseGatePromptHash(m) + `",`
		}
	}
	return strings.Join(lines, "\n")
}

var numberedChapter = regexp.MustCompile(`/04-surfaces/\d+-`)

var literalManifestHash = regexp.MustCompile(`"manifestHash": "sha256:[0-9a-f]{64}"`)

// refreshExampleManifestHash rewrites the example receipt's manifestHash to the
// manifest's sha256 when the example stores a literal one, and leaves a
// placeholder alone.
func refreshExampleManifestHash(example, manifest string) string {
	return literalManifestHash.ReplaceAllLiteralString(example, `"manifestHash": "`+sha256Of(manifest)+`"`)
}

func sha256Of(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func leadingSpace(s string) string { return s[:len(s)-len(strings.TrimLeft(s, " \t"))] }

// jsonString renders s as a JSON string the way the manifest spells its
// strings: UTF-8 as itself, and no HTML escaping.
func jsonString(t *testing.T, s string) string {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}
