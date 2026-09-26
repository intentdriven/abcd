# Security policy

## Reporting a vulnerability

Report vulnerabilities privately through GitHub's private vulnerability
reporting: [Security → Report a
vulnerability](https://github.com/intentdriven/abcd/security/advisories/new).
Never open a public issue or pull request for a security finding — the report
stays private while it is triaged and fixed.

You can expect an acknowledgement within a week. Please include what you found,
where (file or component), and how to reproduce it.

## Scope

The released binaries, the plugin surface (`hooks/`, `commands/`, `agents/`),
and the release pipeline itself are all in scope. Secret scanning, push
protection and dependency review run on the repository, but a human report
beats every scanner — when in doubt, report it.

## Supported versions

The latest release is supported. Older releases receive no separate fixes; a
security fix ships as a new release.
