package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/state"
)

func newDiagnoseCmd() *cobra.Command {
	var sourceID string
	var replicaID string

	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "执行复制故障诊断",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sourceID == "" {
				return fmt.Errorf("--source 为必填参数")
			}
			fmt.Printf("[诊断] source=%s replica=%s\n", sourceID, replicaID)
			checks, err := state.LatestPreflightChecks(sourceID, replicaID)
			if err != nil {
				return err
			}
			if len(checks) == 0 {
				fmt.Println("未发现 preflight 记录，建议先执行 preflight。")
			} else {
				fmt.Println("preflight 检查结果：")
				failCount := 0
				for _, item := range checks {
					status := "PASS"
					if !item.OK {
						status = "FAIL"
						failCount++
					}
					fmt.Printf("- [%s] %s => %s\n", status, item.Name, item.Detail)
					if !item.OK {
						for _, tip := range diagnosePreflightFailure(item.Name, item.Detail) {
							fmt.Printf("    建议: %s\n", tip)
						}
					}
				}
				if failCount == 0 {
					fmt.Println("preflight 全部通过。")
				}
			}

			task, err := state.LatestTask(sourceID, replicaID)
			if err != nil {
				return err
			}
			if task == nil {
				fmt.Println("未发现 bootstrap 任务记录。")
				return nil
			}
			fmt.Printf("latest_task=%s status=%s message=%s\n", task.ID, task.Status, task.Message)
			switch task.Status {
			case "success":
				fmt.Println("建议: 可继续执行 status 或开始后续业务验证。")
			case "partial":
				fmt.Println("建议: 当前为半完成状态，请先使用 --dry-run 校验并等待真实复制步骤接入。")
			default:
				fmt.Println("建议: 重新执行 preflight，修复失败项后再 bootstrap。")
				for _, tip := range diagnoseTaskFailure(task.Message) {
					fmt.Printf("  - %s\n", tip)
				}
				if task.RollbackHint != "" {
					fmt.Println("建议回滚SQL:")
					fmt.Printf("  %s\n", task.RollbackHint)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceID, "source", "", "主库来源 ID")
	cmd.Flags().StringVar(&replicaID, "replica", "", "从库主机 ID（可选）")
	return cmd
}

func diagnosePreflightFailure(name, detail string) []string {
	text := strings.ToLower(name + " " + detail)
	tips := make([]string, 0)
	if strings.Contains(text, "repl_pass") {
		tips = append(tips, "补充 `source add --repl-pass` 或更新来源配置后重试。")
	}
	if strings.Contains(text, "ssh") || strings.Contains(text, "connect") {
		tips = append(tips, "确认控制端到从库的 SSH 网络、端口和密钥权限。")
	}
	if strings.Contains(text, "sudo") {
		tips = append(tips, "为 SSH 用户配置 NOPASSWD sudo，或改用具备必要权限的账号。")
	}
	if strings.Contains(text, "mysql") {
		tips = append(tips, "在从库安装 mysql 客户端，并确认 mysql 命令可执行。")
	}
	if strings.Contains(text, "network") || strings.Contains(text, "主从网络连通") {
		tips = append(tips, "放通从库到主库的 3306 端口及安全组/防火墙策略。")
	}
	if len(tips) == 0 {
		tips = append(tips, "根据失败详情先修复环境，再重跑 preflight。")
	}
	return tips
}

func diagnoseTaskFailure(message string) []string {
	text := strings.ToLower(message)
	tips := make([]string, 0)
	if strings.Contains(text, "access denied") {
		tips = append(tips, "检查 --mysql-user/--mysql-pass 是否正确，并确认从库本地登录权限。")
	}
	if strings.Contains(text, "unknown variable") || strings.Contains(text, "syntax") {
		tips = append(tips, "确认 MySQL 版本兼容 `CHANGE REPLICATION SOURCE TO` 语法（MySQL 8.0+）。")
	}
	if strings.Contains(text, "source_auto_position") || strings.Contains(text, "gtid") {
		tips = append(tips, "检查主从 GTID 设置并确保 SOURCE_AUTO_POSITION 场景可用。")
	}
	if strings.Contains(text, "can't connect") || strings.Contains(text, "connection") {
		tips = append(tips, "检查主库地址、端口及网络策略，确认从库能连到主库。")
	}
	if len(tips) == 0 {
		tips = append(tips, "查看任务 message 原始输出，针对 SQL 报错逐条修复后重试。")
	}
	return tips
}
