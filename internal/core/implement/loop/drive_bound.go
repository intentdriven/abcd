package loop

// drive_bound.go is how long a drive runs when the person names no end
// (itd-2611251809580662 decisions 1 and 9, spc-2611251810508418 step 3):
// seven days, or `drive.until_days`, a whole number from 1 to 31 from the
// repository's .abcd/config.json at the commit origin itself names as its
// default branch's tip (claim.AskOriginDefault), as the object store holds it
// and never from a working tree a lane's agent can write. The surface reads it
// once, before the consent, turns it into an instant shown before the yes, and
// passes that instant on every re-execution, so nothing read later can move a
// drive's end. --until is the person's own bound and is not this file's.
//
// The `drive` namespace holds keys of two owners: the repository's default
// length here, and the person's own label (step 6) in the machine's file.
// driveKeys names each key with the one layer that holds it, so every reader
// claims the whole namespace from one list, and a key in the other party's
// file is refused naming that file, never passed over and never mistaken for
// a misspelt key.

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// BundledDriveDays is how many days a drive runs when nothing sets
// drive.until_days and no --until is given.
const BundledDriveDays = 7

// MaxDriveDays bounds drive.until_days.
const MaxDriveDays = 31

// driveNamespace is the drive's configuration namespace.
const driveNamespace = "drive"

// driveKey is one key of the drive namespace, with the one layer that holds
// it and whose it is, said when another layer sets it.
type driveKey struct {
	name  string
	layer layered.Layer
	owner string
}

// driveKeys are every key the drive namespace holds. Every reader of the
// namespace claims all of them and refuses each one set in a layer that is not
// its own.
var driveKeys = []driveKey{
	{name: "until_days", layer: layered.Repo,
		owner: "how long a drive runs by default is the repository's to say, and --until is the person's"},
}

// driveKeyNames are driveKeys' names, for a claim of the namespace.
func driveKeyNames() []string {
	names := make([]string, len(driveKeys))
	for i, k := range driveKeys {
		names[i] = k.name
	}
	return names
}

// untilDaysKey is the default length's key as a person writes it.
var untilDaysKey = driveNamespace + ".until_days"

// untilDaysForm is the accepted form every refusal of the default names.
var untilDaysForm = "drive.until_days in the repository's .abcd/config.json, read at origin's default branch, is how many days a drive runs when --until is not given, a whole number from 1 to 31 (7, the default, is a week); " +
	abcdhome.Display("config.json") + " does not hold it, and --until <date or RFC 3339 instant> sets one drive's end"

// bundledDriveDays is the default length no file sets.
func bundledDriveDays() PaceValue {
	return PaceValue{Value: BundledDriveDays, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}
}

// resolveDriveDays resolves how many days a drive starting now runs when the
// person gave no --until: the repository's drive.until_days at origin's
// default branch, else the bundled seven, with a note saying why no
// repository figure was read where one could not be. roots name the machine's
// file, which is read only to refuse the key there. A repository value outside
// 1 to 31 refuses, naming the file at the branch and the commit.
// It is unexported until the drive's start calls it, which exports it.
func resolveDriveDays(repoRoot string, roots layered.Roots) (PaceValue, string, error) {
	return resolveDriveDaysAsking(repoRoot, roots, claim.AskOriginDefault)
}

// resolveDriveDaysAsking is resolveDriveDays asking origin through ask.
//
// Where origin cannot be asked, or names no default branch, the bundled seven
// applies and the note says so: the end is shown as an instant before the
// person's yes, so a drive the repository would have made shorter starts only
// on a yes to the end it shows, and an outage, which a drive exists to wait
// through, does not keep one from starting.
func resolveDriveDaysAsking(repoRoot string, roots layered.Roots, ask askOriginDefault) (PaceValue, string, error) {
	if err := refuseMachineDrive(roots); err != nil {
		return PaceValue{}, "", err
	}
	v := bundledDriveDays()
	has, err := claim.HasOrigin(repoRoot)
	if err != nil || !has {
		return v, fmt.Sprintf("this checkout has no remote named %s, so no repository figure for %s was read and the drive runs %d days, abcd's default",
			claim.Remote, untilDaysKey, v.Value), nil
	}
	at, err := ask(repoRoot, untilDaysKey)
	if err != nil {
		return v, fmt.Sprintf("%s could not be asked which branch is its default (%s), so no repository figure for %s was read and the drive runs %d days, abcd's default",
			claim.Remote, termsafe.Sanitize(fsutil.RedactHome(err.Error())), untilDaysKey, v.Value), nil
	}
	if at.None != "" {
		return v, fmt.Sprintf("%s, so it has no repository figure for %s and the drive runs %d days, abcd's default",
			termsafe.Sanitize(at.None), untilDaysKey, v.Value), nil
	}
	s, err := layered.Load(layered.Config, layered.Roots{Repo: repoRoot, RepoAt: at.Tip, RepoAtName: at.Name, RepoAtAsStored: true, HomeRefused: true})
	if err != nil {
		return PaceValue{}, "", untilDaysRefusal(err.Error())
	}
	if err := claimDriveNamespace(s); err != nil {
		return PaceValue{}, "", untilDaysRefusal(err.Error())
	}
	found, err := s.Lookup(untilDaysKey)
	if err != nil {
		return PaceValue{}, "", untilDaysRefusal(err.Error())
	}
	for _, f := range found {
		if f.Layer != layered.Repo {
			continue
		}
		n, err := layered.Decode[int](f.Raw)
		if err == nil {
			err = checkDriveDays(n)
		}
		if err != nil {
			return PaceValue{}, "", untilDaysRefusal(fmt.Sprintf("%s (commit %s): %s is %s: %v", f.Origin, at.Tip[:12], untilDaysKey, compactRaw(f.Raw), err))
		}
		v = PaceValue{Value: n, Layer: layered.Repo.String(), Origin: f.Origin}
	}
	return v, "", nil
}

// refuseMachineDrive claims the drive namespace in the machine's file, loaded
// alone so no working tree is read, and refuses a key there that the
// machine's layer does not hold, naming the file.
func refuseMachineDrive(roots layered.Roots) error {
	if roots.HomeRefused || roots.Home == "" {
		return nil
	}
	s, err := layered.Load(layered.Config, layered.Roots{Home: roots.Home})
	if err != nil {
		return untilDaysRefusal(err.Error())
	}
	if err := claimDriveNamespace(s); err != nil {
		return untilDaysRefusal(err.Error())
	}
	return nil
}

// claimDriveNamespace claims the drive namespace in every layer s holds, with every key
// of driveKeys, and refuses a key set in a layer that is not its own.
func claimDriveNamespace(s *layered.Stack) error {
	if err := s.Claim(driveNamespace, driveKeyNames()...); err != nil {
		return err
	}
	for _, k := range driveKeys {
		key := driveNamespace + "." + k.name
		found, err := s.Lookup(key)
		if err != nil {
			return err
		}
		for _, f := range found {
			if f.Layer != k.layer {
				return fmt.Errorf("%s sets %s, which that file does not hold: %s; remove it from %s", f.Origin, key, k.owner, f.Origin)
			}
		}
	}
	return nil
}

// checkDriveDays holds a figure to 1 to MaxDriveDays.
func checkDriveDays(n int) error {
	if n < 1 || n > MaxDriveDays {
		return fmt.Errorf("want a whole number of days from 1 to %d", MaxDriveDays)
	}
	return nil
}

func untilDaysRefusal(reason string) error {
	return refuse(StageDrain, "", "", termsafe.Sanitize(fsutil.RedactHome(reason)), untilDaysForm)
}
