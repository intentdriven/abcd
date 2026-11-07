package runner

// harness_signing.go reads who signed the harness file a drive runs, with the system's
// own tool, so that a new file can be held to something that does not come from the file.

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// codesignTool is the system's signing tool, by absolute path: nothing on PATH answers
// for it.
const codesignTool = "/usr/bin/codesign"

// codesignTimeout bounds one reading of a signature.
const codesignTimeout = 30 * time.Second

// signingTeamID is a code-signing team's identifier, the one shape put into a requirement.
var signingTeamID = regexp.MustCompile(`^[A-Z0-9]{10}$`)

// HarnessSigning reads and checks who signed a harness file. Neither runs the file or
// reads a login.
type HarnessSigning struct {
	// Team is the team named in path's signature, "" when it names none or cannot be read.
	Team func(path string) string
	// Verify reports whether path's signature holds and is team's.
	Verify func(path, team string) error
}

// Signing is what a drive reads signatures with; a test replaces it.
var Signing = HarnessSigning{Team: codesignTeam, Verify: codesignVerify}

// codesignRun runs the signing tool and returns what it wrote to both streams.
var codesignRun = func(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), codesignTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, codesignTool, args...)
	cmd.Env = []string{}
	return cmd.CombinedOutput()
}

// codesignTeam reads the team from the signature's description.
func codesignTeam(path string) string {
	out, err := codesignRun("-dv", path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if team, ok := strings.CutPrefix(strings.TrimSpace(line), "TeamIdentifier="); ok && signingTeamID.MatchString(team) {
			return team
		}
	}
	return ""
}

// codesignVerify has the system check path's signature and that it chains to a team
// certificate of team.
func codesignVerify(path, team string) error {
	if !signingTeamID.MatchString(team) {
		return errors.New("no signing team to verify against")
	}
	_, err := codesignRun("--verify", `-R=anchor apple generic and certificate leaf[subject.OU] = "`+team+`"`, path)
	return err
}

// verifiedTeam is the team of path's signature when the signature also holds for it, ""
// otherwise: a description alone says what a file claims.
func verifiedTeam(path string) string {
	team := Signing.Team(path)
	if team == "" || Signing.Verify(path, team) != nil {
		return ""
	}
	return team
}
