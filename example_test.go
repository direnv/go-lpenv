// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lpenv_test

import (
	"fmt"
	"log"

	"github.com/direnv/go-lpenv"
)

func ExampleLookPath() {
	path, err := lpenv.LookPath("fortune")
	if err != nil {
		log.Fatal("installing fortune is in your future")
	}
	fmt.Printf("fortune is available at %s\n", path)
}
