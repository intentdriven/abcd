package reflect

// Section names one of the retrospective's five sections, in the order the
// retrospective carries them (spec scope 2). The first four are asked; metrics
// is computed from the seed and never asked.
type Section string

const (
	WentWell     Section = "went_well"
	CouldImprove Section = "could_improve"
	Lessons      Section = "lessons"
	Decisions    Section = "decisions"
	Metrics      Section = "metrics"
)

// AskedSections are the sections the interview asks, in order.
var AskedSections = []Section{WentWell, CouldImprove, Lessons, Decisions}

// AllSections are the five sections the retrospective carries, in order.
var AllSections = []Section{WentWell, CouldImprove, Lessons, Decisions, Metrics}

// Heading is the section's heading in the written retrospective.
func (s Section) Heading() string {
	switch s {
	case WentWell:
		return "What went well"
	case CouldImprove:
		return "What could improve"
	case Lessons:
		return "Lessons learned"
	case Decisions:
		return "Decisions made"
	case Metrics:
		return "Metrics"
	}
	return string(s)
}

// Question is the opening question the interview asks for an asked section.
func (s Section) Question() string {
	switch s {
	case WentWell:
		return "What went well in this release? Name the successes and strengths, with a specific example of each."
	case CouldImprove:
		return "What could improve? Name the issues and gaps, most important first."
	case Lessons:
		return "What did this release teach that a future voyage should carry? One lesson per line, framed for future-you."
	case Decisions:
		return "Which architectural or design decisions crystallised during this release?"
	}
	return ""
}

// FollowUp is the one clarifying question a thin answer is met with (spec
// scope 3).
func (s Section) FollowUp() string {
	switch s {
	case WentWell:
		return "That reads thin. Which specific piece of work went well, and what made it go well?"
	case CouldImprove:
		return "That reads thin. Which specific issue or gap would you change first, and what did it cost?"
	case Lessons:
		return "That reads thin. What would you tell yourself at the start of the next voyage, and why?"
	case Decisions:
		return "That reads thin. Which decision was taken, what were the alternatives, and why this one?"
	}
	return ""
}
