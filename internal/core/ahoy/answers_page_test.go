package ahoy

import (
	"slices"
	"strings"
	"testing"
)

// The words that bound the ahoy page's answers-file paragraph: it opens on
// the file's shape and ends where the page says what --answers replaces.
const (
	answersParagraphOpens = "The file is one JSON object:"
	answersParagraphEnds  = "replaces the piped answer stream"
	runDecidesOpens       = "The questions the run itself decides to ask ("
)

// answersParagraph returns the ahoy page's paragraph on the answers file,
// its line breaks folded so a phrase the page wraps still matches.
func answersParagraph(t *testing.T) string {
	t.Helper()
	page := readAhoyPage(t)
	start := strings.Index(page, answersParagraphOpens)
	if start < 0 {
		t.Fatalf("%s has no paragraph opening %q", ahoyPage, answersParagraphOpens)
	}
	rest := page[start:]
	end := strings.Index(rest, answersParagraphEnds)
	if end < 0 {
		t.Fatalf("%s's answers-file paragraph does not end on %q", ahoyPage, answersParagraphEnds)
	}
	return strings.Join(strings.Fields(rest[:end]), " ")
}

// fileAnswersConfirm reports whether an answers file may answer the approval
// id: every approval setupConfirmIDs declares but the ones no run puts to a
// file. Those are the offers put only to a person at a terminal
// (terminalOnlyGapIDs), which the answers file's prompter never is; the
// commit identity, which stepIdentityEstablish refuses off a terminal; and
// the GitHub settings change, which only `ahoy remote apply` asks, a verb
// that takes no answers file.
func fileAnswersConfirm(id string) bool {
	return id != "git_identity.establish" && id != "remote.apply" && !slices.Contains(terminalOnlyGapIDs, id)
}

// TestAhoyPageNamesEveryApprovalAFileMayAnswer holds the ahoy page to
// setupConfirmIDs (iss-2610241550154898): the page tells a host which ids an
// answers file may carry, so an approval a file may answer and the page does
// not name is one the host cannot answer, and the run stops on it.
func TestAhoyPageNamesEveryApprovalAFileMayAnswer(t *testing.T) {
	page := readAhoyPage(t)
	answerable := 0
	for _, c := range setupConfirmIDs {
		if !fileAnswersConfirm(c.id) {
			continue
		}
		answerable++
		if !strings.Contains(page, "`"+c.id+"`") {
			t.Errorf("%s does not name `%s`, an approval an answers file may answer (setupConfirmIDs)", ahoyPage, c.id)
		}
	}
	if answerable == 0 {
		t.Fatal("no approval in setupConfirmIDs is one a file may answer; the test checks nothing")
	}
	if !fileAnswersConfirm("history.lineage") || !confirmIDKnown("history.lineage") {
		t.Fatal("history.lineage is no longer an approval a file may answer; the page's words on it need the same change")
	}
}

// TestAhoyPageSaysHowAFileAnswersTheLineageQuestion holds the answers-file
// paragraph to the re-founding lineage question: its id, each value a file
// may give it, and its place among the questions checked only when the run
// asks them.
func TestAhoyPageSaysHowAFileAnswersTheLineageQuestion(t *testing.T) {
	para := answersParagraph(t)
	i := strings.Index(para, "`history.lineage` answers")
	if i < 0 {
		t.Fatalf("%s's answers-file paragraph has no sentence opening \"`history.lineage` answers\"", ahoyPage)
	}
	sentences := para[i:]
	if end := strings.Index(sentences, ". The plan answers"); end >= 0 {
		sentences = sentences[:end]
	}
	values, ok := SetupFixedValues("history.lineage")
	if !ok {
		t.Fatal("SetupFixedValues does not know history.lineage")
	}
	for _, v := range values {
		if !strings.Contains(sentences, "`"+v+"`") {
			t.Errorf("%s's words on history.lineage do not name its value `%s`: %q", ahoyPage, v, sentences)
		}
	}
	j := strings.Index(para, runDecidesOpens)
	if j < 0 {
		t.Fatalf("%s's answers-file paragraph has no list opening %q", ahoyPage, runDecidesOpens)
	}
	list := para[j+len(runDecidesOpens):]
	if end := strings.Index(list, ")"); end >= 0 {
		list = list[:end]
	}
	if !strings.Contains(list, "`history.lineage`") {
		t.Errorf("%s does not name `history.lineage` among the questions the run itself decides to ask: %q", ahoyPage, list)
	}
}
