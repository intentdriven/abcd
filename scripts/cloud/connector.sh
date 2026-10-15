#!/usr/bin/env bash
# Loads and validates the host connector declarations (spc-2610120941441282,
# itd-2610120738075547 decision 1, adr-2610120937435856).
#
# A connector is a directory under scripts/cloud/connectors/ holding a
# connector.conf of key=value lines. Everything that names one host lives in
# its connector; every other cloud script asks this one for what it needs, so a
# second host is a second directory and no edit anywhere else (C12).
#
# The declaration is DATA, read line by line and never sourced: a sourced file
# runs whatever it holds, and the push hook reads these declarations too.
#
# Keys (anything else is refused):
#
#   name        the connector's name; must equal its directory's name
#   wip_prefix  the branch prefix its sessions push unchecked work under. It
#               ends in the segment `abcd-wip/` and holds at most one segment
#               before it (the ADR's decision 2), so the hook's receipt-less
#               exception can never widen to a whole namespace or to `main`.
#   is_remote   a script that exits 0 only inside the host's cloud session
#   setup       the script the host runs when a cloud session starts
#   launch      the script that starts a hosted session
#
# name, wip_prefix and is_remote are required. A script key names a regular
# executable file inside the connector's own directory: no absolute path, no
# `..`, and no symlink, so a declaration cannot point the scripts at anything
# outside the connector.
#
# Every verb validates EVERY declaration first, so one bad file refuses them
# all, naming the file: a reader that skipped the broken one would act on a
# partial list without saying so.
#
# Usage:
#   connector.sh validate             # every declaration; exit 0 when all hold
#   connector.sh names                # one connector name per line
#   connector.sh prefixes             # one wip_prefix per line
#   connector.sh default              # the only connector; refuses with two
#   connector.sh get <name> <key>     # one value; a script key as an absolute path
#
# Exit 0 on success, 1 on a refused declaration or lookup, 2 on misuse.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
CONNECTORS_DIR="$here/connectors"
# The file named in every refusal is relative to the checkout, so the message
# reads the same on every machine.
REL_DIR="scripts/cloud/connectors"

WIP_PREFIX_RE='^([a-z0-9][a-z0-9-]*/)?abcd-wip/$'
NAME_RE='^[a-z0-9][a-z0-9-]*$'
SCRIPT_RE='^[A-Za-z0-9_][A-Za-z0-9._-]*(/[A-Za-z0-9_][A-Za-z0-9._-]*)*$'

usage() {
	echo "usage: connector.sh validate | names | prefixes | default | get <name> <key>" >&2
	exit 2
}

refuse() {
	echo "connector: REFUSING $1" >&2
	exit 1
}

# Parallel newline-separated lists, one entry per valid connector: bash 3.2 on
# stock macOS has no associative arrays.
NAMES=""
PREFIXES=""

# read_conf <dir> sets c_name, c_wip_prefix, c_is_remote, c_setup, c_launch from
# <dir>/connector.conf, refusing anything malformed.
read_conf() {
	local dir="$1" conf rel line key value n=0 seen=" "
	conf="$CONNECTORS_DIR/$dir/connector.conf"
	rel="$REL_DIR/$dir/connector.conf"
	c_name="" c_wip_prefix="" c_is_remote="" c_setup="" c_launch=""
	[ -f "$conf" ] && [ ! -L "$conf" ] || refuse "$REL_DIR/$dir/: a connector directory holds a connector.conf, and this one does not."
	while IFS= read -r line || [ -n "$line" ]; do
		n=$((n + 1))
		line="${line%$'\r'}"
		case "$line" in
		'' | '#'*) continue ;;
		esac
		if ! [[ "$line" =~ ^[a-z_]+= ]]; then
			refuse "$rel line $n: not a key=value line."
		fi
		key="${line%%=*}"
		value="${line#*=}"
		case "$seen" in
		*" $key "*) refuse "$rel line $n: $key is given twice." ;;
		esac
		seen="$seen$key "
		case "$key" in
		name) c_name="$value" ;;
		wip_prefix) c_wip_prefix="$value" ;;
		is_remote) c_is_remote="$value" ;;
		setup) c_setup="$value" ;;
		launch) c_launch="$value" ;;
		*) refuse "$rel line $n: unknown key $key (the keys are name, wip_prefix, is_remote, setup, launch)." ;;
		esac
	done <"$conf"

	for key in name wip_prefix is_remote; do
		case "$seen" in
		*" $key "*) ;;
		*) refuse "$rel: the required key $key is missing." ;;
		esac
	done
	[[ "$c_name" =~ $NAME_RE ]] && [ "$c_name" = "$dir" ] ||
		refuse "$rel: name '$c_name' must be lower-case letters, digits and dashes, and equal its directory's name '$dir'."
	[[ "$c_wip_prefix" =~ $WIP_PREFIX_RE ]] ||
		refuse "$rel: wip_prefix '$c_wip_prefix' must end in the segment abcd-wip/ with at most one segment before it (adr-2610120937435856)."
	check_script "$dir" "$rel" is_remote "$c_is_remote"
	[ -z "$c_setup" ] || check_script "$dir" "$rel" setup "$c_setup"
	[ -z "$c_launch" ] || check_script "$dir" "$rel" launch "$c_launch"
}

# check_script refuses a script path that could leave the connector directory,
# or that names no executable file there.
check_script() {
	local dir="$1" rel="$2" key="$3" value="$4" base path parent
	base="$CONNECTORS_DIR/$dir"
	case "/$value/" in
	*/../* | */./*) refuse "$rel: $key '$value' leaves the connector directory." ;;
	esac
	[[ "$value" =~ $SCRIPT_RE ]] || refuse "$rel: $key '$value' must be a relative path inside the connector directory."
	path="$base/$value"
	[ ! -L "$path" ] || refuse "$rel: $key '$value' is a symlink; a connector script is a file inside its own directory."
	[ -f "$path" ] && [ -x "$path" ] || refuse "$rel: $key '$value' is not an executable file in the connector directory."
	parent="$(cd "$(dirname "$path")" && pwd -P)"
	case "$parent/" in
	"$base/"*) ;;
	*) refuse "$rel: $key '$value' resolves outside the connector directory." ;;
	esac
}

load_all() {
	local d dir
	[ -d "$CONNECTORS_DIR" ] || refuse "$REL_DIR/: no connector is declared."
	for d in "$CONNECTORS_DIR"/*; do
		[ -e "$d" ] || continue
		dir="$(basename "$d")"
		if [ ! -d "$d" ] || [ -L "$d" ]; then
			refuse "$REL_DIR/$dir: only connector directories belong here."
		fi
		read_conf "$dir"
		if [ -n "$PREFIXES" ] && printf '%s\n' "$PREFIXES" | grep -Fxq -- "$c_wip_prefix"; then
			refuse "$REL_DIR/$dir/connector.conf: wip_prefix '$c_wip_prefix' is declared by two connectors; each prefix belongs to one."
		fi
		NAMES="${NAMES:+$NAMES
}$c_name"
		PREFIXES="${PREFIXES:+$PREFIXES
}$c_wip_prefix"
	done
	[ -n "$NAMES" ] || refuse "$REL_DIR/: no connector is declared."
}

the_default() {
	local count
	count="$(printf '%s\n' "$NAMES" | grep -c .)"
	if [ "$count" -ne 1 ]; then
		echo "connector: refusing to guess between $count connectors ($(printf '%s\n' "$NAMES" | paste -sd, -)); name one." >&2
		exit 1
	fi
	printf '%s\n' "$NAMES"
}

case "${1:-}" in
validate)
	[ $# -eq 1 ] || usage
	load_all
	echo "connector: $(printf '%s\n' "$NAMES" | grep -c .) connector declaration(s) hold: $(printf '%s\n' "$NAMES" | paste -sd, -)"
	;;
names)
	[ $# -eq 1 ] || usage
	load_all
	printf '%s\n' "$NAMES"
	;;
prefixes)
	[ $# -eq 1 ] || usage
	load_all
	printf '%s\n' "$PREFIXES"
	;;
default)
	[ $# -eq 1 ] || usage
	load_all
	the_default
	;;
get)
	[ $# -eq 3 ] || usage
	load_all
	name="$2"
	key="$3"
	printf '%s\n' "$NAMES" | grep -Fxq -- "$name" || refuse "connector '$name': no such connector is declared."
	read_conf "$name"
	case "$key" in
	name) value="$c_name" ;;
	wip_prefix) value="$c_wip_prefix" ;;
	is_remote) value="$c_is_remote" ;;
	setup) value="$c_setup" ;;
	launch) value="$c_launch" ;;
	*) usage ;;
	esac
	[ -n "$value" ] || refuse "connector '$name' declares no $key."
	case "$key" in
	is_remote | setup | launch) value="$CONNECTORS_DIR/$name/$value" ;;
	esac
	printf '%s\n' "$value"
	;;
*)
	usage
	;;
esac
