#!/usr/bin/env bash
# The two hosted scans CI runs on every pull request, run by `make preflight`
# before the push (iss-2610261151381604): govulncheck over the module and
# gitleaks over the commits the branch would land. Between 2026-10-18 and
# 2026-10-26 they failed 12 of 32 failed CI runs, each found only after a push
# and a hosted run; here a finding stops the preflight as a prerequisite, before
# its test stages wait at the machine's load gate.
#
# Each takes its version from .github/workflows/ci.yml, the one pin, so the scan
# run here is CI's: govulncheck from the `go run …/govulncheck@<version>` line of
# the govulncheck job, gitleaks from GITLEAKS_VERSION, which
# scripts/cloud/connectors/claude/setup.sh installs from as well.
#
# A scan that cannot run here is SKIPPED LOUDLY, never passed in silence: no
# gitleaks on PATH, or a govulncheck that cannot reach its vulnerability
# database offline. Neither stops the preflight, since CI's scan still runs; the
# line says what was not checked.
#
# Usage:
#   preflight-scans.sh vulncheck   # govulncheck over ./... (exit 1 on a finding)
#   preflight-scans.sh secrets     # gitleaks over <merge-base>..HEAD (exit 1 on a finding)
#   preflight-scans.sh pins        # print the two versions ci.yml pins
# Exit 0 clean or skipped, 1 a finding, 2 a usage or environment fault.
set -euo pipefail
export LC_ALL=C

CI_FILE=".github/workflows/ci.yml"

say() { printf 'preflight-scans: %s\n' "$*"; }

usage() {
	echo "usage: preflight-scans.sh vulncheck | secrets | pins" >&2
	exit 2
}

# one_pin prints the one value its sed program takes out of ci.yml, or refuses:
# none, or two that differ, leaves no version to match CI with.
one_pin() {
	local what="$1" prog="$2" pins
	[ -f "$CI_FILE" ] || {
		echo "preflight-scans: no $CI_FILE to read the $what version from" >&2
		exit 2
	}
	pins="$(sed -n "$prog" "$CI_FILE" | sort -u)"
	if [ -z "$pins" ] || [ "$(printf '%s\n' "$pins" | grep -c .)" -ne 1 ]; then
		echo "preflight-scans: $CI_FILE does not pin exactly one $what version, so there is no version to match CI's scan with" >&2
		exit 2
	fi
	printf '%s' "$pins"
}

vuln_pin() {
	one_pin govulncheck 's/.*golang\.org\/x\/vuln\/cmd\/govulncheck@\(v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\).*/\1/p'
}

leaks_pin() {
	one_pin gitleaks 's/^[[:space:]]*GITLEAKS_VERSION:[[:space:]]*\([0-9][0-9.]*\)[[:space:]]*$/\1/p'
}

# default_ref is origin's default branch as this clone last fetched it:
# origin/HEAD when it is set, origin/main otherwise, or nothing.
default_ref() {
	local ref
	ref="$(git symbolic-ref -q --short refs/remotes/origin/HEAD 2>/dev/null || true)"
	[ -n "$ref" ] || ref="origin/main"
	if git rev-parse -q --verify "$ref^{commit}" >/dev/null 2>&1; then
		printf '%s' "$ref"
	fi
}

vulncheck() {
	local version rc=0 out def base fixes
	version="$(vuln_pin)"
	out="$(go run "golang.org/x/vuln/cmd/govulncheck@$version" ./... 2>&1)" || rc=$?
	case "$rc" in
	0)
		say "govulncheck $version: no known vulnerability the module's code reaches"
		return 0
		;;
	3) ;;
	*)
		printf '%s\n' "$out" | tail -n 5 | sed 's/^/  /'
		say "SKIPPED — govulncheck $version could not run (exit $rc), so the module was not checked for known vulnerabilities."
		say "  Offline, it cannot fetch its vulnerability database. CI's govulncheck job still runs; to scan here, run again with the network."
		return 0
		;;
	esac
	printf '%s\n' "$out"
	say "govulncheck $version found a known vulnerability the module's code reaches; the preflight stops before its test stages."
	fixes="$(printf '%s\n' "$out" | sed -n 's/^[[:space:]]*Fixed in:[[:space:]]*//p' | sort -u)"
	if [ -n "$fixes" ]; then
		say "The fixes govulncheck names:"
		printf '%s\n' "$fixes" | sed 's/^/  /'
	fi
	# A branch that moves no dependency did not bring the finding in: a newly
	# published vulnerability, or the toolchain go.mod declares, fails the
	# default branch too, and every branch cut from it (iss-2610261151481013).
	def="$(default_ref)"
	if [ -n "$def" ] && base="$(git merge-base HEAD "$def" 2>/dev/null)" &&
		git diff --quiet "$base" HEAD -- go.mod go.sum; then
		say "This branch changes neither go.mod nor go.sum against $def, so the finding most likely stands on $def itself."
		say "  Fix it there, on a branch of its own (the toolchain line in go.mod, or the module go.sum names), then rebase this one onto it."
	fi
	return 1
}

secrets() {
	local version raw have def base rc=0 out
	version="$(leaks_pin)"
	if ! command -v gitleaks >/dev/null 2>&1; then
		say "SKIPPED — gitleaks is not on PATH, so the commits this branch would land were not scanned for secrets."
		say "  CI's gitleaks job (gitleaks $version) still scans them. Install gitleaks $version to scan here."
		return 0
	fi
	raw="$(gitleaks version 2>/dev/null | head -n 1 || true)"
	have="$(printf '%s' "$raw" | tr -d '[:space:]')"
	have="${have#v}"
	if [ "$have" != "$version" ]; then
		say "WARNING — gitleaks reports version '${raw:-unknown}', not the $version $CI_FILE pins; its rules may differ from CI's, so a clean scan here is not CI's verdict."
	fi
	def="$(default_ref)"
	if [ -z "$def" ] || ! base="$(git merge-base HEAD "$def" 2>/dev/null)"; then
		say "WARNING — no origin default branch to scope the scan to; scanning the full history reachable from HEAD instead."
		base=""
	fi
	if [ -n "$base" ] && [ "$base" = "$(git rev-parse HEAD)" ]; then
		say "gitleaks: this branch adds no commit to $def; nothing to scan"
		return 0
	fi
	local scope="${base:+$base..}HEAD"
	say "gitleaks: scanning $scope"
	out="$(gitleaks git --redact --no-banner --log-opts="$scope" . 2>&1)" || rc=$?
	printf '%s\n' "$out"
	if [ "$rc" -ne 0 ]; then
		say "gitleaks found a secret in $scope (exit $rc); the preflight stops. Remove it from the history (rewrite the branch), and rotate it if it was ever pushed."
		return 1
	fi
	return 0
}

[ $# -eq 1 ] || usage
case "$1" in
vulncheck) vulncheck ;;
secrets) secrets ;;
pins)
	# Assigned first: a failed substitution inside printf's arguments would
	# not stop the script, and an empty pin would print as a version.
	vuln="$(vuln_pin)"
	leaks="$(leaks_pin)"
	printf 'govulncheck %s\ngitleaks %s\n' "$vuln" "$leaks"
	;;
*) usage ;;
esac
