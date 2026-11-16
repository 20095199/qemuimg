package qemuimg

import (
	"context"
	"fmt"
)

// RebaseRequest 描述一次 qemu-img rebase 操作。
type RebaseRequest struct {
	// Filename 是要修改 backing file 的镜像。
	Filename string
	// BackingFile 是新的 backing file。
	BackingFile string
	// BackingFormat 是新的 backing file 格式。空值会触发自动探测。
	BackingFormat Format
	// Format 是镜像格式。空值表示推断。
	Format Format
	// Unsafe 请求使用 -u 标志，允许不安全的 backing 链。
	Unsafe bool
}

// RebaseCommand 为 req 构建 qemu-img rebase 命令。
//
// rebase 始终需要 backing format，因此在构建参数列表之前会先探测缺失的值。
// 探测失败会直接返回错误，而不是静默生成一条 qemu-img 会拒绝的命令。
func (q *QemuImg) RebaseCommand(ctx context.Context, req RebaseRequest) (*Command, error) {
	if req.Filename == "" {
		return nil, ErrEmptyFilename
	}
	if req.BackingFile == "" {
		return nil, ErrEmptyBackingFile
	}
	if err := req.Format.Validate(); err != nil {
		return nil, err
	}
	if err := req.BackingFormat.Validate(); err != nil {
		return nil, err
	}

	backingFormat := req.BackingFormat
	if backingFormat == "" {
		detected, err := q.GetFormat(ctx, req.BackingFile)
		if err != nil {
			return nil, fmt.Errorf("qemuimg: rebase: detect backing format for %q: %w", req.BackingFile, err)
		}
		if detected == "" {
			return nil, fmt.Errorf("qemuimg: rebase: empty format for backing file %q", req.BackingFile)
		}
		backingFormat = detected
	}

	args := []string{"rebase"}
	if req.Format != "" {
		args = append(args, "-f", req.Format.String())
	}
	args = append(args, "-F", backingFormat.String())
	if req.Unsafe {
		args = append(args, "-u")
	}
	args = append(args, "-b", req.BackingFile, req.Filename)

	return &Command{Binary: q.binary, Args: args}, nil
}

// Rebase 构建并执行一次 qemu-img rebase 操作。
func (q *QemuImg) Rebase(ctx context.Context, req RebaseRequest) (Result, error) {
	command, err := q.RebaseCommand(ctx, req)
	if err != nil {
		return Result{}, err
	}
	return q.run(ctx, command)
}
