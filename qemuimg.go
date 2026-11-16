package qemuimg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// ConfigOption 用于自定义 QemuImg 实例。
type ConfigOption func(*QemuImg)

// WithBinary 覆盖 qemu-img 可执行文件路径。空路径会被忽略，
// 保持默认的 "qemu-img"。
func WithBinary(path string) ConfigOption {
	return func(q *QemuImg) {
		if path != "" {
			q.binary = path
		}
	}
}

// WithRunner 覆盖命令执行逻辑。nil runner 会被忽略，
// 保持默认的 LocalRunner。
func WithRunner(r Runner) ConfigOption {
	return func(q *QemuImg) {
		if r != nil {
			q.runner = r
		}
	}
}

// WithLogger 覆盖日志记录器。nil logger 会被忽略，
// 保持默认的空操作 logger。
func WithLogger(l *slog.Logger) ConfigOption {
	return func(q *QemuImg) {
		if l != nil {
			q.logger = l
		}
	}
}

// QemuImg 执行 qemu-img 操作。
//
// QemuImg 可安全地并发使用。探测到的能力会缓存在互斥锁之后。
type QemuImg struct {
	binary string
	runner Runner
	logger *slog.Logger
	mu     sync.Mutex
	caps   *Capabilities
}

// New 返回一个按给定选项配置的 QemuImg。
//
// 默认值为 "qemu-img" 二进制、LocalRunner 和空操作 slog logger，
// 因此除非调用方提供 logger，否则本包保持静默。
func New(opts ...ConfigOption) *QemuImg {
	q := &QemuImg{
		binary: "qemu-img",
		runner: LocalRunner{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(q)
		}
	}
	return q
}

// Binary 返回配置的 qemu-img 可执行文件路径。
func (q *QemuImg) Binary() string {
	if q == nil {
		return ""
	}
	return q.binary
}

// GetFormat 探测 filename 的磁盘镜像格式。
//
// 它首先让 qemu-img 输出 JSON，并读取其中的 "format" 字段。如果该方式失败
// 或未得到格式，则退回到解析人类可读输出。该回退逻辑刻意忽略
// "backing file format:" 行，以免把 overlay 自身的格式与其父镜像的格式混淆。
func (q *QemuImg) GetFormat(ctx context.Context, filename string) (Format, error) {
	if filename == "" {
		return "", ErrEmptyFilename
	}

	jsonCommand := &Command{Binary: q.binary, Args: []string{"info", "--output=json", filename}}
	jsonResult, jsonErr := q.run(ctx, jsonCommand)
	if jsonErr == nil {
		if format, ok := formatFromJSON(jsonResult.Stdout); ok {
			return format, nil
		}
	}

	textCommand := &Command{Binary: q.binary, Args: []string{"info", filename}}
	textResult, textErr := q.run(ctx, textCommand)
	if textErr == nil {
		if format, ok := formatFromText(textResult.Stdout); ok {
			return format, nil
		}
	}

	reasons := make([]string, 0, 2)
	if jsonErr != nil {
		reasons = append(reasons, fmt.Sprintf("info --output=json: %v", jsonErr))
	} else {
		reasons = append(reasons, "info --output=json: response did not contain a format")
	}
	if textErr != nil {
		reasons = append(reasons, fmt.Sprintf("info: %v", textErr))
	} else {
		reasons = append(reasons, "info: response did not contain a file format line")
	}
	return "", fmt.Errorf("qemuimg: unable to detect format of %q: %s", filename, strings.Join(reasons, "; "))
}

func formatFromJSON(data string) (Format, bool) {
	var payload struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return "", false
	}
	format := strings.TrimSpace(payload.Format)
	if format == "" {
		return "", false
	}
	return Format(format), true
}

func formatFromText(data string) (Format, bool) {
	for _, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "backing file format:") {
			continue
		}
		index := strings.Index(trimmed, "file format:")
		if index < 0 {
			continue
		}
		format := strings.TrimSpace(trimmed[index+len("file format:"):])
		if format != "" {
			return Format(format), true
		}
	}
	return "", false
}

// run 执行 cmd，以 debug 级别记录日志，并把非零退出状态转换为
// *CommandError。Runner 启动失败会带上上下文包装，便于调用方同时检查
// 失败原因和尝试执行的命令。
func (q *QemuImg) run(ctx context.Context, cmd *Command) (Result, error) {
	if cmd == nil {
		return Result{}, errors.New("qemuimg: cannot run a nil command")
	}

	q.logger.Debug("qemu-img command", "cmd", cmd.String())

	result, err := q.runner.Run(ctx, cmd.Binary, cmd.Args...)
	if err != nil {
		q.logger.Error("qemu-img command failed to run", "cmd", cmd.String(), "error", err)
		return result, fmt.Errorf("qemuimg: run %s: %w", cmd.String(), err)
	}
	if !result.Success() {
		q.logger.Error("qemu-img command failed",
			"cmd", cmd.String(),
			"exit_code", result.ExitCode,
			"stderr", result.Stderr,
		)
		return result, &CommandError{Command: cmd, Result: result}
	}
	return result, nil
}
