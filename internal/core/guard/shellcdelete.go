package guard

// shellcdelete.go refuses a `bash -c` or `sh -c` script built from variables
// that deletes (iss-2610161612200864). What such a script removes is not
// written on the line: the guard reads the words, but a host's own permission
// check that cannot analyse the script stops and asks a person before it
// runs, and an agent working alone waits at that question. The refusal hands
// the agent the reason and the rewrite instead, so it rewrites the command
// before any host has to ask: the script in a file written in one command and
// run in the next, which the script reading checks line by line, or the
// delete on the line itself with a literal path or a `${VAR:?}`-guarded one.
//
// It is an entry-less verdict, read beside the payload reading, because the
// registry's pattern language matches one command's words and cannot say "a
// `-c` string that holds a variable and a deleting command"
// (dcn-2610161621088050). Its rule on the teaching plane is generated from the
// same words (ShellCLesson), so the refusal and the lesson cannot part.
//
// It is narrow on purpose: only a shell's own `-c` string (not eval, a trap,
// `su -c` or `watch`), only when that string holds a parameter expansion the
// shell will fill in, and only when it runs one of deleteCommands. A `-c`
// script with no variable in it, or one that deletes nothing, passes.

// shellCDeleteEntryID is the reserved id the refusal is reported under. No
// registry entry may claim it.
const shellCDeleteEntryID = "shell-c-variable-delete"

// deleteCommands are the commands whose purpose is to remove files: the ones a
// host's check names when it cannot read the script around them.
var deleteCommands = []string{"rm", "rmdir", "shred", "unlink"}

// shellCDeleteSignal is the refusal, the words the decision and the lesson
// both carry.
func shellCDeleteSignal() payloadSignal {
	return payloadSignal{
		id:      shellCDeleteEntryID,
		verdict: VerdictBlock,
		family:  familyShell,
		reason: "What a `-c` script built from variables deletes is named only when it runs, " +
			"so no check made before it runs can read it, and a host whose own check cannot read " +
			"the script stops to ask a person, where an agent working alone waits.",
		successor: "Write the script to a file in one command and run it in the next (`bash /abs/clean.sh`), " +
			"which is read line by line before it runs; or delete on the line itself, outside any `-c` string, " +
			"with literal paths (`/bin/rm -f -- /abs/path`) or a variable guarded as `\"${VAR:?}\"` " +
			"(`rm -f -- \"${VAR:?}\"/*`), so each command can be checked and nobody has to be asked.",
	}
}

// deletesFromVariables reports whether the segments of one `-c` string hold a
// parameter expansion (segment.variable: a word whose unknown parts are all a
// variable's value, which a single-quoted or escaped `$` is not) and run a
// deleting command at a place a command can sit. A program name nothing on
// the line fixes is not read as a delete: that is program-name-unknown's.
func deletesFromVariables(psegs []segment) bool {
	variable, deletes := false, false
	for _, s := range psegs {
		if len(s.variable) > 0 {
			variable = true
		}
		if deletes {
			continue
		}
		for _, a := range commandSites(s) {
			if tok := s.tokens[a.idx]; !isUnknown(tok) && nameCouldBeAny(tok, deleteCommands) {
				deletes = true
			}
		}
	}
	return variable && deletes
}

// leadsOver reports whether the refusal speaks for a line where the named
// blockers also fired: only when every one of them is an entry about a path
// a variable opens (ShapeUnguardedVariablePath), whose rewrite the refusal's
// own includes. Any other blocker keeps its message, because its hazard is
// not the variable (`rm -rf "$HOME"` is a delete of the home directory before
// it is a script no one can read).
func (r Registry) leadsOver(blockers []string) bool {
	for _, id := range blockers {
		e, ok := r.Entries[id]
		if !ok || !containsString(e.Pattern.ArgShapes, ShapeUnguardedVariablePath) {
			return false
		}
	}
	return true
}

// ShellCLesson is the rule the teaching plane carries for the refusal, in the
// shape every entry's lesson takes: the lead, the id, the command, the why and
// what to run instead. Under a disabled registry it opens with the guard-off
// lead, as every lesson does.
func (r Registry) ShellCLesson() string {
	lead := "Refused by the guard"
	if r.Disabled {
		lead = guardOffLead
	}
	sig := shellCDeleteSignal()
	return lead + " (" + shellCDeleteEntryID + "): a `bash -c` or `sh -c` script built from variables that runs " +
		orList(quoteAll(deleteCommands)) + ". " + sig.reason + " Instead: " + sig.successor
}
