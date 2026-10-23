package autonomy

import "strings"

// The question reroute (spc-2610180914469489, "The question reroute";
// dcn-2610180937177502, dcn-2610181003477194, dcn-2610181016510153).
//
// While the away record holds, no question waits for the person: the guard
// refuses every call to the host's question tool, whatever its chip, and its
// reason routes the question by the act it asks about. An act only the person
// may approve is queued for their return with `abcd mode queue`; every other
// question is decided and recorded with `abcd decide`. Nothing detects the act
// mechanically: the agent classifies its own question from the list, and the
// guard's registry entries enforce only what they name. The text lives here,
// beside the record, so every front door that repeats the list (the guard's
// refusal, the away block, the stop check's refusal of a question asked in
// prose) says the same thing.

// personOnlyActs are the acts only the person may approve, in the spec's
// order, each one line, the catch-all last.
var personOnlyActs = []string{
	"an embargo close's `--approve <digest>`, passed only after the person has read the text and said yes in their own session: never supply it on your own",
	"the embargo triage verdict, which is the person's decision",
	"the drain's hand-back, which is a person's decision",
	"the confirmed apply verb for remote configuration",
	"a release sign-off: a cut, or a step of one, that asks for the person by name, or that no standing instruction of theirs allows (a standing permission for automatic cuts stands, and the away mode adds none)",
	"pushing, opening or merging a pull request without a standing instruction of the person's that allows it: the away mode is never that instruction",
	"publishing anything, changing a repository's visibility, or posting a comment or message outward, other than a pull request a standing instruction allows",
	"adding a dependency",
	"a secret, a credential or an App",
	"a step only a website or a browser can do",
	"a destructive act: deleting what the session did not create, overwriting uncommitted work, a force push, discarding a peer's work",
	"anything that spends money",
	"any other act a command page or a brief invariant names as the person's",
}

// QuestionRoute is how a question is routed while the person is away, one
// line each: the queue route and the person-only acts, then the decide route
// for every other question, then that nothing detects the act and the away
// mode grants nothing. A front door puts its own head line above it.
func QuestionRoute() string {
	var b strings.Builder
	b.WriteString("Route the question by the act it asks about, not by its chip.\n")
	b.WriteString("Queue it with `abcd mode queue \"<step>\"` for the person's return, and carry on with other work, when it asks about an act only the person may approve:\n")
	for _, a := range personOnlyActs {
		b.WriteString("- " + a + ";\n")
	}
	b.WriteString("Decide every other question yourself, product and technical alike, and record it with `abcd decide \"<text>\"`, with its grounds and how to reverse it; then carry on.\n")
	b.WriteString("Nothing detects the act for you: classify your own question from this list. The away mode grants no rights, so it never allows an act on the list.")
	return b.String()
}
