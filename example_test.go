// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lpenv_test

import (
	"fmt"
	"log"
	"os"

	"github.com/direnv/go-lpenv"
)

func ExampleLookPathEnv() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("could not find current working directory")
	}
	path, err := lpenv.LookPathEnv("fortune", cwd, os.Environ())
	if err != nil {
		log.Fatal("installing fortune is in your future")
	}
	fmt.Printf("fortune is available at %s\n", path)
}
