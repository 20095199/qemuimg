package qemuimg

import (
	"context"
	"fmt"
)

// CreateRequest 描述一次 qemu-img create 操作。
type CreateRequest struct {
	// Filename 是要创建的镜像文件。
	Filename string
	// Format 是镜像格式。空值表示推断。
	Format Format
	// Size 是镜像容量，例如 "10G"。它是可选的。
	Size string
	// BackingFile 是 overlay 的可选 backing file。
	BackingFile string
	// BackingFormat 是 backing file 的格式。空值会触发自动探测。
	BackingFormat Format
	// Options 是额外的 -o 选项。
	Options *Options
}

// CreateCommand 为 req 构建 qemu-img create 命令。
//
// 在补入推断出的 backing_fmt 值之前，会先克隆请求中的 Options，因此
// 绝不会修改调用方持有的数据。缺少 backing format 时会自动探测，
// 因为较新的 qemu-img 版本要求在使用 -b 时同时提供该格式。
func (q *QemuImg) CreateCommand(ctx context.Context, req CreateRequest) (*Command, error) {
	if req.Filename == "" {
		return nil, ErrEmptyFilename
	}
	if err := req.Format.Validate(); err != nil {
		return nil, err
	}
	if err := req.BackingFormat.Validate(); err != nil {
		return nil, err
	}

	backingFormat := req.BackingFormat
	if req.BackingFile != "" && backingFormat == "" {
		detected, err := q.GetFormat(ctx, req.BackingFile)
		if err != nil {
			return nil, fmt.Errorf("qemuimg: create: detect backing format for %q: %w", req.BackingFile, err)
		}
		backingFormat = detected
	}

	options := req.Options.Clone()
	if options.Has("backing_file") && !options.Has("backing_fmt") {
		backingFile, _ := options.Get("backing_file")
		detected, err := q.GetFormat(ctx, backingFile)
		if err != nil {
			return nil, fmt.Errorf("qemuimg: create: detect options backing_file format for %q: %w", backingFile, err)
		}
		if detected == "" {
			return nil, fmt.Errorf("qemuimg: create: empty format for options backing_file %q", backingFile)
		}
		options.Set("backing_fmt", detected.String())
	}

	args := []string{"create"}
	if req.Format != "" {
		args = append(args, "-f", req.Format.String())
	}
	if req.BackingFile != "" {
		args = append(args, "-b", req.BackingFile)
	}
	if backingFormat != "" {
		args = append(args, "-F", backingFormat.String())
	}
	if options.Len() > 0 {
		if rendered := options.String(); rendered != "" {
			args = append(args, "-o", rendered)
		}
	}
	args = append(args, req.Filename)
	if req.Size != "" {
		args = append(args, req.Size)
	}

	return &Command{Binary: q.binary, Args: args}, nil
}

// Create 构建并执行一次 qemu-img create 操作。
func (q *QemuImg) Create(ctx context.Context, req CreateRequest) (Result, error) {
	command, err := q.CreateCommand(ctx, req)
	if err != nil {
		return Result{}, err
	}
	return q.run(ctx, command)
}
