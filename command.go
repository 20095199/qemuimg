package qemuimg

import (
	"context"
	"errors"
	"strings"
)

// Command 是一条完全解析好的外部命令。
type Command struct {
	// Binary 是要运行的可执行文件，通常是 "qemu-img"。
	Binary string
	// Args 是按顺序传给 Binary 的参数。
	Args []string
}

// String 把命令渲染成适合日志和展示的单行 shell 引用形式。
//
// 即使参数包含空格或 shell 元字符，输出也可以安全地复制到 POSIX shell 中，
// 但本库自身从不执行这个字符串；Run 始终分别传入 Binary 和 Args。
func (c *Command) String() string {
	if c == nil {
		return ""
	}
	parts := make([]string, 0, len(c.Args)+1)
	parts = append(parts, shellQuote(c.Binary))
	for _, arg := range c.Args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

// Run 通过 r 执行该命令。
//
// 命令的二进制和参数会作为独立的值传给 runner，绝不经过 shell。
func (c *Command) Run(ctx context.Context, r Runner) (Result, error) {
	if c == nil {
		return Result{}, errors.New("qemuimg: cannot run a nil command")
	}
	if r == nil {
		return Result{}, errors.New("qemuimg: cannot run a command with a nil runner")
	}
	return r.Run(ctx, c.Binary, c.Args...)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if isShellSafe(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isShellSafe(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("_-./:=@%+,", r):
		default:
			return false
		}
	}
	return true
}
