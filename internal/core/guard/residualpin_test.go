package guard

import "testing"

// TestNamedResidualsOfTheFixedOutputRead pins the residuals the brief and the
// guard page name for the fixed-output read (review7-guard finding 4,
// iss-2609260115383631), so a change that closes one fails here and the
// sentence naming it is revisited in the same change. A lone substitution
// whose output is unknown, in command position, allows by the recorded
// posture: an allow means no entry matched, and such a name can be any
// program. Only the exact shape `cat <<DELIM`, a newline, the body, the
// delimiter line and blanks is read as fixed output; every other spelling of
// the same document stays unknown, and so allows alone in command position,
// though bash runs what it prints.
func TestNamedResidualsOfTheFixedOutputRead(t *testing.T) {
	const push = "git push --force origin main"
	runVerdictCases(t, []verdictCase{
		{"$(cat msg.txt)", VerdictAllow, ""},
		{"$(date)", VerdictAllow, ""},
		{"$(/bin/cat <<'F'\n" + push + "\nF\n)", VerdictAllow, ""},
		{"$(command cat <<'F'\n" + push + "\nF\n)", VerdictAllow, ""},
		{"$(cat - <<'F'\n" + push + "\nF\n)", VerdictAllow, ""},
		{"$(cat <<'F' 2>/dev/null\n" + push + "\nF\n)", VerdictAllow, ""},
		{"$(cat \\\n<<'F'\n" + push + "\nF\n)", VerdictAllow, ""},
		{"$(cat <<'F'\n" + push + "\nF\ntrue\n)", VerdictAllow, ""},
		{"`cat <<'F'\n" + push + "\nF\n\\\n`", VerdictAllow, ""},
	})
}
