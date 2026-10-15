// Package ghcontract pins the forge client's `--json` field lists, so a test's
// fake forge client refuses a field the real one refuses.
//
// The problem it closes: `abcd land` asked `gh pr view --json` for
// isInMergeQueue, a field gh has no such name for, and a fake forge that
// answered any field hid it; every real queue confirmation failed. The lists
// here are gh's own, as `gh pr view --json`, `gh pr list --json` (the same
// set) and `gh repo view --json` print them with no value, under
// gh-2.100.0/. Refresh them by running those three commands with a newer gh.
//
// It is a test-support package, as gittest is: production code never imports
// it.
package ghcontract

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

// Version is the gh release the lists were read from.
const Version = "2.100.0"

var (
	//go:embed gh-2.100.0/pr-json-fields.txt
	prFields string
	//go:embed gh-2.100.0/repo-view-json-fields.txt
	repoViewFields string
)

// Fields is the `--json` field list gh takes for the command named by the
// first two arguments ("pr view", "pr list", "repo view"), or nil for a
// command whose list is not pinned here.
func Fields(command string) []string {
	switch command {
	case "pr view", "pr list":
		return strings.Fields(prFields)
	case "repo view":
		return strings.Fields(repoViewFields)
	}
	return nil
}

// Check refuses a forge client call (gh's arguments) whose `--json` or
// `--json=` names a field gh does not take for that command, as gh does, and
// a `--json` on a command whose list is not pinned. A call with no `--json`
// passes.
func Check(args []string) error {
	var want []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--json" && i+1 < len(args):
			i++
			want = append(want, strings.Split(args[i], ",")...)
		case a == "--json":
			return fmt.Errorf("gh %v: --json takes one or more comma-separated fields", args)
		case strings.HasPrefix(a, "--json="):
			want = append(want, strings.Split(strings.TrimPrefix(a, "--json="), ",")...)
		}
	}
	if want == nil {
		return nil
	}
	command := strings.Join(args[:min(2, len(args))], " ")
	known := Fields(command)
	if known == nil {
		return fmt.Errorf("gh %v: no --json field list is pinned for %q (gh %s)", args, command, Version)
	}
	for _, f := range want {
		if !slices.Contains(known, f) {
			return fmt.Errorf("gh: Unknown JSON field: %q (gh %s, %s)", f, Version, command)
		}
	}
	return nil
}

// Shell is a POSIX-shell function, ghcontract_check, for a stub forge client
// written as a script: called with the script's arguments ("$@"), it exits 1
// with gh's message on a field Check refuses.
func Shell() string {
	pr := " " + strings.Join(Fields("pr view"), " ") + " "
	repo := " " + strings.Join(Fields("repo view"), " ") + " "
	return `ghcontract_check() {
  ghc_cmd="$1 $2"; ghc_want=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --json) shift; ghc_want="$ghc_want,$1" ;;
      --json=*) ghc_want="$ghc_want,${1#--json=}" ;;
    esac
    shift
  done
  [ -n "$ghc_want" ] || return 0
  case "$ghc_cmd" in
    "pr view"|"pr list") ghc_known="` + pr + `" ;;
    "repo view") ghc_known="` + repo + `" ;;
    *) echo "stub gh: no --json field list is pinned for $ghc_cmd (gh ` + Version + `)" >&2; exit 1 ;;
  esac
  set -f; ghc_ifs=$IFS; IFS=,
  for ghc_f in ${ghc_want#,}; do
    case "$ghc_known" in
      *" $ghc_f "*) ;;
      *) IFS=$ghc_ifs; set +f; echo "Unknown JSON field: \"$ghc_f\" (gh ` + Version + `, $ghc_cmd)" >&2; exit 1 ;;
    esac
  done
  IFS=$ghc_ifs; set +f
}
`
}
