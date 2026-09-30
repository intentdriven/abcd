---
schema_version: 1
id: "iss-2609300055241043"
slug: "abcd-update-stages-the-verified-release-at-the-fixed-name"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
remedy: "Unlink whatever stands at the staging name, then create it with O_CREATE|O_EXCL|O_WRONLY (plus O_NOFOLLOW): O_EXCL refuses any existing entry, a symlink included, on every platform (open(2) POSIX: with O_CREAT and O_EXCL, if path is a symbolic link open fails regardless of its target), so the bytes are never written through a link and the swapped entry is the regular file the open created."
---

abcd update stages the verified release at the fixed name .abcd.new beside the target and opens it with O_CREATE|O_WRONLY|O_TRUNC, which follows a symlink planted at that name: the file the link points at is truncated and filled with the release bytes, and the swap then renames the symlink into the target's name, so the PATH entry becomes a link to that file. Precondition is write access to the install directory (which already lets one replace abcd), and the open was selfupdate.Apply's before the staging moved into Apply, so not a regression, but the open is owned code now.
