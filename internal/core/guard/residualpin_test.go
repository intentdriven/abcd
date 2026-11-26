package guard

import "testing"

// TestNamedResidualsOfTheFixedOutputRead pins the residuals the brief and the
// guard page name for the fixed-output read (review7-guard finding 4,
// iss-2609260115383631), so a change that closes one fails here and the
// sentence naming it is revisited in the same change. A substitution whose
// output is unknown, as a command's program, is refused: which program runs is
// unchecked. Only the exact shape `cat <<DELIM`, a newline, the body, the
// delimiter line and blanks is read as fixed output; every other spelling of
// the same document stays unknown, and so is refused as the program.
func TestNamedResidualsOfTheFixedOutputRead(t *testing.T) {
	const push = "git push --force origin main"
	runVerdictCases(t, []verdictCase{
		{"$(cat msg.txt)", VerdictBlock, unknownProgramEntryID},
		{"$(date)", VerdictBlock, unknownProgramEntryID},
		{"$(/bin/cat <<'F'\n" + push + "\nF\n)", VerdictBlock, unknownProgramEntryID},
		{"$(command cat <<'F'\n" + push + "\nF\n)", VerdictBlock, unknownProgramEntryID},
		{"$(cat - <<'F'\n" + push + "\nF\n)", VerdictBlock, unknownProgramEntryID},
		{"$(cat <<'F' 2>/dev/null\n" + push + "\nF\n)", VerdictBlock, unknownProgramEntryID},
		{"$(cat \\\n<<'F'\n" + push + "\nF\n)", VerdictBlock, unknownProgramEntryID},
		{"$(cat <<'F'\n" + push + "\nF\ntrue\n)", VerdictBlock, unknownProgramEntryID},
		{"`cat <<'F'\n" + push + "\nF\n\\\n`", VerdictBlock, unknownProgramEntryID},
	})
}
