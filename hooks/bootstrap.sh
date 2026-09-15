#!/bin/sh
# abcd plugin bootstrap: provision $CLAUDE_PLUGIN_ROOT/abcd whenever it is
# missing (itd-105 / spc-21), preferring the persistent per-plugin download
# cache in $CLAUDE_PLUGIN_DATA over the network (itd-132 / spc-35).
#
# POSIX sh and the base system only: the abcd binary is exactly what is missing
# on a fresh plugin install and after every plugin update (the harness re-clones
# into a fresh commit-stamped cache directory), so nothing here may depend on it.
#
# The plugin ROOT is transient — every update replaces it and garbage-collection
# later deletes it — while the data dir survives updates and is deleted only on
# full uninstall. So the checksum-verified release artefact is kept ONCE in the
# data dir and each fresh root is provisioned from it by a re-verified copy; the
# network is asked for an artefact only when the released binary itself changed.

set -u

plugin_root="${CLAUDE_PLUGIN_ROOT:-}"
[ -n "$plugin_root" ] || exit 0

binary="$plugin_root/abcd"

# Every instruction printed below names the binary by this ABSOLUTE path, in
# SINGLE quotes — the same form internal/core/ahoy's shSingleQuote produces —
# because the strings are printed to be pasted into a shell by a reader for whom
# `abcd` is not a name the shell can resolve (iss-207), and the plugin root is
# not a path this script chose. Double quotes carry a space or an apostrophe
# safely and then leave $, a backtick and a " live: a path holding one of those
# would expand, substitute, or terminate the string on paste. The sed rewrites
# each embedded ' as '\'' — close, escape, reopen — which is the one form that
# survives every byte a path can contain.
binary_quoted="'$(printf '%s' "$binary" | sed "s/'/'\\\\''/g")'"

# HOME is a filesystem destination here, never a fetch origin: it locates the
# owned-copy provenance record (below) and renders user-scope paths in their
# tilde form. Empty when HOME is unset, which every use guards.
home_dir="${HOME:-}"

# The persistent data dir is taken from the harness or not at all. Its
# documented path shape could be derived from the plugin root, but a wrong
# guess would plant a trusted artefact in an untracked location, so the
# derivation is deliberately not attempted: when the variable is absent or the
# directory unusable, this script degrades LOUDLY to the spc-21 per-root fetch
# below — the success notice says so — and never falls back in silence.
data_dir="${CLAUDE_PLUGIN_DATA:-}"

# The fetch origins are constants of this script, and there is deliberately no
# way to redirect them from the environment. An override here is not a
# convenience, it is a code-execution primitive: the binary AND the
# checksums.txt that verifies it are fetched from the same base, so whoever
# names the origin supplies both the payload and the manifest that "verifies"
# it — verification becomes vacuous — and the binary installed below is then run
# unattended as the Bash shell guard on every tool call. Two successive attempts
# to keep the seam behind a loopback allowlist were both defeated in adversarial
# review (URL userinfo of the form http://127.0.0.1:1@example.invalid slipped
# through the glob, and clearing the transport pin on the override path reopened
# redirect downgrade), so the seam is REMOVED rather than narrowed a third time.
# The tests rewrite these two literals in a throwaway COPY of this file; nothing
# in the shipped script consults the environment to decide where to fetch from.
repo_url="https://github.com/intentdriven/abcd"
releases_url="$repo_url/releases"
api_url="https://api.github.com/repos/intentdriven/abcd"

# Every fetch below is HTTPS-only, redirects included, unconditionally: there is
# no toggle and no code path that clears it. Without the pin a redirect could
# downgrade the transport (or point at file://) and hand both the payload and the
# manifest that verifies it to whoever can rewrite a response.
#
# The pin is not enough on its own, because curl reads a configuration surface
# this script's argv knows nothing about. Unless -q is its FIRST argument, curl
# loads $CURL_HOME/.curlrc (falling back to $HOME/.curlrc), and one `connect-to`
# or `resolve` line there re-points the connection while the URL above still
# literally reads https://github.com/… — the transport pin still holds, and the
# checksum check becomes vacuous, because the same config supplies the binary AND
# the checksums.txt that "verifies" it. Independent review reproduced exactly
# that. So: -q first on every call below, and the other names curl reads without
# being told to — the proxy variables, which can route both fetches through a
# server of the setter's choosing, and the CA overrides, which are what make such
# a route succeed on TLS — are removed here, before any fetch happens. The cost
# is deliberate and accepted: a machine that can only reach the network through a
# proxy no longer bootstraps automatically, and refuse() already names the
# manual install and build-from-source ways out.
unset HTTPS_PROXY https_proxy HTTP_PROXY http_proxy ALL_PROXY all_proxy CURL_HOME
unset CURL_CA_BUNDLE SSL_CERT_FILE SSL_CERT_DIR

lock=''
tmp=''
root_tmp=''
path_tmp=''
auth_tmp=''

# staged records that provisioning BEGAN, and terminal that the run has already
# had its last word (a notice or a refusal). Together they are the contract this
# script owes a reader: once it starts provisioning, exactly one terminal line
# follows, always. The §4 fresh-machine gate failed because
# neither half held — on a cloud-synced home the run produced no output at all,
# neither success nor refusal, and every UserPromptSubmit and PreToolUse hook
# then failed as a raw "No such file or directory" for an entire evening with
# nothing anywhere saying why (iss-253). Silence is the defect these two names
# exist to make impossible.
staged=''
terminal=''

cleanup() {
	[ -n "$tmp" ] && rm -rf "$tmp"
	[ -n "$root_tmp" ] && rm -rf "$root_tmp"
	[ -n "$path_tmp" ] && rm -f "$path_tmp"
	[ -n "$auth_tmp" ] && rm -rf "$auth_tmp"
	[ -n "$lock" ] && rm -rf "$lock"
	return 0
}

# safe strips the control characters a message must never carry to a terminal.
# The values interpolated into the messages below are not this script's own text
# — a plugin-root or data-dir path, a release tag read off a redirect — and a
# raw escape sequence in one of them can recolour, reposition, or visually
# rewrite what the reader is shown. Tab and newline survive; the report is made
# of them.
safe() {
	printf '%s' "$1" | tr -d '\000-\010\013-\037\177'
}

# say_refusal is the single failure message every failing path shares: what is
# missing, what it costs, and the three ways out. A raw shell error ("No such
# file or directory") must never be the whole story a user gets. It is split
# from refuse() because the EXIT trap has to be able to say it WITHOUT calling
# exit from inside a trap.
say_refusal() {
	terminal=yes
	printf 'abcd bootstrap: %s\n\nThe abcd binary is not installed in the plugin root, so the abcd hooks cannot run and the shell-hazard guard is inactive — shell commands run UNGUARDED until it is.\n\nAny one of these fixes it:\n  - start a session with network access, and this script retries by itself;\n  - install the release binary by hand (%s#install) and copy it to %s;\n  - build from source for full trust: go build ./cmd/abcd, then copy the binary to %s.\n' \
		"$(safe "$1")" "$repo_url" "$(safe "$binary")" "$(safe "$binary")" >&2
}

refuse() {
	say_refusal "$1"
	exit 1
}

# notice is a reported condition rather than a fault: the install succeeded, or
# the platform has no released binary. It exits ZERO.
#
# It used to exit 2, on the belief that only a non-zero exit puts a SessionStart
# hook's stderr in front of the human. That belief is false and was measured:
# the harness renders a non-zero SessionStart as an opaque "startup hook error"
# banner followed by a truncated echo of the hooks.json command, and DROPS the
# stderr text. So the exit code did not surface the notice, it replaced the
# notice with an error — reporting a fault in abcd rather than the successful
# install it was announcing (iss-2608251011427187, following the same correction
# for `abcd hook session-start` in iss-2608241115201044).
#
# The half of the original reasoning that was RIGHT is kept, and it is the reason
# this function still writes to stderr rather than stdout: a SessionStart hook's
# stdout becomes model context. These messages interpolate a release tag and
# paths, and an adversarial review demonstrated a directive payload reaching
# context through exactly that channel when the Go hook briefly wrote its notices
# there. Nothing derived goes to stdout.
#
# refuse() keeps its non-zero exit. That split is the point: a genuine fault
# SHOULD raise an error banner, because something is wrong and the session should
# say so; a successful install should not.
notice() {
	terminal=yes
	printf '%s\n' "$(safe "$1")" >&2
	exit 0
}

# on_exit is the EXIT trap: it cleans up, and it converts a SILENT death into a
# refusal. Provisioning below raises `staged` and then finishes through notice()
# or refuse() — so reaching the end of the run with provisioning begun and no
# terminal line means the script died somewhere it does not know about (a signal, an unwritable filesystem faulting a command whose
# failure nothing checks, a killed pipeline). That is precisely the §4 failure
# mode, and the only thing worse than failing is failing quietly: the reader is
# left with hooks that error "No such file or directory" on every prompt and
# nothing that names the cause. A death is reported with the same three ways out
# every other refusal carries. SIGKILL still runs no trap; nothing in a shell
# can cover that.
on_exit() {
	cleanup
	if [ -n "$staged" ] && [ -z "$terminal" ]; then
		say_refusal 'provisioning the abcd binary for this plugin root ended without installing it and without reporting why — the run was interrupted, or a step failed in a way this script could not see'
	fi
	return 0
}

# sha256_file prints the lowercase SHA-256 of the file named by $1, or nothing
# when no hasher is available or the file cannot be read.
sha256_file() {
	if command -v shasum >/dev/null 2>&1; then
		sum_line=$(shasum -a 256 "$1" 2>/dev/null)
	elif command -v sha256sum >/dev/null 2>&1; then
		sum_line=$(sha256sum "$1" 2>/dev/null)
	else
		sum_line=''
	fi
	printf '%s\n' "$sum_line" | sed -n 's/^\([0-9a-fA-F]\{64\}\).*/\1/p' | tr 'ABCDEF' 'abcdef'
}

# meta_field prints the first value recorded for key $2 in the key=value file
# $1. Control characters are stripped and the value capped, so a corrupted
# record can neither smuggle terminal control bytes into a message nor push an
# unbounded value into a rewritten record.
meta_field() {
	sed -n "s/^$2=//p" "$1" 2>/dev/null | head -n 1 | tr -d '\000-\037\177' | cut -c1-120
}

# tilde renders a path under the home directory in its ~ form, the way every
# abcd surface prints a user-scope path (internal/fsutil.RedactHome): a notice
# that names the reader's PATH copy should not carry their username with it.
# Anything else, and everything when HOME is unset, is printed as it is.
tilde() {
	if [ -n "$home_dir" ] && [ "$1" = "$home_dir" ]; then
		printf '~'
	elif [ -n "$home_dir" ] && [ "${1#"$home_dir"/}" != "$1" ]; then
		# The literal ~ is the output wanted, not an expansion left undone.
		# shellcheck disable=SC2088
		printf '~/%s' "${1#"$home_dir"/}"
	else
		printf '%s' "$1"
	fi
}

# 1. Fast path: a steady-state session pays a file test and no network. A
#    regular file, not merely something with an execute bit — a directory is
#    executable too (matching internal/core/ahoy's isExecutableFile).
if [ -f "$binary" ] && [ -x "$binary" ]; then
	# Migration (spc-35, one-way): a root provisioned BEFORE the cache existed
	# holds a verified binary and its .binary-meta; the first run with an empty
	# cache seeds the cache from it, so the next plugin update is served by
	# copy instead of a re-download. Everything here is best-effort and silent
	# — this session's binary is already in place — except the verification:
	# the seed is a promotion into the trusted location, so the binary is
	# re-hashed against the record it claims, and a mismatch seeds nothing.
	# Roots provisioned from the cache carry no .binary-meta, so for them this
	# adds one file test and nothing else.
	[ -n "$data_dir" ] || exit 0
	root_meta="$plugin_root/.binary-meta"
	[ -f "$root_meta" ] || exit 0
	mig_os=$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')
	mig_arch=$(uname -m 2>/dev/null)
	case "$mig_arch" in
		x86_64 | amd64) mig_arch=amd64 ;;
		aarch64 | arm64) mig_arch=arm64 ;;
		*) exit 0 ;;
	esac
	case "$mig_os" in
		darwin | linux) ;;
		*) exit 0 ;;
	esac
	cache_dir="$data_dir/cache"
	cache_binary="$cache_dir/abcd-$mig_os-$mig_arch"
	# `mv -f` onto a DIRECTORY moves the seed INTO it rather than replacing it,
	# and a lying binary-meta then vouches for a directory — every later root
	# then downloads and refuses, running the shell guard UNGUARDED every
	# session until a human removes the directory by hand, and it survives every
	# plugin update (iss-2608210934566229). The main install site refuses this
	# non-regular-file shape (below); the migration seed must too. The fast-path
	# `[ ! -f ]` test alone is TRUE for a directory, so it cannot stand in for it.
	if [ -e "$cache_binary" ] && [ ! -f "$cache_binary" ]; then
		refuse "$cache_binary exists and is not a regular file, so the plugin cache cannot be seeded from this plugin root"
	fi
	[ ! -f "$cache_binary" ] || exit 0
	want=$(meta_field "$root_meta" binary_sha256)
	case "$want" in *[!0-9a-f]*) exit 0 ;; esac
	[ "${#want}" -eq 64 ] || exit 0
	mkdir -p "$cache_dir" 2>/dev/null || exit 0
	# The shared-cache lock (below) is taken, never broken, and lost quietly:
	# a stale lock only defers the seed to a later session, and the main path
	# owns stale-lock recovery.
	lock="$data_dir/.bootstrap.lock"
	if ! mkdir "$lock" 2>/dev/null; then
		lock=''
		exit 0
	fi
	trap on_exit EXIT
	trap 'on_exit; exit 1' HUP INT TERM
	[ ! -f "$cache_binary" ] || exit 0
	tmp="$data_dir/.bootstrap.tmp.$$"
	rm -rf "$tmp"
	mkdir "$tmp" 2>/dev/null || exit 0
	cp "$binary" "$tmp/artefact" 2>/dev/null || exit 0
	got=$(sha256_file "$tmp/artefact")
	if [ "$got" != "$want" ]; then
		# A mismatch seeds nothing — and says so. The seed is a promotion into
		# the trusted location, and itd-132 ac-5 promises every promotion
		# refuses LOUDLY on a mismatch; a silent exit here left a root whose
		# binary had been swapped under its record with no signal at all
		# (iss-2609012111160075). It is a notice, not a refusal: this session's
		# binary is already in place, so the hooks run, and nothing changed.
		# There is no once-per-state memory to draw on, so the line repeats
		# each session the state holds — which is also exactly how long this
		# copy-and-hash was already being repeated for nothing; the state ends
		# at the next plugin update, whose fresh root downloads and fills the
		# cache, or when the stale .binary-meta is removed from this root.
		if [ -n "$got" ]; then
			seed_why='does not match the SHA-256 checksum its provenance record (.binary-meta) vouches for — the binary was replaced after it was verified, or the record is stale'
		else
			seed_why='cannot be hashed, because neither shasum nor sha256sum is available, so it cannot be verified against its provenance record'
		fi
		notice "$(printf 'abcd bootstrap: the plugin cache was not seeded from this plugin root, because the binary at %s %s. Nothing was changed: this session runs the binary already in place, and the next plugin update downloads a fresh checksum-verified copy instead of promoting this one. %s ahoy shows the health of the install.' \
			"$(tilde "$binary")" "$seed_why" "$binary_quoted")"
	fi
	mig_tag=$(meta_field "$root_meta" release_tag)
	mig_sha=$(meta_field "$root_meta" release_sha)
	case "$mig_sha" in *[!0-9a-f]*) mig_sha=unknown ;; esac
	[ "${#mig_sha}" -eq 40 ] || mig_sha=unknown
	mig_at=$(meta_field "$root_meta" fetched_at)
	# The cache meta is the spc-21 record minus plugin_sha: one cache serves
	# many roots, so a provisioning-time root is meaningless — the skew notice
	# compares the LIVE plugin root at render time.
	{
		printf 'release_tag=%s\n' "$mig_tag"
		printf 'release_sha=%s\n' "$mig_sha"
		printf 'binary_sha256=%s\n' "$want"
		printf 'fetched_at=%s\n' "$mig_at"
	} > "$tmp/binary-meta" 2>/dev/null || exit 0
	chmod 0755 "$tmp/artefact" 2>/dev/null || exit 0
	# Re-check the obstruction under the lock: a directory that appeared in the
	# window since the pre-lock guard must not swallow the seed either.
	if [ -e "$cache_binary" ] && [ ! -f "$cache_binary" ]; then
		refuse "$cache_binary exists and is not a regular file, so the plugin cache cannot be seeded from this plugin root"
	fi
	mv -f "$tmp/artefact" "$cache_binary" 2>/dev/null || exit 0
	mv -f "$tmp/binary-meta" "$cache_dir/binary-meta" 2>/dev/null
	exit 0
fi

# `mv -f` onto an existing DIRECTORY moves the file INTO it rather than
# replacing it, so a stray directory at $binary would otherwise "succeed"
# every session while the hooks stay broken.
if [ -e "$binary" ] && [ ! -f "$binary" ]; then
	refuse "$binary exists and is not a regular file, so the release binary cannot be installed there"
fi

# 2. Platform gate. An unsupported platform is a reported condition, not a hook
#    fault, so it changes nothing and never blocks — but it is still said out
#    loud, because the shell guard stays inactive for as long as it holds.
os=$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')
arch=$(uname -m 2>/dev/null)
case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) arch='' ;;
esac
case "$os" in
	darwin | linux) ;;
	*) os='' ;;
esac
if [ -z "$os" ] || [ -z "$arch" ]; then
	notice "$(printf 'abcd bootstrap: no abcd binary is released for this platform (%s %s). Released binaries cover darwin and linux on amd64 and arm64 only, so nothing was downloaded and nothing was changed. To run abcd here, build from source: go build ./cmd/abcd, then copy the binary to %s.' \
		"$(uname -s 2>/dev/null)" "$(uname -m 2>/dev/null)" "$binary")"
fi

# 3. Provisioning mode. The cache mode needs a usable data dir; anything else
#    is the spc-21 per-root fetch, with the degradation named in the success
#    notice (never a silent fallback).
asset="abcd-$os-$arch"
cache_mode=''
degrade_note=' The persistent plugin data directory is unavailable, so the binary was fetched into this plugin root only and will be re-downloaded after the next plugin update.'
cache_dir=''
cache_binary=''
cache_meta=''
if [ -n "$data_dir" ]; then
	cache_dir="$data_dir/cache"
	if mkdir -p "$cache_dir" 2>/dev/null; then
		cache_mode=yes
		degrade_note=''
		cache_binary="$cache_dir/$asset"
		cache_meta="$cache_dir/binary-meta"
	fi
fi

# The owned-copy provenance record is HOME-scoped, never in the data dir: a
# terminal (where `ahoy install`/`uninstall` and `abcd update` run) has HOME but
# not CLAUDE_PLUGIN_DATA, so a record behind that variable would be unreadable
# exactly where those verbs run (iss-2608210934566230, adr-46 decision 4).
# Empty when HOME is unset, which every use below guards.
path_entry=''
[ -n "$home_dir" ] && path_entry="$home_dir/.abcd/path-entry"

# 4. Concurrency lock. mkdir is atomic on POSIX, so the loser of the race is the
#    process whose mkdir fails; it exits quietly rather than racing the winner.
#    In cache mode the lock lives in the DATA dir, because per-root locks cannot
#    serialise two roots writing the one shared cache; degraded mode keeps the
#    per-root lock. A lock older than ten minutes belongs to a run that was
#    killed — without breaking it, provisioning stays blocked forever.
if [ -n "$cache_mode" ]; then
	lock="$data_dir/.bootstrap.lock"
else
	lock="$plugin_root/.bootstrap.lock"
fi
if ! mkdir "$lock" 2>/dev/null; then
	if [ -n "$(find "$lock" -maxdepth 0 -mmin +10 2>/dev/null)" ]; then
		rm -rf "$lock"
	fi
	if ! mkdir "$lock" 2>/dev/null; then
		# A lock DIRECTORY that exists is the race: another run holds it, and
		# staying quiet is right. mkdir failing with no lock directory there is
		# something else — a read-only or unwritable directory, or a
		# non-directory squatting the lock path — and that is a permanent,
		# every-session failure the two cases must not share a silent exit with.
		if [ -d "$lock" ]; then
			lock=''
			exit 0
		fi
		refuse "the bootstrap lock ($lock) cannot be created and no lock directory is there, so this is not a concurrent run — the lock's directory is not writable, or something that is not a directory occupies the lock path"
	fi
fi
# A signal trap that RETURNS resumes the script (POSIX), which would carry on
# against directories cleanup just deleted and report a checksum mismatch that
# never happened. Terminate explicitly; cleanup is idempotent, so the EXIT trap
# firing again after it is a no-op.
trap on_exit EXIT
trap 'on_exit; exit 1' HUP INT TERM
# 4b. Provisioning starts here, and from here the run owes the reader exactly
#     one terminal line — a success notice or a refusal — whichever way it ends.
#     Everything below is the work that produced nothing at all on the §4
#     machine: a directory sweep of a plugin root holding a full source checkout
#     on a cloud-synced filesystem (anomalous stat results, 65535 link counts,
#     multi-minute tree walks), then three network fetches. The EXIT trap is what
#     closes the contract when one of them kills the run.
#
#     There is deliberately NO eager "provisioning…" line printed here, loud as
#     that would be. Only the FIRST line of a hook's stderr reaches the
#     transcript (iss-208, measured on the first manual install), and this
#     script's success notice already spends that line on the one-time
#     `ahoy install` instruction, placed first for exactly that reason
#     (iss-207). An announcement ahead of it would take the line from the
#     success and — far worse — from the REFUSAL's reason on the failing path,
#     which is precisely the silence itd-154 exists to end: the reader would
#     learn that provisioning started and never learn why it did not finish. The
#     announcement is therefore held and spent only where it is the only thing a
#     reader would otherwise have: on_exit's report of a run that died without a
#     word of its own.
#
#     The flag is raised after the lock rather than before it so a session that
#     merely lost the race is not reported as a failed provision: it never
#     provisioned at all.
staged=yes

# SIGKILL runs no trap, so a killed run leaves its PID-stamped temp directory
# behind holding a partially downloaded, UNVERIFIED binary. The lock is held from
# here on, so sweeping them cannot touch a live run's directory.
rm -rf "$plugin_root"/.bootstrap.tmp.* 2>/dev/null
if [ -n "$cache_mode" ]; then
	rm -rf "$data_dir"/.bootstrap.tmp.* 2>/dev/null
	rm -rf "$data_dir"/.bootstrap.auth.* 2>/dev/null
fi

# 5. Refresh detector (spc-35): the recorded cache state plus ONE best-effort
#    resolve of the latest release tag decide whether the network is needed.
#    /releases/latest answers 302 -> /releases/tag/<tag>, and %{redirect_url} on
#    an UNFOLLOWED request is the only URL shape that survives: %{url_effective}
#    after -L lands on the asset CDN, which carries no tag segment at all. The
#    resolve doubles as spc-21 step 4: any download below is pinned to the ONE
#    resolved release, so the binary and the manifest that verifies it can never
#    come from two different releases.
cached_tag=''
cached_sha=''
if [ -n "$cache_mode" ] && [ -f "$cache_binary" ]; then
	cached_tag=$(meta_field "$cache_meta" release_tag)
	cached_sha=$(meta_field "$cache_meta" binary_sha256)
	case "$cached_sha" in *[!0-9a-f]*) cached_sha='' ;; esac
	[ "${#cached_sha}" -eq 64 ] || cached_sha=''
	# An artefact with no verifiable recorded hash is not a cache hit: nothing
	# can be promoted out of it, so it is treated as absent and re-fetched.
	[ -n "$cached_sha" ] || cached_tag=''
fi

resolved_tag=''
if command -v curl >/dev/null 2>&1; then
	redirect=$(curl -q -fsS --proto '=https' --proto-redir '=https' --max-time 15 -o /dev/null -w '%{redirect_url}' "$releases_url/latest" 2>/dev/null) || redirect=''
	resolved_tag=$(printf '%s\n' "$redirect" | sed -n 's|.*/releases/tag/\([^/?#]*\).*|\1|p')
fi

# Decide whether the cache is trusted, and on what evidence. A cached artefact
# and its co-located binary-meta are equally same-UID-writable, so re-hashing
# the artefact against that record proves only corruption, never tamper: an
# attacker who writes BOTH satisfies it, and because the cache is now preferred
# over the network across every update, the implant persists rather than healing
# at the next update (adr-46 decision 3, iss-2608210934566228).
#
#   - Resolve fails (offline): no published manifest is reachable, so the cache
#     is trusted at CORRUPTION-EVIDENCE ONLY (re-hash below). The success notice
#     says so — an unauthenticated cache while offline — never claiming a
#     verification it did not perform. The deliberate availability trade.
#   - Resolve answers the cached tag (online): AUTHENTICATE the cached hash
#     against the release's PUBLISHED checksums.txt (the same same-origin,
#     HTTPS-pinned, -q-first manifest fetch spc-21 step 5 uses) before trusting
#     the cache. Match -> provision from cache, manifest-verified. Mismatch ->
#     the cache is tampered or stale: discard it and fall to the download path.
#     Manifest fetch fails though the tag resolved -> treat as offline.
#   - Resolve answers a different tag -> download path.
# The accepted gap: a release cut with no plugin update never triggers a fetch
# here — the version-skew notice surfaces it, `abcd update` is the explicit path.
use_cache=''
cache_trust=''
stale_note=''
if [ -n "$cached_sha" ]; then
	if [ -z "$resolved_tag" ]; then
		use_cache=yes
		cache_trust=offline
	elif [ "$resolved_tag" = "$cached_tag" ]; then
		auth_tmp="$data_dir/.bootstrap.auth.$$"
		rm -rf "$auth_tmp"
		if command -v curl >/dev/null 2>&1 && mkdir "$auth_tmp" 2>/dev/null &&
			curl -q -fsSL --proto '=https' --proto-redir '=https' --max-time 30 -o "$auth_tmp/checksums.txt" "$releases_url/download/$resolved_tag/checksums.txt" 2>/dev/null; then
			published=$(grep " $asset\$" "$auth_tmp/checksums.txt" 2>/dev/null | head -n 1 | sed -n 's/^\([0-9a-fA-F]\{64\}\).*/\1/p' | tr 'ABCDEF' 'abcdef')
			rm -rf "$auth_tmp"
			auth_tmp=''
			if [ -n "$published" ] && [ "$published" = "$cached_sha" ]; then
				use_cache=yes
				cache_trust=manifest
			else
				# Mismatch or an unlisted asset leaves use_cache empty: the cache
				# is tampered or stale, so it is discarded and the download path
				# below re-fetches and re-verifies over it — and the notice says
				# the cache was discarded, because a re-download that silently
				# heals over a tampered artefact erases the one piece of evidence
				# the reader had (iss-2609012111160075).
				stale_note=' The cached artefact was not the one the published release manifest lists, so it was discarded and replaced by this fresh verified download.'
			fi
		else
			# The tag resolved but the manifest is unreachable: treat as offline.
			rm -rf "$auth_tmp"
			auth_tmp=''
			use_cache=yes
			cache_trust=offline
		fi
	fi
fi

meta_note=''
cache_note=''
path_note=''
from_note=''
stamp_note=''

if [ -n "$use_cache" ]; then
	release_tag="$cached_tag"
	expected_sha="$cached_sha"
	# The notice names the trust the install rests on, never more (adr-46
	# decision 3): a manifest-verified cache says so; an offline one says it was
	# provisioned from an unauthenticated cache, so the reader is never told a
	# verification happened that did not.
	if [ "$cache_trust" = manifest ]; then
		from_note=' No download was needed: the cached artefact was verified against the published release manifest.'
	else
		from_note=' No download was needed: the artefact was provisioned from an unauthenticated cache while offline.'
	fi
else
	command -v curl >/dev/null 2>&1 ||
		refuse 'curl is not available, so the release binary cannot be downloaded'
	[ -n "$resolved_tag" ] ||
		refuse 'the latest release tag could not be resolved, so the download cannot be pinned to a single release — there may be no network'
	release_tag="$resolved_tag"

	# 6. Download into the mode's temp dir — the data dir in cache mode (same
	#    filesystem as the cache, so publishing into it is a rename), the plugin
	#    root otherwise (same filesystem as the install, ditto).
	download_url="$releases_url/download/$release_tag"
	if [ -n "$cache_mode" ]; then
		tmp="$data_dir/.bootstrap.tmp.$$"
	else
		tmp="$plugin_root/.bootstrap.tmp.$$"
	fi
	rm -rf "$tmp"
	# `mkdir -p` succeeds on a directory that already exists — including one
	# that reappeared, symlinked, in the window since the rm -rf above. Plain
	# `mkdir` fails on that, turning a same-name race into a refusal instead of
	# a curl write through a planted symlink.
	mkdir "$tmp" 2>/dev/null ||
		refuse "a temporary directory cannot be created at $tmp"

	curl -q -fsSL --proto '=https' --proto-redir '=https' --max-time 120 -o "$tmp/$asset" "$download_url/$asset" 2>/dev/null ||
		refuse "downloading $asset from release $release_tag failed — there may be no network, or that release may carry no asset for this platform"
	curl -q -fsSL --proto '=https' --proto-redir '=https' --max-time 30 -o "$tmp/checksums.txt" "$download_url/checksums.txt" 2>/dev/null ||
		refuse "downloading checksums.txt from release $release_tag failed, so the download cannot be verified and is not installed"

	# 7. Verification against the same-origin manifest.
	line=$(grep " $asset\$" "$tmp/checksums.txt" 2>/dev/null | head -n 1)
	[ -n "$line" ] ||
		refuse "the release checksums.txt lists no entry for $asset, so the download cannot be verified and is not installed"
	printf '%s\n' "$line" > "$tmp/manifest.txt"
	if command -v shasum >/dev/null 2>&1; then
		verify='shasum -a 256 -c manifest.txt'
	elif command -v sha256sum >/dev/null 2>&1; then
		verify='sha256sum -c manifest.txt'
	else
		refuse 'neither shasum nor sha256sum is available, so the download cannot be verified and is not installed'
	fi
	(cd "$tmp" && $verify) > /dev/null 2>&1 ||
		refuse "the downloaded $asset does not match its SHA-256 checksum in the release checksums.txt — the artefact is corrupted or is not the published one, so nothing was installed"

	# The hash just verified is the provenance every later PROMOTION of this
	# artefact re-checks: cache -> plugin root, cache -> PATH copy, and the
	# migration seed all compare against it and refuse on a mismatch. The fast
	# path deliberately does not recompute it — its whole contract is that a
	# steady-state session pays a file test and no more.
	binary_sha256=$(printf '%s\n' "$line" | sed -n 's/^\([0-9a-fA-F]\{64\}\).*/\1/p' | tr 'ABCDEF' 'abcdef')
	[ -n "$binary_sha256" ] || binary_sha256=unknown
	expected_sha="$binary_sha256"

	# The release commit is read from the API when it answers, and left unknown
	# otherwise: the meta file is the skew notice's only evidence, so it never
	# records a value it did not resolve.
	release_sha=unknown
	body=$(curl -q -fsSL --proto '=https' --proto-redir '=https' --max-time 15 -H 'Accept: application/vnd.github+json' "$api_url/commits/$release_tag" 2>/dev/null) || body=''
	candidate=$(printf '%s\n' "$body" | tr ',' '\n' |
		sed -n 's/.*"sha"[[:space:]]*:[[:space:]]*"\([0-9a-f]\{40\}\)".*/\1/p' | head -n 1)
	[ -n "$candidate" ] && release_sha="$candidate"

	# 8. Publish into the cache (cache mode): artefact then meta, each by rename
	#    on the cache's own filesystem. A hash that failed to parse cannot be
	#    recorded as provenance, so nothing is cached for it and this run
	#    behaves as the degraded per-root install below.
	if [ -n "$cache_mode" ] && [ "$binary_sha256" != unknown ]; then
		if [ -e "$cache_binary" ] && [ ! -f "$cache_binary" ]; then
			refuse "$cache_binary exists and is not a regular file, so the verified artefact cannot be cached or installed"
		fi
		{
			printf 'release_tag=%s\n' "$release_tag"
			printf 'release_sha=%s\n' "$release_sha"
			printf 'binary_sha256=%s\n' "$binary_sha256"
			printf 'fetched_at=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
		} > "$tmp/binary-meta" 2>/dev/null
		chmod 0755 "$tmp/$asset" 2>/dev/null ||
			refuse "the downloaded $asset cannot be made executable"
		mv -f "$tmp/$asset" "$cache_binary" 2>/dev/null ||
			refuse "the verified $asset cannot be published into the plugin cache at $cache_binary"
		if [ -e "$cache_meta" ] && [ ! -f "$cache_meta" ]; then
			cache_note=' (the cache provenance record could not be written because its path is occupied by something that is not a regular file, so the next update may re-download)'
		elif ! mv -f "$tmp/binary-meta" "$cache_meta" 2>/dev/null; then
			cache_note=' (the cache provenance record could not be written, so the next update may re-download)'
		fi

		# Refresh the abcd-owned PATH copy in the same run: a NEW release just
		# landed in the cache, and the copy `ahoy install` recorded would
		# otherwise serve the old one until someone re-ran it. Only a REGULAR FILE
		# that still matches the recorded provenance hash is ever replaced —
		# anything else (a symlink, a mismatching file, or NOTHING at the recorded
		# path) is not a matching owned copy, so it is left alone: the record
		# vouches for the bytes it recorded, and an absent path is not those bytes,
		# so creating one there would claim an ownership the record cannot prove.
		# The replacement is re-verified before the rename, like every promotion.
		#
		# Leaving the copy alone is right; doing it in silence was not
		# (iss-2609012111160075): the reader's abcd then quietly serves a release
		# older than the one just installed, with no signal until `abcd update`
		# or `ahoy` calls it foreign. So every branch that declines names what it
		# left untouched and why, on the success notice — which is one line, and
		# fires once per NEW release: a cache hit never reaches this block.
		if [ -n "$path_entry" ] && [ -f "$path_entry" ]; then
			entry_path=$(sed -n 's/^path=//p' "$path_entry" 2>/dev/null | head -n 1 | tr -d '\000-\037\177')
			entry_sha=$(meta_field "$path_entry" binary_sha256)
			refresh_ok=yes
			# A record with no usable path or hash vouches for nothing, so there
			# is nothing to refresh and no copy to report against; `ahoy` shows
			# the record's own health.
			[ -n "$entry_path" ] || refresh_ok=''
			case "$entry_sha" in *[!0-9a-f]*) refresh_ok='' ;; esac
			[ "${#entry_sha}" -eq 64 ] || refresh_ok=''
			shown_path=$(tilde "$entry_path")
			path_tail=" — $binary_quoted ahoy shows the health of the install."
			if [ -n "$refresh_ok" ]; then
				# Ownership-check unconditionally: an absent recorded path is NOT
				# an owned copy, so it never short-circuits to "refresh it".
				if [ -f "$entry_path" ] && [ ! -L "$entry_path" ]; then
					cur_sha=$(sha256_file "$entry_path")
					if [ "$cur_sha" != "$entry_sha" ]; then
						refresh_ok=''
						path_note=" The abcd command on your PATH ($shown_path) was NOT refreshed: it no longer matches the provenance abcd recorded for it, so it is not abcd's to replace and was left untouched, still serving whatever it holds now$path_tail"
					fi
				else
					refresh_ok=''
					path_note=" The abcd command on your PATH ($shown_path) was NOT refreshed: the recorded path no longer holds a regular file, so there is no owned copy to replace and nothing was written there$path_tail"
				fi
			fi
			if [ -n "$refresh_ok" ]; then
				path_tmp="$(dirname "$entry_path")/.abcd-path-refresh.$$"
				rm -f "$path_tmp"
				# Assumed failed until the rename lands: the copy beside it could
				# not be written, or did not verify, and the old copy stands.
				path_note=" The abcd command on your PATH ($shown_path) was NOT refreshed: the replacement copy could not be written beside it or did not verify against the release checksum, so the copy in place still serves the previous release$path_tail"
				if cp "$cache_binary" "$path_tmp" 2>/dev/null; then
					new_sha=$(sha256_file "$path_tmp")
					if [ "$new_sha" = "$binary_sha256" ] &&
						chmod 0755 "$path_tmp" 2>/dev/null &&
						mv -f "$path_tmp" "$entry_path" 2>/dev/null; then
						path_tmp=''
						# Re-record path + the refreshed hash + the LIVE plugin
						# root: the record's plugin_root is the terminal's route
						# home, and this fresh root replaced the one it named.
						{
							printf 'path=%s\n' "$entry_path"
							printf 'binary_sha256=%s\n' "$new_sha"
							printf 'plugin_root=%s\n' "$plugin_root"
						} > "$tmp/path-entry" 2>/dev/null &&
							mkdir -p "$(dirname "$path_entry")" 2>/dev/null &&
							mv -f "$tmp/path-entry" "$path_entry" 2>/dev/null
						path_note=' The abcd command on your PATH was refreshed to the same release.'
					else
						rm -f "$path_tmp"
						path_tmp=''
					fi
				fi
			fi
		fi
	fi
fi

# 9. Install into the plugin root.
if [ -n "$cache_mode" ] && { [ -n "$use_cache" ] || [ "$expected_sha" != unknown ]; }; then
	# Verified promotion out of the cache: the artefact is COPIED into a temp
	# dir on the plugin root's filesystem, the COPY is hashed against the
	# recorded provenance — one hash covers cache corruption and copy
	# corruption both — and only a match is renamed into place. A mismatch
	# refuses loudly and installs nothing: a tampered or bit-rotted cache must
	# never reach the guard path, and the evidence is left in place, named.
	root_tmp="$plugin_root/.bootstrap.tmp.$$.root"
	rm -rf "$root_tmp"
	mkdir "$root_tmp" 2>/dev/null ||
		refuse "a temporary directory cannot be created in the plugin root ($plugin_root)"
	cp "$cache_binary" "$root_tmp/abcd" 2>/dev/null ||
		refuse "the cached release artefact at $cache_binary cannot be read"
	got_sha=$(sha256_file "$root_tmp/abcd")
	[ -n "$got_sha" ] ||
		refuse 'neither shasum nor sha256sum is available, so the cached artefact cannot be verified and is not installed'
	[ "$got_sha" = "$expected_sha" ] ||
		refuse "the cached artefact at $cache_binary does not match its recorded SHA-256 checksum, so it was refused and nothing was installed — remove that file and start a session with network access to fetch a fresh verified copy"
	chmod 0755 "$root_tmp/abcd" 2>/dev/null ||
		refuse "the verified binary cannot be made executable"
	mv -f "$root_tmp/abcd" "$binary" 2>/dev/null ||
		refuse "the verified binary cannot be installed at $binary"

	# Record, beside the binary, the data dir this root was provisioned from.
	# CLAUDE_PLUGIN_DATA reaches hook processes only, and the one-time `ahoy
	# install` the notice below asks for runs from a terminal, where it is
	# unset: without this stamp that run finds no cache and degrades to a
	# symlink into this root, which dies at the next plugin update — the
	# very shape the cache replaced (iss-2609012111168716). The stamp is a
	# route, never a trust claim: the reader follows it only to an existing
	# absolute directory and re-hashes the artefact against the cache's
	# recorded binary_sha256 before any copy, like every promotion. Written
	# into the same temp dir as the binary and renamed in; a directory at the
	# stamp path would swallow the rename, so it is reported instead.
	stamp_path="$plugin_root/.data-dir"
	if [ -e "$stamp_path" ] && [ ! -f "$stamp_path" ]; then
		stamp_note=' (the .data-dir record could not be written because that path exists and is not a regular file, so `ahoy install` run from a terminal will not find the plugin cache)'
	else
		printf 'data_dir=%s\n' "$data_dir" > "$root_tmp/data-dir" 2>/dev/null &&
			mv -f "$root_tmp/data-dir" "$stamp_path" 2>/dev/null ||
			stamp_note=' (the .data-dir record could not be written, so `ahoy install` run from a terminal will not find the plugin cache)'
	fi
	rm -rf "$root_tmp"
	root_tmp=''
else
	# Degraded install (no usable data dir, or a hash that failed to parse):
	# the spc-21 per-root path, verbatim — the artefact was verified in a temp
	# dir on this same filesystem, so the install is one rename.
	chmod 0755 "$tmp/$asset" 2>/dev/null ||
		refuse "the downloaded $asset cannot be made executable"
	mv -f "$tmp/$asset" "$binary" 2>/dev/null ||
		refuse "the verified $asset cannot be installed at $binary"

	# No data dir provisioned this root, so no stamp may say one did: a stale
	# record from an earlier cache-mode provision of the same root would route
	# a terminal `ahoy install` to a cache this binary did not come from.
	rm -f "$plugin_root/.data-dir" 2>/dev/null

	# The root-local provenance record, written only on this path: cache-mode
	# roots read the cache meta instead, and the skew notice compares the LIVE
	# plugin root — so plugin_sha, a provisioning-time snapshot of the root, is
	# recorded nowhere any more.
	#
	# Written into the temp dir and renamed in, on the same filesystem, for the
	# same reason the binary is: a crash mid-write would otherwise leave a
	# truncated value that still parses, and a skew notice rendered off a
	# truncated commit is a lie.
	meta_path="$plugin_root/.binary-meta"
	if [ -e "$meta_path" ] && [ ! -f "$meta_path" ]; then
		# The same `mv -f` hazard the binary path refuses above, and quieter
		# here: a DIRECTORY at this path swallows the record as
		# .binary-meta/binary-meta, where nothing reads it, while the notice
		# below would claim provenance was recorded. The install itself is
		# genuine and is not undone for this — it is reported instead.
		meta_note=' (the .binary-meta provenance record could not be written because that path exists and is not a regular file, so version-skew reporting stays silent for this plugin root)'
	else
		{
			printf 'release_tag=%s\n' "$release_tag"
			printf 'release_sha=%s\n' "$release_sha"
			printf 'binary_sha256=%s\n' "$binary_sha256"
			printf 'fetched_at=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
		} > "$tmp/binary-meta" 2>/dev/null &&
			mv -f "$tmp/binary-meta" "$meta_path" 2>/dev/null ||
			meta_note=' (the .binary-meta provenance record could not be written, so version-skew reporting stays silent for this plugin root)'
	fi
fi

# Re-stamp the owned-copy record's plugin_root to THIS root, on every provision.
# The record's plugin_root is the route home the terminal uses to reach the
# plugin root (the old PATH symlink's job); a plugin root is a commit-stamped
# cache dir the harness replaces on update, so the recorded one goes stale at
# every update, and the hook — the one context with CLAUDE_PLUGIN_ROOT — is what
# keeps it current. Only an existing, well-formed record is rewritten (this
# re-stamps provenance, never creates it); path + hash are preserved verbatim.
# Runs for cache hits too, where the PATH-copy refresh above did not fire.
if [ -n "$path_entry" ] && [ -f "$path_entry" ]; then
	rec_path=$(sed -n 's/^path=//p' "$path_entry" 2>/dev/null | head -n 1 | tr -d '\000-\037\177')
	rec_sha=$(meta_field "$path_entry" binary_sha256)
	rec_ok=yes
	[ -n "$rec_path" ] || rec_ok=''
	case "$rec_sha" in *[!0-9a-f]*) rec_ok='' ;; esac
	[ "${#rec_sha}" -eq 64 ] || rec_ok=''
	if [ -n "$rec_ok" ]; then
		rec_tmp="$path_entry.rewrite.$$"
		{
			printf 'path=%s\n' "$rec_path"
			printf 'binary_sha256=%s\n' "$rec_sha"
			printf 'plugin_root=%s\n' "$plugin_root"
		} > "$rec_tmp" 2>/dev/null &&
			mv -f "$rec_tmp" "$path_entry" 2>/dev/null ||
			rm -f "$rec_tmp" 2>/dev/null
	fi
fi

# The one place PATH setup is suggested; the PATH entry itself stays owned by
# ahoy. This prints once per plugin root, because every later session takes the
# fast path above.
#
# The instruction names the binary by the ABSOLUTE path this script already
# holds, and does so for a reason worth stating: the sentence is addressed to a
# reader for whom `abcd` is not a name the shell can resolve — that is the whole
# condition it exists to fix — so an instruction reading `abcd ahoy install`
# fails with "command not found" for everyone who needs it. On the first manual
# install the agent that read it invented a `go run` incantation into the
# harness's plugin cache instead (iss-207). $binary is resolvable right now, by
# anyone, with no toolchain. Keep the invocation LAST on the line so it stays
# copy-pasteable, and keep the success leading the first word: only the first
# line of a hook's stderr reaches the transcript.
#
# The path is wrapped in SINGLE quotes (binary_quoted, defined at the top) for
# the reason given there: this string is printed to be pasted into a shell.
notice "$(printf 'abcd bootstrap: installed the checksum-verified abcd binary (release %s) into the plugin root, so the abcd hooks are live for this session.%s%s%s%s%s%s%s For the abcd command in your own terminal, run this once — the path is absolute because abcd is not on your PATH yet, which is exactly what the command fixes: %s ahoy install' \
	"$release_tag" "$from_note" "$stale_note" "$path_note" "$meta_note" "$cache_note" "$stamp_note" "$degrade_note" "$binary_quoted")"
