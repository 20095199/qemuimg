// Package qemuimg 提供对 qemu-img 命令行工具的轻量、无第三方依赖封装。
//
// 本包集中处理每一次 qemu-img 调用，使业务代码无需自行拼接参数列表、
// 探测已安装的 qemu-img 版本，或解析原始退出码。与版本相关的差异
// （例如较新的 convert 命令接受 -F backing format 参数，较新的 commit
// 命令接受 -d 参数）在此统一处理一次。
//
// 命令以确定性方式构建，并可在执行前进行检查。所有执行都通过 Runner 完成，
// 因此无需安装真实的 qemu-img 二进制即可轻松测试本包。
package qemuimg
