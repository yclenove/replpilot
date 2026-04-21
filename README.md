# replpilot

`replpilot` 是一个 MySQL 主从一键化工具，支持 Linux/Windows/macOS 作为控制端，通过 SSH 远程编排 Linux 目标机完成主从配置。

## 当前阶段

- 已完成 Go + Cobra 最小骨架
- 已提供 `init / host / source / preflight / bootstrap / status / diagnose` 命令
- 已落地任务与预检查结果落盘（`~/.replpilot/tasks.json`、`~/.replpilot/preflight_checks.json`）

## MVP 目标

- 零登录主从机器：操作者只在控制端执行命令
- 三步流程：`preflight -> bootstrap -> status/diagnose`
- 默认支持 MySQL 8.0、GTID、单主多从（Linux 目标机）

## 快速开始

```bash
go mod tidy
go run ./cmd/replpilot --help
go run ./cmd/replpilot init
go run ./cmd/replpilot host add --id r1 --address 10.0.0.12 --user root --key-path ~/.ssh/id_rsa
go run ./cmd/replpilot source add --id prod-master --master-host 10.0.0.10 --repl-user repl --repl-pass 'your_repl_pass'
go run ./cmd/replpilot preflight --source prod-master --replica r1
go run ./cmd/replpilot bootstrap --source prod-master --replica r1 --mode auto --dry-run
go run ./cmd/replpilot bootstrap --source prod-master --replica r1 --mode auto --dry-run=false --force --mysql-user root --mysql-pass 'your_mysql_root_pass'
go run ./cmd/replpilot status --source prod-master --replica r1 --mysql-user root --mysql-pass 'your_mysql_root_pass'
go run ./cmd/replpilot diagnose --source prod-master --replica r1
```

## 文档导航

- [架构设计](docs/架构设计.md)
- [开发文档](docs/开发文档.md)
- [开发留痕](docs/开发留痕.md)
