package matrix

// judging.go lays a writing task's copies out for people to judge (spec step
// 6; decisions 21, 23, 24, 26 and 32; adr-2610161733400551). The run draws two
// halves from the copies:
//
//   - the packet, which the judging session is given: the task's criteria,
//     its source material, one good and one bad example to calibrate on, each
//     copy once under an opaque document id, and the judges' subsets, each at
//     most eight copies in an order and under random codes drawn for that
//     subset. It names no cell, model, route, repeat or trial, so the
//     copy-to-model mapping never reaches the session or the relay, and no
//     run, task or repository path either: a source is titled by its place
//     in the task (Source 1, Source 2, ...), never by the path it is read from;
//   - the key, which never leaves the host's machine: the run and task, each
//     document's trial, its length, the packet's digest and the seed the
//     draw was made from.
//
// Every document the packet carries is rendered as inert text through one
// template (renderInert): the text is escaped, never parsed as markup, and the
// document's own policy lets nothing load or run. The session shows it in a
// frame sandboxed with no permission at all (an iframe's srcdoc under an empty
// sandbox attribute), so nothing a model wrote can run in the judge's page.
//
// The subsets bring every pair of builder models together in at least one
// judge's subset (modelBlocks), so the rankings compare every pair; the plan
// asks for that many judges (judgesOf).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// JudgingSchema is the judging packet's and key's schema version.
const JudgingSchema = 1

// maxCopyBytes bounds one document a judge reads: a copy, a source or a
// calibration example.
const maxCopyBytes = 256 << 10

// maxJudgeSlots bounds the subsets one judging draws.
const maxJudgeSlots = 256

// codeAlphabet is what a copy's code is drawn from: capitals and digits a
// judge cannot misread for one another (no I, L, O, S, Z, 0, 1, 2, 5 or 8).
const codeAlphabet = "ABCDEFGHJKMNPQRTUVWXY34679"

// codeLen is a code's length: 26^4 codes, so the few in one subset are
// distinct without many draws.
const codeLen = 4

// writingCopy is one writing copy's text, as its trial produced it, and the
// alias its clone was made under (makeCopyClone).
type writingCopy struct {
	Key   TrialKey
	Alias string
	Text  string
}

// namedText is one document the judges read beside the copies: a source the
// task names, by its path.
type namedText struct {
	Name string
	Text string
}

// judgingInputs is what a writing task's judging is drawn from: the run, the
// task as declared, whether guests may judge (decision 24), the copies that
// came back, the task's source material in the order it names it, the good
// and the bad example the judges calibrate on, and how many subsets to draw.
type judgingInputs struct {
	Run  string
	Task Task
	// Tasks and Cells are every task's id and every cell the run pins: what
	// in the run could identify a copy, withheld from its text. RepoRoot and
	// Store are the run's own checkout and its clone store, each absolute or
	// empty: where a copy ran, withheld from its text too.
	Tasks           []string
	Cells           []CellPin
	RepoRoot, Store string
	GuestsMayJudge  bool
	Copies          []writingCopy
	Sources         []namedText
	Good, Bad       string
	Judges          int
}

// JudgingDoc is one document a judge reads, rendered as inert text.
type JudgingDoc struct {
	ID   string `json:"id"`
	HTML string `json:"html"`
}

// Calibration is the good and the bad example a judge reads before ranking.
type Calibration struct {
	Good JudgingDoc `json:"good"`
	Bad  JudgingDoc `json:"bad"`
}

// SlotCopy is one copy of a judge's subset: the code the judge knows it by and
// the document it is.
type SlotCopy struct {
	Code     string `json:"code"`
	Document string `json:"document"`
}

// JudgeSlot is one judge's subset, its copies in the order that judge sees
// them. The session gives each judge who joins a slot no one else holds.
type JudgeSlot struct {
	ID     string     `json:"id"`
	Copies []SlotCopy `json:"copies"`
}

// JudgingPacket is what the judging session is given. It names no run, task
// or path: the session knows it by its digest, and the key, on the host's
// machine, holds the run and task it is of.
type JudgingPacket struct {
	SchemaVersion  int          `json:"schema_version"`
	Criteria       string       `json:"criteria"`
	GuestsMayJudge bool         `json:"guests_may_judge"`
	Template       ContentRef   `json:"template"`
	Sources        []JudgingDoc `json:"sources"`
	Calibration    Calibration  `json:"calibration"`
	Documents      []JudgingDoc `json:"documents"`
	Slots          []JudgeSlot  `json:"slots"`
}

// KeyedCopy is what one document of a packet is a copy of, and its length,
// which the report shows beside its rank (decision 21).
type KeyedCopy struct {
	Key   TrialKey `json:"key"`
	Words int      `json:"words"`
	Bytes int      `json:"bytes"`
	// Withheld counts the mentions withheld from the copy's text before a
	// judge reads it: of where it ran, of what in the run identifies it, and
	// of models and makers.
	Withheld int `json:"withheld"`
}

// JudgingKey is the host's own half of a packet: never given to the session.
type JudgingKey struct {
	SchemaVersion int                  `json:"schema_version"`
	Run           string               `json:"run"`
	Task          string               `json:"task"`
	Packet        string               `json:"packet"`
	Seed          string               `json:"seed"`
	Documents     map[string]KeyedCopy `json:"documents"`
}

// drawJudging draws a writing task's packet and key from its copies, seeded by
// seed (the caller's, read from crypto/rand; the key keeps it, so the draw can
// be made again). It refuses, drawing nothing, a task that is not a writing
// task, a copy of another task or named twice, a document that is empty, not
// UTF-8 or past maxCopyBytes, sources other than the ones the task names,
// either calibration example missing, and fewer judges than it takes to bring
// every pair of models together.
func drawJudging(in judgingInputs, seed [32]byte) (JudgingPacket, JudgingKey, error) {
	if err := in.validate(); err != nil {
		return JudgingPacket{}, JudgingKey{}, fmt.Errorf("the judging is not drawn: %w", err)
	}
	byCell := map[string][]writingCopy{}
	for _, c := range in.Copies {
		byCell[c.Key.Cell] = append(byCell[c.Key.Cell], c)
	}
	models := make([]string, 0, len(byCell))
	for cell := range byCell {
		models = append(models, cell)
	}
	slices.Sort(models)
	blocks := modelBlocks(len(models))
	if in.Judges < len(blocks) {
		return JudgingPacket{}, JudgingKey{}, fmt.Errorf("the judging is not drawn: %d models need at least %d judges, each ranking at most %d copies, so that every pair of models meets in some judge's subset; %d asked for",
			len(models), len(blocks), maxJudgeSubset, in.Judges)
	}
	rng := rand.New(rand.NewChaCha8(seed))
	// Each model's copies in an order of their own, taken in turn, so the
	// judges who see one model see different copies of it.
	next := map[string]int{}
	for _, m := range models {
		cs := byCell[m]
		rng.Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] })
	}
	var slots [][]writingCopy
	for s := range in.Judges {
		block := blocks[s%len(blocks)]
		per := max(1, maxJudgeSubset/len(block))
		var sub []writingCopy
		for _, mi := range block {
			cs := byCell[models[mi]]
			for range min(per, len(cs)) {
				sub = append(sub, cs[next[models[mi]]%len(cs)])
				next[models[mi]]++
			}
		}
		rng.Shuffle(len(sub), func(i, j int) { sub[i], sub[j] = sub[j], sub[i] })
		slots = append(slots, sub)
	}

	// Each copy any subset shows is one document, under an id drawn apart
	// from its trial.
	var shown []writingCopy
	seen := map[TrialKey]bool{}
	for _, sub := range slots {
		for _, c := range sub {
			if !seen[c.Key] {
				seen[c.Key] = true
				shown = append(shown, c)
			}
		}
	}
	rng.Shuffle(len(shown), func(i, j int) { shown[i], shown[j] = shown[j], shown[i] })
	docOf := map[TrialKey]string{}
	p := JudgingPacket{
		SchemaVersion: JudgingSchema, Criteria: in.Task.Criteria,
		GuestsMayJudge: in.GuestsMayJudge, Template: renderTemplate(),
		Calibration: Calibration{
			Good: JudgingDoc{ID: "good", HTML: renderInert("A good example", in.Good)},
			Bad:  JudgingDoc{ID: "bad", HTML: renderInert("A bad example", in.Bad)},
		},
	}
	k := JudgingKey{SchemaVersion: JudgingSchema, Run: in.Run, Task: in.Task.ID, Seed: hex.EncodeToString(seed[:]), Documents: map[string]KeyedCopy{}}
	// A source is titled by its place: the path it is read from is the
	// repository's, and no path reaches a judge.
	for i, s := range in.Sources {
		p.Sources = append(p.Sources, JudgingDoc{ID: fmt.Sprintf("source-%d", i+1), HTML: renderInert(fmt.Sprintf("Source %d", i+1), s.Text)})
	}
	for i, c := range shown {
		id := fmt.Sprintf("d%d", i+1)
		docOf[c.Key] = id
		text, withheld := in.scrub(c)
		p.Documents = append(p.Documents, JudgingDoc{ID: id, HTML: renderInert("Copy", text)})
		k.Documents[id] = KeyedCopy{Key: c.Key, Words: len(strings.Fields(text)), Bytes: len(text), Withheld: withheld}
	}
	for s, sub := range slots {
		slot := JudgeSlot{ID: fmt.Sprintf("s%d", s+1)}
		codes := map[string]bool{}
		for _, c := range sub {
			code := drawCode(rng, codes)
			slot.Copies = append(slot.Copies, SlotCopy{Code: code, Document: docOf[c.Key]})
		}
		p.Slots = append(p.Slots, slot)
	}
	d, err := p.digest()
	if err != nil {
		return JudgingPacket{}, JudgingKey{}, err
	}
	k.Packet = d
	return p, k, nil
}

// scrub is copy c's text as a judge reads it: every mention of where it ran
// (its clone, the clone store, the run's checkout, the home directory, the
// run's id) and everything the identity scrub withholds (every name in the
// run that could identify a copy, and every name of a model or its maker)
// withheld, as a check copy's own words are (identityScrub), so a copy cannot
// sign its work in front of its judges. It returns how many mentions it
// withheld. A writing copy is the model's own words throughout: no part of it
// is the frozen material's, so nothing of it is kept whole.
func (in judgingInputs) scrub(c writingCopy) (string, int) {
	dir := ""
	if in.Store != "" && c.Alias != "" {
		dir = cloneDir(in.Store, in.Run, c.Alias)
	}
	red := checkRedactor{
		place: newRedactor(reviewInputs{RepoRoot: in.RepoRoot, Store: in.Store, Dir: dir, Run: in.Run}),
		model: newIdentityScrub(in.Run, c.Key, runIdentity{Tasks: append([]string{in.Task.ID}, in.Tasks...), Cells: in.Cells, Alias: c.Alias}),
	}
	return red.text(c.Text)
}

func (in judgingInputs) validate() error {
	if !runIDRe.MatchString(in.Run) {
		return fmt.Errorf("%q is not a run id", echo(in.Run))
	}
	if in.Task.Kind != KindWriting {
		return fmt.Errorf("the task %s is a %s task; people judge a writing task's copies", in.Task.ID, in.Task.Kind)
	}
	if err := in.Task.validate(); err != nil {
		return fmt.Errorf("task %s: %w", in.Task.ID, err)
	}
	if len(in.Copies) == 0 {
		return fmt.Errorf("the task %s has no copy to judge", in.Task.ID)
	}
	if in.Judges < 1 || in.Judges > maxJudgeSlots {
		return fmt.Errorf("%d judges asked for; from 1 to %d", in.Judges, maxJudgeSlots)
	}
	seen := map[TrialKey]bool{}
	for _, c := range in.Copies {
		if err := c.Key.validate(); err != nil {
			return err
		}
		if c.Key.Task != in.Task.ID {
			return fmt.Errorf("the copy %s is not of the task %s", c.Key, in.Task.ID)
		}
		if seen[c.Key] {
			return fmt.Errorf("the copy %s is given twice", c.Key)
		}
		seen[c.Key] = true
		if err := readable("the copy "+c.Key.String(), c.Text); err != nil {
			return err
		}
	}
	if len(in.Sources) != len(in.Task.Source) {
		return fmt.Errorf("the task %s names %d sources; %d given", in.Task.ID, len(in.Task.Source), len(in.Sources))
	}
	for i, s := range in.Sources {
		if s.Name != in.Task.Source[i] {
			return fmt.Errorf("source %d is %q; the task names %q", i+1, echo(s.Name), in.Task.Source[i])
		}
		if err := readable("the source "+s.Name, s.Text); err != nil {
			return err
		}
	}
	if err := readable("the good example", in.Good); err != nil {
		return err
	}
	return readable("the bad example", in.Bad)
}

// readable holds a document a judge reads to text there is something of, valid
// UTF-8, within maxCopyBytes.
func readable(what, s string) error {
	if strings.TrimSpace(s) == "" || len(s) > maxCopyBytes || !utf8.ValidString(s) {
		return fmt.Errorf("%s is empty, not UTF-8 or past %d bytes", what, maxCopyBytes)
	}
	return nil
}

// drawCode draws a code no copy of the subset holds yet.
func drawCode(rng *rand.Rand, taken map[string]bool) string {
	for {
		var b strings.Builder
		for range codeLen {
			b.WriteByte(codeAlphabet[rng.IntN(len(codeAlphabet))])
		}
		if c := b.String(); !taken[c] {
			taken[c] = true
			return c
		}
	}
}

// modelBlocks is which models share a judge's subset, as indexes of n models:
// all of them when they fit one subset; otherwise the models in groups of half
// a subset, and every two groups one block, so any two models share a block
// and no block holds more than a subset's copies. A judge ranks one block.
func modelBlocks(n int) [][]int {
	if n <= 0 {
		return nil
	}
	all := make([]int, n)
	for i := range all {
		all[i] = i
	}
	if n <= maxJudgeSubset {
		return [][]int{all}
	}
	var groups [][]int
	for chunk := range slices.Chunk(all, maxJudgeSubset/2) {
		groups = append(groups, chunk)
	}
	var out [][]int
	for i := range groups {
		for j := i + 1; j < len(groups); j++ {
			out = append(out, slices.Concat(groups[i], groups[j]))
		}
	}
	return out
}

// copyTemplate is the one document every copy, source and example is rendered
// through. Its policy loads nothing and runs nothing, whatever the text holds;
// the text is escaped before it is placed, so it is never markup.
const copyTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'">
<meta name="referrer" content="no-referrer">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{title}}</title>
<style>body{margin:0;padding:16px;font:16px/1.5 Georgia,serif;color:#1a1a1a;background:#fff}pre{margin:0;white-space:pre-wrap;overflow-wrap:anywhere;font:inherit}@media (prefers-color-scheme:dark){body{color:#e8e8e8;background:#161616}}</style>
</head>
<body><pre>{{text}}</pre></body>
</html>
`

// renderTemplate is the template as the run record keeps it, by content.
func renderTemplate() ContentRef { return contentOf("judging-copy.html", []byte(copyTemplate)) }

// renderInert renders text as an inert document: masked of terminal and
// bidirectional controls (shown as '?', so a judge sees that something was
// there), escaped, and placed in the template's one pre-formatted block.
func renderInert(title, text string) string {
	esc := func(s string) string { return html.EscapeString(termsafe.SanitizeBlock(s)) }
	// One pass of a replacer: nothing placed is read again for a placeholder.
	return strings.NewReplacer("{{title}}", esc(title), "{{text}}", esc(text)).Replace(copyTemplate)
}

// digest is the sha256 of the packet's JSON. A packet past the bound its lab
// reads a record back within has none: it could not be written.
func (p JudgingPacket) digest() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("the judging packet cannot be encoded: %w", err)
	}
	if len(data) > lab.MaxStateBytes {
		return "", errors.New("the judging packet is past the bound its lab reads a record back within; judge fewer or shorter copies")
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
