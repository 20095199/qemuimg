package qemuimg

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// Result 包含运行外部命令的结果。
type Result struct {
	// Stdout 是捕获的标准输出。
	Stdout string
	// Stderr 是捕获的标准错误。
	Stderr string
	// ExitCode 是进程退出状态。成功时为零。
	ExitCode int
}

// Success 报告进程是否以零状态退出。
func (r Result) Success() bool {
	return r.ExitCode == 0
}

// Runner 抽象命令执行，使得测试中可以用 fake 替换 qemu-img。
type Runner interface {
	// Run 使用 args 执行 name，并返回捕获的结果。
	//
	// 实现方只有在进程无法启动或未能完成时（例如二进制不存在、
	// context 被取消）才应返回非 nil 的 error。非零的进程退出状态
	// 通过 Result.ExitCode 报告，此时 error 为 nil。
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

// LocalRunner 使用 os/exec 在本机执行命令。
type LocalRunner struct{}

// Run 使用 os/exec.CommandContext 实现 Runner。
//
// 参数会直接传给进程，不经过 shell。这避免了引号转义问题以及通过文件名
// 进行命令注入。非零退出状态通过 Result.ExitCode 返回且 error 为 nil，
// 而启动失败和 context 取消则作为 error 返回。
func (r LocalRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
	if err == nil {
		return result, nil
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return result, ctxErr
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, err
}
