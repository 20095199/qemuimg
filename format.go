package qemuimg

// Format 标识 qemu-img 支持的一种磁盘镜像格式。
type Format string

const (
	// FormatRaw 是 raw 磁盘镜像格式。
	FormatRaw Format = "raw"
	// FormatQcow2 是 QEMU 写时复制（copy-on-write）第 2 版格式。
	FormatQcow2 Format = "qcow2"
)

// ParseFormat 校验 s 并将其转换为 Format。
//
// 空字符串会被接受并返回 ("", nil)，因为许多 qemu-img 选项是可选的，
// 表示“由 qemu-img 自行推断”。除 "", "raw", "qcow2" 之外的任何值都会
// 返回 *InvalidFormatError。
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case "":
		return "", nil
	case FormatRaw:
		return FormatRaw, nil
	case FormatQcow2:
		return FormatQcow2, nil
	default:
		return "", &InvalidFormatError{Value: s}
	}
}

// Validate 报告 f 是否为受支持的格式之一。
//
// 空的 Format 是合法的，表示“推断”；否则 f 必须是 FormatRaw 或
// FormatQcow2。校验失败会返回包装 ErrInvalidFormat 的
// *InvalidFormatError 值。
func (f Format) Validate() error {
	switch f {
	case "", FormatRaw, FormatQcow2:
		return nil
	default:
		return &InvalidFormatError{Value: string(f)}
	}
}

// String 返回底层的格式名称。
func (f Format) String() string {
	return string(f)
}
