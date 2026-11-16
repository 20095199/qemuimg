package qemuimg

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrEmptyFilename 表示某个需要镜像文件名的操作在未提供文件名时被调用。
	ErrEmptyFilename = errors.New("qemuimg: filename is required")
	// ErrEmptyOutputFilename 表示 convert 在未提供目标文件名时被调用。
	ErrEmptyOutputFilename = errors.New("qemuimg: output filename is required")
	// ErrEmptyBackingFile 表示某个需要 backing file 的操作在未提供时被调用。
	ErrEmptyBackingFile = errors.New("qemuimg: backing file is required")
	// ErrEmptySize 表示 resize 在未提供容量时被调用。
	ErrEmptySize = errors.New("qemuimg: size is required")
	// ErrInvalidFormat 是 InvalidFormatError 所包装的哨兵错误。
	ErrInvalidFormat = errors.New("qemuimg: invalid format")
	// ErrSkipEmptyUnsupported 表示已安装的 qemu-img 不支持 commit -d 标志。
	ErrSkipEmptyUnsupported = errors.New("qemuimg: commit -d is not supported by this qemu-img")
	// ErrCapabilitiesUnavailable 表示 qemu-img 能力探测未能产生可用输出。
	ErrCapabilitiesUnavailable = errors.New("qemuimg: unable to detect qemu-img capabilities")
)

// InvalidFormatError 描述不受支持的磁盘镜像格式。
type InvalidFormatError struct {
	// Value 是被拒绝的格式字符串。
	Value string
}

// Error 实现 error 接口。
func (e *InvalidFormatError) Error() string {
	if e == nil {
		return ErrInvalidFormat.Error()
	}
	return fmt.Sprintf("qemuimg: invalid format %q (supported: raw, qcow2)", e.Value)
}

// Unwrap 返回 ErrInvalidFormat，以便调用方使用 errors.Is 判断。
func (e *InvalidFormatError) Unwrap() error {
	return ErrInvalidFormat
}

// CommandError 描述一次已运行但以非零状态退出的 qemu-img 调用。
type CommandError struct {
	// Command 是失败的命令。
	Command *Command
	// Result 包含捕获的 stdout、stderr 和退出码。
	Result Result
}

// Error 实现 error 接口，在可用时包含退出码、渲染后的命令和捕获的 stderr。
func (e *CommandError) Error() string {
	if e == nil {
		return "qemuimg: command failed"
	}
	command := "<nil>"
	if e.Command != nil {
		command = e.Command.String()
	}
	stderr := strings.TrimSpace(e.Result.Stderr)
	if stderr != "" {
		return fmt.Sprintf("qemuimg: command failed with exit code %d: %s: %s", e.Result.ExitCode, command, stderr)
	}
	return fmt.Sprintf("qemuimg: command failed with exit code %d: %s", e.Result.ExitCode, command)
}
