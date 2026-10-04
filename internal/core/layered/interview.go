package layered

import "fmt"

// The interview.list setting (spc-2610030911534855, "The numbered fallback"):
// how a plain-Terminal interview takes a choice from a list. arrows, the
// bundled default, is the arrow-key list with typing to narrow; numbered reads
// whole lines, a number or part of a name, and never touches the terminal's
// modes, which is the mode a screen reader is served by.
const (
	InterviewListKey      = "interview.list"
	InterviewListArrows   = "arrows"
	InterviewListNumbered = "numbered"
)

// InterviewList resolves interview.list from the machine's
// ~/.abcd/config.json, else the bundled arrows. It claims the interview
// namespace, so a misspelt key is refused, and it is the machine's alone: how
// a person reads a list is theirs to say, so a repository's .abcd/config.json
// that sets it is refused naming the machine's file, as a value outside the
// two is, never passed over for the default.
func InterviewList(r Roots) (Value[string], error) {
	s, err := Load(Config, r)
	if err != nil {
		return Value[string]{}, err
	}
	if err := s.Claim("interview", "list"); err != nil {
		return Value[string]{}, err
	}
	found, err := s.Lookup(InterviewListKey)
	if err != nil {
		return Value[string]{}, err
	}
	for _, f := range found {
		if f.Layer == Repo {
			return Value[string]{}, fmt.Errorf("%s (%s layer): %s is set here, but how a list is read is "+
				"the person's own to say; set it in %s and remove it from %s",
				f.Origin, f.Layer, InterviewListKey, Config.MachineOrigin(), f.Origin)
		}
	}
	return Get(s, InterviewListKey, InterviewListArrows, func(v string) error {
		if v != InterviewListArrows && v != InterviewListNumbered {
			return fmt.Errorf("want %q or %q", InterviewListArrows, InterviewListNumbered)
		}
		return nil
	})
}
