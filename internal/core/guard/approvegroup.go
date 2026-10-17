package guard

// approvegroup.go checks the text an approval hands back (runFrom): the
// change into the working tree and the one bash call that runs the command.
// The check reads that text with the guard's own tokenizer and approves it
// only when it is exactly `cd -- <dir>`, joined by && to `exec /bin/bash
// --noprofile --norc -c <command>` with the whole command as its last word,
// and nothing else: no other command, no redirection, nothing after it.
//
// runFrom spells the command as one single-quoted word, in which no shell
// reads anything but the quote that ends it, and spells each quote the
// command holds as a close, an escaped quote and a reopen. So the outer text
// holds no unquoted byte the command chose, and nothing in the command can
// end the bash call's word early or add a command beside it; the check is
// the guard's own reading of that, and a text it reads any other way is
// refused. The quoting never puts a closing quote straight before an opening
// one, so zsh's RC_QUOTES option, which reads two quotes inside a quoted
// string as one quote, reads the word the same way.

import "slices"

// approvedShell is the shell an approved command runs under: bash, whose
// grammar the guard reads, so the command the guard judged is the command
// that runs, whatever the person's login shell is.
const approvedShell = "/bin/bash"

// runFromIntact is runFrom(dir, command) and whether that text is intact:
// the change of directory and the one bash call holding the whole command. A
// dir that is not absolute is not wrapped, and the command is returned as it
// is.
func runFromIntact(dir, command string) (string, bool) {
	text := runFrom(dir, command)
	if text == command {
		return text, true
	}
	return text, wrapIntact(dir, command, text)
}

// wrapIntact reports whether text, read with the guard's own tokenizer, is
// exactly `cd -- dir`, joined by && to `exec /bin/bash --noprofile --norc -c
// command`, with nothing else in it.
func wrapIntact(dir, command, text string) bool {
	all, err := tokenize(text)
	if err != nil {
		return false
	}
	var segs []segment
	for _, s := range all {
		if s.heredocUnterminated || s.substitutionUnread || s.braceGroup || s.parenWord || len(s.redirects) > 0 {
			return false
		}
		// A segment with no words is a note about the whole text (a name
		// such as IFS it spells, read with its quotes ignored), not a
		// command.
		if len(s.tokens) > 0 {
			segs = append(segs, s)
		}
	}
	if len(segs) != 2 {
		return false
	}
	first, run := segs[0], segs[1]
	return slices.Equal(first.tokens, []string{"cd", "--", dir}) &&
		slices.Equal(run.tokens, []string{"exec", approvedShell, "--noprofile", "--norc", "-c", command}) &&
		run.afterAnd && run.chain == first.chain && run.end == len(text)
}
