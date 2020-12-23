// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Use an external test to avoid os/exec -> internal/testenv -> os/exec
// circular dependency.

package lpenv_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/direnv/go-lpenv"
)

func installExe(t *testing.T, dest, src string) {
	fsrc, err := os.Open(src)
	if err != nil {
		t.Fatal("os.Open failed: ", err)
	}
	defer fsrc.Close()
	fdest, err := os.Create(dest)
	if err != nil {
		t.Fatal("os.Create failed: ", err)
	}
	defer fdest.Close()
	_, err = io.Copy(fdest, fsrc)
	if err != nil {
		t.Fatal("io.Copy failed: ", err)
	}
}

func installBat(t *testing.T, dest string) {
	f, err := os.Create(dest)
	if err != nil {
		t.Fatalf("failed to create batch file: %v", err)
	}
	defer f.Close()
	fmt.Fprintf(f, "@echo %s\n", dest)
}

func installProg(t *testing.T, dest, srcExe string) {
	err := os.MkdirAll(filepath.Dir(dest), 0700)
	if err != nil {
		t.Fatal("os.MkdirAll failed: ", err)
	}
	if strings.ToLower(filepath.Ext(dest)) == ".bat" {
		installBat(t, dest)
		return
	}
	installExe(t, dest, srcExe)
}

type lookPathTest struct {
	rootDir   string
	PATH      string
	PATHEXT   string
	files     []string
	searchFor string
	fails     bool // test is expected to fail
}

func (test lookPathTest) runProg(t *testing.T, env []string, args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	cmd.Dir = test.rootDir
	args[0] = filepath.Base(args[0])
	cmdText := fmt.Sprintf("%q command", strings.Join(args, " "))
	out, err := cmd.CombinedOutput()
	if (err != nil) != test.fails {
		if test.fails {
			t.Fatalf("test=%+v: %s succeeded, but expected to fail", test, cmdText)
		}
		t.Fatalf("test=%+v: %s failed, but expected to succeed: %v - %v", test, cmdText, err, string(out))
	}
	if err != nil {
		return "", fmt.Errorf("test=%+v: %s failed: %v - %v", test, cmdText, err, string(out))
	}
	// normalise program output
	p := string(out)
	// trim terminating \r and \n that batch file outputs
	for len(p) > 0 && (p[len(p)-1] == '\n' || p[len(p)-1] == '\r') {
		p = p[:len(p)-1]
	}
	return p, nil
}

func updateEnv(env []string, name, value string) []string {
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), name+"=") {
			env[i] = name + "=" + value
			return env
		}
	}
	return append(env, name+"="+value)
}

func createEnv(dir, PATH, PATHEXT string) []string {
	env := os.Environ()
	env = updateEnv(env, "PATHEXT", PATHEXT)
	// Add dir in front of every directory in the PATH.
	dirs := filepath.SplitList(PATH)
	for i := range dirs {
		dirs[i] = filepath.Join(dir, dirs[i])
	}
	path := strings.Join(dirs, ";")
	env = updateEnv(env, "PATH", os.Getenv("SystemRoot")+"/System32;"+path)
	return env
}

// createFiles copies srcPath file into multiply files.
// It uses dir as prefix for all destination files.
func createFiles(t *testing.T, dir string, files []string, srcPath string) {
	for _, f := range files {
		installProg(t, filepath.Join(dir, f), srcPath)
	}
}

func (test lookPathTest) run(t *testing.T, tmpdir, printpathExe string) {
	test.rootDir = tmpdir
	createFiles(t, test.rootDir, test.files, printpathExe)
	env := createEnv(test.rootDir, test.PATH, test.PATHEXT)
	// Run "cmd.exe /c test.searchFor" with new environment and
	// work directory set. All candidates are copies of printpath.exe.
	// These will output their program paths when run.
	should, errCmd := test.runProg(t, env, "cmd", "/c", test.searchFor)
	// Run the lookpath program with new environment and work directory set.
	have, errLP := lpenv.LookPathEnv(test.searchFor, env)
	// Compare results.
	if errCmd == nil && errLP == nil {
		// both succeeded
		if strings.ToLower(should) != strings.ToLower(have) {
			t.Errorf("test=%+v failed: expected to find %q, but found %q", test, should, have)
		}
		return
	}
	if errCmd != nil && errLP != nil {
		// both failed -> continue
		return
	}
	if errCmd != nil {
		t.Errorf("test=%+v failed: test setup failed with %v", test, errCmd)
	}
	if errLP != nil {
		t.Errorf("test=%+v failed: expected to find %q, but got error %v", test, should, errLP)
	}
}

var lookPathTests = []lookPathTest{
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`, `p2\a`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1.dir;p2.dir`,
		files:     []string{`p1.dir\a`, `p2.dir\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.exe`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\b.exe`},
		searchFor: `b`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\b`, `p2\a`},
		searchFor: `a`,
		fails:     true, // TODO(brainman): do not know why this fails
	},
	// If the command name specifies a path, the shell searches
	// the specified path for an executable file matching
	// the command name. If a match is found, the external
	// command (the executable file) executes.
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `p2\a`,
	},
	// If the command name specifies a path, the shell searches
	// the specified path for an executable file matching the command
	// name. ... If no match is found, the shell reports an error
	// and command processing completes.
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\b.exe`, `p2\a.exe`},
		searchFor: `p2\b`,
		fails:     true,
	},
	// If the command name does not specify a path, the shell
	// searches the current directory for an executable file
	// matching the command name. If a match is found, the external
	// command (the executable file) executes.
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`a`, `p1\a.exe`, `p2\a.exe`},
		searchFor: `a`,
	},
	// The shell now searches each directory specified by the
	// PATH environment variable, in the order listed, for an
	// executable file matching the command name. If a match
	// is found, the external command (the executable file) executes.
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a`,
	},
	// The shell now searches each directory specified by the
	// PATH environment variable, in the order listed, for an
	// executable file matching the command name. If no match
	// is found, the shell reports an error and command processing
	// completes.
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `b`,
		fails:     true,
	},
	// If the command name includes a file extension, the shell
	// searches each directory for the exact file name specified
	// by the command name.
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.exe`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.com`,
		fails:     true, // includes extension and not exact file name match
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1`,
		files:     []string{`p1\a.exe.exe`},
		searchFor: `a.exe`,
	},
	{
		PATHEXT:   `.COM;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.exe`,
	},
	// If the command name does not include a file extension, the shell
	// adds the extensions listed in the PATHEXT environment variable,
	// one by one, and searches the directory for that file name. Note
	// that the shell tries all possible file extensions in a specific
	// directory before moving on to search the next directory
	// (if there is one).
	{
		PATHEXT:   `.COM;.EXE`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p2\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p2\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p1\a.exe`, `p2\a.bat`, `p2\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p2\a.exe`},
		searchFor: `a`,
		fails:     true, // tried all extensions in PATHEXT, but none matches
	},
}

func TestLookPathEnv(t *testing.T) {
	tmp, err := ioutil.TempDir("", "TestLookPathEnv")
	if err != nil {
		t.Fatal("TempDir failed: ", err)
	}
	defer os.RemoveAll(tmp)

	printpathExe := buildPrintPathExe(t, tmp)

	// Run all tests.
	for i, test := range lookPathTests {
		dir := filepath.Join(tmp, "d"+strconv.Itoa(i))
		err := os.Mkdir(dir, 0700)
		if err != nil {
			t.Fatal("Mkdir failed: ", err)
		}
		test.run(t, dir, printpathExe)
	}
}

// buildPrintPathExe creates a Go program that prints its own path.
// dir is a temp directory where executable will be created.
// The function returns full path to the created program.
func buildPrintPathExe(t *testing.T, dir string) string {
	const name = "printpath"
	srcname := name + ".go"
	err := ioutil.WriteFile(filepath.Join(dir, srcname), []byte(printpathSrc), 0644)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	outname := name + ".exe"
	p, err := lpenv.LookPathEnv("go", os.Environ())
	if err != nil {
		t.Fatalf("could not find the Go executable")
	}
	cmd := exec.Command(p, "build", "-o", outname, srcname)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build executable: %v - %v", err, string(out))
	}
	return filepath.Join(dir, outname)
}

const printpathSrc = `
package main

import (
	"os"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func getMyName() (string, error) {
	var sysproc = syscall.MustLoadDLL("kernel32.dll").MustFindProc("GetModuleFileNameW")
	b := make([]uint16, syscall.MAX_PATH)
	r, _, err := sysproc.Call(0, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	n := uint32(r)
	if n == 0 {
		return "", err
	}
	return string(utf16.Decode(b[0:n])), nil
}

func main() {
	path, err := getMyName()
	if err != nil {
		os.Stderr.Write([]byte("getMyName failed: " + err.Error() + "\n"))
		os.Exit(1)
	}
	os.Stdout.Write([]byte(path))
}
`
