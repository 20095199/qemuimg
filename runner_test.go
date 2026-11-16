package qemuimg

import (
	"context"
	"os/exec"
	"testing"
)

func TestLocalRunnerCapturesOutputAndExitCode(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}

	runner := LocalRunner{}
	result, err := runner.Run(context.Background(), sh, "-c", "printf hello")
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if !result.Success() {
		t.Fatalf("Success() = false, exit code %d", result.ExitCode)
	}
	if result.Stdout != "hello" {
		t.Fatalf("Stdout = %q, want %q", result.Stdout, "hello")
	}

	result, err = runner.Run(context.Background(), sh, "-c", "echo oops >&2; exit 7")
	if err != nil {
		t.Fatalf("Run() returned an error for a non-zero exit: %v", err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("ExitCode = %d, want 7", result.ExitCode)
	}
	if result.Success() {
		t.Fatal("Success() = true for exit code 7")
	}
	if result.Stderr != "oops\n" {
		t.Fatalf("Stderr = %q, want %q", result.Stderr, "oops\n")
	}
}

func TestLocalRunnerMissingBinaryReturnsError(t *testing.T) {
	runner := LocalRunner{}
	_, err := runner.Run(context.Background(), "qemuimg-binary-that-does-not-exist", "-h")
	if err == nil {
		t.Fatal("Run() returned no error for a missing binary")
	}
}

func TestLocalRunnerContextCancellationReturnsError(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := LocalRunner{}
	_, err = runner.Run(ctx, sh, "-c", "sleep 1")
	if err == nil {
		t.Fatal("Run() returned no error for a cancelled context")
	}
}
