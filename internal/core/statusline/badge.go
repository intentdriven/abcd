package statusline

// The badge: one slot, three states, and the rule that its meaning never rests
// on colour.
//
// The three states are one thing that changes meaning, not a presence
// indicator plus a separate idle word (iss-168, refined 2026-08-29). Quiet
// means abcd is here and nobody is waiting; the two role states mean the loop
// is parked on the named person for a verdict.
//
// WORD AND INVERSE VIDEO CARRY THE MEANING; COLOUR ONLY REINFORCES. That is
// itd-200's commitment and it is structural here rather than a comment: the
// word is the Element's PLAIN form, so it is what a consumer that renders no
// escapes at all shows, and the three words are distinct prose that names the
// addressee outright. TestBadgeMeaningSurvivesColourRemoval strips every
// escape from a rendered row and requires the three states still to be told
// apart, which is the property stated as a test rather than as an intention.
//
// The presence pair is a SETTING and the two role pairs are FIXED CONSTANTS,
// and the line between them is what each badge is for. A role badge says whose
// question is waiting, and recognition without reading only works if it looks
// the same in every repository; the presence badge says abcd is here, which is
// ambient, and nothing breaks when it differs between projects.

import "github.com/intentdriven/abcd/internal/core/mode"

// State is what the badge is reporting. It is the stored mode's own type
// (`.abcd/.work.local/mode`, one line, spc-70): mode OWNS the vocabulary — the
// three words, their parse and their store — and this package holds only the
// render's view of it. The alias keeps the render's API and tests in their own
// words while making the two packages' states the same type by construction,
// so a state read out of the store reaches Render without a conversion that
// could drift. Importing a type is not touching a store: the render stays pure,
// and mode never imports this package.
type State = mode.State

const (
	// StateManaged: abcd is here and nobody is waiting.
	StateManaged = mode.Managed
	// StateFacilitator: the loop is parked on the facilitator.
	StateFacilitator = mode.Facilitator
	// StateProductThinker: the loop is parked on the product thinker.
	StateProductThinker = mode.ProductThinker
)

// badgeWord is the text inside the badge for each state.
//
// The two role words say "waiting" outright rather than naming the role alone.
// A bare "facilitator" is a label and could as easily mean "you are the
// facilitator"; the status line's one job in that state is to report that an
// answer is OWED, and the word has to carry that where no colour does. The
// cost is width, and width is the one thing the badge is allowed to spend:
// it is element one, so it is what a narrow host keeps.
var badgeWord = map[State]string{
	StateManaged:        "abcd",
	StateFacilitator:    "waiting: facilitator",
	StateProductThinker: "waiting: product thinker",
}

// rolePairs are the two FIXED badge pairs, drawn from internal/livery's
// palette rather than invented here, so the terminal badge and the repository's
// rendered artwork stay one set of colours.
//
// Both role badges are dark text on a light fill, which is the opposite
// polarity to the presence badge's light text on a dark fill. That polarity is
// load-bearing: iss-168 records that the presence gold and the product
// thinker's amber are the nearest two hues in the set, and the polarity
// difference is what holds them apart when the hue does not.
//
// The hexes are the livery palette's 'k' (dark), 'w' (white) and 'y' (yellow).
// The roles' own palette is settled on the design branch under ids that have
// not merged (itd-200, open question 3); until it lands these are abcd's
// existing colours rather than a second, unrelated set. Both pairs are held to
// ContrastBar by TestRoleBadgePairsClearTheBar.
var rolePairs = map[State]Pair{
	StateFacilitator:    {Foreground: "#1c1f26", Background: "#e8e8e8"},
	StateProductThinker: {Foreground: "#1c1f26", Background: "#f0c052"},
}

// fixedPair is the badge pair a state renders in regardless of the setting.
// The quiet state has none — its pair is the configured one — and returns the
// zero Pair.
func fixedPair(s State) Pair { return rolePairs[s] }

// reset ends the badge's colour run. Everything after the badge is plain, so
// the sequence closes on the badge itself rather than at the end of the row:
// a host that truncates mid-row must not leave the rest of its status bar
// painted in abcd's colours.
const reset = "\x1b[0m"

// renderBadge composes the badge element for one state. The plain form is the
// word alone; the rendered form is the word inside a filled block of colour,
// which is what "inverse video with its word inside it" amounts to once the
// pair is explicit.
//
// The padding spaces belong to the rendered form only. They exist to make the
// fill read as a block rather than as coloured text, and a consumer showing
// the plain form — the `/abcd` board's one-line state — wants the word without
// them.
func renderBadge(s State, presence Pair) Element {
	word, ok := badgeWord[s]
	if !ok {
		// A stored mode this binary does not know renders as the quiet state.
		// The alternative — an empty or invented badge — would put a claim on
		// the row that nothing in the record supports.
		s = StateManaged
		word = badgeWord[StateManaged]
	}
	pair := presence
	if fixed := fixedPair(s); fixed != (Pair{}) {
		pair = fixed
	}
	return Element{Key: KeyPresence, Plain: word, Rendered: paint(pair, " "+word+" ")}
}

// paint wraps text in a foreground/background pair. A pair that does not parse
// renders the text uncoloured rather than failing: Load has already refused
// and substituted every pair that reaches here, so this branch is unreachable
// in practice, and the failure mode it guarantees if it ever is reached is a
// legible badge with no colour — which the accessibility commitment says must
// be enough on its own anyway.
func paint(p Pair, text string) string {
	fg, err := ParseColor(p.Foreground)
	if err != nil {
		return text
	}
	bg, err := ParseColor(p.Background)
	if err != nil {
		return text
	}
	return "\x1b[" + fg.sgr(38) + ";" + bg.sgr(48) + "m" + text + reset
}
