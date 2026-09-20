package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/provenance"
)

// TestCreateFromTextSeedsDraft is the itd-46 AC1 core: a quoted-text create files
// a new drafts/itd-N-<slug>.md seeded from the text, with the canonical draft
// frontmatter set, and the seeded body carries the text.
func TestCreateFromTextSeedsDraft(t *testing.T) {
	root := t.TempDir()

	it, err := CreateFromText(root, "I want users to feel the card respects their time", TextOptions{})
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	if !nativeIntentIDRe.MatchString(it.ID) {
		t.Fatalf("minted id = %q, want a native itd-<yymmddHHMMSS><rrrr> id", it.ID)
	}
	if it.Bucket != BucketDrafts {
		t.Fatalf("bucket = %q, want drafts", it.Bucket)
	}
	if !slugRe.MatchString(it.Slug) {
		t.Fatalf("slug %q is not kebab-case", it.Slug)
	}
	if err := Validate(it); err != nil {
		t.Fatalf("created intent fails Validate: %v", err)
	}
	abs := filepath.Join(root, it.Path)
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("created file unreadable: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "I want users to feel the card respects their time") {
		t.Fatalf("seeded body missing the quoted text:\n%s", body)
	}
	// Canonical draft frontmatter: spec_id null, kind null/standalone/bundle-member.
	fields := frontmatter.Fields(strings.Split(body, "\n"))
	if fields["id"].Value != it.ID {
		t.Fatalf("frontmatter id = %q, want %s", fields["id"].Value, it.ID)
	}
	if !frontmatter.IsNull(fields["spec_id"].Value) {
		t.Fatalf("drafts spec_id must be null, got %q", fields["spec_id"].Value)
	}
}

// TestCreateFromTextRefusesEmpty proves empty/whitespace text is refused and
// nothing is written (unrecognized/empty input never writes).
func TestCreateFromTextRefusesEmpty(t *testing.T) {
	root := t.TempDir()
	for _, in := range []string{"", "   ", "\t\n"} {
		if _, err := CreateFromText(root, in, TextOptions{}); err == nil {
			t.Fatalf("CreateFromText(%q) must be refused", in)
		}
	}
	// No drafts file appeared.
	if entries, _ := os.ReadDir(filepath.Join(root, draftsDir)); len(entries) != 0 {
		t.Fatalf("empty-text create wrote %d files, want 0", len(entries))
	}
}

// TestCreateFromTextRedactsSecretsAndHomePaths proves gh-486: a quoted-text
// create must route the caller's text through the ONE canonical scanner before
// it persists, so a secret token or an absolute home path in the intent text
// never lands verbatim in the committed draft — neither in the body nor in the
// derived filename/slug. The spans below are FAKE shapes (a syntactically valid
// ghp_ PAT of 36 chars and a non-caller /Users/<name> path), matched only by
// shape, never real credentials.
func TestCreateFromTextRedactsSecretsAndHomePaths(t *testing.T) {
	root := t.TempDir()

	const fakeToken = "ghp_" + "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // ghp_ + 36 A's
	const fakeHome = "/Users/alice/.ssh/id_rsa"
	text := "leftover " + fakeToken + " and " + fakeHome + " in the install receipt"

	it, err := CreateFromText(root, text, TextOptions{})
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}

	abs := filepath.Join(root, it.Path)
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("created file unreadable: %v", err)
	}
	body := string(data)

	if strings.Contains(body, fakeToken) {
		t.Errorf("draft body persisted the raw secret token verbatim:\n%s", body)
	}
	if strings.Contains(body, "/Users/alice") {
		t.Errorf("draft body persisted the raw home path verbatim:\n%s", body)
	}
	// The slug is derived from the same text and becomes the filename: it must
	// not carry the leaked spans either (the capture-slug leak shape).
	if strings.Contains(it.Path, "users-alice") || strings.Contains(it.Slug, "users-alice") {
		t.Errorf("derived filename/slug leaked the home path: path=%q slug=%q", it.Path, it.Slug)
	}
	// The masked fingerprint deliberately keeps the token's first three runes for
	// triage, so "ghp" surviving in the slug is the REDACTED form — what must not
	// persist is the raw span (the full token, or a long run of its body).
	rawRun := strings.Repeat("a", 8) // 8+ of the token's 36 A's would mean it slipped through
	if strings.Contains(it.Path, fakeToken) || strings.Contains(it.Slug, rawRun) {
		t.Errorf("derived filename/slug leaked the raw secret token: path=%q slug=%q", it.Path, it.Slug)
	}
}

// TestCreateFromTextPassesRecordLint runs the real intent_lifecycle record-lint
// over a freshly seeded draft — the "abcd lint stays green" guarantee.
func TestCreateFromTextPassesRecordLint(t *testing.T) {
	root := t.TempDir()
	if _, err := CreateFromText(root, "seeded from a quoted-text capture", TextOptions{}); err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, fnd := range findings {
		if fnd.RuleID == "intent_lifecycle" {
			t.Fatalf("seeded draft violates intent_lifecycle: %s:%d %s", fnd.File, fnd.Line, fnd.Message)
		}
	}
}

// TestCreateDraftPromotedFromRoundTrip: a draft minted through the promote
// path carries the promoted_from back-edge in its frontmatter, and the intent
// reader parses it back (two-sided edge, spc-24).
func TestCreateDraftPromotedFromRoundTrip(t *testing.T) {
	root := t.TempDir()

	it, err := CreateDraft(root, DraftOptions{
		Slug:         "an-issue-that-grew-up",
		Title:        "An issue that grew up",
		SeedBody:     "Graduated from `iss-7`: an issue that grew up. Read that issue record for the source observation.",
		PromotedFrom: "iss-7",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if it.PromotedFrom != "iss-7" {
		t.Fatalf("created intent PromotedFrom = %q, want iss-7", it.PromotedFrom)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := c.Lookup(it.ID)
	if !ok {
		t.Fatalf("minted draft %s not found by Load", it.ID)
	}
	if got.PromotedFrom != "iss-7" {
		t.Fatalf("parsed-back PromotedFrom = %q, want iss-7", got.PromotedFrom)
	}
	// Absent on every existing record: a draft minted from text has none.
	plain, err := CreateFromText(root, "a plain quoted-text draft", TextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	c, err = Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Lookup(plain.ID); got.PromotedFrom != "" {
		t.Fatalf("text-created draft must carry no promoted_from, got %q", got.PromotedFrom)
	}
}

// TestCreateDraftValidatesInputs: a promote-path mint refuses a malformed slug
// or promoted_from before any path is built.
func TestCreateDraftValidatesInputs(t *testing.T) {
	root := t.TempDir()
	if _, err := CreateDraft(root, DraftOptions{Slug: "../evil", Title: "x", SeedBody: "y"}); err == nil {
		t.Fatalf("CreateDraft must refuse a non-kebab slug")
	}
	if _, err := CreateDraft(root, DraftOptions{Slug: "ok-slug", Title: "x", SeedBody: "y", PromotedFrom: "itd-3"}); err == nil {
		t.Fatalf("CreateDraft must refuse a promoted_from that is not an iss-N id")
	}
	if entries, err := os.ReadDir(filepath.Join(root, IntentsRelDir, BucketDrafts)); err == nil && len(entries) > 0 {
		t.Fatalf("a refused CreateDraft wrote %d file(s)", len(entries))
	}
}

// TestSeedDraftStampsProvenance is the itd-178 core on the intent side: a draft
// written through the command carries BOTH disclosure keys, neither of them
// supplied as free text. origin has no flag at all — it is derived from which
// command ran — and the production mode is a closed choice validated before
// anything is written.
func TestSeedDraftStampsProvenance(t *testing.T) {
	root := t.TempDir()
	it, err := CreateFromText(root, "a draft worth stamping with its provenance", TextOptions{})
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	fields := readDraftFields(t, root, it.Path)
	if got := fields[provenance.KeyOrigin].Value; got != string(provenance.KindResearcherAuthored) {
		t.Errorf("origin = %q, want %q", got, provenance.KindResearcherAuthored)
	}
	if got := fields[provenance.KeyProductionMode].Value; got != string(provenance.DefaultMode) {
		t.Errorf("production_mode = %q, want the default %q", got, provenance.DefaultMode)
	}

	// A declared mode is stamped; the origin does not move with it.
	it2, err := CreateFromText(root, "a dictated draft worth stamping", TextOptions{ProductionMode: "dictated-and-formatted"})
	if err != nil {
		t.Fatalf("CreateFromText with a declared mode: %v", err)
	}
	fields = readDraftFields(t, root, it2.Path)
	if got := fields[provenance.KeyProductionMode].Value; got != "dictated-and-formatted" {
		t.Errorf("production_mode = %q, want dictated-and-formatted", got)
	}
	if got := fields[provenance.KeyOrigin].Value; got != string(provenance.KindResearcherAuthored) {
		t.Errorf("origin = %q, want it unmoved at %q", got, provenance.KindResearcherAuthored)
	}

	// A promote-shaped draft declares the other arrival path.
	it3, err := CreateDraft(root, DraftOptions{
		Slug: "graduated", Title: "Graduated", SeedBody: "from a record",
		PromotedFrom: "iss-1", Origin: provenance.Origin{Kind: provenance.KindExtractedFromRecord},
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	fields = readDraftFields(t, root, it3.Path)
	if got := fields[provenance.KeyOrigin].Value; got != string(provenance.KindExtractedFromRecord) {
		t.Errorf("promoted draft origin = %q, want %q", got, provenance.KindExtractedFromRecord)
	}

	// An out-of-vocabulary mode is refused before anything is written.
	before := draftCount(t, root)
	if _, err := CreateFromText(root, "a draft with a bogus production mode", TextOptions{ProductionMode: "typed"}); err == nil {
		t.Error("an out-of-vocabulary production mode must be refused")
	}
	if after := draftCount(t, root); after != before {
		t.Errorf("a refused create wrote a draft: %d -> %d", before, after)
	}
}

// readDraftFields reads a written record's frontmatter through the shared line
// scanner — the same reader every consumer uses, so a key it cannot see is a
// key that was not really written.
func readDraftFields(t *testing.T, root, rel string) map[string]frontmatter.Field {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return frontmatter.Fields(strings.Split(string(data), "\n"))
}

func draftCount(t *testing.T, root string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, IntentsRelDir, BucketDrafts))
	if err != nil {
		t.Fatalf("reading drafts: %v", err)
	}
	return len(entries)
}

// TestCreateDraftStampsAReadingOrigin — framework 11.3 (linkage): a draft a
// reading occasioned carries the run and the item in `origin`, beside the
// production mode, so both halves of the disclosure pair land in one act.
func TestCreateDraftStampsAReadingOrigin(t *testing.T) {
	root := t.TempDir()
	it, err := CreateDraft(root, DraftOptions{
		Slug: "a-reading-occasioned-draft", Title: "A reading occasioned draft",
		SeedBody:     "Graduated from a reading item.",
		PromotedFrom: "rdi-17",
		Origin: provenance.Origin{
			Kind: provenance.KindContributedByReading, Run: "rdg-3", Item: "rdi-17",
		},
		ProductionMode: "dictated-and-formatted",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	fields := readDraftFields(t, root, it.Path)
	if got, want := fields[provenance.KeyOrigin].Value, "contributed-by-reading rdg-3/rdi-17"; got != want {
		t.Errorf("origin = %q, want %q", got, want)
	}
	if got := fields[provenance.KeyProductionMode].Value; got != "dictated-and-formatted" {
		t.Errorf("production_mode = %q, want dictated-and-formatted", got)
	}
	if got := fields["promoted_from"].Value; got != "rdi-17" {
		t.Errorf("promoted_from = %q, want rdi-17", got)
	}
	// The rendered value is a plain scalar: no ": " inside it, which is what keeps
	// it readable to the same-line frontmatter scanner every record reader uses.
	if _, err := provenance.ParseOrigin(fields[provenance.KeyOrigin].Value); err != nil {
		t.Errorf("the written origin does not parse back: %v", err)
	}
}

// TestCreateDraftRefusesAReadingOriginDisagreeingWithTheBackEdge — framework
// 11.3: the origin's item and the `promoted_from` back-edge are one join written
// twice, so a draft carrying them in disagreement is a state no command
// produced. The mint refuses it rather than writing it.
func TestCreateDraftRefusesAReadingOriginDisagreeingWithTheBackEdge(t *testing.T) {
	root := t.TempDir()
	if _, err := CreateDraft(root, DraftOptions{
		Slug: "seeded", Title: "Seeded", SeedBody: "body",
		Origin: provenance.Origin{Kind: provenance.KindContributedByReading, Run: "rdg-3", Item: "rdi-17"},
	}); err == nil {
		t.Error("a reading origin with no back-edge must be refused")
	}
	if _, err := os.ReadDir(filepath.Join(root, IntentsRelDir, BucketDrafts)); err == nil {
		if n := draftCount(t, root); n != 0 {
			t.Fatalf("a refused mint wrote %d draft(s)", n)
		}
	}
	_, err := CreateDraft(root, DraftOptions{
		Slug: "seeded", Title: "Seeded", SeedBody: "body",
		PromotedFrom: "rdi-18",
		Origin:       provenance.Origin{Kind: provenance.KindContributedByReading, Run: "rdg-3", Item: "rdi-17"},
	})
	if err == nil {
		t.Fatal("a reading origin naming a different item from the back-edge must be refused")
	}
	if !strings.Contains(err.Error(), "rdi-17") || !strings.Contains(err.Error(), "rdi-18") {
		t.Errorf("the refusal must name both halves of the disagreement; got %v", err)
	}
	if _, err := os.ReadDir(filepath.Join(root, IntentsRelDir, BucketDrafts)); err == nil {
		if n := draftCount(t, root); n != 0 {
			t.Fatalf("a refused mint wrote %d draft(s)", n)
		}
	}
}

// TestReadingRouteSeedNamesNoItem — companion 8.3 (no reading sees another's
// output): the Press Release is the first field the intent projection carries to
// the entailment reading, so a reading-route seed naming its rdi-N would put a
// prior reading's output inside the object of the next one. The seed says "a
// reading item" and the join lives in fields no projection reaches.
func TestReadingRouteSeedNamesNoItem(t *testing.T) {
	root := t.TempDir()
	it, err := CreateDraft(root, DraftOptions{
		Slug: "a-reading-occasioned-draft", Title: "A reading occasioned draft",
		SeedBody: "Graduated from `rdi-17` (accepted): a stated constraint.",
		// The back-edge and the origin carry the item; the seed must not.
		PromotedFrom: "rdi-17",
		Origin: provenance.Origin{
			Kind: provenance.KindContributedByReading, Run: "rdg-3", Item: "rdi-17",
		},
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatal(err)
	}
	press := namedSection(t, string(data), "## Press Release")
	if strings.Contains(press, "rdi-") {
		t.Errorf("the reading-route Press Release names a reading item:\n%s", press)
	}
	if !IsSeedNote(seedWords(press)) {
		t.Errorf("the reading-route seed is no longer recognised as a placeholder: %q", seedWords(press))
	}

	// The issue route's seed keeps its iss-N: an issue is something a person
	// noticed, not a reading's output.
	it2, err := CreateDraft(root, DraftOptions{
		Slug: "graduated-from-an-issue", Title: "Graduated from an issue",
		SeedBody: "from a record", PromotedFrom: "iss-1",
		Origin: provenance.Origin{Kind: provenance.KindExtractedFromRecord},
	})
	if err != nil {
		t.Fatalf("CreateDraft (issue route): %v", err)
	}
	data2, err := os.ReadFile(filepath.Join(root, it2.Path))
	if err != nil {
		t.Fatal(err)
	}
	press2 := namedSection(t, string(data2), "## Press Release")
	if !strings.Contains(press2, "iss-1") {
		t.Errorf("the issue-route seed no longer names its issue:\n%s", press2)
	}
	if !IsSeedNote(seedWords(press2)) {
		t.Errorf("the issue-route seed is no longer recognised as a placeholder: %q", seedWords(press2))
	}
}

// sectionBody returns the text under one heading, up to the next heading.
func namedSection(t *testing.T, doc, heading string) string {
	t.Helper()
	_, rest, ok := strings.Cut(doc, heading+"\n")
	if !ok {
		t.Fatalf("the record carries no %q section:\n%s", heading, doc)
	}
	if cut := strings.Index(rest, "\n## "); cut >= 0 {
		rest = rest[:cut]
	}
	return strings.TrimSpace(rest)
}

// seedWords reduces a press-release body to the words IsSeedNote judges: quote
// markers, emphasis markers and whitespace runs removed.
func seedWords(body string) string {
	body = strings.ReplaceAll(body, ">", " ")
	body = strings.ReplaceAll(body, "_", " ")
	return strings.Join(strings.Fields(body), " ")
}

// TestCreateFromTextSeedsPressRelease is the iss-2609170726360399 headline: the
// quoted text IS the press release, so it seeds `## Press Release` as prose
// rather than sitting under a placeholder; the H1 is the text's first sentence,
// not the whole paragraph; and `## Why This Matters` prompts for its own
// content instead of repeating the text a second time.
func TestCreateFromTextSeedsPressRelease(t *testing.T) {
	root := t.TempDir()
	const first = "Teams see who is waiting on whom without asking"
	const second = "Every session prints its owed answer."
	text := first + ". " + second

	it, err := CreateFromText(root, text, TextOptions{})
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)

	if !strings.Contains(doc, "\n# "+first+"\n") {
		t.Errorf("H1 is not the first sentence %q:\n%s", first, doc)
	}
	if strings.Contains(doc, "# "+text) {
		t.Errorf("H1 carries the whole quoted paragraph:\n%s", doc)
	}
	press := namedSection(t, doc, "## Press Release")
	if !strings.Contains(press, first) || !strings.Contains(press, second) {
		t.Errorf("Press Release does not carry the whole quoted text:\n%s", press)
	}
	if IsSeedNote(seedWords(press)) {
		t.Errorf("Press Release is still the seed placeholder: %q", seedWords(press))
	}
	why := namedSection(t, doc, "## Why This Matters")
	if strings.Contains(why, first) || strings.Contains(why, second) {
		t.Errorf("Why This Matters repeats the quoted text:\n%s", why)
	}
	if why == "" {
		t.Errorf("Why This Matters is empty; want the seeded prompt")
	}
}

// TestDeriveTitle pins the sentence split the H1 is derived from: the text up
// to the first `.`, `!` or `?` that is followed by whitespace or ends the text,
// with that terminator dropped (no shipped intent's H1 carries one); a text
// with no terminator is one sentence; a sentence longer than the slug cap is
// cut on a word boundary at or before the cap.
func TestDeriveTitle(t *testing.T) {
	longSentence := "a sentence that runs on well past the sixty-character slug cap without ever reaching a terminator"
	cases := []struct {
		name, text, want string
	}{
		{"two sentences", "Teams see who is waiting. Every session prints its answer.", "Teams see who is waiting"},
		{"no terminator", "Teams see who is waiting on whom", "Teams see who is waiting on whom"},
		{"terminator at the end only", "Teams see who is waiting on whom.", "Teams see who is waiting on whom"},
		{"question mark", "Who is waiting on whom? Everyone.", "Who is waiting on whom"},
		{"exclamation", "Nobody waits! Every session prints its answer.", "Nobody waits"},
		{"a period inside a token is not a boundary", "v0.9 prints its answer. Then more.", "v0.9 prints its answer"},
		{"whitespace collapses", "Teams   see\twho\nis waiting. More.", "Teams see who is waiting"},
		{"an ellipsis leaves no stub", "Wait... what is owed? Everything.", "Wait"},
		{"one long sentence is cut on a word boundary", longSentence, "a sentence that runs on well past the sixty-character slug"},
		{"a long sentence with a terminator is cut the same way", longSentence + ". Then more.", "a sentence that runs on well past the sixty-character slug"},
		{"one unbroken word is cut at the cap", strings.Repeat("x", maxSlugLen+5), strings.Repeat("x", maxSlugLen)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveTitle(tc.text)
			if got != tc.want {
				t.Errorf("deriveTitle(%q) = %q, want %q", tc.text, got, tc.want)
			}
			if n := len([]rune(got)); n > maxSlugLen {
				t.Errorf("derived title is %d runes, over the %d cap", n, maxSlugLen)
			}
		})
	}
}

// TestCreateFromTextTitleOverride: an explicit Title replaces the derived one
// and is held to the text's own bar — non-empty after trimming, one line,
// redacted through the same scanner — while the slug stays derived from the
// text and the Press Release still carries the whole text.
func TestCreateFromTextTitleOverride(t *testing.T) {
	root := t.TempDir()
	text := "Teams see who is waiting on whom without asking. Every session prints its owed answer."

	it, err := CreateFromText(root, text, TextOptions{Title: "  Nobody waits on a silent session  "})
	if err != nil {
		t.Fatalf("CreateFromText with a title: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if !strings.Contains(doc, "\n# Nobody waits on a silent session\n") {
		t.Errorf("H1 is not the explicit title:\n%s", doc)
	}
	if !strings.HasPrefix(it.Slug, "teams-see-who-is-waiting") {
		t.Errorf("slug %q is not derived from the text", it.Slug)
	}
	if press := namedSection(t, doc, "## Press Release"); !strings.Contains(press, "owed answer") {
		t.Errorf("Press Release lost the text under an explicit title:\n%s", press)
	}

	// The title is redacted through the same scanner as the text.
	const fakeHome = "/Users/alice/.ssh/id_rsa"
	it2, err := CreateFromText(root, text, TextOptions{Title: "the key at " + fakeHome + " leaks"})
	if err != nil {
		t.Fatalf("CreateFromText with a title to redact: %v", err)
	}
	data2, err := os.ReadFile(filepath.Join(root, it2.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data2), "/Users/alice") {
		t.Errorf("an explicit title persisted a raw home path:\n%s", data2)
	}

	// Degenerate titles are refused with nothing written. An explicitly set
	// empty title is held to the same bar as a blank one — "non-empty after
	// trimming" — rather than read as no title given.
	before := draftCount(t, root)
	for _, bad := range []string{"   ", "\t\n", "two\nlines", "carriage\rreturn"} {
		if _, err := CreateFromText(root, text, TextOptions{Title: bad}); err == nil {
			t.Errorf("Title %q must be refused", bad)
		}
	}
	if _, err := CreateFromText(root, text, TextOptions{Title: "", TitleSet: true}); err == nil {
		t.Error("an explicitly set empty Title must be refused")
	}
	if after := draftCount(t, root); after != before {
		t.Errorf("a refused title wrote a draft: %d -> %d", before, after)
	}
}

// TestPromotePathSeedUnchanged: the promote route keeps its shape — the Press
// Release is the promotion seed note and the seed body sits under Why This
// Matters — so the site's placeholder exclusion and the by-id pointer a
// promoted draft carries are untouched by the quoted-text route moving.
func TestPromotePathSeedUnchanged(t *testing.T) {
	root := t.TempDir()
	it, err := CreateDraft(root, DraftOptions{
		Slug: "graduated-from-an-issue", Title: "Graduated from an issue",
		SeedBody: "Graduated from `iss-1`: read that issue record for the source observation.", PromotedFrom: "iss-1",
		Origin: provenance.Origin{Kind: provenance.KindExtractedFromRecord},
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if press := namedSection(t, doc, "## Press Release"); !IsSeedNote(seedWords(press)) {
		t.Errorf("the promote route's Press Release is no longer the seed note: %q", seedWords(press))
	}
	if why := namedSection(t, doc, "## Why This Matters"); !strings.Contains(why, "Graduated from `iss-1`") {
		t.Errorf("the promote route's Why This Matters lost its by-id pointer:\n%s", why)
	}
	if !strings.Contains(doc, "\n# Graduated from an issue\n") {
		t.Errorf("the promote route's H1 is not the explicit title:\n%s", doc)
	}
}
