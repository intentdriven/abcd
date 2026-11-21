package matrix

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// cellName is the id of model m's cell in writingInputs: distinctive enough
// that finding it in a packet is a leak, never a coincidence.
func cellName(m int) string { return fmt.Sprintf("leakcell-%02d", m) }

// writingInputs is a writing task with models builder cells of repeats copies
// each, drawn for judges.
func writingInputs(models, repeats, judges int) judgingInputs {
	var copies []writingCopy
	for m := range models {
		for r := 1; r <= repeats; r++ {
			copies = append(copies, writingCopy{
				Key:  TrialKey{Task: "doc-page", Cell: cellName(m), Repeat: r},
				Text: fmt.Sprintf("# Guide\n\nThe page, draft %d of writer %d.\n", r, m),
			})
		}
	}
	return judgingInputs{
		Run:     "mx-261023012949-ab12",
		Task:    Task{ID: "doc-page", Kind: KindWriting, Source: []string{"docs/guide.md"}, Criteria: "accurate against the source first"},
		Copies:  copies,
		Sources: []namedText{{Name: "docs/guide.md", Text: "# Guide\n\nWhat the page explains.\n"}},
		Good:    "# Guide\n\nA page that says what the source says.\n",
		Bad:     "# Guide\n\nA page that says what the source does not.\n",
		Judges:  judges,
	}
}

var seedA = [32]byte{1, 2, 3}

func draw(t *testing.T, in judgingInputs, seed [32]byte) (JudgingPacket, JudgingKey) {
	t.Helper()
	p, k, err := drawJudging(in, seed)
	if err != nil {
		t.Fatalf("drawJudging: %v", err)
	}
	return p, k
}

// A judge's subset holds at most eight copies, and across the subsets every
// pair of builder models meets in at least one judge's subset, so every pair
// is compared (decisions 21 and 26). The judges the plan asks for are enough
// subsets to cover every pair.
func TestAJudgesSubsetHoldsAtMostEightCopiesAndEveryPairOfModelsIsCompared(t *testing.T) {
	for _, models := range []int{1, 2, 3, 5, 8, 9, 12, 17, 30} {
		t.Run(fmt.Sprint(models), func(t *testing.T) {
			need := max(judgeFloor, len(modelBlocks(models)))
			p, k := draw(t, writingInputs(models, 3, need), seedA)
			if len(p.Slots) != need {
				t.Fatalf("%d slots; want the %d judges asked for", len(p.Slots), need)
			}
			met := map[[2]string]bool{}
			seen := map[string]bool{}
			for _, s := range p.Slots {
				if len(s.Copies) == 0 || len(s.Copies) > maxJudgeSubset {
					t.Fatalf("slot %s holds %d copies; want 1 to %d", s.ID, len(s.Copies), maxJudgeSubset)
				}
				codes := map[string]bool{}
				var cells []string
				for _, c := range s.Copies {
					if codes[c.Code] {
						t.Fatalf("slot %s repeats the code %s", s.ID, c.Code)
					}
					codes[c.Code] = true
					kc, ok := k.Documents[c.Document]
					if !ok {
						t.Fatalf("slot %s names document %s the key does not hold", s.ID, c.Document)
					}
					cells = append(cells, kc.Key.Cell)
					seen[kc.Key.Cell] = true
				}
				for _, a := range cells {
					for _, b := range cells {
						if a < b {
							met[[2]string{a, b}] = true
						}
					}
				}
			}
			if len(seen) != models {
				t.Fatalf("the subsets show %d models of %d", len(seen), models)
			}
			for a := range models {
				for b := a + 1; b < models; b++ {
					if !met[[2]string{cellName(a), cellName(b)}] {
						t.Fatalf("models %d and %d meet in no judge's subset", a, b)
					}
				}
			}
			var plan []JudgeNeed
			var builders []Cell
			for m := range models {
				builders = append(builders, Cell{ID: cellName(m), Route: "host"})
			}
			plan = judgesOf(Declaration{Tasks: []Task{writingInputs(1, 1, 1).Task}, Builders: builders})
			if plan[0].Count != need {
				t.Fatalf("the plan asks for %d judges; the draw needs %d", plan[0].Count, need)
			}
		})
	}
}

// Fewer judges than it takes to bring every pair of models together is
// refused, never drawn with a pair left uncompared.
func TestFewerJudgesThanEveryPairNeedsAreRefused(t *testing.T) {
	need := len(modelBlocks(12))
	if need < 2 {
		t.Fatalf("12 models make %d blocks; want several", need)
	}
	if _, _, err := drawJudging(writingInputs(12, 3, need-1), seedA); err == nil || !strings.Contains(err.Error(), "every pair") {
		t.Fatalf("drawJudging with %d judges = %v; want a refusal naming every pair", need-1, err)
	}
}

// Each judge's order and codes are drawn for that judge: two judges of the
// same models see them in different orders under different codes, and the
// packet the session is given names no cell, model, route, repeat or trial,
// so the copy-to-model mapping never reaches it (decision 24,
// adr-2610161733400551). The key on the host's machine holds the mapping.
func TestEachJudgesOrderAndCodesAreDrawnForThemAndThePacketNamesNoAuthor(t *testing.T) {
	in := writingInputs(6, 4, 6)
	p, k := draw(t, in, seedA)
	orders := map[string]bool{}
	codes := map[string]bool{}
	for _, s := range p.Slots {
		var cells, cs []string
		for _, c := range s.Copies {
			cells = append(cells, k.Documents[c.Document].Key.Cell)
			cs = append(cs, c.Code)
		}
		orders[strings.Join(cells, ",")] = true
		codes[strings.Join(cs, ",")] = true
	}
	if len(orders) < 2 || len(codes) != len(p.Slots) {
		t.Fatalf("%d slots show %d orders and %d code sets; want each drawn apart", len(p.Slots), len(orders), len(codes))
	}
	again, _ := draw(t, in, seedA)
	if !slices.EqualFunc(again.Slots, p.Slots, func(a, b JudgeSlot) bool { return slices.Equal(a.Copies, b.Copies) }) {
		t.Fatal("the same seed drew a different packet; the key's seed would not reproduce it")
	}
	other, _ := draw(t, in, [32]byte{9})
	if slices.EqualFunc(other.Slots, p.Slots, func(a, b JudgeSlot) bool { return slices.Equal(a.Copies, b.Copies) }) {
		t.Fatal("another seed drew the same packet")
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for m := range 6 {
		if strings.Contains(string(data), cellName(m)) {
			t.Fatalf("the packet names the cell %s", cellName(m))
		}
	}
	for _, leak := range []string{"trials/", `"repeat"`, `"cell"`, "host", "runner", "model"} {
		if strings.Contains(string(data), leak) {
			t.Fatalf("the packet carries %q, which names an author", leak)
		}
	}
	if len(k.Documents) == 0 || k.Run != in.Run || k.Task != in.Task.ID || len(k.Seed) != 64 {
		t.Fatalf("key = %+v; want the run, task, seed and every document's trial", k)
	}
	if d, err := p.digest(); err != nil || d != k.Packet {
		t.Fatalf("the key names packet %q; the packet's digest is %q (%v)", k.Packet, d, err)
	}
}

// What a judging cannot stand on is refused before anything is drawn.
func TestJudgingIsRefusedWhatItCannotStandOn(t *testing.T) {
	cases := map[string]func(*judgingInputs){
		"not a writing task": func(in *judgingInputs) { in.Task.Kind = KindCode },
		"no copies":          func(in *judgingInputs) { in.Copies = nil },
		"another task":       func(in *judgingInputs) { in.Copies[0].Key.Task = "other-page" },
		"a copy twice":       func(in *judgingInputs) { in.Copies[1] = in.Copies[0] },
		"an empty copy":      func(in *judgingInputs) { in.Copies[0].Text = " \n" },
		"invalid UTF-8":      func(in *judgingInputs) { in.Copies[0].Text = "\xff" },
		"a copy too large":   func(in *judgingInputs) { in.Copies[0].Text = strings.Repeat("a", maxCopyBytes+1) },
		"no good example":    func(in *judgingInputs) { in.Good = "" },
		"no bad example":     func(in *judgingInputs) { in.Bad = "" },
		"a source not named": func(in *judgingInputs) { in.Sources[0].Name = "docs/other.md" },
		"no judge":           func(in *judgingInputs) { in.Judges = 0 },
		"a bad run id":       func(in *judgingInputs) { in.Run = "run-1" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := writingInputs(3, 3, 3)
			mutate(&in)
			if _, _, err := drawJudging(in, seedA); err == nil {
				t.Fatal("drawn; want a refusal")
			}
		})
	}
}

// Every copy, the source and the calibration examples render as inert text
// through one template: nothing a model wrote is markup in the judge's page,
// and the document allows nothing to load or run.
func TestCopiesRenderAsInertText(t *testing.T) {
	hostile := "<script>alert(1)</script>\n<img src=x onerror=alert(2)>\n</pre><iframe src=//x></iframe>\n" +
		"<a href=\"javascript:alert(3)\">x</a>\n<!-- c --> ]]> <style>*{}</style>\x1b[2J‮hidden\n"
	in := writingInputs(2, 2, 3)
	in.Copies[0].Text = hostile
	in.Sources[0].Text = hostile
	in.Good = hostile
	p, k := draw(t, in, seedA)

	var docs []JudgingDoc
	docs = append(docs, p.Sources...)
	docs = append(docs, p.Calibration.Good, p.Calibration.Bad)
	docs = append(docs, p.Documents...)
	hostileDoc := ""
	for id, kc := range k.Documents {
		if kc.Key == in.Copies[0].Key {
			hostileDoc = id
		}
	}
	found := false
	for _, d := range docs {
		h := d.HTML
		for _, bad := range []string{"<script", "<img", "<iframe", "<a ", "<style>*", "<!-- c", "\x1b", "‮", "javascript:alert(3)\">"} {
			if strings.Contains(h, bad) {
				t.Fatalf("document %s carries %q raw:\n%s", d.ID, bad, h)
			}
		}
		if !strings.Contains(h, `content="default-src 'none'`) {
			t.Fatalf("document %s carries no policy that blocks every load and script:\n%s", d.ID, h)
		}
		if d.ID == hostileDoc {
			found = strings.Contains(h, "&lt;script&gt;alert(1)&lt;/script&gt;")
		}
	}
	if hostileDoc == "" || !found {
		t.Fatal("the hostile copy is not in the packet as escaped text")
	}
	if p.Template != renderTemplate() || p.Template.SHA256 == "" {
		t.Fatalf("the packet names template %+v; want the one copies render through", p.Template)
	}
	// One template: a document is the template with its text filled in.
	if got := renderInert("x", ""); !strings.HasPrefix(got, "<!doctype html>") || strings.Contains(got, "{{") {
		t.Fatalf("an empty render = %q; want the template filled", got)
	}
	tpl := renderInert("", "")
	for _, d := range docs {
		pre, _, _ := strings.Cut(tpl, "<title>")
		if !strings.HasPrefix(d.HTML, pre) {
			t.Fatalf("document %s is not rendered through the template", d.ID)
		}
	}
}

// The packet a judging session is given carries no repository path, run or
// task id: a source is titled by its place in the task ("Source 1"), and the
// path the declaration names it by stays in the host's own inputs. Every field
// the packet carries is swept, the rendered documents included.
func TestThePacketCarriesNoSourcePathRunOrTaskID(t *testing.T) {
	const path = "docs/leaksource-guide.md"
	in := writingInputs(3, 3, 3)
	in.Task.ID = "leaktask-page"
	for i := range in.Copies {
		in.Copies[i].Key.Task = in.Task.ID
	}
	in.Task.Source = []string{path, "notes/leaksource-two.txt"}
	in.Sources = []namedText{
		{Name: path, Text: "# Guide\n\nWhat the page explains.\n"},
		{Name: "notes/leaksource-two.txt", Text: "More of what it explains.\n"},
	}
	p, k := draw(t, in, seedA)
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{path, "leaksource", ".md", ".txt", "docs/", "notes/", in.Run, in.Task.ID, "leaktask"} {
		if strings.Contains(string(data), leak) {
			t.Fatalf("the packet carries %q:\n%s", leak, data)
		}
	}
	for i, s := range p.Sources {
		if want := fmt.Sprintf("<title>Source %d</title>", i+1); !strings.Contains(s.HTML, want) {
			t.Fatalf("source %s is not titled by its place (%s):\n%s", s.ID, want, s.HTML)
		}
	}
	if k.Run != in.Run || k.Task != in.Task.ID {
		t.Fatalf("key = %+v; the host's half still names the run and task", k)
	}
}

// A writing copy cannot sign its own work in front of its judges: its text is
// passed through the same identity scrub a check copy's words are, so a copy
// that names its model, its maker, its cell, its route, its runner, its run,
// its task or its alias reaches the judge with each of them withheld, and
// the key, on the host's machine, counts what was withheld and measures the
// copy as the judge reads it.
func TestAWritingCopyCannotSignItsWorkInFrontOfItsJudges(t *testing.T) {
	in := writingInputs(2, 1, 3)
	in.Cells = []CellPin{
		{Cell: cellName(0), Role: RoleBuilder, Source: SourceOpenRouter, Route: "builder=runner:opencode@openrouter/acme/wordsmith-7b",
			Runner: "opencode", Model: "openrouter/acme/wordsmith-7b", HarnessVersion: "1.2.3", ProviderPin: &ProviderPin{Order: []string{"fastinfer"}}},
		{Cell: cellName(1), Role: RoleBuilder, Source: SourceHost, Route: "host", Runner: "host"},
	}
	in.Copies[0].Alias = "copy_" + strings.Repeat("ab", 16)
	signed := "# Guide\n\nWritten by wordsmith-7b (openrouter/acme/wordsmith-7b) through opencode 1.2.3 on fastinfer, " +
		"cell " + cellName(0) + ", task doc-page of run " + in.Run + ", in " + in.Copies[0].Alias + ".\n" +
		"Also signed: Claude, Anthropic, GPT-4o, Gemini and o3.\n"
	in.Copies[0].Text = signed
	p, k := draw(t, in, seedA)
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"wordsmith", "acme", "openrouter", "opencode", "1.2.3", "fastinfer", cellName(0), "leakcell",
		"doc-page", in.Run, in.Copies[0].Alias, "Claude", "Anthropic", "GPT", "Gemini", "o3"} {
		if strings.Contains(string(data), leak) {
			t.Fatalf("the packet carries %q, which the copy signed its work with", leak)
		}
	}
	if !strings.Contains(string(data), withheldMark) {
		t.Fatalf("the packet shows no mark where the copy's signature was withheld")
	}
	var withheld, size int
	for _, kc := range k.Documents {
		if kc.Key == in.Copies[0].Key {
			withheld, size = kc.Withheld, kc.Bytes
		}
	}
	if withheld == 0 {
		t.Fatalf("the key counts nothing withheld from the signed copy: %+v", k.Documents)
	}
	if size == len(signed) {
		t.Fatalf("the key measures the signed copy at %d bytes, the copy as written; want the copy as the judge reads it", size)
	}
}
