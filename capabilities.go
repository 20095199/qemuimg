package qemuimg

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// Capabilities 描述不同 qemu-img 版本之间存在差异的可选特性。
type Capabilities struct {
	// ConvertBackingFormat 表示 convert 是否接受 -F。较新的 qemu-img 版本
	// 在使用 -B 时要求提供 backing format，而 qemu-img 5 完全拒绝 -F。
	ConvertBackingFormat bool
	// CommitSkipEmpty 表示 commit 是否接受 -d。
	CommitSkipEmpty bool
}

var (
	convertBackingFormatFlag = regexp.MustCompile(`(^|\s)\[?-F(\s|\]|$)`)
	commitSkipEmptyFlag      = regexp.MustCompile(`(^|\s)\[?-d(\s|\]|$)`)
)

// Capabilities 探测并缓存所配置 qemu-img 二进制支持的可选特性。
//
// 探测过程解析 "qemu-img -h" 打印的 "Command syntax" 行。qemu-img 通常把帮助
// 写到 stdout，但某些构建或包装脚本可能写到 stderr，因此两个流都会被检查。
// 结果会被缓存，因为已安装的二进制在进程生命周期内不会改变，且探测输出很少变化。
func (q *QemuImg) Capabilities(ctx context.Context) (Capabilities, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.caps != nil {
		return *q.caps, nil
	}

	result, err := q.runner.Run(ctx, q.binary, "-h")
	if err != nil {
		return Capabilities{}, fmt.Errorf("qemuimg: detect capabilities with %q -h: %w", q.binary, err)
	}

	output := result.Stdout + "\n" + result.Stderr
	if strings.TrimSpace(output) == "" {
		return Capabilities{}, fmt.Errorf("%w: %q -h produced no output", ErrCapabilitiesUnavailable, q.binary)
	}

	caps := Capabilities{
		ConvertBackingFormat: syntaxLineSupportsFlag(output, "convert", convertBackingFormatFlag),
		CommitSkipEmpty:      syntaxLineSupportsFlag(output, "commit", commitSkipEmptyFlag),
	}
	q.caps = &caps
	return caps, nil
}

func syntaxLineSupportsFlag(help, command string, flag *regexp.Regexp) bool {
	for _, line := range strings.Split(help, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, command+" ") || strings.HasPrefix(trimmed, command+"\t") {
			return flag.MatchString(trimmed)
		}
	}
	return false
}
