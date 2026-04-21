## 2026-04-21
- Summary: 初始化 replpilot 项目骨架，落地 Go+Cobra 命令入口与首批文档。
- Affected: `go.mod`, `cmd/replpilot/main.go`, `internal/command/*`, `README.md`, `docs/架构设计.md`, `docs/开发文档.md`, `docs/开发留痕.md`
- Impact: 项目具备可编译的最小 CLI 结构和文档基线，可继续实现 SSH 编排与主从流程。

## 2026-04-21
- Summary: 安装 Go 并实现 init/host/source 本地配置管理命令，打通配置读写基础能力。
- Affected: `internal/config/store.go`, `internal/command/config_cmd.go`, `internal/command/host.go`, `internal/command/source.go`, `internal/command/root.go`, `README.md`, `docs/开发文档.md`, `docs/开发留痕.md`
- Impact: 工具可在本地初始化配置并维护主机与来源信息，为后续 preflight/ssh 编排提供输入数据。

## 2026-04-21
- Summary: 实现 preflight 初版真实检查，支持按 source/replica 读取配置并执行 SSH/sudo/mysql 可用性验证。
- Affected: `internal/command/preflight.go`, `internal/config/store.go`, `docs/开发文档.md`, `docs/开发留痕.md`
- Impact: 工具已具备主从初始化前的基础可行性检查能力，可在执行 bootstrap 前提前暴露环境问题。

## 2026-04-21
- Summary: 增加 host/source remove 命令，并在 preflight 中补充从库到主库端口连通检查。
- Affected: `internal/command/host.go`, `internal/command/source.go`, `internal/command/preflight.go`, `docs/开发文档.md`, `docs/开发留痕.md`
- Impact: 配置生命周期更完整，且可提前识别主从网络不通导致的初始化失败风险。

## 2026-04-21
- Summary: 打通 bootstrap/status/diagnose 的任务状态链路，并持久化 preflight 结果供诊断复用。
- Affected: `internal/state/task.go`, `internal/command/bootstrap.go`, `internal/command/preflight.go`, `internal/command/status.go`, `internal/command/diagnose.go`, `README.md`, `docs/开发文档.md`, `docs/开发留痕.md`
- Impact: 工具具备基础任务可观测能力，能追踪最近执行状态并输出诊断建议，便于后续接入真实复制执行。

## 2026-04-21
- Summary: 接入 bootstrap 真执行链路，支持通过 SSH 在从库执行 STOP/CHANGE/START REPLICA。
- Affected: `internal/command/bootstrap.go`, `internal/command/ssh_exec.go`, `internal/command/preflight.go`, `internal/command/source.go`, `internal/config/store.go`, `README.md`, `docs/开发文档.md`, `docs/开发留痕.md`
- Impact: 工具从“仅编排可观测”升级为“可触发真实复制配置”，可在受控环境完成主从初始化闭环。
