package embargo

import (
	"errors"
	"strings"
)

// AnnouncementCheck is the embargo check an announcement is held to
// (itd-2609151838312703 C5), in the shape claim.EmbargoCheck takes: `abcd
// embargo check` on the announcement's whole message and the record it names.
// Any finding holds it, and a check that cannot run is an error, which refuses
// it too. `abcd announce` and the announcements abcd's own pushes make
// (`abcd land`, a build or drain lane's landing) all pass through it, so no
// announcement is judged by a second copy of the check.
func AnnouncementCheck(root, home, text string, records []string) (bool, error) {
	if home == "" {
		return false, errors.New("the home directory cannot be named, so the embargo store cannot be found")
	}
	res, err := Check(CheckRequest{Home: home, RepoRoot: root, Files: []string{"-"},
		Stdin: strings.NewReader(text), Records: records})
	if err != nil {
		return false, err
	}
	return !res.Clean(), nil
}
