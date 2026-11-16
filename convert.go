package qemuimg

import (
	"context"
	"fmt"
)

// ConvertRequest 描述一次 qemu-img convert 操作。
type ConvertRequest struct {
	// Filename 是源镜像。
	Filename string
	// OutputFilename 是目标镜像。
	OutputFilename string
	// Format 是源镜像格式。空值表示推断。
	Format Format
	// OutputFormat 是目标镜像格式。空值表示推断。
	OutputFormat Format
	// BackingFile 是目标镜像的可选 backing file。
	BackingFile string
	// BackingFormat 是 backing file 的格式。当已安装的 qemu-img 支持 -F 时，
	// 空值会触发自动探测。
	BackingFormat Format
	// Options 是额外的 -o 选项。
	Options *Options
	// Progress 请求使用 -p 进度标志。
	Progress bool
}

// ConvertCommand 为 req 构建 qemu-img convert 命令。
//
// 只有在提供了 backing file 时才检查 Capabilities，因为 -F 标志仅在这种
// 情况下有意义。对于拒绝 -F 的 qemu-img 版本，即使调用方提供了
// BackingFormat 也会省略该标志，以保证命令有效。
func (q *QemuImg) ConvertCommand(ctx context.Context, req ConvertRequest) (*Command, error) {
	if req.Filename == "" {
		return nil, ErrEmptyFilename
	}
	if req.OutputFilename == "" {
		return nil, ErrEmptyOutputFilename
	}
	if err := req.Format.Validate(); err != nil {
		return nil, err
	}
	if err := req.OutputFormat.Validate(); err != nil {
		return nil, err
	}
	if err := req.BackingFormat.Validate(); err != nil {
		return nil, err
	}

	needBackingFormat := false
	if req.BackingFile != "" {
		caps, err := q.Capabilities(ctx)
		if err != nil {
			return nil, fmt.Errorf("qemuimg: convert: %w", err)
		}
		needBackingFormat = caps.ConvertBackingFormat
	}

	backingFormat := req.BackingFormat
	if needBackingFormat && backingFormat == "" {
		detected, err := q.GetFormat(ctx, req.BackingFile)
		if err != nil {
			return nil, fmt.Errorf("qemuimg: convert: detect backing format for %q: %w", req.BackingFile, err)
		}
		if detected == "" {
			return nil, fmt.Errorf("qemuimg: convert: empty format for backing file %q", req.BackingFile)
		}
		backingFormat = detected
	}

	options := req.Options.Clone()
	if options.Has("backing_file") && !options.Has("backing_fmt") {
		backingFile, _ := options.Get("backing_file")
		detected, err := q.GetFormat(ctx, backingFile)
		if err != nil {
			return nil, fmt.Errorf("qemuimg: convert: detect options backing_file format for %q: %w", backingFile, err)
		}
		if detected == "" {
			return nil, fmt.Errorf("qemuimg: convert: empty format for options backing_file %q", backingFile)
		}
		options.Set("backing_fmt", detected.String())
	}

	args := []string{"convert"}
	if req.Format != "" {
		args = append(args, "-f", req.Format.String())
	}
	if req.OutputFormat != "" {
		args = append(args, "-O", req.OutputFormat.String())
	}
	if req.BackingFile != "" {
		args = append(args, "-B", req.BackingFile)
		if needBackingFormat {
			args = append(args, "-F", backingFormat.String())
		}
	}
	if options.Len() > 0 {
		if rendered := options.String(); rendered != "" {
			args = append(args, "-o", rendered)
		}
	}
	if req.Progress {
		args = append(args, "-p")
	}
	args = append(args, req.Filename, req.OutputFilename)

	return &Command{Binary: q.binary, Args: args}, nil
}

// Convert 构建并执行一次 qemu-img convert 操作。
func (q *QemuImg) Convert(ctx context.Context, req ConvertRequest) (Result, error) {
	command, err := q.ConvertCommand(ctx, req)
	if err != nil {
		return Result{}, err
	}
	return q.run(ctx, command)
}
