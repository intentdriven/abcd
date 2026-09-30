---
schema_version: 1
id: "iss-2609300905174086"
slug: "consent-gates-judge-dev-null-a-terminal-term-isterminal-is-a"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/term/term.go"
remedy: "Make term.IsTerminal ask the kernel: a termios get (ioctl TIOCGETA on darwin, TCGETS on linux, through the standard library syscall package, fail-closed false elsewhere) succeeds only on a real terminal, as isatty(3) does; route the hand-rolled ModeCharDevice check in readKey through it. Grounds: POSIX isatty is defined by tcgetattr succeeding, and /dev/null is a character device that is not a terminal."
---

Consent gates judge /dev/null a terminal: term.IsTerminal is a character-device test, so stdin redirected from /dev/null (and a closed fd 0, which the Go runtime reopens on /dev/null) reads as a person at a terminal. The tool-install question is printed to stderr, EOF reads as no, and the decline says 'answered no at the terminal' instead of 'no terminal to ask at'; the itd-131 identity offer asks the same way, and the provider-key reader in ahoy_connect.go hand-rolls the same check and refuses /dev/null as 'stdin is a terminal'.
