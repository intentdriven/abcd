package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// TestBodyMentionOfADecisionIsAMentionEdge pins the site's half of step 1
// (spc-2610110931381316): a record body naming a decision record by its dcn
// handle, in any spelling, relates the two records the way a body naming an
// issue does. Where decision nodes are published is step 5's question; whether
// a mention of one is read is this one's.
func TestBodyMentionOfADecisionIsAMentionEdge(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		".abcd/work/decisions/dcn-2610141200001111-a-ruling.md": "---\nid: dcn-2610141200001111\n---\n2026-10-14 — a ruling\n",
		".abcd/work/issues/open/iss-7-cites-it.md":              "---\nid: iss-7\n---\nThe ruling DCN-02610141200001111 settled it.\n",
	}
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	nodes := []lint.RecordNode{
		{ID: "dcn-2610141200001111", Type: "decision", Path: ".abcd/work/decisions/dcn-2610141200001111-a-ruling.md"},
		{ID: "iss-7", Type: "issue", Lifecycle: "open", Path: ".abcd/work/issues/open/iss-7-cites-it.md"},
	}
	index := map[string]int{"dcn-2610141200001111": 0, "iss-7": 1}
	mentions, _, err := scanMentions(root, nodes, index, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mentions) != 1 {
		t.Fatalf("mentions = %+v; want the issue's mention of the decision", mentions)
	}
	m := mentions[0]
	if !(m.From == "dcn-2610141200001111" && m.To == "iss-7") && !(m.From == "iss-7" && m.To == "dcn-2610141200001111") {
		t.Errorf("mention = %+v; want it between the decision and the issue", m)
	}
}
