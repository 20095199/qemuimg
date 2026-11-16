# qemu-img Go 封装库

一个**无第三方依赖**的 Go 库，统一封装 `qemu-img` 的常用操作：`create`、`convert`、`rebase`、`commit`、`resize`，以及镜像格式探测。

业务代码不再需要拼接 `qemu-img` 命令、判断 qemu-img 版本、解析退出码。**即使只有一行 `qemu-img info`，也通过本库调用。**

## 背景

在实际项目中升级 qemu-img 时，曾遇到参数不兼容问题：

- `qemu-img 6+` 的 `convert -B backing_file` 要求同时提供 `-F backing_fmt`；
- `qemu-img 5` 的 `convert` 不接受 `-F`；
- `commit -d` 在部分旧版本中不存在。

当时的调用分散在业务代码各处，版本差异出现后改动难找、代码难看。解决方案是把所有 qemu-img 操作收敛到一个封装层：

- **统一调用，单点适配。** 后续 qemu-img 参数再变化，只改封装层，不改业务代码。
- **参数结构化、可校验。** 格式、文件名、backing file 等参数集中校验，减少低级错误。
- **日志统一。** 每条命令、每次失败都有统一记录。
- **错误统一。** 非零退出码、启动失败、版本不支持等错误可以结构化识别。
- **方便做切面。** 统一调用后，可以集中处理重试、超时、审计、指标等横切逻辑。

## 特性

- 覆盖常用的镜像操作：`create`、`convert`、`rebase`、`commit`、`resize`，以及镜像格式探测。
- **自动适配 qemu-img 版本差异**：`convert` 的 `-F`、`commit` 的 `-d`。
- **命令可预览**：`CreateCommand` / `ConvertCommand` 等返回结构化 `*Command`，执行前即可查看、记录、测试。
- **Runner 抽象**：本地执行、SSH 远程执行、自定义执行器均可替换；版本探测和格式探测也走同一个 Runner。
- **Options 有序**：`-o` / `--image-opts` 参数顺序确定，命令和日志可复现。
- **只依赖 Go 标准库**，要求 Go 1.21+。
- **单元测试不需要安装 qemu-img**；集成测试可选开启。
- **支持 context**：超时、取消会透传到子进程。

## 安装

```bash
go get github.com/20095199/qemuimg
```

## 快速开始

```go
package main

import (
	"context"
	"log"

	"github.com/20095199/qemuimg"
)

func main() {
	ctx := context.Background()
	q := qemuimg.New()

	// 创建 10G 基础镜像
	if _, err := q.Create(ctx, qemuimg.CreateRequest{
		Filename: "/data/base.qcow2",
		Format:   qemuimg.FormatQcow2,
		Size:     "10G",
	}); err != nil {
		log.Fatal(err)
	}

	// 创建 overlay。BackingFormat 留空会自动探测。
	if _, err := q.Create(ctx, qemuimg.CreateRequest{
		Filename:    "/data/overlay.qcow2",
		Format:      qemuimg.FormatQcow2,
		BackingFile: "/data/base.qcow2",
	}); err != nil {
		log.Fatal(err)
	}

	// 查看将执行的命令
	cmd, err := q.CreateCommand(ctx, qemuimg.CreateRequest{
		Filename: "/data/preview.qcow2",
		Format:   qemuimg.FormatQcow2,
		Size:     "1G",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(cmd.String())
	// 输出: qemu-img create -f qcow2 /data/preview.qcow2 1G
}
```

## API 对照

| 操作 | 方法 | 生成命令 |
| --- | --- | --- |
| 创建镜像 | `CreateCommand` / `Create` | `qemu-img create ...` |
| 转换镜像 | `ConvertCommand` / `Convert` | `qemu-img convert ...` |
| 重设 backing | `RebaseCommand` / `Rebase` | `qemu-img rebase ...` |
| 提交 overlay | `CommitCommand` / `Commit` | `qemu-img commit ...` |
| 调整容量 | `ResizeCommand` / `Resize` | `qemu-img resize ...` |
| 探测格式 | `GetFormat` | `qemu-img info ...` |
| 版本能力 | `Capabilities` | `qemu-img -h` |

`*Command` 方法只构建命令，不执行；对应的无 `Command` 后缀方法会构建并通过 Runner 执行。

## 版本自适应

不同 qemu-img 版本的参数差异由 `Capabilities` 统一处理：

```go
caps, err := q.Capabilities(ctx)
if err != nil {
	log.Fatal(err)
}

log.Printf("convert supports -F: %v", caps.ConvertBackingFormat)
log.Printf("commit supports -d: %v", caps.CommitSkipEmpty)
```

检测方式：执行 `qemu-img -h`，解析 `convert` 和 `commit` 的命令语法行，判断是否包含 `-F` / `-d`。结果按 `QemuImg` 实例缓存，避免每次操作都探测。

| 能力 | 检测方式 | 行为 |
| --- | --- | --- |
| `convert -F` | `qemu-img -h` 的 `convert` 行是否含 `-F` | 支持时，`-B` 后自动补 `-F`；不支持时不传 `-F` |
| `commit -d` | `qemu-img -h` 的 `commit` 行是否含 `-d` | 不支持时，`SkipEmpty: true` 返回 `ErrSkipEmptyUnsupported` |

当 `BackingFile` 已提供但 `BackingFormat` 为空时，本库会调用 `GetFormat` 自动探测 backing file 格式。探测失败会返回错误，而不是生成一条现代 qemu-img 会拒绝的命令。

## Options

`-o` 和 `--image-opts` 使用有序的 `Options`，避免 Go map 遍历顺序导致命令不确定：

```go
opts := qemuimg.NewOptions(
	qemuimg.Option{Key: "compat", Value: "1.1"},
	qemuimg.Option{Key: "lazy_refcounts", Value: "on"},
)
// -> -o compat=1.1,lazy_refcounts=on
```

从 map 构造时按键排序：

```go
opts := qemuimg.OptionsFromMap(map[string]string{
	"compat":         "1.1",
	"lazy_refcounts": "on",
})
// -> -o compat=1.1,lazy_refcounts=on
```

如果 `options` 中包含 `backing_file` 但不包含 `backing_fmt`，本库会自动探测并补上 `backing_fmt`：

```go
_, err := q.Convert(ctx, qemuimg.ConvertRequest{
	Filename:       "/data/src.qcow2",
	OutputFilename: "/data/dst.qcow2",
	Format:         qemuimg.FormatQcow2,
	OutputFormat:   qemuimg.FormatQcow2,
	Options: qemuimg.NewOptions(
		qemuimg.Option{Key: "backing_file", Value: "/data/base.qcow2"},
	),
})
// -> qemu-img convert -f qcow2 -O qcow2 -o backing_file=/data/base.qcow2,backing_fmt=qcow2 ...
```

## 错误处理

所有操作返回 `(Result, error)`。命令构建阶段的参数错误直接返回；执行阶段的非零退出码返回 `*CommandError`；进程无法启动（例如二进制不存在、context 取消）返回包装后的原始错误。

```go
_, err := q.Commit(ctx, qemuimg.CommitRequest{
	Filename:  "/data/overlay.qcow2",
	Format:    qemuimg.FormatQcow2,
	SkipEmpty: true,
})

if errors.Is(err, qemuimg.ErrSkipEmptyUnsupported) {
	// 当前 qemu-img 不支持 commit -d，可降级为不带 -d 的提交
}

var cmdErr *qemuimg.CommandError
if errors.As(err, &cmdErr) {
	log.Printf("exit=%d stderr=%s", cmdErr.Result.ExitCode, cmdErr.Result.Stderr)
}
```

常用哨兵错误：

| 错误 | 含义 |
| --- | --- |
| `ErrEmptyFilename` | 缺少镜像文件名 |
| `ErrEmptyOutputFilename` | `convert` 缺少目标文件名 |
| `ErrEmptyBackingFile` | `rebase` 缺少 backing file |
| `ErrEmptySize` | `resize` 缺少容量 |
| `ErrInvalidFormat` | 格式不是 `raw` / `qcow2` |
| `ErrSkipEmptyUnsupported` | 当前 qemu-img 不支持 `commit -d` |
| `ErrCapabilitiesUnavailable` | 版本能力探测失败 |

## 远程执行与自定义 Runner

远程执行通过 `Runner` 接口实现，本地、SSH 或任意执行器都可以替换：

```go
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}
```

约定：只有**进程无法启动**（二进制不存在、context 取消等）才返回非 `nil` error；**非零退出码**通过 `Result.ExitCode` 返回，`error` 为 `nil`，由上层转换为 `*CommandError`。

一个最小的 SSH Runner 示例：

```go
type sshRunner struct {
	host string
}

func (r sshRunner) Run(ctx context.Context, name string, args ...string) (qemuimg.Result, error) {
	remoteArgs := append([]string{r.host, name}, args...)
	cmd := exec.CommandContext(ctx, "ssh", remoteArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := qemuimg.Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	return result, err
}

q := qemuimg.New(qemuimg.WithRunner(sshRunner{host: "192.0.2.10"}))
```

版本探测 `qemu-img -h`、格式探测 `qemu-img info` 和实际操作都会经过同一个 Runner，因此远程主机上必须安装 qemu-img。

## 行为细节

- **`resize` 的容量原样透传。** `"10G"` 是绝对目标容量，`"+2G"` 是相对增长，与 `qemu-img resize` 命令行语义一致。缩减（`"-1G"` 或小于当前容量的绝对值）会被较新的 qemu-img 拒绝，因为它要求 `--shrink` 标志，而本库目前不支持传该标志。
- **日志默认静默。** 命令执行记录在 `log/slog` 的 debug 级别，失败记录在 error 级别；默认 logger 丢弃全部输出，需要时用 `WithLogger` 注入自己的 logger。

## 目录结构

```text
.
├── doc.go            # 包说明
├── format.go         # Format 类型与校验
├── options.go        # 有序 Options
├── runner.go         # Runner 接口与 LocalRunner
├── command.go        # Command 结构、String()、Run()
├── errors.go         # 哨兵错误与 CommandError
├── capabilities.go   # qemu-img 版本能力探测与缓存
├── qemuimg.go        # QemuImg 构造、GetFormat、统一执行
├── create.go         # create
├── convert.go        # convert
├── rebase.go         # rebase
├── commit.go         # commit
├── resize.go         # resize
└── *_test.go         # 单元测试与可选集成测试
```

## 测试

单元测试使用 fake Runner，不需要安装 qemu-img：

```bash
go test ./...
go test -race ./...
```

针对真实 qemu-img 的集成测试默认跳过，需要显式开启：

```bash
QEMUIMG_INTEGRATION=1 go test -run TestIntegration -v ./...
```

集成测试覆盖：创建基础镜像、创建 overlay、探测格式、convert、rebase、commit、resize。测试会根据 `Capabilities` 自动适配当前 qemu-img 版本。

## License

MIT
