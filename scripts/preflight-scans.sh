#!/usr/bin/env bash
# The two hosted scans CI runs on every pull request, run by `make preflight`
# before the push (iss-2610261151381604): govulncheck over the module and
# gitleaks over the commits the branch would land. Between 2026-10-18 and
# 2026-10-26 they failed 12 of 32 failed CI runs, each found only after a push
# and a hosted run; here a finding stops the preflight as a prerequisite, before
# its build and test stages. The one exception is a govulncheck finding on a
# branch that moves no dependency, when the same scan of the fork point's tree
# reports every vulnerability it names: govulncheck is not a required check,
# and the branch did not bring that finding in, so stopping would hold every
# branch for a fix none of them can make. It warns.
# The comparison is by vulnerability id: a branch that newly reaches an id the
# fork point already reaches, by another path, is warned about, not stopped.
#
# Each takes its version from .github/workflows/ci.yml, the one pin, so the scan
# run here is CI's: govulncheck from the `go run …/govulncheck@<version>` line of
# the govulncheck job, gitleaks from GITLEAKS_VERSION, which
# scripts/cloud/connectors/claude/setup.sh installs from as well. A gitleaks on
# PATH at another version, or none, is replaced by CI's release, checked against
# the sha256 in scripts/gitleaks-checksums.txt (iss-2611081341084748): a newer
# gitleaks passed a token fixture CI's refused. When that release cannot be
# fetched or verified the secret scan REFUSES (exit 2), naming the skew.
# A gitleaks on PATH that reports CI's version runs as it is, with no digest
# check: it is the person's own PATH, trusted exactly as it was before the pin
# was enforced. Only the release this script fetches is checked against
# scripts/gitleaks-checksums.txt.
#
# A govulncheck that cannot be built at CI's version, cannot run, or runs and
# cannot reach its vulnerability database offline is SKIPPED LOUDLY, never
# passed in silence: govulncheck is not a required check and CI's scan still
# runs; the lines say what was not checked and why. A build the module system
# refuses on a checksum that fails verification REFUSES (exit 2) instead, as a
# gitleaks release that fails its check does. It runs as its own binary, so its
# finding exit (3) is told apart from a run that failed (see vulncheck).
#
# Usage:
#   preflight-scans.sh vulncheck   # govulncheck over ./... (exit 1 on a finding)
#   preflight-scans.sh secrets     # gitleaks over <merge-base>..HEAD (exit 1 on a finding)
#   preflight-scans.sh pins        # print the two versions ci.yml pins
# Exit 0 clean, skipped or warned, 1 a finding, 2 a usage or environment fault
# (a gitleaks that ran and could not scan among them: CI's would fail too;
# CI's gitleaks that could not be fetched or verified; and a govulncheck whose
# build failed its checksum verification).
set -euo pipefail
export LC_ALL=C

CI_FILE=".github/workflows/ci.yml"
SUMS_FILE="scripts/gitleaks-checksums.txt"

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

# vuln_ids prints the vulnerability ids a govulncheck report names, one a line.
vuln_ids() {
	printf '%s\n' "$1" | sed -n 's/^Vulnerability #[0-9]*: \(GO-[0-9][0-9]*-[0-9][0-9]*\).*/\1/p' | sort -u
}

# scan_tree is the fork point's tree while ids_not_at scans it, removed however
# the script ends: an interrupted scan leaves no copy of the module behind. It is
# made and removed by vulncheck itself, never inside the command substitution
# ids_not_at runs in: a subshell's variables never reach this EXIT trap, and an
# interrupt ends the subshell before its own cleanup runs.
scan_tree=""
leaks_tmp=""
vuln_bin=""
trap 'trap "" HUP INT TERM; [ -z "$scan_tree" ] || rm -rf "$scan_tree"; [ -z "$leaks_tmp" ] || rm -rf "$leaks_tmp"; [ -z "$vuln_bin" ] || rm -rf "$vuln_bin"' EXIT
# Each signal trap ignores the three signals before it exits, and the EXIT trap
# ignores them before its first removal: a signal while the removals run would
# take a signal trap, whose exit ends the EXIT trap at once and leaves whatever
# was not yet removed. The EXIT trap's own ignore covers a script that ended by
# itself; the signal traps' covers the moment before the EXIT trap starts.
trap 'trap "" HUP INT TERM; exit 129' HUP
trap 'trap "" HUP INT TERM; exit 130' INT
trap 'trap "" HUP INT TERM; exit 143' TERM

# set_tmp_root sets tmp_root to the temporary directory every private directory
# here is made in, as an absolute path: TMPDIR, or /tmp. A relative TMPDIR is
# made absolute against the working directory, since go install refuses a
# relative GOBIN; one that names no directory refuses, naming it. TMPDIR is
# exported as that absolute path, so a tool this script starts sees it too:
# the fork point's scan runs inside the extracted tree, where the relative one
# names nothing.
tmp_root=""
set_tmp_root() {
	local d="${TMPDIR:-/tmp}"
	case "$d" in
	/*) tmp_root="$d" ;;
	*)
		tmp_root="$(CDPATH='' cd -- "$d" 2>/dev/null && pwd -P)" || {
			echo "preflight-scans: TMPDIR is $d, a relative path that names no directory here; set it to an absolute path" >&2
			exit 2
		}
		;;
	esac
	export TMPDIR="$tmp_root"
}

# ids_not_at prints the ids report $3 names that the same scan of commit $1's
# tree, extracted into the empty directory $4, by the govulncheck binary $2,
# does not; it fails when that scan cannot run, with the scan's last lines on
# stderr.
ids_not_at() {
	local commit="$1" tool="$2" report="$3" tree="$4" rc=0 out
	git archive "$commit" | tar -x -C "$tree" || return 1
	out="$(cd "$tree" && "$tool" ./... 2>&1)" || rc=$?
	case "$rc" in
	0 | 3) ;;
	*)
		printf '%s\n' "$out" | tail -n 5 | sed 's/^/  /' >&2
		return 1
		;;
	esac
	comm -23 <(vuln_ids "$report") <(vuln_ids "$out")
}

# vuln_skipped says, loudly, that the module was not checked: $1 is the lines
# that say why, $2 what failed, and $3 what those lines are, which differs by
# caller: go's own when the tool could not be built, the system's when the
# built binary could not run or was ended, the tool's when it ran and stopped.
vuln_skipped() {
	printf '%s\n' "$1" | tail -n 5 | sed 's/^/  /'
	say "SKIPPED — $2, so the module was not checked for known vulnerabilities."
	say "  $3"
	say "  CI's govulncheck job still runs."
}

vulncheck() {
	local version rc=0 out def base fixes tool
	version="$(vuln_pin)"
	# govulncheck runs as its own binary, never through `go run`: go run
	# reports any non-zero exit of the program it ran as its own exit 1, with
	# an "exit status N" line, so a finding (govulncheck's exit 3) arrived as
	# the same exit 1 an offline run gives, and every finding was reported as
	# a skip while CI's scan failed. Reading that line back would judge go's
	# wording, not the tool's exit; `go install <pkg>@<version>` builds the
	# very program `go run <pkg>@<version>` builds (same module, version and
	# toolchain choice, from the same directory) into a private directory the
	# EXIT trap removes, and the tool's exit then arrives unchanged.
	set_tmp_root
	vuln_bin="$(mktemp -d "$tmp_root/preflight-govulncheck.XXXXXX")" || {
		echo "preflight-scans: no temporary directory to build govulncheck $version into" >&2
		exit 2
	}
	out="$(GOBIN="$vuln_bin" go install "golang.org/x/vuln/cmd/govulncheck@$version" 2>&1)" || rc=$?
	if [ "$rc" -ne 0 ]; then
		# A download that does not match the checksum database (go prints
		# "SECURITY ERROR"), a checksum database that fails its own proofs
		# ("security error: misbehaving server"), or a module the module cache
		# holds altered ("checksum mismatch") is not a machine that is
		# offline: what would run is not what CI runs, and something changed
		# it. Skipping would pass the preflight over a tool supply that failed
		# its verification, so it refuses, as a gitleaks release that fails its
		# sha256 check does. Exactly two strings are matched, ignoring case,
		# in a lower-cased copy of go's lines: "security error" and "checksum
		# mismatch". A verification failure go words otherwise is not matched
		# and takes the loud skip below; either way no binary is run. A
		# checksum database go cannot reach (no network, a 404 or 410) is
		# neither, and skips.
		local lc
		lc="$(printf '%s' "$out" | tr '[:upper:]' '[:lower:]')"
		case "$lc" in
		*"security error"* | *"checksum mismatch"*)
			# go's report of a mismatch runs to about ten lines, the
			# module and both hashes first: all of it is shown.
			printf '%s\n' "$out" | tail -n 15 | sed 's/^/  /'
			say "REFUSING to scan for known vulnerabilities: go would not build govulncheck $version, because a module it fetched failed its checksum verification (the lines above are go's)."
			say "  NOT skipping: what would run is not what CI runs. If the module cache's copy was altered, removing it (go clean -modcache) clears it; if it repeats, the download itself does not match, and the scan waits until it does."
			exit 2
			;;
		esac
		vuln_skipped "$out" "govulncheck $version could not be built (go install exit $rc)" \
			"The lines above are go's: it could not fetch or build the tool (with no network for its module proxy, for one)."
		return 0
	fi
	if [ ! -x "$vuln_bin/govulncheck" ]; then
		vuln_skipped "$out" "govulncheck $version could not be built (go install exited 0 and left no govulncheck in $vuln_bin)" \
			"go reported success but wrote no binary there, so there is nothing to run."
		return 0
	fi
	tool="$vuln_bin/govulncheck"
	out="$("$tool" ./... 2>&1)" || rc=$?
	case "$rc" in
	0)
		say "govulncheck $version: no known vulnerability the module's code reaches"
		return 0
		;;
	3) ;;
	126 | 127)
		vuln_skipped "$out" "the govulncheck $version built in $vuln_bin could not be executed (exit $rc)" \
			"The lines above are the system's; a temporary directory mounted noexec, or a binary built for another system, does this. Set TMPDIR to a directory that allows execution (here it is $tmp_root)."
		return 0
		;;
	*)
		if [ "$rc" -gt 128 ]; then
			vuln_skipped "$out" "govulncheck $version was ended by signal $((rc - 128)) (exit $rc) before it finished" \
				"Nothing it printed is a result; a signal from outside, or the system ending it for memory, does this."
		else
			vuln_skipped "$out" "govulncheck $version ran and stopped with exit $rc" \
				"The lines above are its own: offline it cannot fetch its vulnerability database, and a module it cannot load stops it too."
		fi
		return 0
		;;
	esac
	printf '%s\n' "$out"
	fixes="$(printf '%s\n' "$out" | sed -n 's/^[[:space:]]*Fixed in:[[:space:]]*//p' | sort -u)"
	# A branch that moves no dependency most likely did not bring the finding
	# in: a newly published vulnerability, or the toolchain go.mod declares,
	# fails the default branch too, and every branch cut from it
	# (iss-2610261151481013). But govulncheck judges reachability, so this
	# branch's own code may be what newly reaches a module already required:
	# the fork point's tree is scanned the same way, and only a finding it
	# carries too warns (iss-2610262259078996).
	def="$(default_ref)"
	if [ -n "$def" ] && base="$(git merge-base HEAD "$def" 2>/dev/null)" &&
		git diff --quiet "$base" HEAD -- go.mod go.sum; then
		local added="" scanned=0
		if [ -n "$(vuln_ids "$out")" ] &&
			scan_tree="$(mktemp -d "$tmp_root/preflight-scans.XXXXXX")"; then
			added="$(ids_not_at "$base" "$tool" "$out" "$scan_tree")" && scanned=1
			rm -rf "$scan_tree"
			scan_tree=""
		fi
		if [ -z "$(vuln_ids "$out")" ]; then
			say "The report names no vulnerability id this script can read, so whether $def carries the finding too cannot be told."
		elif [ "$scanned" -ne 1 ]; then
			say "The fork point with $def could not be scanned, so whether this branch brought the finding in cannot be told."
		elif [ -z "$added" ]; then
			say "WARNING — govulncheck $version found a known vulnerability the module's code reaches; this branch changes neither go.mod nor go.sum"
			say "  against $def, and the same scan of the fork point's tree reports every one of its ids too, so this branch did not bring it in."
			say "  It does not stop the preflight: govulncheck is not a required check. If $def has moved since the fork point, rebase first:"
			say "  it may carry the fix already. Otherwise fix it on a branch of its own (the toolchain line in go.mod, or the module go.sum names)."
			if [ -n "$fixes" ]; then
				say "The fixes govulncheck names:"
				printf '%s\n' "$fixes" | sed 's/^/  /'
			fi
			return 0
		else
			say "This branch changes neither go.mod nor go.sum against $def, but its own code newly reaches what the fork point's does not:"
			printf '%s\n' "$added" | sed 's/^/  /'
		fi
	fi
	say "govulncheck $version found a known vulnerability the module's code reaches; the preflight stops before its test stages."
	if [ -n "$fixes" ]; then
		say "The fixes govulncheck names:"
		printf '%s\n' "$fixes" | sed 's/^/  /'
	fi
	return 1
}

# pinned_gitleaks sets leaks_cmd to a gitleaks at version $1, the one ci.yml
# pins, or refuses with exit 2 naming the skew (iss-2611081341084748). A
# gitleaks on PATH reporting exactly that version is CI's and runs as it is.
# Anything else, another version or none, is set aside for CI's own release:
# the tarball for this machine is kept in the abcd home, in a directory made
# 0700, and on every run copied into a private temporary directory the EXIT
# trap removes; that copy is the one checked against the sha256 the repository
# holds in $SUMS_FILE (never a checksum list fetched beside it) and the one
# unpacked, so what runs is exactly what was verified. A fetch lands in the
# private directory first and is kept only once it matches. The kept directory
# and tarball must belong to this user, be writable by no one else and not be
# symlinks, or the scan refuses naming the path. A fetch or a check that fails
# refuses: falling back to the PATH copy would judge the branch with rules CI
# does not run, the false pass this exists to remove (loud staging).
pinned_gitleaks() {
	local version="$1" have="none on PATH" raw os arch asset digest n cache got
	if command -v gitleaks >/dev/null 2>&1; then
		raw="$(gitleaks version 2>/dev/null | head -n 1 || true)"
		raw="$(printf '%s' "$raw" | tr -d '[:space:]')"
		if [ "${raw#v}" = "$version" ]; then
			leaks_cmd="gitleaks"
			return 0
		fi
		have="${raw:-a version it does not report} on PATH"
	fi
	refuse() {
		say "REFUSING to scan for secrets: $CI_FILE pins gitleaks $version, and the gitleaks here is $have."
		say "  $*"
		say "  NOT falling back to the gitleaks on PATH: another version's rules pass what CI's refuses."
		exit 2
	}
	case "$(uname -s)" in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) refuse "gitleaks $version is fetched for macOS and Linux only, not $(uname -s); install gitleaks $version on PATH." ;;
	esac
	case "$(uname -m)" in
	arm64 | aarch64) arch=arm64 ;;
	x86_64 | amd64) arch=x64 ;;
	*) refuse "gitleaks $version is fetched for arm64 and x86_64 only, not $(uname -m); install gitleaks $version on PATH." ;;
	esac
	asset="gitleaks_${version}_${os}_${arch}.tar.gz"
	# The asset is matched as a whole field, never as a pattern: its dots are
	# literal.
	digest=""
	if [ -f "$SUMS_FILE" ]; then
		digest="$(awk -v a="$asset" 'NF == 2 && $2 == a && length($1) == 64 && $1 !~ /[^0-9a-f]/ { print $1 }' "$SUMS_FILE")" ||
			refuse "$SUMS_FILE could not be read, so there is no sha256 to verify $asset against."
	fi
	n="$(printf '%s' "$digest" | grep -c . || true)"
	if [ "$n" -ne 1 ]; then
		refuse "$SUMS_FILE holds no single sha256 for $asset, so there is nothing to verify a download against; copy its line from the release's checksum list when bumping GITLEAKS_VERSION."
	fi
	if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
		refuse "Neither sha256sum nor shasum is on PATH, so $asset cannot be verified against $SUMS_FILE."
	fi
	[ -n "${HOME:-}" ] || refuse "HOME is not set, so there is no abcd home to keep CI's release in."
	cache="$HOME/.abcd.noindex/gate-tools/gitleaks/$version"
	(umask 077 && mkdir -p "$cache") || refuse "$cache could not be made."
	private_to_me "$cache" d directory
	set_tmp_root
	leaks_tmp="$(mktemp -d "$tmp_root/preflight-gitleaks.XXXXXX")" || refuse "no temporary directory to unpack $asset into."
	local copy="$leaks_tmp/$asset" fetched=0
	if [ -e "$cache/$asset" ] || [ -L "$cache/$asset" ]; then
		private_to_me "$cache/$asset" f file
		cat "$cache/$asset" >"$copy" || refuse "The cached $cache/$asset could not be read."
	else
		say "gitleaks $version: fetching CI's release ($asset), to verify against $SUMS_FILE"
		if ! curl -sSLf --retry 2 -o "$copy" "https://github.com/gitleaks/gitleaks/releases/download/v$version/$asset"; then
			refuse "The download of $asset failed (curl's error is above; the fetch needs network)."
		fi
		fetched=1
	fi
	got="$(sha256_of "$copy")" || refuse "The hasher on PATH could not hash $copy (its error is above), so $asset cannot be verified against $SUMS_FILE."
	if [ "$got" != "$digest" ]; then
		if [ "$fetched" -eq 1 ]; then
			refuse "The downloaded $asset has sha256 $got, not the $digest $SUMS_FILE pins; nothing was kept or run."
		fi
		refuse "The cached $cache/$asset has sha256 $got, not the $digest $SUMS_FILE pins; remove it to fetch it again."
	fi
	if [ "$fetched" -eq 1 ]; then
		local part="$cache/$asset.part.$$"
		if (umask 077 && cat "$copy" >"$part") && mv -f "$part" "$cache/$asset"; then
			:
		else
			rm -f "$part"
			say "gitleaks $version: the verified $asset could not be kept in $cache; the next run fetches it again."
		fi
	fi
	tar -xzf "$copy" -C "$leaks_tmp" gitleaks || refuse "$asset holds no gitleaks to unpack."
	# A binary the system will not execute (a TMPDIR mounted noexec, where
	# the shell exits 126 with a permission error) reports no version, and
	# that is not the release's fault: the refusal says so and where.
	local ran=0 why
	raw="$("$leaks_tmp/gitleaks" version 2>"$leaks_tmp/version.err")" || ran=$?
	why="$(head -n 1 "$leaks_tmp/version.err" 2>/dev/null || true)"
	case "$ran:$why" in
	0:*) ;;
	126:* | *"Permission denied"* | *"Operation not permitted"*)
		refuse "The verified $asset unpacked a gitleaks into $leaks_tmp that could not be executed (${why:-exit $ran}); a temporary directory mounted noexec does this. Set TMPDIR to a directory that allows execution (the temporary directory here is $tmp_root) and run again."
		;;
	esac
	raw="$(printf '%s\n' "$raw" | head -n 1 | tr -d '[:space:]')"
	[ "${raw#v}" = "$version" ] || refuse "The verified $asset unpacked a gitleaks that reports '${raw:-nothing}', not $version."
	say "gitleaks: $have, not CI's $version; scanning with CI's gitleaks $version, sha256 verified against $SUMS_FILE"
	leaks_cmd="$leaks_tmp/gitleaks"
}

# private_to_me refuses, through pinned_gitleaks's refuse, unless path $1 is
# of find type $2 (d or f), is not a symlink, belongs to this user and is
# writable by no one else; $3 names the kind in the refusal.
private_to_me() {
	local p="$1" type="$2" kind="$3" me
	if [ -L "$p" ]; then
		refuse "The kept gitleaks $kind $p is a symlink; it is used only as a real $kind of this user's. Remove it to fetch CI's release again."
	fi
	me="$(id -u)"
	if [ -z "$(find "$p" -prune -type "$type" -user "$me" ! -perm -020 ! -perm -002 -print 2>/dev/null)" ]; then
		refuse "The kept gitleaks $kind $p is not a $kind owned by uid $me that only it can write. Remove it to fetch CI's release again."
	fi
}

# sha256_of prints the sha256 of file $1 with whichever tool the system has;
# pinned_gitleaks has refused already when there is neither.
sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

secrets() {
	local version def base rc=0 out leaks_cmd=""
	version="$(leaks_pin)"
	pinned_gitleaks "$version"
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
	# gitleaks exits 1 on a leak and on its own errors alike, so a leak is given
	# a code of its own.
	out="$("$leaks_cmd" git --redact --no-banner --exit-code 77 --log-opts="$scope" . 2>&1)" || rc=$?
	printf '%s\n' "$out"
	case "$rc" in
	0) return 0 ;;
	77)
		say "gitleaks found a secret in $scope; the preflight stops. Remove it from the history (rewrite the branch), and rotate it if it was ever pushed."
		return 1
		;;
	*)
		say "gitleaks could not scan $scope (exit $rc); the lines above say why. CI's gitleaks job, a required check, would most likely fail the same way, so the preflight stops."
		exit 2
		;;
	esac
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
