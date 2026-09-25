package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// oracleBoardCheckout lays an unmanaged checkout and a fresh home, and
// writes the routing files it is given ("" writes none).
func oracleBoardCheckout(t *testing.T, repoTable, machineTable string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	put := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if repoTable != "" {
		put(filepath.Join(repo, ".abcd", "config", "oracle-routing.json"), repoTable)
	}
	if machineTable != "" {
		put(filepath.Join(home, ".abcd", "oracle-routing.json"), machineTable)
	}
}

func oracleLines(text string) []string {
	var out []string
	in := false
	for _, l := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(l, "  oracle:"):
			in = true
			out = append(out, l)
		case in && strings.HasPrefix(l, "    "):
			out = append(out, l)
		default:
			in = false
		}
	}
	return out
}

// TestBoardOracleLinesShowEveryLayerWithTheWinnerMarked is AC 6's board half:
// a repository row and a machine row that disagree are both shown, beside the
// bundled proposal, and the repository's is marked as the one that applies.
func TestBoardOracleLinesShowEveryLayerWithTheWinnerMarked(t *testing.T) {
	oracleBoardCheckout(t,
		`{"schema_version":1,"agents":{"scribe":{"tier":"frontier"}}}`,
		`{"schema_version":1,"agents":{"scribe":{"tier":"local"}}}`)

	stdout, stderr, err := runCLISplit(t)
	if err != nil {
		t.Fatalf("board: %v\n%s", err, stderr)
	}
	lines := oracleLines(stdout)
	if len(lines) != 16 {
		t.Fatalf("want the oracle heading and one line per agent (15), got %d:\n%s", len(lines), stdout)
	}
	var scribe string
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "scribe ") {
			scribe = l
		}
	}
	for _, want := range []string{"repo=frontier*", "machine=local", "bundled=economy"} {
		if !strings.Contains(scribe, want) {
			t.Fatalf("scribe line %q does not carry %q", scribe, want)
		}
	}
	if strings.Count(scribe, "*") != 1 {
		t.Fatalf("scribe line %q marks more than one winner", scribe)
	}

	var got struct {
		Oracle []struct {
			Agent  string `json:"agent"`
			Winner string `json:"winner"`
			Layers []struct {
				Layer string `json:"layer"`
				Tier  string `json:"tier"`
			} `json:"layers"`
		} `json:"oracle"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range got.Oracle {
		if r.Agent != "scribe" {
			continue
		}
		found = true
		if r.Winner != "repo" || len(r.Layers) != 3 || r.Layers[0].Layer != "repo" || r.Layers[1].Tier != "local" {
			t.Fatalf("scribe = %+v", r)
		}
	}
	if !found {
		t.Fatalf("--json carries no oracle row for scribe: %+v", got.Oracle)
	}
}

// TestBoardOmitsOracleLinesWhenNothingIsAccepted: with no routing table the
// board is unchanged, and the JSON omits the member rather than nulling it.
func TestBoardOmitsOracleLinesWhenNothingIsAccepted(t *testing.T) {
	oracleBoardCheckout(t, "", "")
	stdout, _, err := runCLISplit(t)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "oracle:") {
		t.Fatalf("a board with nothing accepted rendered oracle lines:\n%s", stdout)
	}
	if strings.Contains(string(runCLI(t, "--json")), `"oracle"`) {
		t.Fatal("--json carries an oracle member with nothing accepted")
	}
}

// TestBoardReportsAnOrphanRowAndAMalformedTableOnStderr: the orphan is named
// and the rest render (AC 9 at the surface); a malformed table omits the lines
// with its reason, and the board still succeeds.
func TestBoardReportsAnOrphanRowAndAMalformedTableOnStderr(t *testing.T) {
	oracleBoardCheckout(t, `{"schema_version":1,"agents":{"ghost-agent":{"tier":"local"}}}`, "")
	stdout, stderr, err := runCLISplit(t)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, `"ghost-agent"`) || len(oracleLines(stdout)) != 16 {
		t.Fatalf("stderr %q / stdout:\n%s", stderr, stdout)
	}

	oracleBoardCheckout(t, `{"schema_version":1,"agents":{"scribe":{"tier":"cheap"}}}`, "")
	stdout, stderr, err = runCLISplit(t)
	if err != nil {
		t.Fatalf("a malformed routing table failed the board: %v", err)
	}
	if strings.Contains(stdout, "oracle:") || !strings.Contains(stderr, "oracle lines are omitted") ||
		!strings.Contains(stderr, "cheap") {
		t.Fatalf("stderr %q / stdout:\n%s", stderr, stdout)
	}
}

// TestBoardReadsTheRoutingTableFromTheRulesRoot is review-tier1 F2 at the
// board: a monorepo member with its own .abcd/ governs its subtree, so the
// board run inside it reads the member's routing table, the same root the
// member's rules and guard are read from, not git's toplevel.
func TestBoardReadsTheRoutingTableFromTheRulesRoot(t *testing.T) {
	oracleBoardCheckout(t, "", "")
	top, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	member := filepath.Join(top, "member")
	table := filepath.Join(member, ".abcd", "config", "oracle-routing.json")
	if err := os.MkdirAll(filepath.Dir(table), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(table, []byte(`{"schema_version":1,"agents":{"scribe":{"tier":"local"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(member)
	stdout, stderr, err := runCLISplit(t)
	if err != nil {
		t.Fatalf("board: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "repo=local*") {
		t.Fatalf("the board inside the member did not read the member's routing table:\n%s\nstderr: %s", stdout, stderr)
	}
}
