package ahoy

import (
	"os"
	"slices"

	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/releasedigest"
)

// What waits on this repository after an update (spc-2610170635097509, step
// 5): the setup questions this repository has never answered, and the
// settings whose default a release in the update changed and which this
// repository takes from the default. Both are read from disk and the binary
// alone; nothing here writes.

// SetupInterview is the setup interview's name, as its answers records are
// named (interview.Write) and read (interview.Latest).
const SetupInterview = "setup"

// The kinds of thing that wait.
const (
	// WaitingQuestion is a setup question this repository never answered.
	WaitingQuestion = "question"
	// WaitingDefault is a setting whose default changed in the update and
	// which the repository takes from the default.
	WaitingDefault = "default"
)

// WaitingItem is one thing that waits: its setup key (the id its answer is
// recorded under), a title in core's own words, and, for a changed default,
// the record that changed it and the release that shipped the record.
type WaitingItem struct {
	Kind    string `json:"kind"`
	Key     string `json:"key"`
	Title   string `json:"title"`
	Record  string `json:"record,omitempty"`
	Release string `json:"release,omitempty"`
}

// WaitingSet is everything that waits, the questions in the order setup asks
// them, then the changed defaults, newest release first.
type WaitingSet struct {
	Questions []WaitingItem `json:"questions"`
	Defaults  []WaitingItem `json:"defaults"`
}

// The seams the tests stand in for: the digest's range, the default-change
// table, and the home whose records hold the machine-wide answers.
var (
	waitingRange          = releasedigest.Range
	waitingDefaultChanges = defaultChanges
	waitingHome           = os.UserHomeDir
)

// waiting is waitingIn over a fresh detection of cwd.
func waiting(cwd, from, to string) (WaitingSet, error) {
	det, err := Detect(cwd)
	if err != nil {
		return WaitingSet{}, err
	}
	return waitingIn(cwd, det, from, to)
}

// waitingIn returns what waits on the repository at cwd, whose detection is
// det, for an update from the release from to the release to: the questions
// det's gaps arm that this repository has no recorded answer to (a question
// answered with a value, or deferred with "decide later", is recorded), and
// the settings whose default a release after from up to to changed, which
// the repository's .abcd/config.json leaves unset and no waiting question
// already covers.
//
// A category approval and the adoption are not waiting questions: they are
// consent to the changes the questions and gaps make, asked whenever there is
// a change to make, not a setting the repository answers once.
func waitingIn(cwd string, det DetectionResult, from, to string) (WaitingSet, error) {
	set := WaitingSet{Questions: []WaitingItem{}, Defaults: []WaitingItem{}}
	repoAnswers, err := interview.Latest(interview.Place{Repo: cwd}, SetupInterview)
	if err != nil {
		return WaitingSet{}, err
	}
	machineAnswers := map[string]string{}
	if home, herr := waitingHome(); herr == nil && home != "" {
		if machineAnswers, err = interview.Latest(interview.Place{Home: home}, SetupInterview); err != nil {
			return WaitingSet{}, err
		}
	}
	for _, q := range armedQuestions(cwd, det) {
		answers := repoAnswers
		if SetupMachineWide(q.Key) {
			answers = machineAnswers
		}
		if _, recorded := answers[q.Key]; recorded {
			continue
		}
		set.Questions = append(set.Questions, q)
	}

	changes, err := waitingDefaultChanges()
	if err != nil || len(changes) == 0 {
		return set, err
	}
	sum, err := waitingRange(from, to)
	if err != nil {
		return WaitingSet{}, err
	}
	cfg, err := readConfig(cwd)
	if err != nil {
		return WaitingSet{}, err
	}
	covered := map[string]bool{}
	for _, q := range set.Questions {
		covered[q.Key] = true
	}
	for _, rel := range sum.Releases {
		cited := map[string]bool{}
		for _, e := range rel.Entries {
			for _, r := range e.Records {
				cited[r.ID] = true
			}
		}
		for _, c := range changes {
			if covered[c.Key] || !cited[c.Record] {
				continue
			}
			slot := configValueSlots[c.Key]
			if _, isSet := subMap(cfg, slot[0])[slot[1]]; isSet {
				continue
			}
			covered[c.Key] = true
			set.Defaults = append(set.Defaults, WaitingItem{
				Kind: WaitingDefault, Key: c.Key, Record: c.Record, Release: rel.Tag,
				Title: "the default of " + slot[0] + "." + slot[1] + " changed in " + rel.Tag + " (" + c.Record + ")",
			})
		}
	}
	return set, nil
}

// armedQuestions is every setup question det's gaps arm in the repository at
// cwd, in the order setup asks them, each titled in core's words: the config
// values the install collects, the artefact kind, the house-style question,
// the git identity, the lineage link, the optional offers, and the
// retirement of each tool conventions file that only repeats AGENTS.md.
func armedQuestions(cwd string, det DetectionResult) []WaitingItem {
	gaps := map[string]Gap{}
	for _, g := range det.Gaps {
		if _, seen := gaps[g.ID]; !seen {
			gaps[g.ID] = g
		}
	}
	var out []WaitingItem
	add := func(key, title string) {
		out = append(out, WaitingItem{Kind: WaitingQuestion, Key: key, Title: title})
	}
	byGap := func(gapID, key string) {
		if g, ok := gaps[gapID]; ok {
			add(key, g.Title)
		}
	}
	byGap("config.visibility_missing", "visibility")
	byGap("config.docs_target_missing", "docs_target")
	if oracleBackendAsked() {
		byGap("config.oracle_backend_missing", "oracle_backend")
	}
	byGap("config.scan_deep_missing", "scan_deep")
	if !hasPluginManifest(cwd) {
		byGap(ArtefactMissingGapID, artefactKindKey)
	}
	if _, ok := gaps["banlist.public_family_missing"]; ok && publicPathIsWritable(cwd) {
		add(emDashPromptKey, "the house-style question: an em dash in a list item")
	}
	if slices.ContainsFunc(establishGapIDs, func(id string) bool { _, ok := gaps[id]; return ok }) {
		add("git_identity.establish", "who commits to this repository")
	}
	byGap("identity.refounding_candidate", "history.lineage")
	byGap(ReadmeOfferGapID, ReadmePromptKey)
	for _, id := range []string{StatusLineOfferGapID, OracleRoutingMachineGapID, OracleRoutingRepoGapID, DrainRuleOfferGapID} {
		byGap(id, id)
	}
	if _, ok := gaps[ConventionsRetireGapID]; ok {
		s := newConventionsScan(cwd)
		defer s.close()
		for _, f := range toolConventionsFiles {
			at := s.classify(f.Rel)
			at.release()
			if at.class == toolFileRepeats {
				add(retirePromptKey(at.shown, at.how), f.Rel+" only repeats AGENTS.md")
			}
		}
	}
	return out
}
