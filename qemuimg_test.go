package qemuimg

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGetFormatJSON(t *testing.T) {
	q, runner := newTestQemuImg(func(_ string, args []string) (Result, error) {
		if len(args) >= 2 && args[0] == "info" && args[1] == "--output=json" {
			return Result{Stdout: `{"filename":"/x","format":"qcow2"}`}, nil
		}
		return Result{Stdout: "file format: raw\n"}, nil
	})

	format, err := q.GetFormat(context.Background(), "/x")
	if err != nil {
		t.Fatalf("GetFormat() returned error: %v", err)
	}
	if format != FormatQcow2 {
		t.Fatalf("GetFormat() = %q, want qcow2", format)
	}

	calls := runner.snapshot()
	if len(calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(calls))
	}
	wantArgs := []string{"info", "--output=json", "/x"}
	if got := calls[0].args; len(got) != len(wantArgs) || got[0] != wantArgs[0] || got[1] != wantArgs[1] || got[2] != wantArgs[2] {
		t.Fatalf("args = %#v, want %#v", got, wantArgs)
	}
}

func TestGetFormatTextFallback(t *testing.T) {
	q, _ := newTestQemuImg(func(_ string, args []string) (Result, error) {
		if len(args) >= 2 && args[0] == "info" && args[1] == "--output=json" {
			return Result{Stdout: "this is not json"}, nil
		}
		return Result{Stdout: "image: /x\nfile format: qcow2\nvirtual size: 1 GiB\n"}, nil
	})

	format, err := q.GetFormat(context.Background(), "/x")
	if err != nil {
		t.Fatalf("GetFormat() returned error: %v", err)
	}
	if format != FormatQcow2 {
		t.Fatalf("GetFormat() = %q, want qcow2", format)
	}
}

func TestGetFormatTextIgnoresBackingFormat(t *testing.T) {
	q, _ := newTestQemuImg(func(_ string, args []string) (Result, error) {
		if len(args) >= 2 && args[0] == "info" && args[1] == "--output=json" {
			return Result{Stdout: `{"format":""}`}, nil
		}
		return Result{Stdout: "backing file format: raw\nfile format: qcow2\n"}, nil
	})

	format, err := q.GetFormat(context.Background(), "/overlay")
	if err != nil {
		t.Fatalf("GetFormat() returned error: %v", err)
	}
	if format != FormatQcow2 {
		t.Fatalf("GetFormat() = %q, want qcow2", format)
	}
}

func TestGetFormatEmptyFilename(t *testing.T) {
	q, _ := newTestQemuImg(nil)

	_, err := q.GetFormat(context.Background(), "")
	if !errors.Is(err, ErrEmptyFilename) {
		t.Fatalf("GetFormat() error = %v, want ErrEmptyFilename", err)
	}
}

func TestGetFormatFailure(t *testing.T) {
	q, _ := newTestQemuImg(func(_ string, _ []string) (Result, error) {
		return Result{ExitCode: 1, Stderr: "cannot open image"}, nil
	})

	format, err := q.GetFormat(context.Background(), "/missing")
	if err == nil {
		t.Fatal("GetFormat() returned no error")
	}
	if format != "" {
		t.Fatalf("GetFormat() format = %q, want empty", format)
	}
	if !strings.Contains(err.Error(), "/missing") {
		t.Fatalf("error %q does not mention the filename", err)
	}
}

func TestCreateCommand(t *testing.T) {
	tests := []struct {
		name string
		req  CreateRequest
		want string
	}{
		{
			name: "basic",
			req:  CreateRequest{Filename: "/x", Format: FormatQcow2, Size: "1G"},
			want: "qemu-img create -f qcow2 /x 1G",
		},
		{
			name: "explicit backing format",
			req: CreateRequest{
				Filename:      "/overlay",
				Format:        FormatQcow2,
				BackingFile:   "/base",
				BackingFormat: FormatQcow2,
			},
			want: "qemu-img create -f qcow2 -b /base -F qcow2 /overlay",
		},
		{
			name: "detected backing format",
			req: CreateRequest{
				Filename:    "/overlay",
				Format:      FormatQcow2,
				BackingFile: "/base",
			},
			want: "qemu-img create -f qcow2 -b /base -F qcow2 /overlay",
		},
		{
			name: "options backing format autofill",
			req: CreateRequest{
				Filename: "/x",
				Format:   FormatQcow2,
				Size:     "1G",
				Options:  NewOptions(Option{Key: "backing_file", Value: "/base"}),
			},
			want: "qemu-img create -f qcow2 -o backing_file=/base,backing_fmt=qcow2 /x 1G",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, _ := newTestQemuImg(fixtureHandler(modernHelp))
			command, err := q.CreateCommand(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("CreateCommand() returned error: %v", err)
			}
			if got := command.String(); got != tt.want {
				t.Fatalf("command = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateCommandDoesNotMutateOptions(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(modernHelp))
	options := NewOptions(Option{Key: "backing_file", Value: "/base"})

	_, err := q.CreateCommand(context.Background(), CreateRequest{
		Filename: "/x",
		Format:   FormatQcow2,
		Options:  options,
	})
	if err != nil {
		t.Fatalf("CreateCommand() returned error: %v", err)
	}
	if options.Has("backing_fmt") {
		t.Fatal("CreateCommand mutated the caller's Options")
	}
}

func TestCreateCommandValidation(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(modernHelp))

	if _, err := q.CreateCommand(context.Background(), CreateRequest{Format: FormatQcow2}); !errors.Is(err, ErrEmptyFilename) {
		t.Fatalf("empty filename error = %v, want ErrEmptyFilename", err)
	}
	if _, err := q.CreateCommand(context.Background(), CreateRequest{Filename: "/x", Format: "vmdk"}); !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("invalid format error = %v, want ErrInvalidFormat", err)
	}
	if _, err := q.CreateCommand(context.Background(), CreateRequest{Filename: "/x", BackingFormat: "vmdk"}); !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("invalid backing format error = %v, want ErrInvalidFormat", err)
	}
}

func TestConvertCommand(t *testing.T) {
	tests := []struct {
		name string
		help string
		req  ConvertRequest
		want string
	}{
		{
			name: "modern supports backing format",
			help: modernHelp,
			req: ConvertRequest{
				Filename:       "/in",
				OutputFilename: "/out",
				Format:         FormatQcow2,
				OutputFormat:   FormatQcow2,
				BackingFile:    "/base",
			},
			want: "qemu-img convert -f qcow2 -O qcow2 -B /base -F qcow2 /in /out",
		},
		{
			name: "legacy omits backing format",
			help: legacyHelp,
			req: ConvertRequest{
				Filename:       "/in",
				OutputFilename: "/out",
				Format:         FormatQcow2,
				OutputFormat:   FormatQcow2,
				BackingFile:    "/base",
				BackingFormat:  FormatQcow2,
			},
			want: "qemu-img convert -f qcow2 -O qcow2 -B /base /in /out",
		},
		{
			name: "progress",
			help: modernHelp,
			req: ConvertRequest{
				Filename:       "/in",
				OutputFilename: "/out",
				Format:         FormatRaw,
				OutputFormat:   FormatQcow2,
				Progress:       true,
			},
			want: "qemu-img convert -f raw -O qcow2 -p /in /out",
		},
		{
			name: "output format only",
			help: legacyHelp,
			req: ConvertRequest{
				Filename:       "/in",
				OutputFilename: "/out",
				OutputFormat:   FormatRaw,
			},
			want: "qemu-img convert -O raw /in /out",
		},
		{
			name: "options",
			help: modernHelp,
			req: ConvertRequest{
				Filename:       "/in",
				OutputFilename: "/out",
				Options:        NewOptions(Option{Key: "compat", Value: "1.1"}),
			},
			want: "qemu-img convert -o compat=1.1 /in /out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, _ := newTestQemuImg(fixtureHandler(tt.help))
			command, err := q.ConvertCommand(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("ConvertCommand() returned error: %v", err)
			}
			if got := command.String(); got != tt.want {
				t.Fatalf("command = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertCommandValidation(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(modernHelp))

	if _, err := q.ConvertCommand(context.Background(), ConvertRequest{OutputFilename: "/out"}); !errors.Is(err, ErrEmptyFilename) {
		t.Fatalf("empty filename error = %v, want ErrEmptyFilename", err)
	}
	if _, err := q.ConvertCommand(context.Background(), ConvertRequest{Filename: "/in"}); !errors.Is(err, ErrEmptyOutputFilename) {
		t.Fatalf("empty output filename error = %v, want ErrEmptyOutputFilename", err)
	}
	if _, err := q.ConvertCommand(context.Background(), ConvertRequest{Filename: "/in", OutputFilename: "/out", OutputFormat: "vmdk"}); !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("invalid output format error = %v, want ErrInvalidFormat", err)
	}
}

func TestRebaseCommand(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(modernHelp))

	command, err := q.RebaseCommand(context.Background(), RebaseRequest{
		Filename:      "/overlay",
		BackingFile:   "/newbase",
		BackingFormat: FormatQcow2,
		Format:        FormatQcow2,
		Unsafe:        true,
	})
	if err != nil {
		t.Fatalf("RebaseCommand() returned error: %v", err)
	}
	want := "qemu-img rebase -f qcow2 -F qcow2 -u -b /newbase /overlay"
	if got := command.String(); got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestRebaseCommandDetectsBackingFormat(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(modernHelp))

	command, err := q.RebaseCommand(context.Background(), RebaseRequest{
		Filename:    "/overlay",
		BackingFile: "/newbase",
	})
	if err != nil {
		t.Fatalf("RebaseCommand() returned error: %v", err)
	}
	want := "qemu-img rebase -F qcow2 -b /newbase /overlay"
	if got := command.String(); got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestRebaseCommandValidation(t *testing.T) {
	q, _ := newTestQemuImg(fixtureHandler(modernHelp))

	if _, err := q.RebaseCommand(context.Background(), RebaseRequest{BackingFile: "/base"}); !errors.Is(err, ErrEmptyFilename) {
		t.Fatalf("empty filename error = %v, want ErrEmptyFilename", err)
	}
	if _, err := q.RebaseCommand(context.Background(), RebaseRequest{Filename: "/x"}); !errors.Is(err, ErrEmptyBackingFile) {
		t.Fatalf("empty backing file error = %v, want ErrEmptyBackingFile", err)
	}
}

func TestCommitCommand(t *testing.T) {
	t.Run("skip empty supported", func(t *testing.T) {
		q, _ := newTestQemuImg(fixtureHandler(modernHelp))
		command, err := q.CommitCommand(context.Background(), CommitRequest{
			Filename:  "/overlay",
			Format:    FormatQcow2,
			SkipEmpty: true,
		})
		if err != nil {
			t.Fatalf("CommitCommand() returned error: %v", err)
		}
		want := "qemu-img commit -f qcow2 -d /overlay"
		if got := command.String(); got != want {
			t.Fatalf("command = %q, want %q", got, want)
		}
	})

	t.Run("skip empty unsupported", func(t *testing.T) {
		q, _ := newTestQemuImg(fixtureHandler(legacyHelp))
		_, err := q.CommitCommand(context.Background(), CommitRequest{
			Filename:  "/overlay",
			SkipEmpty: true,
		})
		if !errors.Is(err, ErrSkipEmptyUnsupported) {
			t.Fatalf("error = %v, want ErrSkipEmptyUnsupported", err)
		}
		if err == nil || !strings.Contains(err.Error(), "qemu-img") {
			t.Fatalf("error = %v, want it to mention the binary", err)
		}
	})

	t.Run("image opts with filename", func(t *testing.T) {
		q, _ := newTestQemuImg(fixtureHandler(modernHelp))
		command, err := q.CommitCommand(context.Background(), CommitRequest{
			Filename: "/overlay",
			ImageOpts: NewOptions(
				Option{Key: "driver", Value: "qcow2"},
				Option{Key: "file", Value: "/x"},
			),
		})
		if err != nil {
			t.Fatalf("CommitCommand() returned error: %v", err)
		}
		want := "qemu-img commit --image-opts driver=qcow2,file=/x /overlay"
		if got := command.String(); got != want {
			t.Fatalf("command = %q, want %q", got, want)
		}
	})

	t.Run("image opts without filename", func(t *testing.T) {
		q, _ := newTestQemuImg(fixtureHandler(modernHelp))
		command, err := q.CommitCommand(context.Background(), CommitRequest{
			ImageOpts: NewOptions(Option{Key: "driver", Value: "qcow2"}),
		})
		if err != nil {
			t.Fatalf("CommitCommand() returned error: %v", err)
		}
		want := "qemu-img commit --image-opts driver=qcow2"
		if got := command.String(); got != want {
			t.Fatalf("command = %q, want %q", got, want)
		}
	})

	t.Run("requires a target", func(t *testing.T) {
		q, _ := newTestQemuImg(fixtureHandler(modernHelp))
		_, err := q.CommitCommand(context.Background(), CommitRequest{})
		if !errors.Is(err, ErrEmptyFilename) {
			t.Fatalf("error = %v, want ErrEmptyFilename", err)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		q, _ := newTestQemuImg(fixtureHandler(modernHelp))
		_, err := q.CommitCommand(context.Background(), CommitRequest{Filename: "/x", Format: "vmdk"})
		if !errors.Is(err, ErrInvalidFormat) {
			t.Fatalf("error = %v, want ErrInvalidFormat", err)
		}
	})
}

func TestResizeCommand(t *testing.T) {
	tests := []struct {
		name string
		size string
		want string
	}{
		{name: "bare size is absolute", size: "1024M", want: "qemu-img resize /x 1024M"},
		{name: "explicit shrink", size: "-1G", want: "qemu-img resize /x -1G"},
		{name: "explicit growth", size: "+2G", want: "qemu-img resize /x +2G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, _ := newTestQemuImg(nil)
			command, err := q.ResizeCommand(context.Background(), ResizeRequest{
				Filename: "/x",
				Size:     tt.size,
			})
			if err != nil {
				t.Fatalf("ResizeCommand() returned error: %v", err)
			}
			if got := command.String(); got != tt.want {
				t.Fatalf("command = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResizeCommandValidation(t *testing.T) {
	q, _ := newTestQemuImg(nil)

	if _, err := q.ResizeCommand(context.Background(), ResizeRequest{Size: "1G"}); !errors.Is(err, ErrEmptyFilename) {
		t.Fatalf("empty filename error = %v, want ErrEmptyFilename", err)
	}
	if _, err := q.ResizeCommand(context.Background(), ResizeRequest{Filename: "/x"}); !errors.Is(err, ErrEmptySize) {
		t.Fatalf("empty size error = %v, want ErrEmptySize", err)
	}
}

func TestExecutorReturnsCommandError(t *testing.T) {
	q, _ := newTestQemuImg(func(_ string, _ []string) (Result, error) {
		return Result{ExitCode: 1, Stderr: "boom\n"}, nil
	})

	_, err := q.Create(context.Background(), CreateRequest{Filename: "/x", Format: FormatQcow2})
	if err == nil {
		t.Fatal("Create() returned no error for a non-zero exit")
	}

	var commandErr *CommandError
	if !errors.As(err, &commandErr) {
		t.Fatalf("error type = %T, want *CommandError", err)
	}
	if commandErr.Result.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", commandErr.Result.ExitCode)
	}
	if commandErr.Result.Stderr != "boom\n" {
		t.Fatalf("Stderr = %q, want boom", commandErr.Result.Stderr)
	}
	if commandErr.Command == nil {
		t.Fatal("CommandError.Command is nil")
	}
	message := commandErr.Error()
	if !strings.Contains(message, "exit code 1") {
		t.Fatalf("Error() = %q, want exit code", message)
	}
	if !strings.Contains(message, "boom") {
		t.Fatalf("Error() = %q, want stderr", message)
	}
}

func TestExecutorWrapsRunnerStartError(t *testing.T) {
	startErr := errors.New("start failed")
	q, _ := newTestQemuImg(func(_ string, _ []string) (Result, error) {
		return Result{}, startErr
	})

	_, err := q.Create(context.Background(), CreateRequest{Filename: "/x"})
	if !errors.Is(err, startErr) {
		t.Fatalf("error = %v, want it to wrap %v", err, startErr)
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) {
		t.Fatal("start failure was reported as *CommandError")
	}
}

func TestNewDefaults(t *testing.T) {
	q := New()
	if got := q.Binary(); got != "qemu-img" {
		t.Fatalf("Binary() = %q, want qemu-img", got)
	}

	q = New(WithBinary(""), WithRunner(nil), WithLogger(nil))
	if got := q.Binary(); got != "qemu-img" {
		t.Fatalf("Binary() after empty options = %q, want qemu-img", got)
	}

	q = New(WithBinary("/opt/qemu-img"))
	if got := q.Binary(); got != "/opt/qemu-img" {
		t.Fatalf("Binary() = %q, want /opt/qemu-img", got)
	}

	var nilQemuImg *QemuImg
	if got := nilQemuImg.Binary(); got != "" {
		t.Fatalf("nil Binary() = %q, want empty", got)
	}
}
