#!/usr/bin/env bash
# Exit 0 only inside one of this host's hosted cloud sessions, which set
# CLAUDE_CODE_REMOTE=true; exit 1 everywhere else, the person's own computer
# included (itd-2610120738075547 C3). It prints nothing: the caller words the
# refusal, so the host is named only here.
#
# The comparison is exact. An unset, empty, `1` or `TRUE` value is not the
# marker the host sets, and a check that read any of them as "remote" would let
# a brief written for the cloud start work on the person's computer.
set -euo pipefail

[ "${CLAUDE_CODE_REMOTE:-}" = true ]
