package qemuimg

import (
	"context"
	"fmt"
)

// CommitRequest 描述一次 qemu-img commit 操作。
type CommitRequest struct {
	// Filename 是要提交的镜像。当改由 ImageOpts 标识镜像时，该字段可以为空。
	Filename string
	// ImageOpts 通过 --image-opts 传入。
	ImageOpts *Options
	// Format 是镜像格式。空值表示推断。
	Format Format
	// SkipEmpty 请求使用 -d，使提交后的镜像保持为空。
	SkipEmpty bool
}

// CommitCommand 为 req 构建 qemu-img commit 命令。
//
// -d 标志仅在较新的 qemu-img 版本中可用。当请求了 SkipEmpty 但不受支持时，
// 会返回包装 ErrSkipEmptyUnsupported 的错误，便于调用方通过 errors.Is
// 识别该情况并选择降级方案。
func (q *QemuImg) CommitCommand(ctx context.Context, req CommitRequest) (*Command, error) {
	if err := req.Format.Validate(); err != nil {
		return nil, err
	}

	imageOpts := ""
	if req.ImageOpts != nil {
		imageOpts = req.ImageOpts.String()
	}
	if req.Filename == "" && imageOpts == "" {
		return nil, fmt.Errorf("%w: commit requires a filename or image options", ErrEmptyFilename)
	}

	skipSupported := false
	if req.SkipEmpty {
		caps, err := q.Capabilities(ctx)
		if err != nil {
			return nil, fmt.Errorf("qemuimg: commit: %w", err)
		}
		if !caps.CommitSkipEmpty {
			return nil, fmt.Errorf("%w (binary: %s)", ErrSkipEmptyUnsupported, q.binary)
		}
		skipSupported = true
	}

	args := []string{"commit"}
	if req.Format != "" {
		args = append(args, "-f", req.Format.String())
	}
	if skipSupported {
		args = append(args, "-d")
	}
	if imageOpts != "" {
		args = append(args, "--image-opts", imageOpts)
	}
	if req.Filename != "" {
		args = append(args, req.Filename)
	}

	return &Command{Binary: q.binary, Args: args}, nil
}

// Commit 构建并执行一次 qemu-img commit 操作。
func (q *QemuImg) Commit(ctx context.Context, req CommitRequest) (Result, error) {
	command, err := q.CommitCommand(ctx, req)
	if err != nil {
		return Result{}, err
	}
	return q.run(ctx, command)
}
