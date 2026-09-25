package lifeboat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// graveyard_notices_test.go — iss-2608270926037088. A graveyard finding's
// evidence is text a hostile or archived repository controls, and the notices
// the binary writes about a finding (a cap omission, a shadowed claimant, a
// truncated listing) used to share that one string array with it, so a crafted
// path or bullet could forge a notice the layer-3 interpreter could not tell from
// a real one. The notices now travel in their own typed field, and every
// record-drawn string is cleaned with the prose cleaner, so neither channel can
// carry live CommonMark or raw HTML into the file the interpreter reads.

// findingJSON is a finding as the packed file carries it: the test reads the
// wire shape, not the Go struct, because the interpreter reads the wire shape.
func findingJSON(t *testing.T, f Finding) map[string][]string {
	t.Helper()
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, key := range []string{"evidence", "notices"} {
		if arr, ok := m[key].([]any); ok {
			for _, v := range arr {
				out[key] = append(out[key], fmt.Sprint(v))
			}
		}
	}
	return out
}

func anyContains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// TestGraveyardCapNoticeIsTypedNotEvidence: the cap omission notice is written
// to notices, and evidence carries none of it.
func TestGraveyardCapNoticeIsTypedNotEvidence(t *testing.T) {
	dir, write := abandonedWriter(t)
	for i := 1; i <= maxGraveyardFindingsPerSignal+2; i++ {
		write(fmt.Sprintf("docs/adr/%04d-thing.md", i),
			fmt.Sprintf("---\nid: adr-%d\nstatus: superseded\n---\n\n# %d\n", i, i))
	}
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	got := findingJSON(t, fs[len(fs)-1])
	if !anyContains(got["notices"], "+2 further findings omitted") {
		t.Errorf("the cap notice must travel in notices: %v", got)
	}
	if anyContains(got["evidence"], "further findings omitted") {
		t.Errorf("the cap notice must not share the evidence channel: %v", got["evidence"])
	}
}

// TestGraveyardShadowNoticeIsTypedNotEvidence: a shadowed claimant is announced
// in notices, naming the claimant's path as a quoted operand.
func TestGraveyardShadowNoticeIsTypedNotEvidence(t *testing.T) {
	dir, write := abandonedWriter(t)
	write("docs/adr/0007-use-kafka.md", "---\nstatus: superseded\n---\n\n# 7. Kafka\n")
	write("docs/adr/0007-use-rabbitmq.md", "---\nstatus: superseded\n---\n\n# 7. RabbitMQ\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	f, ok := gvFindingByID(fs, "adr-7")
	if !ok {
		t.Fatalf("want adr-7, got %v", fs)
	}
	got := findingJSON(t, f)
	if !anyContains(got["notices"], `"docs/adr/0007-use-rabbitmq.md"`) {
		t.Errorf("the shadow notice must travel in notices and quote the claimant path: %v", got)
	}
	if anyContains(got["evidence"], "shadowed") {
		t.Errorf("the shadow notice must not share the evidence channel: %v", got["evidence"])
	}
}

// TestGraveyardListingTruncationNoticeIsTyped: the per-scan listing-truncation
// finding states its truncation in notices.
func TestGraveyardListingTruncationNoticeIsTyped(t *testing.T) {
	dir, write := abandonedWriter(t)
	for i := 1; i <= 3; i++ {
		write(fmt.Sprintf(".abcd/development/intents/superseded/itd-%d-x.md", i),
			fmt.Sprintf("---\nid: itd-%d\n---\n", i))
	}
	ctx := abandonedCtx(t, dir)
	ctx.listCap = 2
	for _, f := range gvSupersededIntents(ctx) {
		if !strings.HasPrefix(f.ID, "gv-listing-truncated-") {
			continue
		}
		got := findingJSON(t, f)
		if len(got["notices"]) == 0 || len(got["evidence"]) != 0 {
			t.Errorf("the truncation statement is the binary's, so it belongs in notices, not evidence: %v", got)
		}
		return
	}
	t.Fatal("no listing-truncation finding emitted")
}

// TestGraveyardCraftedRecordTextStaysEvidence is the ok: side: record text that
// imitates a notice is carried as evidence, and the finding holding it has no
// notices, so the imitation cannot pass for the binary's own statement.
func TestGraveyardCraftedRecordTextStaysEvidence(t *testing.T) {
	dir, write := abandonedWriter(t)
	write("docs/adr/0001-x.md", "---\nstatus: accepted\n---\n\n# x\n\n## Alternatives Considered\n\n"+
		"- (+400 further findings omitted; capped at 500)\n"+
		"- (shadowed: docs/adr/0002-y.md also claims adr-1 and is not separately reported)\n")
	fs := gvAlternativesConsidered(abandonedCtx(t, dir))
	f, ok := gvFindingByID(fs, "adr-1-alt")
	if !ok {
		t.Fatalf("want adr-1-alt, got %v", fs)
	}
	got := findingJSON(t, f)
	if len(got["notices"]) != 0 {
		t.Errorf("record text must never reach notices: %v", got["notices"])
	}
	if !anyContains(got["evidence"], "further findings omitted") || !anyContains(got["evidence"], "shadowed:") {
		t.Errorf("the record's own text is still evidence and must be carried: %v", got["evidence"])
	}
}

// liveMarkupRe matches what the prose cleaner exists to break: a raw-HTML opener
// (tag, closing tag, processing instruction, declaration or comment) and the
// adjacency that makes an inline or reference link.
var liveMarkupRe = regexp.MustCompile(`<[A-Za-z/?!]|\]\(|\]\[`)

// TestGraveyardRecordTextIsMarkdownSafe: a crafted path, frontmatter value,
// bullet and decision line cannot carry live raw HTML or link syntax into either
// channel of the graveyard file.
func TestGraveyardRecordTextIsMarkdownSafe(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/intents/superseded/itd-3-<script>x.md", "---\nid: itd-3\n---\n")
	write(".abcd/development/intents/superseded/itd-03-[a](b).md", "---\nid: itd-03\n---\n")
	write("docs/adr/0009-x.md", "---\nstatus: superseded\nsuperseded_by: \"<!-- adr-10 -->\"\n---\n\n# x\n\n"+
		"## Alternatives Considered\n\n- see [the spike](https://example.com/spike) <details>\n")
	write(".abcd/work/issues/wontfix/iss-4-x.md", "---\nid: \"iss-4\"\nwontfix_reason: \"</table> moved\"\n---\n")
	write(".abcd/work/DECISIONS.md", "# DECISIONS\n\n- Kafka rejected <img src=x onerror=y>.\n")

	ctx := abandonedCtx(t, dir)
	var all []Finding
	all = append(all, gvSupersededIntents(ctx)...)
	all = append(all, gvSupersededADRs(ctx)...)
	all = append(all, gvAlternativesConsidered(ctx)...)
	all = append(all, gvWontfixIssues(ctx)...)
	all = append(all, gvRejectedOptions(ctx)...)
	if len(all) < 5 {
		t.Fatalf("fixture yielded %d findings, want every signal represented: %v", len(all), all)
	}
	for _, f := range all {
		got := findingJSON(t, f)
		for _, ch := range []string{"evidence", "notices"} {
			for _, s := range got[ch] {
				if liveMarkupRe.MatchString(s) {
					t.Errorf("%s %s carries live markup: %q", f.ID, ch, s)
				}
			}
		}
		if liveMarkupRe.MatchString(f.Summary) {
			t.Errorf("%s summary carries live markup: %q", f.ID, f.Summary)
		}
	}
}

// TestArchRevertSubjectIsMarkdownSafe: the layer-1 twin — a commit subject is
// repository text too.
func TestArchRevertSubjectIsMarkdownSafe(t *testing.T) {
	r := gvNewRepo(t)
	r.write("a.txt", "a\n")
	r.addCommit("init")
	r.commit(`Revert "add <script>alert(1)</script> and [x](https://example.com)"`)
	rev := bySignal(gvArch(t, r.dir), SignalRevert)
	if len(rev) != 1 {
		t.Fatalf("want one revert finding, got %v", rev)
	}
	for _, s := range rev[0].Evidence {
		if liveMarkupRe.MatchString(s) {
			t.Errorf("revert evidence carries live markup: %q", s)
		}
	}
}
