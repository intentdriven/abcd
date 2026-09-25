package cli

import (
	"errors"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// noLaunchPayloadGuidance is what a repository with no launch payload is told
// (iss-2608270559313719). It is not misconfigured: a repository that ships no
// plugin bundle has no include config, and its releases go through the
// changelog-driven path instead, which the refusal names.
const noLaunchPayloadGuidance = "this repository declares no launch payload (.abcd/config/launch-payload.json), " +
	"so there is no plugin bundle to preview, gate or stage. A repository without one releases through the " +
	"changelog-driven path: `abcd launch scaffold` installs the release workflows, `abcd launch ship` derives the " +
	"version and writes the dated CHANGELOG heading, and the auto-release workflow tags that commit on merge. " +
	"A repository that does ship a plugin declares its payload in that file"

// launchPayloadRefusal turns a launch error into what the operator reads: the
// release-path guidance for a repository with no payload, the scrubbed error
// otherwise.
func launchPayloadRefusal(err error) string {
	if errors.Is(err, launch.ErrNoLaunchPayload) {
		return noLaunchPayloadGuidance
	}
	return scrubPaths(err)
}
