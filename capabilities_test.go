package qemuimg

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestCapabilitiesModern(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(modernHelp))

	caps, err := q.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() returned error: %v", err)
	}
	if !caps.ConvertBackingFormat {
		t.Fatal("ConvertBackingFormat = false, want true")
	}
	if !caps.CommitSkipEmpty {
		t.Fatal("CommitSkipEmpty = false, want true")
	}
}

func TestCapabilitiesLegacy(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(legacyHelp))

	caps, err := q.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() returned error: %v", err)
	}
	if caps.ConvertBackingFormat {
		t.Fatal("ConvertBackingFormat = true, want false")
	}
	if caps.CommitSkipEmpty {
		t.Fatal("CommitSkipEmpty = true, want false")
	}
}

func TestCapabilitiesUsesStderr(t *testing.T) {
	q, _ := newTestQemuImg(func(_ string, _ []string) (Result, error) {
		return Result{Stderr: modernHelp}, nil
	})

	caps, err := q.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() returned error: %v", err)
	}
	if !caps.ConvertBackingFormat || !caps.CommitSkipEmpty {
		t.Fatalf("Capabilities from stderr = %+v, want both true", caps)
	}
}

func TestCapabilitiesCached(t *testing.T) {
	var helpCalls int32
	q, _ := newTestQemuImg(func(_ string, args []string) (Result, error) {
		if len(args) == 1 && args[0] == "-h" {
			atomic.AddInt32(&helpCalls, 1)
			return Result{Stdout: modernHelp}, nil
		}
		return Result{}, nil
	})

	first, err := q.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("first Capabilities() returned error: %v", err)
	}
	second, err := q.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("second Capabilities() returned error: %v", err)
	}
	if first != second {
		t.Fatalf("cached capabilities differ: %+v vs %+v", first, second)
	}
	if got := atomic.LoadInt32(&helpCalls); got != 1 {
		t.Fatalf("-h invocations = %d, want 1", got)
	}
}

func TestCapabilitiesEmptyOutput(t *testing.T) {
	q, _ := newTestQemuImg(func(_ string, _ []string) (Result, error) {
		return Result{}, nil
	})

	_, err := q.Capabilities(context.Background())
	if err == nil {
		t.Fatal("Capabilities() returned no error for empty help output")
	}
	if !errors.Is(err, ErrCapabilitiesUnavailable) {
		t.Fatalf("error %v does not wrap ErrCapabilitiesUnavailable", err)
	}
}

func TestCapabilitiesIgnoresFlagsOutsideCommandLine(t *testing.T) {
	const help = `Command syntax:
  convert [--image-opts] [-B backing_file] [-o options] filename output_filename
  commit [-f fmt] filename
Command parameters:
  -F is accepted by compare and amend only
  -d is accepted by bench only
`
	q, _ := newTestQemuImg(func(_ string, _ []string) (Result, error) {
		return Result{Stdout: help}, nil
	})

	caps, err := q.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() returned error: %v", err)
	}
	if caps.ConvertBackingFormat || caps.CommitSkipEmpty {
		t.Fatalf("Capabilities = %+v, want both false", caps)
	}
}
