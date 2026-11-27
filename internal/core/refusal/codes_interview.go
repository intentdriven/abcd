package refusal

// codes_interview.go — the codes of the written interviews, which `abcd intent
// interview` and `abcd reflect interview` both run (spc-2610301431108175
// step 3). The prefix is the sub-verb both commands share.

const (
	InterviewUnreachableRole Code = "interview/unreachable-role"
	InterviewInvalidAnswers  Code = "interview/invalid-answers"
	InterviewMissingTerminal Code = "interview/missing-terminal"
	InterviewInvalidConfig   Code = "interview/invalid-config"
)

func init() {
	register(
		Entry{
			Code:    InterviewUnreachableRole,
			Summary: "no route reaches the interviewer role, so nothing was run",
			Explanation: "The interview runs its role on a runner, and no layer of the configuration\n" +
				"routes the role to one: the refusal names the key to set and the file. Run\n" +
				"`abcd ahoy install` or set the role's runner, then run the interview again.",
			Next:     "abcd ahoy install",
			RaisedBy: []string{"abcd intent interview", "abcd reflect interview"},
		},
		Entry{
			Code:    InterviewInvalidAnswers,
			Summary: "the answers given cannot answer the interview's questions",
			Explanation: "Off a terminal the questions are answered from --answers by ordinal, and the file\n" +
				"cannot be read, does not parse, runs out before a question, or gives a value the\n" +
				"question does not offer: the refusal names the question. Nothing was recorded.",
			Next:     explainCommand,
			RaisedBy: []string{"abcd intent interview", "abcd reflect interview"},
		},
		Entry{
			Code:    InterviewMissingTerminal,
			Summary: "the questions are drawn only at a terminal, and none is attached",
			Explanation: "With no --answers file the interview draws each question at a terminal, and\n" +
				"stdin, stdout or stderr is not one. Run it at a terminal, or answer from a file\n" +
				"with --answers. Nothing was run.",
			Next:     explainCommand,
			RaisedBy: []string{"abcd intent interview", "abcd reflect interview"},
		},
		Entry{
			Code:    InterviewInvalidConfig,
			Summary: "the runner or question configuration the interview reads is refused",
			Explanation: "A layer of the configuration sets a runner, a question limit or a local tier\n" +
				"the interview cannot use as it stands: the refusal names the key and the file.\n" +
				"Fix that layer, then run the interview again. Nothing was run.",
			Next:     NoNext,
			RaisedBy: []string{"abcd intent interview", "abcd reflect interview"},
		},
	)
}
