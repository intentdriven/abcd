package question

// Fields is the field view of one call to the host's question tool: one
// question, or up to four parts of one thing shown as tabs. It is what every
// front door finally shows, so the limits are checked on it. The guard hook
// decodes the host's tool input into it (the JSON key names are the host's, so
// the decoding lives in the surface); the companion's structured question type
// is mapped onto it by that type's own method.
type Fields struct {
	Tabs []Tab
}

// Tab is one question of a call: the four fields the host's question tool
// takes. A call with one question has one tab.
type Tab struct {
	Header  string   // the chip: whom the question is for and which it is
	Text    string   // the material, the Now: and Change later: lines, then the question
	Options []Choice // the answers, the decide-later option last
	// Typed is the prompt of the question's typed part, taken in the host's
	// free-text row: it is no option the host lists, and the limits count it
	// as one toward their floor. The host's own input carries none.
	Typed string
}

// Choice is one option of a question, as the host shows it.
type Choice struct {
	Label       string // a few words
	Description string // what choosing it means
	Preview     string // the host's side preview, which abcd's questions never carry (rule 14)
}

// Person is whom a question is addressed to, as the mode records it.
type Person int

const (
	// Unnamed is no mode naming anyone: a repository abcd does not manage has
	// no mode store, and a managed mode parks on nobody. The chip's role word
	// then stands in, since the chip names whom the question is for.
	Unnamed Person = iota
	// ProductThinker is the mode naming the product thinker.
	ProductThinker
	// Facilitator is the mode naming the technical facilitator.
	Facilitator
)

// Addressee is the check's input about whom a question is for. Verbs is the
// binary's verb list, which the surface reads from its command tree and passes
// in, so the core holds no copy of it (rule 9).
type Addressee struct {
	Person Person
	Verbs  []string
}
