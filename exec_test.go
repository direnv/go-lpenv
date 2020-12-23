// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Use an external test to avoid os/exec -> net/http -> crypto/x509 -> os/exec
// circular dependency on non-cgo darwin.

package lpenv_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/direnv/go-lpenv"
)

// TestHelperProcess isn't a real test. It's used as a helper process
// for TestParameterRun.
func TestHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "lookpath":
		p, err := lpenv.LookPath(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "LookPath failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(p)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}
