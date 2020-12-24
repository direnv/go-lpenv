package lpenv_test

import (
	"os"
	"testing"

	"github.com/direnv/go-lpenv"
)

func TestGetenv(t *testing.T) {
	env := []string{"PATH=foobar"}

	path := lpenv.Getenv(env, "PATH")
	if path != "foobar" {
		t.Errorf("expected %s to be 'foobar'", path)
	}

	path = lpenv.Getenv(os.Environ(), "PATH")
	if path == "" {
		t.Errorf("expected to find PATH in %+v", os.Environ())
	}
}
