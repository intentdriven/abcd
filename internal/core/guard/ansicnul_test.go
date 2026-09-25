package guard

import (
	"fmt"
	"strings"
	"testing"
)

// TestForgedMarkIsTruncatedLikeBash — review3-guard finding 1. An ANSI-C
// escape could decode to the byte the tokenizer uses as its mark, AFTER Check
// had stripped the line's own: `$'\x00'git` read as an unknown word, and an
// unknown command matched as text allowed every blocker. bash ends an ANSI-C
// string at its first NUL, so `$'\x00'git` is `git`, and the guard reads it so.
func TestForgedMarkIsTruncatedLikeBash(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`$'\x00'git push --force origin main`, VerdictBlock, "git-push-force"},
		{`$'\0'git push --force origin main`, VerdictBlock, "git-push-force"},
		{`$'\u0000'git push --force origin main`, VerdictBlock, "git-push-force"},
		{`$'\U00000000'git push --force origin main`, VerdictBlock, "git-push-force"},
		{`$'\c@'git push --force origin main`, VerdictBlock, "git-push-force"},
		{`$'\x00'gh repo delete o/r`, VerdictBlock, "gh-repo-delete"},
		{`$'\x00'pkill -f x`, VerdictBlock, "pkill-by-pattern"},
		{`cd s && $'\x00'rm -rf *`, VerdictBlock, "rm-rf-after-cd-chain"},
		{`$'\x00'sudo git push --force origin main`, VerdictBlock, "git-push-force"},
		{`sh -c "$'\x00'git push --force origin main"`, VerdictBlock, "git-push-force"},
		// Everything after the NUL is dropped, up to the closing quote.
		{`git push $'--force\x00ignored' origin main`, VerdictBlock, "git-push-force"},
		{`$'git\x00ls' push --force origin main`, VerdictBlock, "git-push-force"},
	})
}

// TestAnsiCEscapeNeverDecodesTheMark is the forged mark's property, over every
// escape form bash decodes: no ANSI-C string, whatever it spells, puts the mark
// into a word. The mark stays unforgeable by construction — Check drops the
// line's own NULs, and this is the only other way a byte reaches a word.
func TestAnsiCEscapeNeverDecodesTheMark(t *testing.T) {
	var forms []string
	for v := 0; v < 256; v++ {
		forms = append(forms, fmt.Sprintf(`\x%02x`, v), fmt.Sprintf(`\x%x`, v))
	}
	for v := 0; v < 01000; v++ {
		forms = append(forms, fmt.Sprintf(`\%o`, v), fmt.Sprintf(`\%03o`, v))
	}
	for v := 0; v < 0x10000; v += 0x7f {
		forms = append(forms, fmt.Sprintf(`\u%04x`, v), fmt.Sprintf(`\u%x`, v))
	}
	forms = append(forms, `\u0000`, `\u0`, `\U0`, `\U00000000`, `\U0000`, `\U110000`)
	for c := 0x20; c < 0x7f; c++ {
		form := `\c` + string(rune(c))
		if c == '\\' {
			form += `\` // a backslash after \c is one of a pair, as in any escape
		}
		forms = append(forms, form)
	}
	forms = append(forms, `\c\'`, `\c`)
	for _, form := range forms {
		lines := []string{"a$'" + form + "'b", "$'" + form + "'", "$'x" + form + "y" + form + "'"}
		if form == `\c'` {
			// `\c'` is `\c` and the string's closing quote, as bash reads it
			// (review4-guard finding 2): the form closes its own string.
			lines = []string{"a$'" + form + "b", "$'" + form, "$'x" + form + "y$'" + form}
		}
		for _, line := range lines {
			segs, err := tokenize(line)
			if err != nil {
				t.Fatalf("tokenize(%q): %v", line, err)
			}
			for _, s := range segs {
				for _, tok := range s.tokens {
					if isUnknown(tok) {
						t.Fatalf("tokenize(%q) put the mark into the word %q: an escape forged it", line, tok)
					}
				}
			}
		}
	}
}

// TestUnparsableDecisionBlocks — review4-guard finding 2, the half behind the
// decoder. A line Check cannot split is answered on the hook by
// UnparsableDecision: a block under a reserved id that no registry entry may
// claim, with the way past, never a pass.
func TestUnparsableDecisionBlocks(t *testing.T) {
	_, err := Defaults().Check(`rm -rf "unterminated`)
	if err == nil {
		t.Fatal("an unterminated double quote must stay unparsable in command text")
	}
	d := UnparsableDecision(err)
	if d.Verdict != VerdictBlock || d.EntryID != unparsableEntryID || !contains(d.Matches, unparsableEntryID) {
		t.Errorf("UnparsableDecision = %q via %q (matches %v), want block via %q", d.Verdict, d.EntryID, d.Matches, unparsableEntryID)
	}
	if d.Successor == "" || d.Why == "" || !strings.Contains(d.Why, "unterminated double quote") {
		t.Errorf("the block must say what did not parse and the way past: why %q, successor %q", d.Why, d.Successor)
	}
	if !containsString(reservedEntryIDs, unparsableEntryID) {
		t.Errorf("reservedEntryIDs = %v, want %q listed", reservedEntryIDs, unparsableEntryID)
	}
}

// TestAnsiCStringEndingInAnEscapeCloses — review4-guard finding 2. bash finds
// the end of an ANSI-C string first, stepping each backslash pair, and only
// then decodes what it holds, so no escape can consume the closing quote. The
// decoder took `\c` and the byte after it whatever that byte was, so `$'\c'`
// swallowed its own quote, the line did not parse, and the hook ran it
// unchecked. `\c\\` is one escape (control-backslash), as bash decodes it.
func TestAnsiCStringEndingInAnEscapeCloses(t *testing.T) {
	const push = "git push --force origin main"
	var cases []verdictCase
	for _, str := range []string{`$'\c'`, `$'\c\\'`, `$'\\'`, `$'\x'`, `$'\u'`, `$'\U'`, `$'\0'`, `$'a\c'`, `$'\c\'x'`} {
		for _, sep := range []string{"; ", " && ", " || ", "\n", " | "} {
			cases = append(cases,
				verdictCase{str + sep + push, VerdictBlock, "git-push-force"},
				verdictCase{"x=" + str + sep + push, VerdictBlock, "git-push-force"},
				verdictCase{"echo " + str + sep + "gh repo delete o/r", VerdictBlock, "gh-repo-delete"},
			)
		}
	}
	cases = append(cases,
		verdictCase{`$'\c'; cd s && rm -rf *`, VerdictBlock, "rm-rf-after-cd-chain"},
		verdictCase{`echo $'\c'; ls`, VerdictAllow, ""},
	)
	runVerdictCases(t, cases)

	for line, want := range map[string]string{
		`$'\c\\'`:  "\x1c",
		`$'\c\\x'`: "\x1cx",
		`$'\c\'x'`: "\x1c'x",
		`$'\c?'`:   "\x7f",
		`$'\ca'`:   "\x01",
	} {
		segs, err := tokenize(line)
		if err != nil || len(segs) != 1 || len(segs[0].tokens) != 1 || segs[0].tokens[0] != want {
			t.Errorf("tokenize(%q) = %+v, %v; want the one word %q", line, segs, err, want)
		}
	}
}
