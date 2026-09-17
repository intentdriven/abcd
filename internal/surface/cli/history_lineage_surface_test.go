package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/history"
)

// history_lineage_surface_test.go — iss-2609091915475296.
//
// Reaching a session's sub-agents from the session identifier worked in the core
// and had no operator surface. `ListForSession` existed and was tested, `show`
// deliberately returns the main-thread record alone, and the human render of
// `list`/`show` printed neither the agent identifier nor the agent type — so the
// only route to a session's whole set was `list --json` plus hand-filtering on a
// field the plugin page did not document. By this repository's "wired or it isn't
// done" rule that is a gap, and it is closed at the verb the core seam belongs
// to: `list` is the set verb, so `list --session <id>` is ListForSession's front
// door, and both renders now name which agent produced each record.

// lineageStore captures one session's spine plus two sub-agents of it, and a
// record of a DIFFERENT session, so a set query can be seen to select.
func lineageStore(t *testing.T) (repo, rootSHA string) {
	t.Helper()
	repo, rootSHA = sessionEndRepo(t)
	t.Chdir(repo)
	for _, rec := range []history.CaptureMeta{
		{SessionID: "sess-lineage", Kind: "native"},
		{SessionID: "sess-lineage", Kind: "native", AgentID: "agent-one",
			AgentType: "ruthless-reviewer", SpawnDepth: 1, LineageSource: "hook",
			SpawnAttribution: "sidecar"},
		{SessionID: "sess-lineage", Kind: "native", AgentID: "agent-two",
			AgentType: "sota-researcher", SpawnDepth: 1, LineageSource: "hook",
			SpawnAttribution: "sidecar"},
		{SessionID: "sess-other", Kind: "native"},
	} {
		body := "assistant: " + rec.SessionID + " " + rec.AgentID + "\n"
		if _, err := history.Capture(repo, rootSHA, []byte(body), rec); err != nil {
			t.Fatalf("Capture %+v: %v", rec, err)
		}
	}
	return repo, rootSHA
}

// TestHistoryListSessionReachesTheWholeSessionSet is the headline: from the
// session identifier alone, an operator gets the spine and every sub-agent, and
// nothing belonging to another session.
func TestHistoryListSessionReachesTheWholeSessionSet(t *testing.T) {
	lineageStore(t)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"history", "list", "--session", "sess-lineage", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history list --session exited %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var got []history.Record
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("--json listing is not an array of records: %v\n%s", err, stdout.String())
	}
	if len(got) != 3 {
		t.Fatalf("want the spine and both sub-agents (3 records), got %d: %+v", len(got), got)
	}
	// The main thread leads, because the branches are only legible against it.
	if got[0].AgentID != "" {
		t.Errorf("the main-thread record must lead the set, got agent %q first", got[0].AgentID)
	}
	agents := map[string]string{}
	for _, r := range got {
		if r.SessionID != "sess-lineage" {
			t.Errorf("another session's record reached the set: %+v", r)
		}
		if r.AgentID != "" {
			agents[r.AgentID] = r.AgentType
		}
	}
	if agents["agent-one"] != "ruthless-reviewer" || agents["agent-two"] != "sota-researcher" {
		t.Errorf("the set must carry each sub-agent with its type, got %v", agents)
	}
}

// TestHistoryListHumanRenderNamesTheAgentAndItsType is the other half of the
// gap: a person reading the listing could not tell a sub-agent's record from the
// spine, let alone what kind of agent produced it.
func TestHistoryListHumanRenderNamesTheAgentAndItsType(t *testing.T) {
	lineageStore(t)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"history", "list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history list exited %d\nstderr: %s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"agent-one", "ruthless-reviewer", "agent-two", "sota-researcher"} {
		if !strings.Contains(out, want) {
			t.Errorf("the human listing never names %q:\n%s", want, out)
		}
	}
}

// TestHistoryShowHumanRenderNamesTheAgentAndItsType: `show` still returns ONE
// record — that is deliberate — but a reader must be able to tell which agent's
// record they are holding.
func TestHistoryShowHumanRenderNamesTheAgentAndItsType(t *testing.T) {
	lineageStore(t)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"history", "show", "agent-one"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history show exited %d\nstderr: %s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"agent-one", "ruthless-reviewer"} {
		if !strings.Contains(out, want) {
			t.Errorf("the shown record never names %q:\n%s", want, out)
		}
	}
}

// TestHistoryListSessionSaysSoWhenTheSessionIsUnknown: an empty set for a NAMED
// session is a question that got no answer, so the render names the session. The
// plain `list` may legitimately be empty — a repo simply has no transcripts — and
// a mistyped identifier must not read as that. It is not a refusal: an operator
// asking about a session the store has never seen has asked a legal question.
func TestHistoryListSessionSaysSoWhenTheSessionIsUnknown(t *testing.T) {
	lineageStore(t)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"history", "list", "--session", "sess-nope"}, &stdout, &stderr); code != 0 {
		t.Fatalf("an unknown session is an empty listing, not a refusal: exit %d\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "sess-nope") {
		t.Errorf("an empty set must name the session it found nothing for:\n%s", stdout.String())
	}
}

// TestHistoryPluginPageDocumentsTheSessionSet holds the second front door. The
// capability is not delivered until the plugin markdown reaches it, and the
// record's own complaint was that the JSON route was undocumented.
func TestHistoryPluginPageDocumentsTheSessionSet(t *testing.T) {
	page := filepath.Join(testRepoRoot(), pluginCommandsDir, "history.md")
	body, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	// Scoped to the List section, because `agent_id` and `agent_type` already
	// appear under Staged: a page-wide substring match would pass on a page that
	// documents the staging lane and says nothing about reaching a stored set.
	section := sectionOf(t, string(body), "## List")
	for _, want := range []string{"--session", "agent_id", "agent_type"} {
		if !strings.Contains(section, want) {
			t.Errorf("the List section of commands/history.md never mentions %q, so the plugin front door cannot reach a session's sub-agents:\n%s", want, section)
		}
	}
}

// sectionOf returns one `## `-delimited section of a markdown page, heading
// included.
func sectionOf(t *testing.T, body, heading string) string {
	t.Helper()
	start := strings.Index(body, heading+"\n")
	if start < 0 {
		t.Fatalf("the page has no %q section", heading)
	}
	rest := body[start+len(heading):]
	if end := strings.Index(rest, "\n## "); end >= 0 {
		return heading + rest[:end]
	}
	return heading + rest
}
