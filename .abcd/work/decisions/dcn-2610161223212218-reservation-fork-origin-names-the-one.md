---
id: dcn-2610161223212218
slug: reservation-fork-origin-names-the-one
date: 2026-10-16
---
2026-10-16 — reservation.fork_origin names the one fork whose reservations it allows, as "<owner>/<name>", instead of a bare "allow" (facilitator-on-delegation, drain-many-accounts): the key takes "refuse", the bundled default, or one owner and one name, each a plain GitHub name, read as before through the layered configuration and only once the forge has named an origin push URL a fork; a fork the forge names that is not the allowed one, compared without regard to case, is refused naming both, and a waived check is never silent: reserve, a run record and a drain lane say "origin is a fork the repository allows (<owner>/<name>)"; the bare "allow" is dropped, since it never shipped (grounds: a committed "allow" in a repository that is itself a fork is inherited by every fork of it, so a fork of the fork would reserve silently where no one working on the repository can see it; naming the fork keeps the waiver to the one repository the setting was written for; refines dcn-2610161141338923).
