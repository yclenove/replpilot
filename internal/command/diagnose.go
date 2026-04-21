package command

import (
	"fmt"

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
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceID, "source", "", "主库来源 ID")
	cmd.Flags().StringVar(&replicaID, "replica", "", "从库主机 ID（可选）")
	return cmd
}
