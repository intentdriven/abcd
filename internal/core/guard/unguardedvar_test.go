package guard

import (
	"strings"
	"testing"
)

// TestRmOnAnUnguardedVariablePathBlocks — iss-2610091942156774. An rm whose
// operand begins with a variable followed by `/` (`"$VAR"/*`) deletes from the
// filesystem root when the variable is empty or unset, and the host raises its
// own dangerous-rm prompt for exactly that shape, which the person must
// approve blind. The guard refuses it first, with the rewrite as its
// successor, so the agent changes the command before the host prompts:
// `"${VAR:?}"/...` stops the shell on an empty value, and a literal path
// cannot be empty. A variable later in the path cannot climb to the root, and
// a bare `"$VAR"` operand empties to no path at all, so neither is it.
func TestRmOnAnUnguardedVariablePathBlocks(t *testing.T) {
	const id = "rm-unguarded-variable-path"
	runVerdictCases(t, []verdictCase{
		// The four spellings the ruling names, with and without -r.
		{`rm -f "$VAR"/*`, VerdictBlock, id},
		{`rm -rf $VAR/build`, VerdictBlock, id},
		{`rm -rf "${VAR}"/build`, VerdictBlock, id},
		{`rm -rf ${VAR}/build`, VerdictBlock, id},
		{`rm "$VAR"/notes.txt`, VerdictBlock, id},
		// The slash inside the quotes is the same word to bash.
		{`rm -rf "$VAR/build"`, VerdictBlock, id},
		{`rm -rf "$TMPDIR/abcd-x"`, VerdictBlock, id},
		// A trailing slash alone empties to the root itself.
		{`rm -rf "$VAR"/`, VerdictBlock, id},
		// After the operand terminator, behind a launcher, in a chain, among
		// other operands.
		{`rm -rf -- "$VAR"/*`, VerdictBlock, id},
		{`sudo rm -rf "$VAR"/*`, VerdictBlock, id},
		{`/bin/rm -f "$VAR"/*`, VerdictBlock, id},
		{`make clean && rm -rf "$OUT"/*`, VerdictBlock, id},
		{`rm -rf ./build "$OUT"/cache`, VerdictBlock, id},
		// Inside a string handed to a shell.
		{`sh -c 'rm -rf "$VAR"/*'`, VerdictBlock, id},
		{`bash -c "rm -rf \"\$VAR\"/build"`, VerdictBlock, id},
		{`sh -c "rm -rf $VAR/build"`, VerdictBlock, id},
		// A positional parameter, two variables that can both be empty, an
		// expansion that can print nothing, and a default whose word is itself
		// a variable that can be empty.
		{`rm -rf "$1"/build`, VerdictBlock, id},
		{`rm -rf "$A$B"/x`, VerdictBlock, id},
		{`rm -rf ${VAR:-}/build`, VerdictBlock, id},
		{`rm -rf ${VAR:+x}/build`, VerdictBlock, id},
		{`rm -rf "${XDG:-$CACHE}"/x`, VerdictBlock, id},

		// The rewrites the successor names.
		{`rm -rf "${VAR:?}"/build`, VerdictAllow, ""},
		{`rm -f "${VAR:?}"/*`, VerdictAllow, ""},
		{`rm -rf "${VAR:?VAR is unset}"/build`, VerdictAllow, ""},
		{`rm -rf ${VAR:?}/build`, VerdictAllow, ""},
		{`/bin/rm -f -- "${VAR:?}"/*`, VerdictAllow, ""},
		{`rm -rf /tmp/build`, VerdictAllow, ""},
		// A default that cannot be empty guards the value as `:?` does.
		{`rm -rf "${TMPDIR:-/tmp}/abcd-x"`, VerdictAllow, ""},
		{`rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/abcd"`, VerdictAllow, ""},
		// A trim, a replacement, a substring or a case change is an everyday
		// expansion the guard leaves allowed.
		{`rm -rf "${VAR%/}"/x`, VerdictAllow, ""},
		{`rm -rf "${p%/*}/build"`, VerdictAllow, ""},
		// A variable later in the path cannot climb to the root.
		{`rm -rf ./build/$name`, VerdictAllow, ""},
		{`rm -rf build/"$name"`, VerdictAllow, ""},
		{`rm -rf "./out/$VAR"/cache`, VerdictAllow, ""},
		// A bare variable operand empties to no path at all.
		{`rm -rf "$tmp"`, VerdictAllow, ""},
		{`rm -f $file`, VerdictAllow, ""},
		{`rm -rf "${VAR}"`, VerdictAllow, ""},
		// The home and the working directory are set by the login and the
		// shell; their own entries own a delete of them.
		{`rm -rf "$HOME/.cache/abcd-test"`, VerdictAllow, ""},
		{`rm -rf "$PWD/build"`, VerdictAllow, ""},
		// The words as data: a here-document body, a commit message, a quoted
		// argument, and other commands.
		{"cat > NOTES.md <<'EOF'\nrm -rf \"$VAR\"/*\nEOF", VerdictAllow, ""},
		{"tee NOTES.md <<'EOF'\nrm -f $VAR/build\nEOF", VerdictAllow, ""},
		{`git commit -m 'never run rm -rf "$VAR"/* again'`, VerdictAllow, ""},
		{`printf '%s\n' 'rm -rf "$VAR"/*'`, VerdictAllow, ""},
		{`echo "$VAR"/build`, VerdictAllow, ""},
		{`ls "$VAR"/`, VerdictAllow, ""},
	})
}

// TestRmOnAnUnguardedVariablePathTeachesTheRewrite pins what the refusal says:
// the successor names the `"${VAR:?}"/...` rewrite and the literal path, so the
// block message is the lesson the host's prompt would otherwise give the
// person instead of the agent.
func TestRmOnAnUnguardedVariablePathTeachesTheRewrite(t *testing.T) {
	d := verdictOf(t, `rm -f "$VAR"/*`)
	if d.Verdict != VerdictBlock || d.EntryID != "rm-unguarded-variable-path" {
		t.Fatalf("verdict = %q via %q, want a block via rm-unguarded-variable-path", d.Verdict, d.EntryID)
	}
	for _, want := range []string{`"${VAR:?}"/`, "literal path"} {
		if !strings.Contains(d.Successor, want) {
			t.Errorf("successor %q does not name %q", d.Successor, want)
		}
	}
}

// TestRmRewriteInsideAShellStringIsNotThisEntry: the successor's rewrite
// handed to a shell does not fire the entry. The string as a whole is still
// a warn of the execute-a-string family, which reads `${VAR:?}` there as
// syntax it does not inspect; that verdict is not this entry's and is not
// pinned here.
func TestRmRewriteInsideAShellStringIsNotThisEntry(t *testing.T) {
	for _, cmd := range []string{
		`sh -c 'rm -rf "${VAR:?}"/build'`,
		`bash -c 'rm -f "${VAR:?}"/*'`,
	} {
		d := verdictOf(t, cmd)
		if d.Verdict == VerdictBlock || contains(d.Matches, "rm-unguarded-variable-path") {
			t.Errorf("Check(%q) = %q via %v, want no rm-unguarded-variable-path match", cmd, d.Verdict, d.Matches)
		}
	}
}

// TestArgShapesRejectsAnUnknownShape: a shape name the matcher does not know
// would describe an operand nothing can be, the silent defang the other
// operand checks refuse.
func TestArgShapesRejectsAnUnknownShape(t *testing.T) {
	r := Defaults()
	e := r.Entries["rm-unguarded-variable-path"]
	e.Pattern.ArgShapes = []string{"unguarded-variabel-path"}
	r.Entries["rm-unguarded-variable-path"] = e
	if err := Validate(r); err == nil {
		t.Fatal("Validate accepted an unknown arg_shapes name")
	}
}

// TestRmGuardedRewriteInsideADoubleQuotedShellString — iss-2610100938485695
// shape 1. Inside a double-quoted string the enclosing shell expands the
// variable, and `${X:?}` stops it there on an empty value, so the string's
// re-read of `${X:?}/y` names no path from the root; neither does a default
// that cannot be empty, nor a guard nested in a default. Written out for the
// re-read, `${X:?}` is the `${X}` it prints, which alone can be empty: the
// refusal told the agent to write what it had written. A default that can
// be empty, a plain reference, an escaped `\${X}` the string re-reads
// itself, and a name the string also writes unguarded still refuse.
func TestRmGuardedRewriteInsideADoubleQuotedShellString(t *testing.T) {
	const id = "rm-unguarded-variable-path"
	for _, cmd := range []string{
		`sh -c "rm -rf ${X:?}/y"`,
		`bash -c "rm -f \"${VAR:?}\"/*"`,
		`sh -c "rm -rf ${X:-/tmp}/y"`,
		`sh -c "rm -rf ${X:=/tmp}/y"`,
		`sh -c "rm -rf ${X:-${Y:?}}/y"`,
		`sh -c "rm -rf ${X:-${Y:-/tmp}}/y"`,
	} {
		d := verdictOf(t, cmd)
		if d.Verdict == VerdictBlock || contains(d.Matches, id) {
			t.Errorf("Check(%q) = %q via %v, want no %s match", cmd, d.Verdict, d.Matches, id)
		}
	}
	runVerdictCases(t, []verdictCase{
		{`sh -c "rm -rf ${X}/y"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:-}/y"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:-$Y}/y"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:-${Y}}/y"`, VerdictBlock, id},
		{`sh -c "rm -rf \${X}/y ${X:?}/a"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:?}/a ${X}/b"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:?}/a $X/b"`, VerdictBlock, id},
		// A default the string's shell empties is no guard there: its quotes
		// come off (`${X:-""}`) and its unquoted blanks split (`${X:- }`).
		{`sh -c "rm -rf ${X:-\"\"}/y"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:- }/y"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:-a }/y"`, VerdictBlock, id},
		{`sh -c "rm -rf ${X:-\$Y}/y"`, VerdictBlock, id},
		// The home's own entry still reads the guarded spelling.
		{`sh -c "rm -rf ${HOME:?}"`, VerdictBlock, "rm-rf-root-or-home"},
		{`sh -c "rm -rf '${HOME:?}'"`, VerdictBlock, "rm-rf-root-or-home"},
	})
}

// TestRmNestedGuardIsRead — iss-2610100938485695 shape 2. A default whose
// word is itself guarded (`${VAR:-${OTHER:?}}`, `${VAR:-${OTHER:-/tmp}}`)
// never prints nothing, so the path is not one from the root. A nested
// reference that can be empty still refuses, and so does a name the
// expansion also writes unguarded: `${VAR:-${VAR}}` prints VAR's empty value.
func TestRmNestedGuardIsRead(t *testing.T) {
	const id = "rm-unguarded-variable-path"
	runVerdictCases(t, []verdictCase{
		{`rm -rf "${VAR:-${OTHER:?}}"/y`, VerdictAllow, ""},
		{`rm -rf "${VAR:-${OTHER:-/tmp}}"/y`, VerdictAllow, ""},
		{`rm -rf "${VAR:=${OTHER:?}}"/y`, VerdictAllow, ""},
		{`rm -rf "${VAR:-${OTHER:?}/tmp}"/y`, VerdictAllow, ""},
		{`rm -rf "${VAR:-${OTHER}}"/y`, VerdictBlock, id},
		{`rm -rf "${VAR:-${OTHER:-}}"/y`, VerdictBlock, id},
		{`rm -rf "${VAR:-${OTHER-/tmp}}"/y`, VerdictBlock, id},
		{`rm -rf "${VAR:-${VAR}}"/y`, VerdictBlock, id},
		{`rm -rf "${VAR:-$VAR}"/y`, VerdictBlock, id},
	})
}

// TestRmVariableBeforeASubstitutionIsFollowed — iss-2610100938485695 shape
// 3. A command substitution can print nothing as the variable can, so
// `$VAR$(true)/x` is `/x` with VAR empty: the site is followed past the
// substitution to the `/`. An arithmetic expansion always prints a number,
// so `$VAR$((1))/x` is never a path from the root.
func TestRmVariableBeforeASubstitutionIsFollowed(t *testing.T) {
	const id = "rm-unguarded-variable-path"
	runVerdictCases(t, []verdictCase{
		{`rm -rf $VAR$(true)/x`, VerdictBlock, id},
		{"rm -rf $VAR`true`/x", VerdictBlock, id},
		{`rm -rf "$VAR$(true)"/x`, VerdictBlock, id},
		{`rm -rf "$VAR$(true)$(true)/x"`, VerdictBlock, id},
		{`rm -rf "$VAR$(true)$B"/x`, VerdictBlock, id},
		{`sh -c 'rm -rf $VAR$(true)/x'`, VerdictBlock, id},
		{`rm -rf $VAR$((1))/x`, VerdictAllow, ""},
		{`rm -rf "${VAR:?}$(true)"/x`, VerdictAllow, ""},
		{`rm -rf "$VAR$(date +%s)"`, VerdictAllow, ""},
	})
}

// TestRmDefaultThatSplitsAwayIsNoGuard — a default written unquoted is split
// on its blanks, so `${X:- }/y` with X unset is the field `/y` in bash, sh and
// dash, and `${X:-a }/y` is the fields `a` and `/y`: the blank leaves the `/`
// opening a field of its own. Quoted, the blank is text (`"${X:- }"/y` is
// ` /y`), and so is an escaped one (`${X:-\ }/y`) or an ANSI-C string's
// (`${X:-$'\x20'}/y`).
func TestRmDefaultThatSplitsAwayIsNoGuard(t *testing.T) {
	const id = "rm-unguarded-variable-path"
	runVerdictCases(t, []verdictCase{
		{`rm -rf ${X:- }/y`, VerdictBlock, id},
		{`rm -rf ${X:-a }/y`, VerdictBlock, id},
		{`rm -rf "${X:- }"/y`, VerdictAllow, ""},
		{`rm -rf ${X:- a}/y`, VerdictAllow, ""},
		{`rm -rf ${X:-\ }/y`, VerdictAllow, ""},
		{`rm -rf ${X:-$'\x20'}/y`, VerdictAllow, ""},
		{`rm -rf ${X:-/tmp}/y`, VerdictAllow, ""},
	})
}
