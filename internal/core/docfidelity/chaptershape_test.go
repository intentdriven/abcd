package docfidelity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRecordAndApplyAgreeOnTheChapterShape — iss-2609302306153318. Record
// checked only that a failing entry's chapter was non-empty, so it saved a HOLD
// naming `04-surfaces/17-guard.md`, which Apply's one-component name then
// refused: a verdict the recorder accepted could not be applied. Both now read
// the chapter through one parse, which admits the bare file name and the same
// name under its surfaces directory, and refuses anything else at record time.
func TestRecordAndApplyAgreeOnTheChapterShape(t *testing.T) {
	for _, chapter := range []string{"06-capture.md", "04-surfaces/06-capture.md", ChaptersDir + "/06-capture.md"} {
		t.Run(chapter, func(t *testing.T) {
			root := armedRepo(t)
			write(t, root, ChaptersDir+"/06-capture.md", "### `abcd capture`\n\n"+falseLine+" It lists.\n")
			git(t, root, "add", "-A")
			git(t, root, "commit", "-q", "-m", "c1")
			hold := `{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": [` +
				`{"doc": "brief", "chapter": "` + chapter + `", "sentence": "` + falseLine + `", "replacement": "Capture prints JSON.", "evidence": "cli.go:1", "disposition": "confirmed"}]}`
			if _, _, err := Record(root, []byte(hold), at); err != nil {
				t.Fatalf("Record: %v", err)
			}
			v, _, err := Gate(root, tree, []string{"itd-1"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(v.Proposed) != 1 {
				t.Fatalf("proposed edits = %+v", v.Proposed)
			}
			if _, err := Apply(root, v.Proposed, "c1", at); err != nil {
				t.Fatalf("Apply refused the edit Record accepted: %v", err)
			}
			data, _ := os.ReadFile(filepath.Join(root, ChaptersDir, "06-capture.md"))
			if !strings.Contains(string(data), "Capture prints JSON. It lists.") {
				t.Fatalf("the chapter was not edited:\n%s", data)
			}
			v, _, err = Gate(root, tree, []string{"itd-1"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if v.Refuse || len(v.Applied) != 1 {
				t.Fatalf("the applied and flagged edit still refuses: %+v", v.Reasons)
			}
		})
	}

	for _, chapter := range []string{"../06-capture.md", "brief/06-capture.md", "04-surfaces/sub/06-capture.md", "06-capture.txt", "/06-capture.md"} {
		t.Run("refused "+chapter, func(t *testing.T) {
			root := armedRepo(t)
			hold := `{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": [` +
				`{"doc": "brief", "chapter": "` + chapter + `", "sentence": "` + falseLine + `", "replacement": "Capture prints JSON.", "evidence": "cli.go:1", "disposition": "confirmed"}]}`
			if _, _, err := Record(root, []byte(hold), at); err == nil {
				t.Fatalf("Record accepted the chapter %q, which Apply cannot write", chapter)
			}
			if _, err := Apply(root, []Edit{{Chapter: chapter, Sentence: falseLine, Replacement: "x"}}, "c1", at); err == nil {
				t.Fatalf("Apply accepted the chapter %q", chapter)
			}
		})
	}
}
