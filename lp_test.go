// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lpenv

import (
	"testing"
)

var nonExistentPaths = []string{
	"some-non-existent-path",
	"non-existent-path/slashed",
}

func TestLookPathEnvNotFound(t *testing.T) {
	for _, name := range nonExistentPaths {
		path, err := LookPathEnv(name, ".", []string{})
		if err == nil {
			t.Fatalf("LookPathEnv found %q in $PATH", name)
		}
		if path != "" {
			t.Fatalf("LookPathEnv path == %q when err != nil", path)
		}
		perr, ok := err.(*Error)
		if !ok {
			t.Fatal("LookPathEnv error is not an exec.Error")
		}
		if perr.Name != name {
			t.Fatalf("want Error name %q, got %q", name, perr.Name)
		}
	}
}
