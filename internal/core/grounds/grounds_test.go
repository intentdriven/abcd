package grounds

import (
	"strconv"
	"strings"
	"testing"
)

// conjecture is a text that clears the substance floor, used wherever a test is
// about something other than the text.
const conjecture = "we expect a stamped identity to survive rewording"

// TestGroundsParseVocabulary holds the closed set: the three values parse, and
// everything else is refused with nothing recorded.
func TestGroundsParseVocabulary(t *testing.T) {
	for _, tok := range []Token{Pursued, Deferred, Declined} {
		g, err := Parse(string(tok) + ": " + conjecture)
		if err != nil {
			t.Fatalf("Parse(%q) = %v, want the token accepted", tok, err)
		}
		if g.Token != tok {
			t.Fatalf("Parse(%q) token = %q, want %q", tok, g.Token, tok)
		}
		if g.Text != conjecture {
			t.Fatalf("Parse(%q) text = %q, want %q", tok, g.Text, conjecture)
		}
	}
	for _, bad := range []string{
		"planned: " + conjecture,
		"PURSUED!: " + conjecture,
		"",
		conjecture,
		": " + conjecture,
	} {
		if _, err := Parse(bad); err == nil {
			t.Fatalf("Parse(%q) = nil error, want a refusal", bad)
		}
	}
}

// TestGroundsParseSplitsOnFirstColon: the text is prose and may itself carry a
// colon; only the FIRST colon separates the token from the text.
func TestGroundsParseSplitsOnFirstColon(t *testing.T) {
	const text = "the gate refuses it: nothing else reads the record"
	g, err := Parse("pursued: " + text)
	if err != nil {
		t.Fatal(err)
	}
	if g.Token != Pursued {
		t.Fatalf("token = %q, want pursued", g.Token)
	}
	if g.Text != text {
		t.Fatalf("text = %q, want %q", g.Text, text)
	}
}

// TestGroundsParseChecksGrammarNotTheFloor: Parse is the reader's gate over a
// value already written, so it judges the grammar and the vocabulary and leaves
// the substance floor to New — a reader that applied the floor would skip
// records the ledger has always accepted.
func TestGroundsParseChecksGrammarNotTheFloor(t *testing.T) {
	if _, err := Parse("declined: out of scope"); err != nil {
		t.Fatalf("Parse of a short but well-formed value = %v, want it accepted", err)
	}
	if _, err := Parse("declined:   "); err == nil {
		t.Fatal("Parse of a value with no text = nil error, want a refusal")
	}
}

// TestGroundsRefusesDegenerateText pins the substance floor: the shape check
// that refuses the cases carrying no reasoning at all. It is a floor, not a
// judgement — whether a text really names a conjecture is review's question.
func TestGroundsRefusesDegenerateText(t *testing.T) {
	for name, text := range map[string]string{
		"empty":       "",
		"whitespace":  "   \t  ",
		"below floor": "because",
		"token only":  "pursued",
		"token twice": "pursued pursued pursued pursued",
		"verb only":   "promote promote promote promote",
	} {
		if _, err := New(Pursued, text); err == nil {
			t.Fatalf("%s: New(%q) = nil error, want a refusal", name, text)
		}
	}
	if _, err := New(Pursued, conjecture); err != nil {
		t.Fatalf("New(%q) = %v, want the conjecture accepted", conjecture, err)
	}
}

// TestGroundsRenderRoundTrip: what the renderer writes is what the parser
// reads, so the intent writer, the ledger writer and the lint share one copy of
// the grammar rather than three spellings of it.
func TestGroundsRenderRoundTrip(t *testing.T) {
	g, err := New(Declined, "declined because the ledger already carries the reason")
	if err != nil {
		t.Fatal(err)
	}
	back, err := Parse(g.String())
	if err != nil {
		t.Fatalf("Parse(%q) = %v", g.String(), err)
	}
	if back != g {
		t.Fatalf("round trip = %+v, want %+v", back, g)
	}
	if got, want := g.Bullet(), "- "+g.String(); got != want {
		t.Fatalf("Bullet() = %q, want %q", got, want)
	}
}

// TestGroundsCollapsesWhitespace: the value is written as one YAML scalar and
// one Markdown bullet, so a multi-line operand is folded rather than carried
// into a shape neither surface can hold.
func TestGroundsCollapsesWhitespace(t *testing.T) {
	g, err := Parse("pursued:\n  we expect a stamped identity\n  to survive rewording\n")
	if err != nil {
		t.Fatal(err)
	}
	if g.Text != "we expect a stamped identity to survive rewording" {
		t.Fatalf("text = %q, want the folded single line", g.Text)
	}
}

// TestGroundsFloorCountsLettersNotRunes is iss-2608301206034359: the floor
// measured rune LENGTH, so a text with no letters in it at all cleared it —
// twenty zero-width spaces, twenty dots, twenty digits. The refusal this
// vocabulary was promoted to make is not a floor if it is answerable with
// nothing.
func TestGroundsFloorCountsLettersNotRunes(t *testing.T) {
	for name, text := range map[string]string{
		"zero-width spaces": strings.Repeat("​", 20),
		"dots":              strings.Repeat(".", 20),
		"digits":            "12345678901234567890",
		"one long word":     "internationalisation",
		"two words":         "unrecoverable regressions",
		"padded one word":   "supercalifragilistic ... 123",
	} {
		if _, err := New(Pursued, text); err == nil {
			t.Fatalf("%s: New(%q) = nil error, want a refusal", name, text)
		}
		// The reader holds a hand-typed bullet to the same floor.
		if err := ValidateText(Fold(text)); err == nil {
			t.Fatalf("%s: ValidateText(%q) = nil error, want a refusal", name, text)
		}
	}
	if _, err := New(Pursued, conjecture); err != nil {
		t.Fatalf("New(%q) = %v, want the conjecture accepted", conjecture, err)
	}
}

// TestGroundsRefusesControlCharacters is iss-2608301206032013: Go's RE2 `\s`
// excludes the vertical tab, so Fold carried U+000B through and the pre-mint
// gate passed a value the frontmatter serialiser then rejected under the ledger
// lock — after the draft had been minted. The argument boundary refuses exactly
// what yamlScalar refuses, so no route reaches the serialiser carrying one.
func TestGroundsRefusesControlCharacters(t *testing.T) {
	for name, text := range map[string]string{
		"vertical tab": "we expect the gate\vto refuse a control character",
		"NUL":          "we expect the gate\x00to refuse a control character",
		"bell":         "we expect the gate\ato refuse a control character",
		"escape":       "we expect the gate\x1bto refuse a control character",
		"tab-adjacent": "we expect the gate to\v refuse a control character",
	} {
		if _, err := New(Pursued, text); err == nil {
			t.Fatalf("%s: New(%q) = nil error, want a refusal", name, text)
		}
		if err := ValidateText(Fold(text)); err == nil {
			t.Fatalf("%s: ValidateText = nil error, want a refusal", name)
		}
	}
	// The whitespace Fold already normalises is not a control-character refusal:
	// a multi-line operand is folded, not rejected, and a control character at
	// either END is trimmed away by Fold before the text is judged — it is not in
	// the value, so there is nothing to refuse.
	for name, text := range map[string]string{
		"folded multi-line":  "we expect a stamped identity\n\tto survive rewording",
		"trimmed at the end": "\vwe expect a stamped identity to survive rewording\v",
	} {
		if _, err := New(Pursued, text); err != nil {
			t.Fatalf("%s: New = %v, want it accepted", name, err)
		}
	}
}

// TestGroundsFloorCountsScriptioContinuaCharacters is iss-2608301301044588: the
// floor was measured in letter-RUNS, and a script written without inter-word
// spaces has exactly one of them however much it says. A substantive Chinese or
// Japanese conjecture was refused as "1 word", and because the reader applies
// the same floor the bullet was then silently dropped — so the readiness gate
// reported no recorded grounds about a record that visibly carried one. The
// argument is mandatory, so that refused an operator writing in those scripts
// from promoting or resolving anything at all.
func TestGroundsFloorCountsScriptioContinuaCharacters(t *testing.T) {
	for name, text := range map[string]string{
		"chinese":  "我们预期带有戳记的身份能够在改写之后继续存活下来",
		"japanese": "スタンプされた識別子は改名されても生き残ると予期している",
		"mixed":    "識別子 ABCD は改名されても生き残ると予期している",
		// Natural prose in most of these scripts is inert as a pin: its combining
		// marks are Mn rather than L, so they broke the OLD letter-runs too and
		// the text was never refused (the issue record calls this "Thai survives
		// by accident"). Measured, prose leaves seven of the nine entries free to
		// be deleted with the suite still green. A MARK-FREE run of a script's
		// letters is one run under the old count and one unit per letter under
		// this one, so each of these fails if its entry leaves the set — which is
		// the only thing that pins the set. They are alphabet runs, not prose:
		// what they assert is the counting, and readability is not the floor's
		// claim.
		"hiragana letter run": "あいうえおかきくけこさしすせそたちつてと",
		"katakana letter run": "アイウエオカキクケコサシスセソタチツテト",
		"thai letter run":     "กขคงจฉชซฌญฎฏฐฑฒณดตถทธน",
		"lao letter run":      "ກຂຄງຈສຊຍດຕຖທນບປຜຝພຟມ",
		"khmer letter run":    "កខគឃងចឆជឈញដឋឌឍណតថទធន",
		"myanmar letter run":  "ကခဂဃငစဆဇဈဉညဋဌဍဎဏတထဒဓ",
		"tibetan letter run":  "ཀཁགངཅཆཇཉཏཐདནཔཕབམཙཚཛཝ",
		"javanese letter run": "ꦲꦤꦕꦫꦏꦢꦠꦱꦮꦭꦥꦝꦗꦪꦚꦩꦒꦧꦛꦔ",
		// The prose cases stay as the shapes an operator would actually type.
		"thai prose":    "เราคาดว่าตัวระบุที่ประทับตราไว้จะอยู่รอดหลังการเขียนใหม่",
		"khmer prose":   "យើងរំពឹងថាអត្តសញ្ញាណដែលមានត្រាបោះនឹងនៅរស់រានក្រោយការសរសេរឡើងវិញ",
		"lao prose":     "ພວກເຮົາຄາດວ່າຕົວລະບຸທີ່ມີຕາປະທັບຈະຢູ່ລອດຫຼັງການຂຽນຄືນໃໝ່",
		"burmese prose": "တံဆိပ်ခတ်ထားသောအမှတ်အသားသည်ပြန်လည်ရေးသားပြီးနောက်ရှင်သန်နေမည်ဟုမျှော်လင့်သည်",
		"tibetan prose": "ཐེལ་ཙེ་བརྒྱབ་པའི་ངོས་འཛིན་དེ་ཡང་བསྐྱར་འབྲི་རྗེས་གནས་ཐུབ་པར་རེ་བ་བྱེད",
	} {
		if _, err := New(Pursued, text); err != nil {
			t.Fatalf("%s: New(%q) = %v, want the conjecture accepted", name, text, err)
		}
		// The reader inherits the same floor, so the bullet is read back rather
		// than silently dropped under the record's `## Grounds` heading.
		if err := ValidateText(Fold(text)); err != nil {
			t.Fatalf("%s: ValidateText(%q) = %v, want it accepted", name, text, err)
		}
	}
}

// TestGroundsScriptioContinuaFloorRefusesPadding pins the half that must NOT
// move: counting one character of a script without word breaks as one unit is
// an admission, and an admission is where a closed hole reopens. A text padded
// with characters that render as emptiness, or with punctuation, carries no
// units however long it is; and appending one ideograph to a single long word
// does not buy the two units it lacks.
func TestGroundsScriptioContinuaFloorRefusesPadding(t *testing.T) {
	for name, text := range map[string]string{
		"zero-width padded ideograph":  strings.Repeat("​", 19) + "中",
		"ideographic punctuation":      strings.Repeat("、。", 10),
		"one long word plus ideograph": "supercalifragilistic 中",
		"fullwidth digits":             "１２３４５６７８９０１２３４５６７８９０",
		// A Unicode SCRIPT table is not a letter table: it carries the script's
		// own digits, punctuation and combining marks too. Counting a script's
		// characters therefore has to count its LETTERS, or the digit-and-dot
		// padding iss-2608301206034359 closed reopens once per script — and the
		// Tibetan tsheg case is twenty dots exactly.
		"thai digits":            strings.Repeat("๐", 20),
		"lao digits":             strings.Repeat("໐", 20),
		"tibetan tsheg":          strings.Repeat("་", 20),
		"khmer khan":             strings.Repeat("។", 20),
		"myanmar section mark":   strings.Repeat("၊", 20),
		"javanese pada":          strings.Repeat("꧈", 20),
		"thai vowel signs":       strings.Repeat("่", 20),
		"thai digits + one word": "supercalifragilistic ๐๐",
		// Korean is written WITH inter-word spaces, so it is judged by the word
		// count like any other spaced script: two words is two words.
		"korean two words": "안녕하십니까여러분들 안녕하십니까여러분들",
	} {
		if _, err := New(Pursued, text); err == nil {
			t.Fatalf("%s: New(%q) = nil error, want a refusal", name, text)
		}
		if err := ValidateText(Fold(text)); err == nil {
			t.Fatalf("%s: ValidateText(%q) = nil error, want a refusal", name, text)
		}
	}
}

// TestGroundsFloorRefusalNamesTheUnitThatRefused: the message a caller is shown
// must be true of the text that was refused. Telling an operator writing a
// script with no word breaks to add words names a remedy their language cannot
// supply, and the message the gate prints is the only instruction they get.
func TestGroundsFloorRefusalNamesTheUnitThatRefused(t *testing.T) {
	_, err := New(Pursued, strings.Repeat("​", 19)+"中")
	if err == nil {
		t.Fatal("New = nil error, want a refusal")
	}
	if strings.Contains(err.Error(), "word floor") || !strings.Contains(err.Error(), "letter(s)") {
		t.Fatalf("refusal of a text without word breaks = %q, want it to ask for letters and not for words", err)
	}
	for _, text := range []string{
		"supercalifragilistic ... 123",
		// A text that carries a word ALONGSIDE an ideograph has words, and can
		// honestly be told to add one — so the character message is wrong here,
		// and would also report a twenty-letter word as one character.
		"supercalifragilistic 中",
	} {
		_, err = New(Pursued, text)
		if err == nil {
			t.Fatalf("New(%q) = nil error, want a refusal", text)
		}
		if !strings.Contains(err.Error(), "word floor") {
			t.Fatalf("refusal of %q = %q, want the word floor named", text, err)
		}
	}
}

// TestGroundsNoWordBreaksRefusalIsOnlyForScriptsWithout is iss-2608301455387795:
// scriptioContinuaOnly picks the refusal that asks for LETTERS instead of words,
// and two of its conditions had nothing holding them. Without the empty-unit
// guard, a text carrying no letters at all takes the branch vacuously — "every
// unit is a scriptio-continua letter" is true of no units. Without the letter
// test, so does a text whose units are single-letter words of a script that does
// separate words. Either way the caller is told the floor asks for letters
// "where the script has no word breaks", which is not true of the text in front
// of them and names a remedy that is not theirs; the word floor is.
func TestGroundsNoWordBreaksRefusalIsOnlyForScriptsWithout(t *testing.T) {
	for name, text := range map[string]string{
		"no letters at all":   strings.Repeat(".", 20),
		"single-letter words": "a b" + strings.Repeat(".", 18),
	} {
		_, err := New(Pursued, text)
		if err == nil {
			t.Fatalf("%s: New(%q) = nil error, want a refusal", name, text)
		}
		if strings.Contains(err.Error(), "no word breaks") {
			t.Fatalf("%s: refusal of %q = %q, want a text whose script has word breaks not told otherwise", name, text, err)
		}
		if !strings.Contains(err.Error(), "word floor") {
			t.Fatalf("%s: refusal of %q = %q, want the word floor named", name, text, err)
		}
	}
}

// TestControlRefusalClaimsOnlyWhatItChecks is iss-2608301646042379, re-anchored
// by iss-2608301657350399. The refusal said the rune was one "no record field
// can hold". The check is r < 0x20, congruent with yamlScalar; the store holds
// DEL, C1, a line separator and a bidi override quite happily, and the round-5
// security review round-tripped all four through committed grounds and
// wontfix_reason scalars.
//
// The wording is narrowed rather than the check widened, because widening would
// put this floor out of step with the serialiser it deliberately mirrors, which
// is the gate-versus-reader split this repository treats as its own defect class.
//
// The guard is NOT a ban on the one sentence the defect used. That was defeated
// by any synonym — rewriting it to "no record field can CARRY" restores the
// defect verbatim in meaning and leaves such a test green. What is asserted here
// instead is the property that sentence got wrong, which is SCOPE: the refused
// set is exactly the runes below U+0020, proved in both directions; the message
// names the narrow authority the check mirrors; and it may not name the record,
// the store or a record field as the thing that cannot hold the rune, whatever
// verb it reaches for.
func TestControlRefusalClaimsOnlyWhatItChecks(t *testing.T) {
	const carrier = "we expect the conjecture to outlive the session and be read later"

	// Scope, in both directions. A message describing a class wider than this is
	// describing something this floor does not do.
	for r := rune(0); r < 0x20; r++ {
		if err := ValidateText(carrier + string(r)); err == nil {
			t.Errorf("U+%04X is below the serialiser's floor and must be refused", r)
		}
	}
	for _, held := range []rune{0x7F, 0x9B, 0x2028, 0x202E} {
		if err := ValidateText(carrier + string(held)); err != nil {
			t.Errorf("U+%04X round-trips through a committed scalar, so this floor must not refuse it: %v", held, err)
		}
	}

	err := ValidateText("we expect the conjecture to outlive the session\x01 and be read later")
	if err == nil {
		t.Fatal("a text carrying U+0001 must be refused")
	}
	msg := strings.ToLower(err.Error())
	// The positive half: the message names the authority the check actually
	// mirrors. A rewrite that widens the claim must drop this or contradict it,
	// and a synonym for the verb cannot satisfy it.
	if !strings.Contains(msg, "serialiser") {
		t.Errorf("the refusal does not name the frontmatter serialiser, the only authority the check mirrors: %q", err)
	}
	// The negative half is over the NOUNS that make a claim store-wide, not over
	// one whole sentence: no choice of verb makes "no record field can ..."
	// acceptable, and swapping the verb is exactly how the first guard fell.
	for _, overclaim := range []string{"record field", "no record", "any record", "record can", "store can", "ledger can"} {
		if strings.Contains(msg, overclaim) {
			t.Errorf("the refusal claims a store-wide rule the store does not have (%q): %q", overclaim, err)
		}
	}
}

// TestScriptioContinuaOnlyNeedsTheWholeUnit is iss-2608301620343236. The
// unit-length half of scriptioContinuaOnly's test has no fixture reaching it
// through ValidateText: textUnits gives every scriptio-continua letter a unit of
// its own, so a multi-rune unit never begins with one and the sibling letter
// test already rejects it. Deleting the length test therefore leaves the whole
// suite green.
//
// It is kept rather than deleted because it is not defensive scaffolding: it is
// half of what the predicate MEANS. "Every unit is a single scriptio-continua
// letter" and "every unit begins with one" are different claims that agree only
// because of how textUnits splits today, and the weaker one would go on
// agreeing silently if that ever changed. So the guard is pinned here directly,
// at the helper, which is the only level a fixture for it exists at.
func TestScriptioContinuaOnlyNeedsTheWholeUnit(t *testing.T) {
	if scriptioContinuaOnly([]string{"字a"}) {
		t.Error("a multi-rune unit beginning with an ideograph is not a script without word breaks; " +
			"the predicate must read the whole unit, not its first rune")
	}
	if !scriptioContinuaOnly([]string{"字", "文"}) {
		t.Error("units that are each one ideograph are a script without word breaks")
	}
	if scriptioContinuaOnly(nil) {
		t.Error("no units is not a script without word breaks")
	}
}

// TestScriptioContinuaRefusalNamesBothFloors is iss-2608301620346560. The
// no-word-breaks refusal named the unit floor alone, and for a script with no
// word breaks that is never the floor that binds: every unit is one letter, so
// twenty letters is twenty units and the three-unit floor is cleared long
// before the letter floor is. An author told "the floor asks for 3" who adds an
// ideograph is refused a second time, by a number the first refusal did not
// mention, in the same noun it had just counted.
//
// A refusal states a number so the author knows what would be enough. Stating
// one that would not be enough is the cycle's standing class reached by
// omission rather than by assertion.
func TestScriptioContinuaRefusalNamesBothFloors(t *testing.T) {
	_, err := New(Pursued, "字文")
	if err == nil {
		t.Fatalf("New(%q) = nil error, want a refusal", "字文")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no word breaks") {
		t.Fatalf("refusal of a two-ideograph text = %q, want the no-word-breaks refusal", msg)
	}
	for _, want := range []string{strconv.Itoa(MinTextWords), strconv.Itoa(MinTextLetters)} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal %q does not name the floor %q; an author satisfying only the floor it names is refused again", msg, want)
		}
	}
}

// TestGroundsFloorCountsLettersNotPadding is iss-2608301455387735: the character
// half of the floor counted RUNES, so anything that occupied a rune answered it.
// Three one-letter words and seventeen dots cleared a twenty-"character" floor
// while carrying three letters of reasoning; and a text of Hangul fillers —
// letters Unicode marks default-ignorable, which render as nothing — cleared the
// whole floor while being invisible from end to end. A floor a caller can
// satisfy with characters that say nothing is not a floor.
func TestGroundsFloorCountsLettersNotPadding(t *testing.T) {
	const zeroWidth = "​"
	const filler = "ㅤ"
	for name, text := range map[string]string{
		"one-letter words padded with dots":     "a b c" + strings.Repeat(".", 17),
		"one-letter words padded with digits":   "a b c 1234567890123456",
		"invisible letters, zero-width padding": filler + strings.Repeat(zeroWidth, 9) + filler + strings.Repeat(zeroWidth, 9) + filler,
		"twenty invisible letters":              strings.TrimSuffix(strings.Repeat(filler+zeroWidth, 20), zeroWidth),
		"invisible letters, spaced":             strings.TrimSuffix(strings.Repeat(filler+" ", 20), " "),
	} {
		if _, err := New(Pursued, text); err == nil {
			t.Fatalf("%s: New(%q) = nil error, want a refusal", name, text)
		}
		// The reader holds a hand-typed bullet to the same floor.
		if err := ValidateText(Fold(text)); err == nil {
			t.Fatalf("%s: ValidateText(%q) = nil error, want a refusal", name, text)
		}
	}
	// A text with the units but not the letters is told which of the two it is
	// short of, so a caller who padded it learns what the padding did not buy.
	_, err := New(Pursued, "a b c"+strings.Repeat(".", 17))
	if err == nil {
		t.Fatal("New = nil error, want a refusal")
	}
	if !strings.Contains(err.Error(), "letter floor") {
		t.Fatalf("refusal of a padded text = %q, want the letter floor named", err)
	}
	if _, err := New(Pursued, conjecture); err != nil {
		t.Fatalf("New(%q) = %v, want the conjecture accepted", conjecture, err)
	}
}

// TestGroundsFloorRefusesEveryVocabularyWordAlone is the gate the degeneracy set
// exists to be: a text made SOLELY of a vocabulary value carries no reasoning,
// for EVERY value the set holds. It loops the vocabulary rather than naming the
// three, so a fourth value that reached Vocabulary without reaching the
// degeneracy set fails here instead of silently passing the floor
// (iss-2608301836222858).
func TestGroundsFloorRefusesEveryVocabularyWordAlone(t *testing.T) {
	for _, v := range Vocabulary {
		text := strings.TrimSpace(strings.Repeat(string(v)+" ", 4))
		if _, err := New(Pursued, text); err == nil {
			t.Errorf("New(%q) = nil error, want a refusal — the text is only the vocabulary", text)
		}
	}
	for _, verb := range askingVerbs {
		text := strings.TrimSpace(strings.Repeat(verb+" ", 5))
		if _, err := New(Pursued, text); err == nil {
			t.Errorf("New(%q) = nil error, want a refusal — the text is only the verb's own name", text)
		}
	}
}

// TestDegenerateWordsCoversVocabulary asserts the derivation directly. The
// behavioural test above is the one that matters, but this says WHY it passes,
// so a failure names the missing key rather than an accepted text.
func TestDegenerateWordsCoversVocabulary(t *testing.T) {
	for _, v := range Vocabulary {
		if !degenerateWords[string(v)] {
			t.Errorf("degenerateWords is missing the vocabulary value %q", v)
		}
	}
}

// TestVocabularyRenderings pins the three spellings the surfaces render, so a
// change to the closed set fails here — beside the set — rather than in a
// regenerated reference page nobody read. The literals are deliberate: they are
// what a search for the usage text lands on now that no surface types it.
func TestVocabularyRenderings(t *testing.T) {
	if got, want := UsageSpelling(), "<pursued|deferred|declined>"; got != want {
		t.Errorf("UsageSpelling() = %q, want %q", got, want)
	}
	if got, want := ProseList(), "pursued | deferred | declined"; got != want {
		t.Errorf("ProseList() = %q, want %q", got, want)
	}
	if got, want := vocabularyList(), "pursued/deferred/declined"; got != want {
		t.Errorf("vocabularyList() = %q, want %q", got, want)
	}
	for _, v := range Vocabulary {
		if !strings.Contains(UsageSpelling(), string(v)) || !strings.Contains(ProseList(), string(v)) {
			t.Errorf("the renderings do not carry the vocabulary value %q", v)
		}
	}
}
