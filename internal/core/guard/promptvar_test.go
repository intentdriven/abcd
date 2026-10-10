package guard

import "testing"

// bash expands PS4 before every traced command and PS0, PS1, PS2 at an
// interactive prompt, command substitutions included, and runs
// PROMPT_COMMAND as a command line before each prompt, so a line that assigns
// one and turns tracing on (SHELLOPTS=xtrace, BASHOPTS, `bash -x`) or starts
// an interactive shell runs the text in it although the -c string is harmless
// (iss-2610090925390900). The guard now judges such a value wherever the line
// assigns it: as a prefix, an env operand, or a declaration builtin's
// argument. A prompt is decoded first the way bash decodes it, so an octal
// escape that spells `$` is read as `$`.
func TestPromptVariableTextIsJudged(t *testing.T) {
	sub := "$(" + hazardLine + ")"
	runVerdictCases(t, []verdictCase{
		{"SHELLOPTS=xtrace PS4='" + sub + "' bash -c true", VerdictBlock, "git-push-force"},
		{"env SHELLOPTS=xtrace PS4='" + sub + "' bash -c true", VerdictBlock, "git-push-force"},
		{"PS4='" + sub + "' bash -xc true", VerdictBlock, "git-push-force"},
		{"BASHOPTS=xtrace PS4='`" + hazardLine + "`' bash -c true", VerdictBlock, "git-push-force"},
		{"PS1='" + sub + "' bash -i -c true", VerdictBlock, "git-push-force"},
		{"PS0='" + sub + "' bash -i", VerdictBlock, "git-push-force"},
		{"PROMPT_COMMAND='" + hazardLine + "' bash -i", VerdictBlock, "git-push-force"},
		{"export PS4='" + sub + "'; bash -xc true", VerdictBlock, "git-push-force"},
		{"PS4='\\044(" + hazardLine + ")' bash -xc true", VerdictBlock, "git-push-force"},
		{"PS4='+ ' bash -xc true", VerdictAllow, ""},
		// A substitution in a prompt keeps the verdict the same substitution
		// gets in any string the shell runs (`bash -c 'echo $(date)'`).
		{"PS4='$(date) ' bash -xc true", VerdictWarn, "execute-string-uninspectable"},
		{"PS1='\\u@\\h \\$ ' bash -i -c true", VerdictAllow, ""},
		{"PROMPT_COMMAND='history -a' bash -i", VerdictAllow, ""},
	})
}
