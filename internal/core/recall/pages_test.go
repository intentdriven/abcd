package recall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/rules"
)

// pagesFixture is a home and a repository with an empty page store in each
// scope: the workspace's .abcd/memory/ and the machine's ~/.abcd.noindex/memory/.
func pagesFixture(t *testing.T) (home, repo, workspace, machine string) {
	t.Helper()
	home = t.TempDir()
	repo = t.TempDir()
	workspace = filepath.Join(repo, ".abcd", "memory")
	machine = abcdhome.Path(home, "memory")
	for _, dir := range []string{workspace, machine} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return home, repo, workspace, machine
}

// writePage writes one memory page into dir: a recall: list when recall is
// given, a source block, then body.
func writePage(t *testing.T, dir, name, body string, recall ...string) {
	t.Helper()
	fm := "---\n"
	if len(recall) > 0 {
		fm += "recall: [" + strings.Join(recall, ", ") + "]\n"
	}
	fm += "source:\n  class: session_memory\n---\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(fm+body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// noteJSON is a machine-layer note recalled everywhere (root null).
func noteJSON(id, title, body string, recall ...string) string {
	return fmt.Sprintf(`{"id": %q, "title": %q, "body": %q, "recall": %s, "root": null, "origin": "agent",
	"written_by": "Example:model-1", "written_at": "2026-10-11T12:00:00Z", "session": null}`, id, title, body, jsonList(recall))
}

// tiedNoteJSON is a machine-layer note tied to the repository whose root
// commit is root.
func tiedNoteJSON(id, title, body, root string, recall ...string) string {
	return fmt.Sprintf(`{"id": %q, "title": %q, "body": %q, "recall": %s, "root": %q, "origin": "agent",
	"written_by": "Example:model-1", "written_at": "2026-10-11T12:00:00Z", "session": null}`, id, title, body, jsonList(recall), root)
}

// projectNoteJSON is a note in the project's file, which carries no root.
func projectNoteJSON(id, title, body string, recall ...string) string {
	return fmt.Sprintf(`{"id": %q, "title": %q, "body": %q, "recall": %s, "origin": "agent",
	"written_by": "Example:model-1", "written_at": "2026-10-11T12:00:00Z", "session": null}`, id, title, body, jsonList(recall))
}

func jsonList(words []string) string {
	b, err := json.Marshal(words)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func readAndRecall(t *testing.T, home, repo, prompt string, ledger map[string]string) Result {
	t.Helper()
	items, refused := ReadItems(home, repo)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v", refused)
	}
	return Recall(Request{Items: items, Prompt: prompt, Ledger: ledger, Budget: rules.InjectionBudgetBytes})
}

// TestAPageDeclaringAuthComesBackAndNoOther (P1): a page declaring
// recall: [auth, login] comes back on a prompt containing "auth", and no
// other page does.
func TestAPageDeclaringAuthComesBackAndNoOther(t *testing.T) {
	home, repo, workspace, machine := pagesFixture(t)
	writePage(t, workspace, "fact_eng_auth-flow.md", "# Auth uses short-lived tokens\n\nRefresh them before the hour is out.\n", "auth", "login")
	writePage(t, workspace, "fact_eng_billing.md", "# Billing runs nightly\n\nThe auth of the billing job is separate.\n", "billing", "invoice")
	writePage(t, machine, "pref_eng_editor.md", "# Prefer short commit subjects\n", "commit")

	res := readAndRecall(t, home, repo, "why does auth keep failing?", nil)
	if !slices.Equal(res.Recalled, []string{"page:fact_eng_auth-flow.md"}) {
		t.Fatalf("recalled %v, want only the auth page\n%s", res.Recalled, res.Text)
	}
	for _, want := range []string{"## MEMORY", "page:fact_eng_auth-flow.md", "Auth uses short-lived tokens", "Refresh them before the hour is out."} {
		if !strings.Contains(res.Text, want) {
			t.Fatalf("the block does not carry %q:\n%s", want, res.Text)
		}
	}
	for _, not := range []string{"Billing runs nightly", "Prefer short commit subjects"} {
		if strings.Contains(res.Text, not) {
			t.Fatalf("a page that does not declare auth came back (%q):\n%s", not, res.Text)
		}
	}
}

// TestRepeatedPromptInjectsMemoryOnce (P3): the same prompt repeated within
// a session injects the matched page once; the loader's backstop prompt,
// which clears both ledgers, brings it back.
func TestRepeatedPromptInjectsMemoryOnce(t *testing.T) {
	home, repo, workspace, _ := pagesFixture(t)
	writePage(t, workspace, "fact_eng_auth-flow.md", "# Auth uses short-lived tokens\n", "auth")
	rs := rules.RuleSet{SchemaVersion: 1, Domains: map[string]rules.Domain{}}
	const prompt = "fix the auth bug"
	const backstop = 3

	var state rules.SessionState
	var sent []int
	for turn := 1; turn <= backstop; turn++ {
		inj := rules.Inject(rs, prompt, state, backstop)
		res := readAndRecall(t, home, repo, prompt, inj.State.Recalled)
		inj.State.Recalled = res.Ledger
		state = inj.State
		if strings.Contains(res.Text, "Auth uses short-lived tokens") {
			sent = append(sent, turn)
		}
	}
	if !slices.Equal(sent, []int{1, backstop}) {
		t.Fatalf("the page came back on turns %v, want the first and the backstop's %v", sent, []int{1, backstop})
	}
}

// TestWorkspacePageSuppressesTheMachinePage (P4): a recall word that hits a
// page of the same name in both scopes brings the workspace page back and
// suppresses the machine page.
func TestWorkspacePageSuppressesTheMachinePage(t *testing.T) {
	home, repo, workspace, machine := pagesFixture(t)
	writePage(t, workspace, "fact_eng_auth-flow.md", "# The project's auth page\n", "auth")
	writePage(t, machine, "fact_eng_auth-flow.md", "# The person's own auth page\n", "auth")
	writePage(t, machine, "fact_eng_auth-keys.md", "# Where the person keeps auth keys\n", "auth")

	res := readAndRecall(t, home, repo, "rotate the auth tokens", nil)
	want := []string{"page:fact_eng_auth-flow.md", "page:fact_eng_auth-keys.md"}
	if !slices.Equal(slices.Sorted(slices.Values(res.Recalled)), want) {
		t.Fatalf("recalled %v, want %v", res.Recalled, want)
	}
	if !strings.Contains(res.Text, "The project's auth page") || strings.Contains(res.Text, "The person's own auth page") {
		t.Fatalf("the workspace page did not win over the machine page of the same name:\n%s", res.Text)
	}
	if !strings.Contains(res.Text, "Where the person keeps auth keys") {
		t.Fatalf("a machine page with no workspace twin was suppressed:\n%s", res.Text)
	}
}

// TestPageWithoutRecallIsNeverACandidate: a page with no recall: list is
// silent whatever the prompt says, even its own words.
func TestPageWithoutRecallIsNeverACandidate(t *testing.T) {
	home, repo, workspace, machine := pagesFixture(t)
	writePage(t, workspace, "fact_eng_auth-flow.md", "# Auth uses short-lived tokens\n")
	writePage(t, machine, "fact_eng_auth-keys.md", "# Auth keys live in the keychain\n")

	items, _ := ReadItems(home, repo)
	if len(items) != 0 {
		t.Fatalf("a page with no recall list is an item: %+v", items)
	}
	if res := readAndRecall(t, home, repo, "auth tokens short-lived keychain", nil); res.Text != "" {
		t.Fatalf("a page with no recall list came back:\n%s", res.Text)
	}
}

// TestPageItemsCarryTheirIngestDate: a page item's At, the ranking's recency
// tie-break, is the page's latest ingested_at.
func TestPageItemsCarryTheirIngestDate(t *testing.T) {
	home, repo, workspace, _ := pagesFixture(t)
	if err := os.WriteFile(filepath.Join(workspace, "fact_eng_auth-flow.md"),
		[]byte("---\nrecall: [auth]\nsource:\n  class: session_memory\n  ingested_at: 2026-10-01\n---\n# Auth\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	items, _ := ReadItems(home, repo)
	if len(items) != 1 || items[0].At.Format("2006-01-02") != "2026-10-01" {
		t.Fatalf("items = %+v", items)
	}
}
