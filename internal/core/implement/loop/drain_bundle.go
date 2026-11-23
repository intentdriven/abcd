package loop

// drain_bundle.go is the drain's cap on a bundle and the rule for who joins
// one (itd-2611201450052326, spc-2611201455051897 step 3): `drain.bundle_max`,
// resolved once when a drain starts through the one layered configuration
// reader and kept on the drain's state with its layer and origin, as the pace
// is; and bundleCandidates, the issues after a lead in the drain order that
// are of its kind and in its place. Nothing here forms a bundle: a lane with
// members is opened by the drain's forming moves (step 4), so a drain opens
// one issue's lane whatever the cap until then.

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// BundledBundleMax is the most issues one drain lane takes when nothing sets
// drain.bundle_max: 1, so a drain bundles only where a person sets the key,
// until the split and the drain's figures exist (decision 12).
const BundledBundleMax = 1

// maxBundleMax bounds drain.bundle_max: a bundle past ten issues is no longer
// one reviewable pull request.
const maxBundleMax = 10

// The configuration key the cap claims, under the `drain` namespace of
// .abcd/config.json and ~/.abcd.noindex/config.json.
const (
	drainNamespace = "drain"
	keyBundleMax   = "bundle_max"
)

// bundleMaxLayers are the layers a stored cap may name: the ones
// resolveBundleMax writes. No flag sets it.
var bundleMaxLayers = []string{layered.Repo.String(), layered.Machine.String(), layered.Bundled.String()}

// bundleMaxForm is the accepted form every refusal of the cap names.
var bundleMaxForm = "drain.bundle_max in .abcd/config.json or " + abcdhome.Display("config.json") +
	" is the most issues one drain lane takes, a whole number from 1 to 10 (1, the default, takes one issue per lane)"

// bundledBundleMax is the cap a drain carries when nothing set the key, and
// the one a drain state of version 3 is read with.
func bundledBundleMax() PaceValue {
	return PaceValue{Value: BundledBundleMax, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}
}

// resolveBundleMax reads drain.bundle_max through the layered configuration:
// the machine's file over the bundled default sets the cap, and the
// repository's file may lower it and never raise it. The repository's file is
// whatever its last committer wrote, while the cap bounds how many of the
// person's own reservations one lane holds, so a repository figure above the
// machine's, or above the bundled default where the machine sets none, is
// ignored and named in the returned note, as a repository's launch ledger
// retention below the machine's is (runner.readLedger). It claims the `drain`
// namespace, so a misspelt key is refused naming the file it sits in rather
// than letting the default apply unannounced, and a value outside 1 to 10 in
// any layer is refused naming its origin, never replaced by another layer.
func resolveBundleMax(roots layered.Roots) (PaceValue, string, error) {
	s, err := layered.Load(layered.Config, roots)
	if err != nil {
		return PaceValue{}, "", bundleMaxRefusal(err.Error())
	}
	if err := s.Claim(drainNamespace, keyBundleMax); err != nil {
		return PaceValue{}, "", bundleMaxRefusal(err.Error())
	}
	key := drainNamespace + "." + keyBundleMax
	found, err := s.Lookup(key)
	if err != nil {
		return PaceValue{}, "", bundleMaxRefusal(err.Error())
	}
	v, ignored := bundledBundleMax(), ""
	// Lowest precedence first: the machine's figure is what a repository's
	// is held against.
	for _, f := range slices.Backward(found) {
		n, err := layered.Decode[int](f.Raw)
		if err == nil && (n < 1 || n > maxBundleMax) {
			err = fmt.Errorf("want a whole number of issues from 1 to %d", maxBundleMax)
		}
		if err != nil {
			return PaceValue{}, "", bundleMaxRefusal(fmt.Sprintf("%s (%s layer): %s is %s: %v", f.Origin, f.Layer, key, compactRaw(f.Raw), err))
		}
		if f.Layer == layered.Repo && n > v.Value {
			ignored = fmt.Sprintf("%s (%s) is %d, above the %d %s; a repository's %s may only lower the cap, so it is ignored",
				f.Origin, key, n, v.Value, capSource(v), key)
			continue
		}
		v = PaceValue{Value: n, Layer: f.Layer.String(), Origin: f.Origin}
	}
	return v, termsafe.Sanitize(fsutil.RedactHome(ignored)), nil
}

// capSource names where the cap a repository figure is held against came
// from.
func capSource(v PaceValue) string {
	if v.Layer == layered.Bundled.String() {
		return "abcd bundles when this machine sets none"
	}
	return "this machine sets in " + v.Origin
}

// compactRaw is a configuration value for a refusal, bounded so a file cannot
// flood it.
func compactRaw(raw []byte) string {
	const most = 64
	t := strings.TrimSpace(string(raw))
	if len(t) > most {
		t = t[:most] + "..."
	}
	return t
}

func bundleMaxRefusal(reason string) error {
	return refuse(StageDrain, "", "", termsafe.Sanitize(reason), bundleMaxForm)
}

// bundleMaxWritten reports whether a stored cap is one resolveBundleMax
// writes: a value in range, a layer it writes and an origin with no control,
// bidi or zero-width rune, so a hand edit is refused when the state is read
// rather than echoed to the --json stream.
func bundleMaxWritten(v *PaceValue) bool {
	return v == nil || v.Value >= 1 && v.Value <= maxBundleMax &&
		slices.Contains(bundleMaxLayers, v.Layer) && termsafe.Sanitize(v.Origin) == v.Origin
}

// bundleCap is the cap the drain started with: the bundled default for a
// state that records none.
func (s DrainState) bundleCap() int {
	if s.BundleMax == nil {
		return BundledBundleMax
	}
	return s.BundleMax.Value
}

// issueArea is the place an issue's fix is made, as a bundle reads it: the
// directory holding the path its found_at names in the checkout at repoRoot,
// or that path itself when it is a directory there. "" when found_at names no
// path; "." for a file at the repository's root.
func issueArea(repoRoot string, iss capture.Issue) string {
	fp, ok := issueFootprint(repoRoot, iss)
	if !ok {
		return ""
	}
	return footprintArea(repoRoot, fp[0])
}

// footprintArea is the area of one footprint path p in the checkout at root:
// p itself when it is a directory there, its directory otherwise. A path the
// checkout does not hold, or cannot be read, is read as a file, as is every
// path when root is "".
func footprintArea(root, p string) string {
	if root != "" {
		if fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p))); err == nil && fi.IsDir() {
			return p
		}
	}
	return path.Dir(p)
}

// bundleable reports whether an issue of category and area may be in a
// bundle at all: never a security issue, and never one whose place is the
// repository's root or unknown.
func bundleable(category capture.Category, area string) bool {
	return category != drainrule.SecurityCategory && area != "" && area != "."
}

// bundleCandidates is who would join the lead's lane: the eligible issues
// after the lead in the drain order (order is the plan's dispositions, the
// drain order's), of the lead's category exactly and in its area, never a
// security issue or one at the repository's root, at most limit-1 of them,
// so the lane holds limit issues with its lead. A member's severity may differ from
// the lead's. A lead that is not itself bundleable, or a cap of 1, has none.
// area reads an issue's area by id (issueArea); it is the only thing read
// beyond order, so the rule is a pure function of the plan.
func bundleCandidates(order []capture.DrainVerdict, lead string, limit int, area func(id string) string) []string {
	if limit <= 1 {
		return nil
	}
	at := slices.IndexFunc(order, func(v capture.DrainVerdict) bool { return v.ID == lead })
	if at < 0 {
		return nil
	}
	l := order[at]
	where := area(l.ID)
	if l.Outcome != capture.DrainEligible || !bundleable(l.Category, where) {
		return nil
	}
	var members []string
	for _, v := range order[at+1:] {
		if len(members) == limit-1 {
			break
		}
		if v.Outcome != capture.DrainEligible || v.Category != l.Category {
			continue
		}
		if a := area(v.ID); a != where || !bundleable(v.Category, a) {
			continue
		}
		members = append(members, v.ID)
	}
	return members
}

// DrainBundlePreview is one lane the next moves would open with members, as
// `abcd drain --dry-run` shows it: the lead, and the members it would take.
type DrainBundlePreview struct {
	Lead    string   `json:"lead"`
	Members []string `json:"members"`
}

// bundleSmallerNote says what the dry run's bundles cannot know.
const bundleSmallerNote = "the judgements, the reservations and reservation.max_live may make a bundle smaller, and a lead's members are taken only as its lane forms"

// previewBundles is every lane with members the plan would open under limit: each
// eligible issue that would lead, in the drain order, with its candidates; an
// issue a lead earlier in the order takes never leads, and is not taken twice.
func previewBundles(plan capture.DrainPlan, limit int, area func(id string) string) []DrainBundlePreview {
	if limit <= 1 {
		return nil
	}
	taken := map[string]bool{}
	var out []DrainBundlePreview
	for _, v := range plan.Dispositions {
		if v.Outcome != capture.DrainEligible || taken[v.ID] {
			continue
		}
		rest := slices.DeleteFunc(slices.Clone(plan.Dispositions), func(d capture.DrainVerdict) bool { return taken[d.ID] })
		members := bundleCandidates(rest, v.ID, limit, area)
		if len(members) == 0 {
			continue
		}
		taken[v.ID] = true
		for _, m := range members {
			taken[m] = true
		}
		out = append(out, DrainBundlePreview{Lead: v.ID, Members: members})
	}
	return out
}

// planArea reads an issue's area from the plan's open records, against the
// checkout at repoRoot.
func planArea(repoRoot string, plan capture.DrainPlan) func(id string) string {
	return func(id string) string {
		iss, ok := plan.OpenIssue(id)
		if !ok {
			return ""
		}
		return issueArea(repoRoot, iss)
	}
}
