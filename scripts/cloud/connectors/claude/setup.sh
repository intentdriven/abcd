#!/usr/bin/env bash
# Prepare a hosted Claude Code cloud session's machine for this repository
# (spc-2610120941441282 step 4, itd-2610120738075547 C1 and C11, decisions 8,
# 10, 17 and 18). The host runs it from the SessionStart hook in the committed
# .claude/settings.json, which every session in this repository reads, the
# person's own included. So its FIRST act is the connector's remote test: off
# the cloud it exits 0, silently, having run nothing (C11).
#
# In the cloud it runs as root (the container's user), and it is idempotent: a
# session that starts again finds each prerequisite in place and leaves it so.
#
#   1. a normal user `abcd`, with no sudo and no extra group, owning the
#      checkout. If the user cannot be made it says so and exits 1, before
#      anything else: every check runs as that user or not at all (decision 8).
#      Then the checkout is trusted for root in git's system configuration
#      (`git config --system --add safe.directory <checkout>`, once): from git
#      2.35.2 git refuses a repository another user owns, so without it every
#      git command run as root in the checkout, the host's own included, would
#      stop there;
#   1b. the proxy's CA bundle, readable by `abcd`. A hosted session sends
#      HTTPS through a proxy and names its CA bundle in SSL_CERT_FILE,
#      GIT_SSL_CAINFO, CURL_CA_BUNDLE, REQUESTS_CA_BUNDLE or
#      NODE_EXTRA_CA_CERTS, under root's home, which `abcd` cannot read. For
#      each of those that names a file `abcd` cannot read, root copies exactly
#      that bundle to <dir>/<VARIABLE>.pem, root-owned and 0444, in a
#      root-owned 0755 directory no one else can write (<dir> is
#      ABCD_CLOUD_CA_DIR, by default /usr/local/share/abcd/cloud-ca): a
#      temporary file in that directory renamed over the copy, never through a
#      symlink, and only when the content differs. The variable is pointed at
#      the copy for every command run here as `abcd` and, through
#      CLAUDE_ENV_FILE (once), for the session's later shells, root's
#      included, which is why `abcd` must never be able to write it. A bundle
#      `abcd` can read is left exactly as it is; verification is never turned
#      off and nothing else, the system's trust store included, is trusted in
#      its place;
#   2. gitleaks at the version .github/workflows/ci.yml pins (GITLEAKS_VERSION,
#      read from that file as data, so there is one pin), downloaded and
#      checksum-verified exactly as the CI job does, then installed to
#      /usr/local/bin;
#   3. as `abcd`: the toolchain go.mod declares, resolved through
#      scripts/pinned-toolchain.sh (so `go run ./cmd/abcd` works), and
#      core.hooksPath=.githooks in the checkout, so the committed hooks run;
#   3b. as `abcd`: abcd itself, built once from the checkout into its local
#      tier, .abcd/.work.local/cloud/bin/abcd (`go build`, rebuilt at every
#      session start, so it is the checkout's own as the session found it; a
#      session that changes abcd's own checks is judged by the abcd it started
#      with until the next start). The pre-push hook checks
#      every commit of a work-in-progress push with `abcd lint outbound`, and
#      through `go run` each check would build abcd inside the push, past the
#      push transport's idle timeout on a cold cache. The path is handed to the
#      session's later shells through the host's file of variables
#      (CLAUDE_ENV_FILE, which every later Bash command runs first):
#      ABCD_OUTBOUND_BIN for the attribution gate (scripts/check-attribution.sh)
#      and ABCD_BIN for the cloud scripts (scripts/cloud/abcd-bin.sh), each
#      written once;
#   4. the checkout's LOCAL author identity, from the session's forge token,
#      through the one primitive: `go run ./cmd/abcd reserve --holder --json`,
#      which prints {"login","id","noreply","source"}. user.name is the login
#      and user.email the no-reply address, written with `git config --local`
#      in the checkout and nowhere else (decisions 17 and 18).
#
#      The lookup runs as `abcd` through `runuser -u abcd --`, which is not a
#      login: the session's environment, the forge token in it included,
#      passes through unchanged but for HOME, SHELL, USER and LOGNAME. It runs
#      as `abcd` rather than as root because it builds and runs the checkout's
#      own code (nothing of the checkout runs as root but this script), because
#      the build lands in the cache every later command of the session reuses,
#      and because the identity it reads is the one the user that commits
#      sees. Its answer is captured, judged here, and written as `abcd`.
#
#      It sets NOTHING when the lookup fails, when its answer is not a valid
#      login and that login's own no-reply address, or when it named the
#      account from somewhere other than the token while the checkout's
#      identity is not a person's (the attribution gate's own judgement). Then
#      it prints one line saying the identity could not be taken from the
#      forge token, and the brief's first step stops before any work. The
#      identity the container came with is never printed: it is read only into
#      variables and judged by exit code, and the verb's own diagnostics are
#      discarded unread;
#   4b. the holder id this session's reservations and announcements carry
#      (iss-2610290809526033), from the same answer's holder_id. A hosted
#      session's home is not kept, so unless the environment's own settings
#      set ABCD_HOLDER_ID, which runuser passes through to abcd as it passes
#      the token, the lookup above mints a fresh id every session, and holds
#      made under an earlier session's id read as another holder's until they
#      lapse. The step says which: the id came from ABCD_HOLDER_ID, or it was
#      minted here, naming it so the person can copy it into the environment's
#      settings. It never chooses or writes an id itself, and a variable set
#      to anything but one id is named as such, never echoed; abcd refuses the
#      lookup under such a variable, so step 4 then sets no identity and says
#      the variable, not the token, is why;
#   5. one line per prerequisite, on standard output, which the host hands to
#      the session.
#
# Exit 0 every prerequisite is in place (or this is not a cloud session);
# 1 one could not be put in place (each is still attempted, but the user,
# without which nothing else is).
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# The place: off the cloud, nothing at all.
"$here/is-remote.sh" || exit 0

root="$(cd "$here/../../../.." && pwd)"
cd "$root"

USER_NAME=abcd
LOGIN_RE='^[A-Za-z0-9]([A-Za-z0-9-]{0,37}[A-Za-z0-9])?$'
failed=0

say() {
	printf 'setup: %s\n' "$1"
}
miss() {
	printf 'setup: NOT IN PLACE: %s\n' "$1"
	failed=1
}

# as_abcd runs a command as the normal user, environment kept (not a login),
# with the CA variables step 1b re-pointed at the user's own copies (ca_env,
# empty until then and when no copy was needed).
ca_env=()
as_abcd() {
	env ${ca_env[@]+"${ca_env[@]}"} runuser -u "$USER_NAME" -- "$@"
}

if [ "$(id -u)" != 0 ]; then
	say "STOPPING: the setup runs as root, as the host starts it, to make the normal user; this is uid $(id -u). Nothing was changed."
	exit 1
fi

# --- 1. The user ------------------------------------------------------------------

if id -u "$USER_NAME" >/dev/null 2>&1; then
	made="already exists"
else
	rc=0
	useradd -m -s /bin/bash "$USER_NAME" >/dev/null 2>&1 || rc=$?
	if [ "$rc" -ne 0 ] || ! id -u "$USER_NAME" >/dev/null 2>&1; then
		say "STOPPING: the normal user $USER_NAME could not be created (useradd exited $rc), so nothing can run as a user other than root; nothing else was set up."
		exit 1
	fi
	made="created"
fi
if ! chown -R "$USER_NAME:$USER_NAME" "$root"; then
	say "STOPPING: the checkout could not be given to the user $USER_NAME; nothing else was set up."
	exit 1
fi
say "user $USER_NAME ($made, no sudo) owns the checkout; run every later command as it (runuser -u $USER_NAME --)."

# The checkout now belongs to another user than root, which git refuses to
# work in as root unless the checkout is trusted. Read first, so it is added
# once however often the setup runs.
trusted="$(git config --system --get-all safe.directory 2>/dev/null)" || trusted=""
if grep -Fxq -- "$root" <<<"$trusted"; then
	say "git trusts the checkout for root (safe.directory, system configuration) already."
elif git config --system --add safe.directory "$root"; then
	say "git trusts the checkout for root (safe.directory, system configuration), so git run as root works in it."
else
	miss "git's system configuration could not be given the checkout as a safe.directory, so git run as root in it refuses it."
fi

# --- 1b. The proxy's CA bundle, readable by the user ----------------------------------

# In a hosted session HTTPS goes through a proxy, and the session names the
# proxy's CA bundle in the variables below. runuser passes them to abcd
# unchanged, but the bundle sits under root's home, which abcd cannot read, so
# every networked command run as abcd (git, gh, abcd, the toolchain download)
# would fail its certificate check. Where a variable names a file abcd cannot
# read, root copies exactly that bundle, read-only and root's own, into a
# directory only root can write (ABCD_CLOUD_CA_DIR, by default
# /usr/local/share/abcd/cloud-ca); the variable is re-pointed at that copy for
# the commands run here as abcd and, through CLAUDE_ENV_FILE, for the session's
# later shells, root's included. So the copy root's own TLS then trusts can never be
# written by abcd. Verification is never turned off, and nothing else is
# trusted in its place. A bundle abcd can already read is left exactly as it is.
CA_VARS="SSL_CERT_FILE GIT_SSL_CAINFO CURL_CA_BUNDLE REQUESTS_CA_BUNDLE NODE_EXTRA_CA_CERTS"
ca_dir="${ABCD_CLOUD_CA_DIR:-/usr/local/share/abcd/cloud-ca}"

# ca_dir_ready makes the copies' directory, root's own, mode 0755, and proves it
# is a real directory root owns that no one else can write. The path is
# absolute, so no argument below can read as an option.
ca_dir_ready() {
	case "$ca_dir" in /*) ;; *) return 1 ;; esac
	mkdir -p "$ca_dir" && [ -d "$ca_dir" ] && [ ! -L "$ca_dir" ] &&
		chown 0:0 "$ca_dir" && chmod 0755 "$ca_dir" && [ -O "$ca_dir" ] &&
		[ -z "$(find "$ca_dir" -maxdepth 0 \( -perm -020 -o -perm -002 \) -print)" ]
}

# ca_copy writes <src> to <dst> as root: a temporary file in the same
# directory, made 0444 and root's, then renamed over the destination. A
# destination that is a symlink or a directory is refused, never followed.
ca_copy() {
	local src="$1" dst="$2" tmp
	{ [ ! -L "$dst" ] && [ ! -d "$dst" ]; } || return 1
	tmp="$(mktemp "$ca_dir/.ca.XXXXXX")" || return 1
	if cat -- "$src" >"$tmp" && chown 0:0 "$tmp" && chmod 0444 "$tmp" &&
		{ [ ! -L "$dst" ] && [ ! -d "$dst" ]; } && mv -f -- "$tmp" "$dst"; then
		return 0
	fi
	/bin/rm -f -- "$tmp"
	return 1
}

ca_step() {
	local var src dst copied="" left="" line dir_ok=""
	for var in $CA_VARS; do
		src="${!var:-}"
		[ -n "$src" ] || continue
		# Not a file root can read either: the user is no worse off than root.
		if [ ! -f "$src" ] || [ ! -r "$src" ] || as_abcd test -r "$src" 2>/dev/null; then
			left="$left $var"
			continue
		fi
		dst="$ca_dir/$var.pem"
		case "$dst" in
		*"'"* | *$'\n'*)
			miss "the CA bundle $var names cannot be read by $USER_NAME, and $ca_dir cannot be written as a variable, so no copy was made; networked commands run as $USER_NAME fail their certificate check."
			return 0
			;;
		esac
		if [ -z "$dir_ok" ]; then
			if ! ca_dir_ready; then
				miss "the CA bundle $var names ($src) cannot be read by $USER_NAME, and $ca_dir could not be made a directory only root can write, so no copy was made; networked commands run as $USER_NAME fail their certificate check."
				return 0
			fi
			dir_ok=1
		fi
		# Rewritten only when it differs, or is not the plain file it must be.
		if [ -L "$dst" ] || [ ! -f "$dst" ] || ! cmp -s -- "$src" "$dst"; then
			if ! ca_copy "$src" "$dst" || ! cmp -s -- "$src" "$dst"; then
				miss "the CA bundle $var names ($src) cannot be read by $USER_NAME, and a copy could not be made at $dst; networked commands run as $USER_NAME fail their certificate check."
				return 0
			fi
		fi
		if ! as_abcd test -r "$dst" 2>/dev/null; then
			miss "the copy of the CA bundle $var names, $dst, cannot be read by $USER_NAME either; networked commands run as $USER_NAME fail their certificate check."
			return 0
		fi
		ca_env+=("$var=$dst")
		copied="$copied $var"
		if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
			line="export $var='$dst'"
			if ! grep -Fxq -- "$line" "$CLAUDE_ENV_FILE" 2>/dev/null; then
				printf '%s\n' "$line" >>"$CLAUDE_ENV_FILE" || {
					miss "a read-only copy of the CA bundle $var names is at $dst, but $var could not be written to the file of variables for later shells; point it there in each shell."
					return 0
				}
			fi
		fi
	done
	if [ -n "$copied" ]; then
		if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
			say "the CA bundle named by${copied} is not readable by $USER_NAME, so a read-only copy of exactly it, root's own, is in $ca_dir, and those variables point there for $USER_NAME's commands and later shells."
		else
			say "the CA bundle named by${copied} is not readable by $USER_NAME, so a read-only copy of exactly it, root's own, is in $ca_dir; no file of variables for later shells was given, so point those variables there in each shell."
		fi
	elif [ -n "$left" ]; then
		say "the CA bundle named by${left} is left as it is: $USER_NAME can read it, or root cannot either."
	else
		say "no CA bundle is named in the environment, so $USER_NAME's commands use the system's trust, as root's do."
	fi
}
ca_step

# --- 2. gitleaks ----------------------------------------------------------------------

gitleaks_step() {
	local pins version arch base url tmp have rc
	pins="$(sed -n 's/^[[:space:]]*GITLEAKS_VERSION:[[:space:]]*\([0-9][0-9.]*\)[[:space:]]*$/\1/p' .github/workflows/ci.yml 2>/dev/null | sort -u)"
	if [ -z "$pins" ] || [ "$(printf '%s\n' "$pins" | grep -c .)" -ne 1 ]; then
		miss "gitleaks: .github/workflows/ci.yml does not pin one GITLEAKS_VERSION, so there is no version to install."
		return 0
	fi
	version="$pins"
	have="$(gitleaks version 2>/dev/null | tr -d 'v[:space:]')" || have=""
	if [ "$have" = "$version" ]; then
		say "gitleaks $version (the version ci.yml pins) is already installed."
		return 0
	fi
	case "$(uname -m)" in
	x86_64 | amd64) arch=x64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*)
		miss "gitleaks: no release is published for this machine's architecture, $(uname -m)."
		return 0
		;;
	esac
	tmp="$(mktemp -d "${TMPDIR:-/tmp}/abcd-setup.XXXXXX")"
	base="gitleaks_${version}_linux_${arch}.tar.gz"
	url="https://github.com/gitleaks/gitleaks/releases/download/v${version}"
	# The CI job's own steps (ci.yml, job gitleaks), in a directory of our own.
	# Chained with &&, not left to set -e: errexit is off inside a subshell
	# whose status is tested, and an unverified download must never reach
	# the install.
	rc=0
	(
		set -o pipefail
		cd "$tmp" &&
			curl -sSLf -o "$base" "$url/$base" &&
			curl -sSLf -o checksums.txt "$url/gitleaks_${version}_checksums.txt" &&
			grep " $base\$" checksums.txt | sha256sum -c - >/dev/null &&
			tar -xzf "$base" gitleaks &&
			install gitleaks /usr/local/bin/gitleaks
	) >/dev/null 2>&1 || rc=$?
	/bin/rm -rf -- "$tmp"
	if [ "$rc" -ne 0 ]; then
		miss "gitleaks $version: the download, its checksum or the install failed (exit $rc); nothing unverified was installed."
		return 0
	fi
	say "gitleaks $version (the version ci.yml pins) installed, checksum verified."
}
gitleaks_step

# --- 3. The toolchain and the hooks, as the user ------------------------------------------

version="$(sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' go.mod | head -n 1)"
if [ -z "$version" ]; then
	miss "the toolchain: go.mod declares no go <version> line."
elif goroot="$(as_abcd bash "$root/scripts/pinned-toolchain.sh" "$version" 2>/dev/null)"; then
	say "toolchain go$version (the one go.mod declares) resolved for $USER_NAME at $goroot."
else
	miss "the toolchain go$version could not be resolved for $USER_NAME (scripts/pinned-toolchain.sh $version says why)."
fi

if as_abcd git -C "$root" config --local core.hooksPath .githooks; then
	say "core.hooksPath=.githooks is set in the checkout, so the committed hooks run."
else
	miss "core.hooksPath could not be set in the checkout."
fi

# --- 3b. abcd, built once for the session's later shells ------------------------------

bin_rel=".abcd/.work.local/cloud/bin/abcd"
abcd_step() {
	local bin="$root/$bin_rel" var line
	if ! as_abcd mkdir -p "$root/.abcd/.work.local/cloud/bin" ||
		! as_abcd go build -o "$bin" ./cmd/abcd </dev/null >/dev/null 2>&1 || [ ! -x "$bin" ]; then
		miss "abcd could not be built from the checkout (go build ./cmd/abcd, as $USER_NAME), so every push builds it per commit."
		return 0
	fi
	case "$bin" in
	*"'"* | *$'\n'*)
		say "abcd built at $bin_rel; its path cannot be written as a variable, so set ABCD_OUTBOUND_BIN and ABCD_BIN to it in each shell."
		return 0
		;;
	esac
	if [ -z "${CLAUDE_ENV_FILE:-}" ]; then
		say "abcd built at $bin_rel; no file of variables for later shells was given, so set ABCD_OUTBOUND_BIN and ABCD_BIN to it in each shell."
		return 0
	fi
	for var in ABCD_OUTBOUND_BIN ABCD_BIN; do
		line="export $var='$bin'"
		if ! grep -Fxq -- "$line" "$CLAUDE_ENV_FILE" 2>/dev/null; then
			printf '%s\n' "$line" >>"$CLAUDE_ENV_FILE" || {
				miss "abcd built at $bin_rel, but $var could not be written to the file of variables for later shells."
				return 0
			}
		fi
	done
	say "abcd built at $bin_rel and handed to later shells as ABCD_OUTBOUND_BIN and ABCD_BIN."
}
abcd_step

# --- 4. The author identity, from the forge token ----------------------------------------

no_identity() {
	miss "the author identity could not be taken from the forge token, so none was set; the brief's first step stops before any work."
}

# person says whether the checkout's identity, as the user sees it, is set and
# a person's. Read into variables only; the gate's output is discarded unread.
person() {
	local name mail
	name="$(as_abcd git -C "$root" config --get user.name 2>/dev/null)" || return 1
	mail="$(as_abcd git -C "$root" config --get user.email 2>/dev/null)" || return 1
	[ -n "$name" ] && [ -n "$mail" ] || return 1
	bash "$root/scripts/check-attribution.sh" identity author "$name" "$mail" >/dev/null 2>&1
}

# field <name> reads one string field of the verb's flat JSON answer.
field() {
	printf '%s' "$answer" | sed -n 's/.*"'"$1"'"[[:space:]]*:[[:space:]]*"\([^"\\]*\)".*/\1/p' | head -n 1
}

# holder_env_malformed says whether ABCD_HOLDER_ID is set to anything but one
# holder id, which abcd refuses (never echoing it) before it names any account.
holder_env_malformed() {
	[ -n "${ABCD_HOLDER_ID:-}" ] && ! [[ "$ABCD_HOLDER_ID" =~ ^h_[0-9a-f]{32}$ ]]
}
identity_step() {
	local login noreply source id
	answer="$(as_abcd go run ./cmd/abcd reserve --holder --json 2>/dev/null)" || {
		if holder_env_malformed; then
			# The lookup refused on the variable, not the token: no login
			# is printed, so none can be set as the author.
			miss "the author identity was not set: ABCD_HOLDER_ID is set but is not a holder id (h_ and 32 lower-case hex characters), so \`abcd reserve --holder\` refuses, as every reservation and announcement does, before it names the forge token's account; set ABCD_HOLDER_ID to the id \`abcd reserve --holder\` prints, or unset it, and run the setup again. The brief's first step stops before any work."
		else
			no_identity
		fi
		return 0
	}
	answer="$(printf '%s' "$answer" | tr -d '\r\n')"
	login="$(field login)"
	noreply="$(field noreply)"
	source="$(field source)"
	if ! [[ "$login" =~ $LOGIN_RE ]] || ! [[ "$noreply" =~ ^([0-9]+)\+(.*)@users\.noreply\.github\.com$ ]]; then
		no_identity
		return 0
	fi
	id="${BASH_REMATCH[1]}"
	if [ "$(printf '%s' "${BASH_REMATCH[2]}" | tr '[:upper:]' '[:lower:]')" != "$(printf '%s' "$login" | tr '[:upper:]' '[:lower:]')" ] ||
		[ "$id" = 0 ]; then
		no_identity
		return 0
	fi
	if [ "$source" != token ] && ! person; then
		no_identity
		return 0
	fi
	if as_abcd git -C "$root" config --local user.name "$login" &&
		as_abcd git -C "$root" config --local user.email "$noreply"; then
		say "author identity set in the checkout from the forge token's account: $login <$noreply>."
	else
		no_identity
	fi
}
# holder_step says where the holder id this session acts under came from
# (4b), from the identity step's answer.
holder_step() {
	local held
	if [ -n "${ABCD_HOLDER_ID:-}" ]; then
		if [[ "$ABCD_HOLDER_ID" =~ ^h_[0-9a-f]{32}$ ]]; then
			say "holder id: taken from ABCD_HOLDER_ID, so this session's reservations and announcements carry the id of earlier sessions."
		else
			say "holder id: ABCD_HOLDER_ID is set but is not a holder id (h_ and 32 lower-case hex characters), so abcd refuses every reservation and announcement; set it to the id \`abcd reserve --holder\` prints, or unset it."
		fi
		return 0
	fi
	held="$(field holder_id)"
	[[ "$held" =~ ^h_[0-9a-f]{32}$ ]] || return 0
	say "holder id: $held, kept in this session's home only; ABCD_HOLDER_ID is not set, so the next session mints another, and holds made under this one read as another holder's until they lapse. To keep one id, set ABCD_HOLDER_ID=$held in the environment's settings."
}
answer=""
identity_step
holder_step

exit "$failed"
