package recall

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/rules"
)

var update = flag.Bool("update", false, "rewrite testdata/ranking.golden from this run's scores")

// item is a test item: a note-shaped key in the project's layer, written at
// the zero time, unless the caller changes it.
func item(key string, recall []string, title, body string) Item {
	return Item{Key: key, Kind: KindNote, Layer: LayerProject, Title: title, Body: body, Recall: recall}
}

func rankedKeys(rs []Ranked) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Item.Key)
	}
	return out
}

// TestRareMatchingWordOutranksACommonOne: the case a raw count of matching
// recall words gets wrong. Every item shares "deploy" and "service"; one
// alone carries "kerberos". The prompt names all three, so a count puts the
// two-word matches first, while the ranking puts the rare word's item first.
func TestRareMatchingWordOutranksACommonOne(t *testing.T) {
	items := []Item{
		item("mem-2610111200000001", []string{"deploy", "service"}, "Deploys roll one service at a time", "Wait for health checks."),
		item("mem-2610111200000002", []string{"deploy", "service"}, "A service deploy needs a ticket", "File it first."),
		item("mem-2610111200000003", []string{"deploy", "service"}, "Deploy windows are mornings", "Never on a Friday."),
		item("mem-2610111200000004", []string{"deploy", "service"}, "Service logs rotate daily", "Grab them early."),
		item("mem-2610111200000005", []string{"deploy", "service"}, "Each service owns its deploy script", "Keep it in the repo."),
		item("mem-2610111200000006", []string{"kerberos"}, "Kerberos tickets expire in ten hours", "Renew before the long job."),
	}
	const prompt = "deploy the service behind kerberos"

	// The fixture is the case a count gets wrong: by matching words, the
	// kerberos item is last.
	count := slices.Clone(candidates(items, prompt))
	slices.SortStableFunc(count, func(a, b Candidate) int { return len(b.Hits) - len(a.Hits) })
	if count[0].Item.Key == "mem-2610111200000006" {
		t.Fatal("the fixture does not make a raw count put the common words first")
	}

	ranked := Rank(items, prompt)
	if len(ranked) != len(items) || ranked[0].Item.Key != "mem-2610111200000006" {
		t.Fatalf("ranked %v, want the rare word's item first", rankedKeys(ranked))
	}
	if ranked[0].Score <= ranked[1].Score {
		t.Fatalf("the rare word's item does not outscore the rest: %v vs %v", ranked[0].Score, ranked[1].Score)
	}
}

// TestRecallFieldOutweighsBody: a term found in an item's recall words
// counts three times, in its title twice, in its body once. Three items of
// the same weighted length differ only in where "auth" is found.
func TestRecallFieldOutweighsBody(t *testing.T) {
	items := []Item{
		item("mem-2610111200000001", []string{"token"}, "", "auth filler filler filler"), // body: 3 + 4 = 7
		item("mem-2610111200000002", []string{"token"}, "auth", "filler filler"),         // title: 3 + 2 + 2 = 7
		item("mem-2610111200000003", []string{"token", "auth"}, "", "filler"),            // recall: 6 + 1 = 7
	}
	ranked := Rank(items, "the auth token")
	want := []string{"mem-2610111200000003", "mem-2610111200000002", "mem-2610111200000001"}
	if got := rankedKeys(ranked); !slices.Equal(got, want) {
		t.Fatalf("ranked %v, want recall over title over body %v", got, want)
	}
	if !(ranked[0].Score > ranked[1].Score && ranked[1].Score > ranked[2].Score) {
		t.Fatalf("scores do not fall with the field's weight: %v %v %v", ranked[0].Score, ranked[1].Score, ranked[2].Score)
	}
}

// TestTiesBreakOnScopeThenRecencyThenID: items of equal score come in the
// narrower scope first, then the more recent, then by key.
func TestTiesBreakOnScopeThenRecencyThenID(t *testing.T) {
	older := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	mk := func(key, layer string, at time.Time) Item {
		it := item(key, []string{"auth"}, "The same lesson", "The same words.")
		it.Layer, it.At = layer, at
		return it
	}
	items := []Item{
		mk("mem-2610111200000001", LayerMachine, newer),
		mk("mem-2610111200000004", LayerProject, older),
		mk("mem-2610111200000003", LayerProject, newer),
		mk("mem-2610111200000002", LayerProject, newer),
		mk("mem-2610111200000005", LayerProject, time.Time{}),
	}
	ranked := Rank(items, "auth")
	for _, r := range ranked[1:] {
		if r.Score != ranked[0].Score {
			t.Fatalf("the fixture's scores differ, so it tests no tie: %v", ranked)
		}
	}
	want := []string{"mem-2610111200000002", "mem-2610111200000003", "mem-2610111200000004", "mem-2610111200000005", "mem-2610111200000001"}
	if got := rankedKeys(ranked); !slices.Equal(got, want) {
		t.Fatalf("ranked %v, want %v", got, want)
	}
}

// TestRankingIsDeterministicAcrossRuns: the same items in any order give the
// same ranking, scores and reasons included.
func TestRankingIsDeterministicAcrossRuns(t *testing.T) {
	items := goldenCorpus()
	const prompt = "the auth tokens expire before the deploy"
	want := fmt.Sprint(Rank(items, prompt))
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 50; i++ {
		shuffled := slices.Clone(items)
		r.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		if got := fmt.Sprint(Rank(shuffled, prompt)); got != want {
			t.Fatalf("run %d ranked differently:\n%s\n---\n%s", i, got, want)
		}
	}
}

// TestRankingDecidesWhichItemsFitTheBudget: when only one of two candidates
// fits, the higher-ranked survives, not the one first in id order.
func TestRankingDecidesWhichItemsFitTheBudget(t *testing.T) {
	filler := strings.TrimSpace(strings.Repeat("words ", 150))
	items := []Item{
		item("mem-2610111200000001", []string{"deploy"}, "A deploy lesson", filler),
		item("mem-2610111200000002", []string{"deploy", "kerberos"}, "A kerberos deploy lesson", filler),
	}
	budget := len(headerLine) + len(renderItem(items[1])) + 10
	res := Recall(Request{Items: items, Prompt: "deploy with kerberos", Budget: budget})
	if !slices.Equal(res.Recalled, []string{"mem-2610111200000002"}) || !slices.Equal(res.Omitted, []string{"mem-2610111200000001"}) {
		t.Fatalf("recalled %v omitted %v, want the higher-ranked item to fit", res.Recalled, res.Omitted)
	}
	// With room for both, they come back in ranked order.
	all := Recall(Request{Items: items, Prompt: "deploy with kerberos", Budget: rules.InjectionBudgetBytes})
	if !slices.Equal(all.Recalled, []string{"mem-2610111200000002", "mem-2610111200000001"}) {
		t.Fatalf("recalled %v, want ranked order", all.Recalled)
	}
}

// TestRankReasonsAddUpToTheScore: each matched term names the fields it was
// found in, and the contributions sum to the score.
func TestRankReasonsAddUpToTheScore(t *testing.T) {
	for _, r := range Rank(goldenCorpus(), "the auth tokens expire before the deploy") {
		sum := 0.0
		for _, ts := range r.Terms {
			if ts.Recall+ts.Title+ts.Body == 0 || ts.Contribution <= 0 {
				t.Fatalf("%s: a term with no field or no contribution: %+v", r.Item.Key, ts)
			}
			sum += ts.Contribution
		}
		if math.Abs(sum-r.Score) > 1e-12 {
			t.Fatalf("%s: contributions sum to %v, score %v", r.Item.Key, sum, r.Score)
		}
	}
}

// goldenCorpus is a small fixed corpus of notes and pages.
func goldenCorpus() []Item {
	at := func(s string) time.Time {
		tm, err := time.Parse(time.RFC3339, s)
		if err != nil {
			panic(err)
		}
		return tm
	}
	return []Item{
		{Key: "mem-2610111200000001", Kind: KindNote, Layer: LayerMachine, At: at("2026-10-11T12:00:00Z"),
			Title: "Auth tokens expire hourly", Body: "Refresh the token before the hour is out.", Recall: []string{"auth", "token"}},
		{Key: "mem-2610111200000002", Kind: KindNote, Layer: LayerProject, At: at("2026-10-10T09:30:00Z"),
			Title: "Deploys need the dry-run flag", Body: "Pass --dry-run to the deploy script first.", Recall: []string{"deploy"}},
		{Key: "mem-2610111200000003", Kind: KindNote, Layer: LayerMachine, At: at("2026-10-09T08:00:00Z"),
			Title: "The tests need the race flag", Body: "Run the race-enabled tests before saying they pass.", Recall: []string{"test", "race"}},
		{Key: "page:fact_eng_auth-flow.md", Kind: KindPage, Layer: LayerProject, At: at("2026-10-01T00:00:00Z"),
			Title: "Auth uses short-lived tokens", Body: "Tokens expire after an hour; the deploy rotates the signing key.", Recall: []string{"auth", "login"}},
		{Key: "page:fact_eng_deploy-window.md", Kind: KindPage, Layer: LayerMachine, At: at("2026-09-20T00:00:00Z"),
			Title: "Deploy windows", Body: "Deploy in the morning, never before a holiday.", Recall: []string{"deploy", "release"}},
	}
}

// TestRankingGoldenScores: the scores of a small fixed corpus, so a change to
// the formula, its weights or its parameters is a visible diff of
// testdata/ranking.golden (go test -run TestRankingGoldenScores -update).
func TestRankingGoldenScores(t *testing.T) {
	var b strings.Builder
	for _, prompt := range []string{
		"the auth tokens expire before the deploy",
		"deploy the release",
		"run the tests with the race detector",
		"login fails after the token refresh",
	} {
		fmt.Fprintf(&b, "prompt: %s\n", prompt)
		for _, r := range Rank(goldenCorpus(), prompt) {
			fmt.Fprintf(&b, "  %s %.6f\n", r.Item.Key, r.Score)
			for _, ts := range r.Terms {
				fmt.Fprintf(&b, "    %s recall=%d title=%d body=%d %.6f\n", ts.Term, ts.Recall, ts.Title, ts.Body, ts.Contribution)
			}
		}
	}
	golden := filepath.Join("testdata", "ranking.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update to write it)", err)
	}
	if b.String() != string(want) {
		t.Fatalf("the ranking's scores changed; if the change is meant, run with -update and review the diff:\n%s", b.String())
	}
}

// TestRankingFindsATermWhereTheMatcherHits: a single-word recall word is
// found among the prompt's query terms exactly when the loader's term rule
// hits it, so the ranking weighs the words the matcher matched and no others.
func TestRankingFindsATermWhereTheMatcherHits(t *testing.T) {
	words := []string{"test", "tests", "merge", "commit", "attest", "docs", "push", "release"}
	prompts := []string{
		"the tests keep failing", "merging the branch", "an attestation", "committing the fix",
		"pushes to the remote", "update the docs", "testing the release", "paint a landscape",
	}
	for _, p := range prompts {
		hit := rules.TermMatcher(p)
		terms := rules.PromptTerms(p)
		for _, w := range words {
			stems := rules.TextStems(w)
			found := false
			for _, term := range terms {
				if len(stems) == 1 && slices.Contains(term.Roots, stems[0]) {
					found = true
				}
			}
			if found != hit(w) {
				t.Errorf("%q in %q: found among the query terms = %v, the term rule hits = %v", w, p, found, hit(w))
			}
		}
	}
}

// TestAnItemLargerThanTheWholeBudgetDoesNotBlockTheRest: an item whose block
// alone is larger than the whole per-turn budget can never come back in
// full, so it is set aside as too large, never recorded, and the items
// ranked below it still come back; one that only misses this turn's
// remainder still stops the take, as before.
func TestAnItemLargerThanTheWholeBudgetDoesNotBlockTheRest(t *testing.T) {
	huge := strings.TrimSpace(strings.Repeat("kerberos deploy notes ", rules.InjectionBudgetBytes/20))
	items := []Item{
		item("page:fact_eng_huge.md", []string{"deploy", "kerberos"}, "Every kerberos deploy ever", huge),
		item("mem-2610111200000002", []string{"deploy"}, "Deploys need the dry-run flag", "Pass it first."),
	}
	ranked := Rank(items, "deploy with kerberos")
	if ranked[0].Item.Key != "page:fact_eng_huge.md" {
		t.Fatalf("the fixture does not rank the huge page first: %v", rankedKeys(ranked))
	}
	res := Recall(Request{Items: items, Prompt: "deploy with kerberos", Budget: rules.InjectionBudgetBytes})
	if !slices.Equal(res.Recalled, []string{"mem-2610111200000002"}) || !slices.Equal(res.TooLarge, []string{"page:fact_eng_huge.md"}) ||
		len(res.Omitted) != 0 {
		t.Fatalf("recalled %v too large %v omitted %v", res.Recalled, res.TooLarge, res.Omitted)
	}
	if _, ok := res.Ledger["page:fact_eng_huge.md"]; ok {
		t.Fatal("an item that never came back was recorded as sent")
	}
	if len(res.Text) > rules.InjectionBudgetBytes {
		t.Fatalf("the block is over the budget: %d", len(res.Text))
	}
	// With the rules leaving too little for the note this turn, the note is
	// omitted and tried again, and the huge page is still only too large.
	tight := Recall(Request{Items: items, Prompt: "deploy with kerberos", Budget: 10})
	if len(tight.Recalled) != 0 || !slices.Equal(tight.Omitted, []string{"mem-2610111200000002"}) || !slices.Equal(tight.TooLarge, []string{"page:fact_eng_huge.md"}) {
		t.Fatalf("recalled %v too large %v omitted %v", tight.Recalled, tight.TooLarge, tight.Omitted)
	}
}
