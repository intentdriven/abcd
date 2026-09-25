package guard

import (
	"fmt"
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
		if c == '\'' {
			continue
		}
		forms = append(forms, `\c`+string(rune(c)))
	}
	for _, form := range forms {
		for _, line := range []string{"a$'" + form + "'b", "$'" + form + "'", "$'x" + form + "y" + form + "'"} {
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
