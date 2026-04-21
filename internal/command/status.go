package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/state"
)

func newStatusCmd() *cobra.Command {
	var sourceID string
	var replicaID string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "查看复制通道状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sourceID == "" {
				return fmt.Errorf("--source 为必填参数")
			}
			task, err := state.LatestTask(sourceID, replicaID)
			if err != nil {
				return err
			}
			if task == nil {
				fmt.Printf("[状态] source=%s replica=%s\n", sourceID, replicaID)
				fmt.Println("暂无 bootstrap 任务记录")
				return nil
			}
			fmt.Printf("[状态] source=%s replica=%s\n", task.SourceID, task.ReplicaID)
			fmt.Printf("latest_task=%s status=%s mode=%s dry_run=%v\n", task.ID, task.Status, task.Mode, task.DryRun)
			fmt.Printf("updated_at=%s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("message=%s\n", task.Message)
			if len(task.Steps) > 0 {
				fmt.Println("steps:")
				for idx, step := range task.Steps {
					fmt.Printf("  %d. %s\n", idx+1, step)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceID, "source", "", "主库来源 ID")
	cmd.Flags().StringVar(&replicaID, "replica", "", "从库主机 ID（可选）")
	return cmd
}
