package qemuimg

import (
	"context"
	"sync"
)

type runnerCall struct {
	name string
	args []string
}

type fakeRunner struct {
	mu      sync.Mutex
	calls   []runnerCall
	handler func(name string, args []string) (Result, error)
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, runnerCall{name: name, args: append([]string(nil), args...)})
	handler := f.handler
	f.mu.Unlock()

	if handler == nil {
		return Result{}, nil
	}
	return handler(name, args)
}

func (f *fakeRunner) snapshot() []runnerCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := make([]runnerCall, len(f.calls))
	copy(calls, f.calls)
	return calls
}

const modernHelp = `qemu-img version 8.2.0
usage: qemu-img [standard options] command [command options]
Command syntax:
  convert [--object objectdef] [--image-opts] [-f fmt] [-O output_fmt] [-B backing_file [-F backing_fmt]] [-o options] filename output_filename
  commit [--object objectdef] [--image-opts] [-q] [-f fmt] [-t cache] [-b base] [-d] [-p] filename
  create [--object objectdef] [-q] [-f fmt] [-b backing_file] [-F backing_fmt] [-o options] filename [size]
`

const legacyHelp = `qemu-img version 5.0.0
usage: qemu-img [standard options] command [command options]
Command syntax:
  convert [--object objectdef] [--image-opts] [-f fmt] [-O output_fmt] [-B backing_file] [-o options] filename output_filename
  commit [--object objectdef] [--image-opts] [-q] [-f fmt] [-t cache] [-b base] [-p] filename
  create [--object objectdef] [-q] [-f fmt] [-b backing_file] [-F backing_fmt] [-o options] filename [size]
`

func fixtureHandler(help string) func(string, []string) (Result, error) {
	return func(_ string, args []string) (Result, error) {
		switch {
		case len(args) == 1 && args[0] == "-h":
			return Result{Stdout: help}, nil
		case len(args) >= 2 && args[0] == "info" && args[1] == "--output=json":
			return Result{Stdout: `{"format":"qcow2","virtual-size":16777216}`}, nil
		case len(args) >= 1 && args[0] == "info":
			return Result{Stdout: "image: /x\nfile format: qcow2\n"}, nil
		default:
			return Result{}, nil
		}
	}
}

func newTestQemuImg(handler func(string, []string) (Result, error)) (*QemuImg, *fakeRunner) {
	runner := &fakeRunner{handler: handler}
	return New(WithRunner(runner)), runner
}
