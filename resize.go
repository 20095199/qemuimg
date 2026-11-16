package qemuimg

import "context"

// ResizeRequest 描述一次 qemu-img resize 操作。
type ResizeRequest struct {
	// Filename 是要调整容量的镜像。
	Filename string
	// Size 原样传给 qemu-img：开头的 "+" 或 "-" 表示相对当前容量的增长或
	// 缩减，不带符号时是绝对目标容量。
	Size string
}

// ResizeCommand 为 req 构建 qemu-img resize 命令。
func (q *QemuImg) ResizeCommand(_ context.Context, req ResizeRequest) (*Command, error) {
	if req.Filename == "" {
		return nil, ErrEmptyFilename
	}
	if req.Size == "" {
		return nil, ErrEmptySize
	}

	args := []string{"resize", req.Filename, req.Size}
	return &Command{Binary: q.binary, Args: args}, nil
}

// Resize 构建并执行一次 qemu-img resize 操作。
func (q *QemuImg) Resize(ctx context.Context, req ResizeRequest) (Result, error) {
	command, err := q.ResizeCommand(ctx, req)
	if err != nil {
		return Result{}, err
	}
	return q.run(ctx, command)
}
