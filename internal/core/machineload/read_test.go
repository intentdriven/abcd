package machineload

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestReadSeesThisProcess runs the real reader on whichever platform runs the
// test (CI's macOS and Linux legs cover both): the snapshot holds this process
// under its own effective uid, at least one core, and non-negative loads.
func TestReadSeesThisProcess(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no reader on %s; TestMachineLoadCompilesElsewhere covers the stub", runtime.GOOS)
	}
	snap, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !snap.HasLoad || snap.Load1 < 0 || snap.Load5 < 0 || snap.Load15 < 0 {
		t.Fatalf("loads = %v %v %v (has %v)", snap.Load1, snap.Load5, snap.Load15, snap.HasLoad)
	}
	if snap.Cores < 1 {
		t.Fatalf("cores = %d", snap.Cores)
	}
	if !snap.HasProcs {
		t.Fatal("no process table")
	}
	for _, p := range snap.Procs {
		if p.PID == os.Getpid() {
			if p.UID != uint32(os.Geteuid()) {
				t.Fatalf("this process reads as uid %d, want %d", p.UID, os.Geteuid())
			}
			if p.Name == "" || strings.Contains(p.Name, "/") {
				t.Fatalf("this process's name %q is not a base name", p.Name)
			}
			return
		}
	}
	t.Fatalf("this process (pid %d) is not in the snapshot of %d processes", os.Getpid(), len(snap.Procs))
}

// TestMachineLoadImportsOnlyTheStandardLibrary: the package is a leaf that adds
// no dependency, not even one of this module's own packages.
func TestMachineLoadImportsOnlyTheStandardLibrary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v", err)
	}
	fset := token.NewFileSet()
	for _, f := range files {
		file, err := parser.ParseFile(fset, f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			first, _, _ := strings.Cut(p, "/")
			if strings.Contains(first, ".") {
				t.Errorf("%s imports %s, which is not the standard library", f, p)
			}
		}
	}
}

// TestMachineLoadCompilesElsewhere: on a platform with no reader the package
// still builds, so the check can say it could not check rather than fail to
// compile.
func TestMachineLoadCompilesElsewhere(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go tool")
	}
	cmd := exec.Command("go", "vet", ".")
	cmd.Env = append(os.Environ(), "GOOS=freebsd", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("GOOS=freebsd go vet: %v\n%s", err, out)
	}
}
