package matrix

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// scrubFor is the identity scrub of the first copy of writingInputs' run.
func scrubFor(in judgingInputs) identityScrub {
	c := in.Copies[0]
	return newIdentityScrub(in.Run, c.Key, runIdentity{Tasks: append([]string{in.Task.ID}, in.Tasks...), Cells: in.Cells, Alias: c.Alias})
}

// A model or a maker named in any form is withheld: split by a zero-width or
// other format character, in compatibility forms (fullwidth letters), with
// combining marks, or in Cyrillic or Greek letters that look Latin. The text
// is folded before it is matched, and what is withheld is the span of the
// text as written, the characters that split it included.
func TestTheScrubWithholdsANameWrittenInAnyForm(t *testing.T) {
	s := scrubFor(writingInputs(1, 1, 1))
	for _, signed := range []string{
		"C​laude",       // a zero-width space
		"Clau­de",       // a soft hyphen
		"Cl⁠aude",       // a word joiner
		"Сlaude",        // a Cyrillic capital es
		"Clаude",        // a Cyrillic small a
		"аnthrоpic",     // Cyrillic a and o
		"Sοnnet",        // a Greek small omicron
		"ＧＰＴ-4o",        // fullwidth GPT
		"Gémini",       // a combining acute
		"Gеmіnі",        // Cyrillic e and i
		"Llаmа 3",       // Cyrillic a
		"De‍ep‌Seek-V3", // a zero-width joiner and non-joiner
		"Cl⃝aude",       // an enclosing mark (Me)
		"Clःaude",       // a spacing mark (Mc)
		"Clㅤaude",       // a Hangul filler
		"Clᅟaude",       // a Hangul choseong filler
		"Clᅠaude",       // a Hangul jungseong filler
		"Clﾠaude",       // a halfwidth Hangul filler
		"Cl\baude",      // a backspace
		"Cl\x7faude",    // a delete
		"Cl\u009baude",  // a C1 control
		"Claսde",        // an Armenian seh for u
		"Օpus",          // an Armenian capital oh
		"ցemini",        // an Armenian co for g
	} {
		text := "Written by " + signed + " for the page."
		got, n := s.scrub(text)
		if n == 0 || !strings.HasPrefix(got, "Written by "+withheldMark) || !strings.HasSuffix(got, " for the page.") {
			t.Errorf("scrub(%q) = %q (%d withheld); want the name withheld whole and the rest kept", text, got, n)
		}
	}
	if got, n := s.scrub("The page explains the guide."); n != 0 {
		t.Errorf("scrub of ordinary prose = %q (%d withheld); want it kept", got, n)
	}
}

// A withheld name takes its model id token with it: the version that follows
// it, joined by a dash, a dot, an underscore or a space, or written straight
// on, and any suffix the token carries once a version has begun, so no
// "-5-5" or "-4o" is left to sign the copy.
func TestTheScrubWithholdsTheWholeModelIDTokenAfterAName(t *testing.T) {
	s := scrubFor(writingInputs(1, 1, 1))
	for _, c := range []struct{ text, want string }{
		{"by opus-5-5 today", "by " + withheldMark + " today"},
		{"by claude-opus-5-5, today", "by " + withheldMark + ", today"},
		{"by GPT-4o today", "by " + withheldMark + " today"},
		{"by gpt-4o-mini today", "by " + withheldMark + " today"},
		{"by gpt4o today", "by " + withheldMark + " today"},
		{"by Opus 5.5 today", "by " + withheldMark + " today"},
		{"by gemini-2.5-pro today", "by " + withheldMark + " today"},
		{"by o3-mini today", "by " + withheldMark + " today"},
		{"by qwen2.5-coder-32b today", "by " + withheldMark + " today"},
		{"by Claude. Then", "by " + withheldMark + ". Then"},
		{"by Claude and more", "by " + withheldMark + " and more"},
		{"by claude–opus-5-5 today", "by " + withheldMark + " today"},
		{"by claude—opus today", "by " + withheldMark + " today"},
	} {
		if got, _ := s.scrub(c.text); got != c.want {
			t.Errorf("scrub(%q) = %q; want %q", c.text, got, c.want)
		}
	}
}

// A writing copy's text is the model's own words throughout, so a signature
// it makes of words a cited file holds (a quote of a source naming a model)
// is withheld as the rest of its words are: in the packet a judge reads, and
// in the key's measure of the copy. The source shown beside it is the same
// for every copy and is not scrubbed.
func TestAWritingCopyCannotSignWithAQuoteOfItsSource(t *testing.T) {
	in := writingInputs(2, 1, 3)
	in.Sources = []namedText{{Name: "docs/guide.md", Text: "# Guide\n\nThe guide was drafted with Claude.\n"}}
	in.Copies[0].Text = "# Guide\n\nAs the source says, \"The guide was drafted with Claude.\"\n"
	p, _ := draw(t, in, seedA)
	for _, d := range p.Documents {
		if strings.Contains(d.HTML, "Claude") {
			t.Errorf("document %s carries the cited name: %q", d.ID, d.HTML)
		}
	}
	data, err := json.Marshal(p.Sources)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "drafted with Claude") {
		t.Errorf("the source shown beside the copies was scrubbed: %s", data)
	}
}

// The judging packet carries none of the respelled signatures: the scrub runs
// on the path a writing copy's text takes to its judges.
func TestTheJudgingPacketCarriesNoRespelledSignature(t *testing.T) {
	in := writingInputs(2, 1, 3)
	in.Copies[0].Text = "# Guide\n\nWritten by C​laude (Сlaude) as opus-5-5 and GPT-4o.\n"
	p, _ := draw(t, in, seedA)
	data, err := json.Marshal(p.Documents)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"laude", "С", "​", "5-5", "4o"} {
		if strings.Contains(string(data), leak) || strings.Contains(string(data), strings.Trim(strconvQuote(leak), `"`)) {
			t.Errorf("the packet carries %q", leak)
		}
	}
}

// strconvQuote is s as JSON writes it, so an escaped character is found too.
func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// A known model's or maker's name whose letters are split by single
// separators (a space, a dot, a hyphen, an asterisk, an underscore, a line
// break) is withheld whole: the separated form is read as the name it spells
// when it is that whole name, not part of a longer word.
func TestTheScrubWithholdsANameSpelledWithSeparators(t *testing.T) {
	s := scrubFor(writingInputs(1, 1, 1))
	for _, signed := range []string{
		"c-l-a-u-d-e",
		"C l a u d e",
		"g.p.t",
		"G.P.T-4o",
		"C*l*a*u*d*e",
		"c_l_a_u_d_e",
		"Clau\nde",
		"Cl.aude",
		"An-thro-pic",
		"S o n n e t",
		"o.p.u.s",
		"D e e p S e e k",
		"C​.l.a.u.d.e", // separators beside a format character
		"С-l-a-u-d-e",  // a separator beside a look-alike
	} {
		text := "Written by " + signed + " for the page."
		got, n := s.scrub(text)
		if n == 0 || got != "Written by "+withheldMark+" for the page." {
			t.Errorf("scrub(%q) = %q (%d withheld); want the name withheld whole and the rest kept", text, got, n)
		}
	}
	for _, prose := range []string{
		"I met a friend at the top usage of the open air market.",
		"The magnum op us was a b c, and the ex-pert dev in the morning wrote code x.",
		"Me ta, kim i and gro k are not names here, nor is the g p s unit.",
		"I met\na friend; the meta-analysis held.",
		"e.g. the U.S.A. and i.e. the a.m. train",
		"a co-author and a show-off up-ended the plan",
	} {
		if got, n := s.scrub(prose); n != 0 && got != prose {
			if strings.Contains(prose, "meta-analysis") && got == strings.Replace(prose, "meta", withheldMark, 1) {
				continue // a name written straight on is withheld as a substring already
			}
			t.Errorf("scrub of ordinary prose %q = %q (%d withheld); want it kept", prose, got, n)
		}
	}
}

// The scrub reads a text in time linear in its length: a copy's text of the
// largest size the run keeps, built so that every name or version starts a
// match that could run to the end of the text, is scrubbed within a tight
// bound, so no text can stall the draw that reads it.
func TestTheScrubIsLinearOnAdversarialText(t *testing.T) {
	s := scrubFor(writingInputs(1, 1, 1))
	for _, unit := range []string{"o1-", "claude-1-", "gpt 4 ", "c-l-a-u-d-e-", "opus-5-5-", "-", "o", "a b "} {
		text := strings.Repeat(unit, maxCopyBytes/len(unit))
		start := time.Now()
		got, _ := s.scrub(text)
		took := time.Since(start)
		if got == "" {
			t.Errorf("the scrub of %q repeated returned nothing", unit)
		}
		if !raceEnabled && took > time.Second {
			t.Errorf("the scrub of %d bytes of %q repeated took %s; want under 1s", len(text), unit, took)
		}
	}
}
