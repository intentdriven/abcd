// Package cloudscripts holds the tests of the cloud-session scripts under
// scripts/cloud/ (spc-2610120941441282, itd-2610120738075547), and nothing
// else: it has no code of its own.
//
// Each test runs a committed script the way a person or a hosted session
// would, in a throwaway clone carrying copies of the scripts, with a bare
// repository as its remote and fakes first on PATH for every tool whose
// answer the test decides. The tests are slow by nature, each one several git
// processes and a shell, so they sit in a package of their own: go test gives
// them their own time budget and runs them beside the other packages, rather
// than adding minutes of serial time to internal/surface/cli's.
package cloudscripts
