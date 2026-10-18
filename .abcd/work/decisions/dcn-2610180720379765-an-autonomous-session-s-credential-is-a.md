---
id: dcn-2610180720379765
slug: an-autonomous-session-s-credential-is-a
date: 2026-10-18
---
2026-10-18 — An autonomous session's credential is a GitHub App token (facilitator-on-delegation, itd-2610170634394993 planning prep): each session gets a token the repository's App mints for it, short-lived and scoped by repository and permission; a fine-grained personal token is not used, because none can be created through an API and none expires with the session (grounds: docs.github.com authenticating-as-a-github-app-installation; docs.github.com managing-your-personal-access-tokens)
