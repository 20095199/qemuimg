package qemuimg

import (
	"context"
	"testing"
)

func TestCommandStringQuotesArguments(t *testing.T) {
	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "simple",
			cmd:  &Command{Binary: "qemu-img", Args: []string{"create", "/x", "1G"}},
			want: "qemu-img create /x 1G",
		},
		{
			name: "space",
			cmd:  &Command{Binary: "qemu-img", Args: []string{"create", "/tmp/my image", "1G"}},
			want: "qemu-img create '/tmp/my image' 1G",
		},
		{
			name: "single quote",
			cmd:  &Command{Binary: "qemu-img", Args: []string{"create", "it's"}},
			want: `qemu-img create 'it'\''s'`,
		},
		{
			name: "empty argument",
			cmd:  &Command{Binary: "qemu-img", Args: []string{""}},
			want: "qemu-img ''",
		},
		{
			name: "binary with space",
			cmd:  &Command{Binary: "/opt/my tools/qemu-img", Args: []string{"-h"}},
			want: "'/opt/my tools/qemu-img' -h",
		},
		{
			name: "nil command",
			cmd:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommandRunPassesArgumentsSeparately(t *testing.T) {
	runner := &fakeRunner{
		handler: func(_ string, args []string) (Result, error) {
			return Result{Stdout: "ok", ExitCode: 0}, nil
		},
	}
	command := &Command{Binary: "qemu-img", Args: []string{"info", "/x"}}

	result, err := command.Run(context.Background(), runner)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("Stdout = %q, want ok", result.Stdout)
	}

	calls := runner.snapshot()
	if len(calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(calls))
	}
	if calls[0].name != "qemu-img" {
		t.Fatalf("name = %q, want qemu-img", calls[0].name)
	}
	if len(calls[0].args) != 2 || calls[0].args[0] != "info" || calls[0].args[1] != "/x" {
		t.Fatalf("args = %#v, want [info /x]", calls[0].args)
	}
}

func TestCommandRunRejectsNilRunner(t *testing.T) {
	command := &Command{Binary: "qemu-img"}
	if _, err := command.Run(context.Background(), nil); err == nil {
		t.Fatal("Run() with nil runner returned no error")
	}
	var nilCommand *Command
	if _, err := nilCommand.Run(context.Background(), LocalRunner{}); err == nil {
		t.Fatal("nil Command.Run() returned no error")
	}
}
