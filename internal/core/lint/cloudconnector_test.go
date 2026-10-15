package lint_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The cloud-session scripts (spc-2610120941441282, itd-2610120738075547) hold
// two static properties here, beside the preflight gate pins; the scripts
// themselves are run in throwaway clones under internal/surface/cli.

// TestGateToolsNamesOnlyRealPreflightGates holds scripts/gate-tools.tsv, the
// list the cloud environment check reads (C2, decision 5), to the Makefile: a
// gate it names is one of preflight's prerequisites, or `preflight` itself for
// the steps its own recipe runs (build, vet, test). A renamed or retired gate
// then fails here rather than leaving the list quietly naming nothing.
func TestGateToolsNamesOnlyRealPreflightGates(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	real := map[string]bool{"preflight": true}
	for _, p := range preflightPrereqs(t, root) {
		real[p] = true
	}
	seen := map[string]bool{}
	gated := 0
	for i, line := range strings.Split(readRepoFile(t, root, "scripts/gate-tools.tsv"), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			t.Errorf("line %d has %d tab-separated fields, want 3 (tool, gates, setup step): %q", i+1, len(f), line)
			continue
		}
		tool, gates, setup := f[0], f[1], f[2]
		if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`).MatchString(tool) {
			t.Errorf("line %d: %q is not a command name", i+1, tool)
		}
		if seen[tool] {
			t.Errorf("line %d: %s is listed twice", i+1, tool)
		}
		seen[tool] = true
		if strings.TrimSpace(setup) == "" {
			t.Errorf("line %d: %s names no setup step that supplies it", i+1, tool)
		}
		if gates == "-" {
			continue
		}
		gated++
		for _, g := range strings.Split(gates, ",") {
			if !real[g] {
				t.Errorf("line %d: %s names %q, which is not a preflight prerequisite in the Makefile", i+1, tool, g)
			}
		}
	}
	if gated == 0 {
		t.Error("the list names no tool a preflight gate needs, so the environment check could never stop a session")
	}
	if !seen["go"] {
		t.Error("the list does not name go, which every Go gate needs")
	}
}

// hostMarkers are the first host's markers (itd-2610120738075547 C12): its
// name, its command-line tool, the variable that says a session is remote, its
// work-in-progress prefix and the folder it reads its settings from. A test
// may name them; the scripts outside the connector may not.
//
// The command-line tool is matched as a lower-case word standing alone. A
// model identifier (`claude-opus-5`) or the vendor half of an attribution
// trailer (`Assisted-by: Claude:...`) is not the host: the attribution gate,
// which this spec reuses, names the model vendor in exactly those shapes.
var hostMarkers = map[string]*regexp.Regexp{
	"the host's name":         regexp.MustCompile(`(?i)\bclaude[ _-]?code\b`),
	"the remote variable":     regexp.MustCompile(`CLAUDE_CODE_REMOTE`),
	"the host's wip prefix":   regexp.MustCompile(`(?i)claude/abcd-wip`),
	"the host's settings dir": regexp.MustCompile(`(?i)\.claude\b`),
	"the host's command":      regexp.MustCompile(`(^|[^A-Za-z0-9_.-])claude($|[^A-Za-z0-9_:-])`),
}

// hostConnectorDir and hostSettingsFile are the only places the markers may
// appear: the connector, and the settings file the host reads at a fixed path.
const (
	hostConnectorDir = "scripts/cloud/connectors/claude"
	hostSettingsFile = ".claude/settings.json"
)

// cloudSharedFiles are the files outside scripts/cloud/ that this spec adds or
// changes. A file not written yet is skipped, so the lanes that add them are
// covered the moment they land.
var cloudSharedFiles = []string{
	"scripts/gate-tools.tsv",
	"scripts/check-attribution.sh",
	"scripts/check-wip-source.sh",
	"scripts/check-wip-source-cases.sh",
	".githooks/pre-push",
	".github/workflows/ci.yml",
	".abcd/work/rulesets/main-protection.json",
}

func hostMarkerHits(text string) []string {
	var hits []string
	for i, line := range strings.Split(text, "\n") {
		for what, re := range hostMarkers {
			if re.MatchString(line) {
				hits = append(hits, what+" at line "+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
	}
	return hits
}

// TestNoHostMarkerOutsideConnector is C12's search: every file the cloud spec
// adds or changes outside the host connector and the host's settings file is
// free of the first host's markers, so a second host is a new connector and no
// edit anywhere else. Everything under scripts/cloud/ is searched, other
// connectors included, so a later lane's script is covered without an edit
// here.
func TestNoHostMarkerOutsideConnector(t *testing.T) {
	// The scanner itself: each marker fires on its own shape and none fires on
	// the attribution shapes the gate script legitimately carries.
	for _, s := range []string{
		"Claude Code", "claude-code", "CLAUDE_CODE_REMOTE=true", "push to claude/abcd-wip/iss-1",
		".claude/settings.json", "claude -p brief", "command -v claude", "x=$(claude --version)",
	} {
		if len(hostMarkerHits(s)) == 0 {
			t.Errorf("the scanner misses the host marker in %q", s)
		}
	}
	for _, s := range []string{
		"Assisted-by: Claude:claude-opus-5", "claude-opus-4-8[1m]", "`Claude <noreply@anthropic.com>`", "abcd-wip/",
	} {
		if hits := hostMarkerHits(s); len(hits) != 0 {
			t.Errorf("the scanner flags an attribution shape as a host marker: %v", hits)
		}
	}

	root := filepath.Join("..", "..", "..")
	var files []string
	cloud := filepath.Join(root, "scripts", "cloud")
	if _, err := os.Stat(filepath.Join(cloud, "connector.sh")); err != nil {
		t.Fatalf("scripts/cloud/connector.sh is missing, so there is nothing host-agnostic to search: %v", err)
	}
	err := filepath.WalkDir(cloud, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == hostConnectorDir {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range cloudSharedFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			files = append(files, rel)
		}
	}
	for _, rel := range files {
		if rel == hostSettingsFile || strings.HasPrefix(rel, hostConnectorDir+"/") {
			continue
		}
		for _, hit := range hostMarkerHits(readRepoFile(t, root, rel)) {
			t.Errorf("%s names the first host outside its connector: %s", rel, hit)
		}
	}

	// The connector is where the markers live; a search that finds nothing
	// there is not reading what it claims to.
	conn := readRepoFile(t, root, hostConnectorDir+"/is-remote.sh") + readRepoFile(t, root, hostConnectorDir+"/connector.conf")
	if len(hostMarkerHits(conn)) == 0 {
		t.Error("the host connector carries none of the host's markers, so the search proves nothing")
	}
}
