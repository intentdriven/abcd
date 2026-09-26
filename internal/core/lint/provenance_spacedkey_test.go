package lint

import (
	"path/filepath"
	"testing"
)

// A command-written key hand-spelled with a space before its colon (`held :`)
// is normalised by the scanner and honoured by every reader, but no write path
// produces it, and `intent unhold`'s remover matches only the exact spelling.
// record_provenance reports the spelling for each key it judges as a command's
// write — the hold and the disclosure pair (iss-2609210748122003).
func TestRecordProvenanceReportsACommandKeySpelledWithASpaceBeforeTheColon(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/drafts/itd-1-held.md",
		"---\nid: itd-1\nkind: null\nspec_id: null\nheld : \"awaiting the rethink\"\n---\n# draft\n")
	writeFile(t, root, "rec/intents/drafts/itd-2-origin.md",
		"---\nid: itd-2\nkind: null\nspec_id: null\norigin\t: researcher-authored\nproduction_mode: hand-written\n---\n# draft\n")
	writeFile(t, root, "rec/intents/drafts/itd-3-legal.md",
		"---\nid: itd-3\nkind: null\nspec_id: null\nheld: \"awaiting the rethink\"\norigin: researcher-authored\nproduction_mode: hand-written\n---\n# draft\n")
	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-1-held.md"), ruleRecordProvenance, "space before its colon") {
		t.Errorf("`held :` not reported: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-2-origin.md"), ruleRecordProvenance, "space before its colon") {
		t.Errorf("`origin<TAB>:` not reported: %+v", fs)
	}
	for _, f := range fs {
		if f.File == filepath.Join("rec/intents/drafts", "itd-3-legal.md") {
			t.Errorf("the verbs' own spelling was reported: %+v", f)
		}
	}
}
