package cli

import (
	"strings"
	"testing"
)

// TestLibraryPageCarriesTheConsultAndIngestWork: the judgement halves of the
// retired consult and ingest pages live on the library page (step 6), and the
// page says plainly which sub-verb registers a document, so no reader takes
// the memory store's ingest for it (iss-2610110233253365).
func TestLibraryPageCarriesTheConsultAndIngestWork(t *testing.T) {
	bodies := commandFileBodies(t)
	for _, gone := range []string{"consult", "ingest"} {
		if _, ok := bodies[gone]; ok {
			t.Errorf("commands/%s.md still exists; its work folded into commands/library.md", gone)
		}
	}
	page := strings.Join(strings.Fields(bodies["library"]), " ")
	for _, want := range []string{
		"### Search the corpus",
		"### Read the document first",
		"### Decide the class and the key",
		"### Quality-check the extraction",
		"identifying paraphrase",
		"`library add` registers a document",
		"`library ingest` registers nothing",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("commands/library.md does not carry %q", want)
		}
	}
}
