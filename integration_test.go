package qemuimg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIntegration(t *testing.T) {
	if os.Getenv("QEMUIMG_INTEGRATION") == "" {
		t.Skip("set QEMUIMG_INTEGRATION=1 to run")
	}
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not installed")
	}

	ctx := context.Background()
	q := New()
	dir := t.TempDir()

	base := filepath.Join(dir, "base.qcow2")
	overlay := filepath.Join(dir, "overlay.qcow2")
	commitOverlay := filepath.Join(dir, "commit-overlay.qcow2")
	converted := filepath.Join(dir, "converted.qcow2")

	if _, err := q.Create(ctx, CreateRequest{
		Filename: base,
		Format:   FormatQcow2,
		Size:     "16M",
	}); err != nil {
		t.Fatalf("create base: %v", err)
	}

	format, err := q.GetFormat(ctx, base)
	if err != nil {
		t.Fatalf("get format: %v", err)
	}
	if format != FormatQcow2 {
		t.Fatalf("base format = %q, want qcow2", format)
	}

	if _, err := q.Create(ctx, CreateRequest{
		Filename:    overlay,
		Format:      FormatQcow2,
		BackingFile: base,
	}); err != nil {
		t.Fatalf("create overlay: %v", err)
	}

	if _, err := q.Convert(ctx, ConvertRequest{
		Filename:       base,
		OutputFilename: converted,
		Format:         FormatQcow2,
		OutputFormat:   FormatQcow2,
	}); err != nil {
		t.Fatalf("convert: %v", err)
	}

	if _, err := q.Rebase(ctx, RebaseRequest{
		Filename:      overlay,
		BackingFile:   converted,
		BackingFormat: FormatQcow2,
		Format:        FormatQcow2,
	}); err != nil {
		t.Fatalf("rebase: %v", err)
	}

	caps, err := q.Capabilities(ctx)
	if err != nil {
		t.Fatalf("capabilities: %v", err)
	}

	if _, err := q.Create(ctx, CreateRequest{
		Filename:    commitOverlay,
		Format:      FormatQcow2,
		BackingFile: base,
	}); err != nil {
		t.Fatalf("create commit overlay: %v", err)
	}

	if _, err := q.Commit(ctx, CommitRequest{
		Filename:  commitOverlay,
		Format:    FormatQcow2,
		SkipEmpty: caps.CommitSkipEmpty,
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if _, err := q.Resize(ctx, ResizeRequest{
		Filename: base,
		Size:     "+1M",
	}); err != nil {
		t.Fatalf("resize: %v", err)
	}
}
