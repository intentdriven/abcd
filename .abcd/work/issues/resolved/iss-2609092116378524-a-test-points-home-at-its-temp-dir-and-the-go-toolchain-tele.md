---
schema_version: 1
id: "iss-2609092116378524"
slug: "a-test-points-home-at-its-temp-dir-and-the-go-toolchain-tele"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "v0.8.0 auto-release run 34403815697"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hooks_install_shapes_test.go"
impact: internal
resolution: "sandboxHome replaces t.TempDir wherever a test hands a HOME to a process abcd does not control, at four sites; removal becomes best-effort because the Go toolchain writes telemetry counters there on its own schedule and neither GOTELEMETRY nor GOTELEMETRYDIR suppresses it from the environment (measured against go1.27.1)"
---

`TestHooksRunEveryShapeAhoyInstalls/dev_shim` failed the release gate's
race-enabled lane and stopped the v0.8.0 release from publishing. The tag was
already cut, so the repository sat with a `v0.8.0` tag and no Release and no
binaries until the job was re-run.

The test body passed. The only failure was in cleanup:

```
testing.go:1464: TempDir RemoveAll cleanup:
  unlinkat /tmp/TestHooksRunEveryShapeAhoyInstallsdev_shim.../001/.config/go/telemetry/local: directory not empty
```

## Mechanism, measured

Three release-lane runs failed the same subtest. Two name
`.config/go/telemetry` directly and one names the temp root one directory up,
which is the same race seen from a level higher:

```
run 1: unlinkat .../001/.config/go/telemetry/local: directory not empty
run 2: unlinkat /tmp/TestHooksRunEveryShapeAhoyInstallsdev_shim1376441102: directory not empty
run 3: unlinkat .../001/.config/go/telemetry: directory not empty
```

`runAhoyInstall` hands the subtest a HOME from `t.TempDir()`. The `dev shim`
shape is the only one of the three that executes a shim, and the shim rebuilds
from source, so it runs the Go toolchain with that HOME. The toolchain writes
telemetry counters under `HOME/<user-config>/go/telemetry/local` on its own
schedule rather than synchronously, and `t.TempDir()` FAILS THE TEST when its
removal does not succeed. A counter file landing during removal therefore fails a
subtest whose body has already passed.

It reproduces on Linux and not on macOS because the shim gets `PATH=/usr/bin:/bin`
and the release runner has a `go` there. That also explains why the same tree
passed `make preflight` locally and the merge-queue `ci` run minutes earlier.

**Neither documented knob prevents it from the environment.** Measured against
go1.27.1 with a bare `go version` under a fake HOME:

| Environment | Files left under the fake HOME |
| --- | --- |
| default | 3 (`upload.token`, `weekends`, a `.count` file) |
| `GOTELEMETRY=off` | 3 |
| `GOTELEMETRY=local` | 3 |
| `GOTELEMETRYDIR=<outside>` | 3, and 0 in the redirect target |

An earlier revision of this record prescribed `GOTELEMETRY=off`. That would not
have worked, and the table is here so the next reader does not try it again.

## Fix

`sandboxHome(t)` replaces `t.TempDir()` wherever a test hands a HOME to a process
abcd does not control: it creates the directory with `os.MkdirTemp` and removes
it in `t.Cleanup` on a best-effort basis. The directory cannot be kept clean, so
its removal stops being an assertion. What a test is entitled to assert is what
abcd wrote, never what a foreign process left behind.

Applied at four sites and swept rather than patched at the one that fired: the
install-shapes helper, the self-provision helper's default HOME, and the two
session-start subprocesses. The last two are the more exposed, since they hand
the child the full `PATH` and so reach a `go` on any machine. The bootstrap
tests already build a deliberately go-free PATH and are left alone.

## Why it is major rather than a nuisance

It fails the release lane nondeterministically, and the release lane is the one
that runs after the tag is cut. The same test passed in the merge-queue `ci` run,
in the `check` job on both operating systems, and in `make preflight` locally on
the same tree, minutes apart. A gate that passes four times and fails the fifth,
at the one point in the pipeline where a failure leaves a tag without a release,
costs more than the flake suggests.

It is also load-dependent, which is why it surfaces here and not locally: the
release job runs the race lane over the whole of `internal/`, so the window
between the toolchain's write and the cleanup is wider.

## Shape of the fix

First identify the writer, then close it. The diagnosis step is to reproduce
under the release lane's conditions and see what exists in the sandboxed HOME at
cleanup, rather than inferring it from the error path.

If the writer is the Go toolchain, keep it out of the fake HOME entirely
(`GOTELEMETRY=off`, or `GOTELEMETRYDIR` pointing outside the temp tree) rather
than tolerating the race: the fake HOME exists so the test can observe what abcd
writes, and a foreign writer in it defeats that whether or not it also breaks
cleanup. If the writer is a child process the shim starts, the fix is to wait for
it before the body returns, since a test that leaves a process running has a
race that cleanup merely reveals.

Either way, sweep instead of patching one call site: `t.Setenv("HOME"` across the
test tree finds every test that hands a temp directory to something that may
write to it.

## Acceptance

- **Given** the `dev shim` shape, **when** it runs the toolchain under a fake
  HOME, **then** no file appears under `$HOME/.config/go/` and cleanup cannot
  race a writer.
- **Given** the race lane over all of `internal/`, **when** it runs repeatedly,
  **then** this test does not fail in cleanup.

## Grounds

- pursued: we expect the release lane to stop failing in cleanup because the test no longer asserts on the emptiness of a directory a foreign process writes to; what would show this wrong is a cleanup failure naming a path the test itself created, which would mean the writer is abcd rather than the toolchain
