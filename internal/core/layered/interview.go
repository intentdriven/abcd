package layered

import "fmt"

// The interview namespace holds two keys, each one party's to set.
//
// interview.list (spc-2610030911534855, "The numbered fallback"): how a
// plain-Terminal interview takes a choice from a list. arrows, the bundled
// default, is the arrow-key list with typing to narrow; numbered reads whole
// lines, a number or part of a name, and never touches the terminal's modes,
// which is the mode a screen reader is served by. It is the machine's alone.
//
// interview.redrafts (iss-2611011402496418, dcn-2611012123240576): how many
// times in a row an AI-written interview hands a question or a done outcome
// abcd's check refused back to its interviewer to redraft, before it shows the
// question as it stands, repaired where it can be, or ends on the outcome. It
// is the repository's alone.
const (
	InterviewListKey      = "interview.list"
	InterviewListArrows   = "arrows"
	InterviewListNumbered = "numbered"

	InterviewRedraftsKey = "interview.redrafts"
	// InterviewRedraftsDefault is the bound when no file sets it.
	InterviewRedraftsDefault = 3
	// InterviewRedraftsMax is the highest bound a repository may set; 0, the
	// lowest, shows a refused question at once.
	InterviewRedraftsMax = 10
)

// interviewKeys is every key the interview namespace takes. Both readers
// claim all of it, so a file setting one key is never refused by the other
// reader's claim.
var interviewKeys = []string{"list", "redrafts"}

// The refusals of a key set in the other party's file.
const (
	listOwner     = "how a list is read is the person's own to say"
	redraftsOwner = "how many times a refused question is redrafted is the repository's to say"
)

// loadInterview loads the configuration and claims the interview namespace.
func loadInterview(r Roots) (*Stack, error) {
	s, err := Load(Config, r)
	if err != nil {
		return nil, err
	}
	if err := s.Claim("interview", interviewKeys...); err != nil {
		return nil, err
	}
	return s, nil
}

// refuseLayer refuses key held in the layer it does not belong to: owner says
// whose it is, wrong is the layer that may not set it, and right is the file
// where it belongs.
func refuseLayer(s *Stack, key string, wrong Layer, owner, right string) error {
	found, err := s.Lookup(key)
	if err != nil {
		return err
	}
	for _, f := range found {
		if f.Layer == wrong {
			return fmt.Errorf("%s (%s layer): %s is set here, but %s; set it in %s and remove it from %s",
				f.Origin, f.Layer, key, owner, right, f.Origin)
		}
	}
	return nil
}

// InterviewList resolves interview.list from the machine's
// ~/.abcd.noindex/config.json, else the bundled arrows. It claims the interview
// namespace, so a misspelt key is refused, and it is the machine's alone: how
// a person reads a list is theirs to say, so a repository's .abcd/config.json
// that sets it is refused naming the machine's file, as a value outside the
// two is, never passed over for the default. The machine's file setting
// interview.redrafts, the repository's key, is refused the same way, so a
// setting nothing would read never passes unannounced.
func InterviewList(r Roots) (Value[string], error) {
	s, err := loadInterview(r)
	if err != nil {
		return Value[string]{}, err
	}
	if err := refuseLayer(s, InterviewListKey, Repo, listOwner, Config.MachineOrigin()); err != nil {
		return Value[string]{}, err
	}
	if err := refuseLayer(s, InterviewRedraftsKey, Machine, redraftsOwner, Config.RepoOrigin()); err != nil {
		return Value[string]{}, err
	}
	return Get(s, InterviewListKey, InterviewListArrows, func(v string) error {
		if v != InterviewListArrows && v != InterviewListNumbered {
			return fmt.Errorf("want %q or %q", InterviewListArrows, InterviewListNumbered)
		}
		return nil
	})
}

// InterviewRedrafts resolves interview.redrafts from the repository's
// .abcd/config.json, else the bundled InterviewRedraftsDefault. It claims the
// interview namespace as InterviewList does, and it is the repository's
// alone: how hard its interviews hold an interviewer to the asking rules
// before showing what it wrote is the repository's to say, so the machine's
// file setting it is refused naming the repository's. A value that is not a
// whole number from 0 to InterviewRedraftsMax is refused naming the key, the
// value and the file, never passed over for the default.
func InterviewRedrafts(r Roots) (Value[int], error) {
	s, err := loadInterview(r)
	if err != nil {
		return Value[int]{}, err
	}
	if err := refuseLayer(s, InterviewRedraftsKey, Machine, redraftsOwner, Config.RepoOrigin()); err != nil {
		return Value[int]{}, err
	}
	return Get(s, InterviewRedraftsKey, InterviewRedraftsDefault, func(n int) error {
		if n < 0 || n > InterviewRedraftsMax {
			return fmt.Errorf("outside 0 to %d", InterviewRedraftsMax)
		}
		return nil
	})
}
