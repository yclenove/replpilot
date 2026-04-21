package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/config"
	"github.com/yclenove/replpilot/internal/state"
)

func newStatusCmd() *cobra.Command {
	var sourceID string
	var replicaID string
	var timeoutSec int
	var mysqlUser string
	var mysqlPass string

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

			realReplicaID := replicaID
			if realReplicaID == "" {
				realReplicaID = task.ReplicaID
			}
			if realReplicaID == "" {
				return nil
			}

			cfgPath, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			host, ok := cfg.FindHost(realReplicaID)
			if !ok {
				fmt.Printf("replica 主机不存在，跳过远程状态查询: %s\n", realReplicaID)
				return nil
			}
			raw, err := runSSHCommand(host, timeoutSec, buildShowReplicaStatusCmd(mysqlUser, mysqlPass))
			if err != nil {
				fmt.Printf("远程复制状态查询失败: %s\n", raw)
				return nil
			}
			statusMap := parseReplicaStatus(raw)
			if len(statusMap) == 0 {
				fmt.Println("未解析到 SHOW REPLICA STATUS 结果，请确认从库已配置复制。")
				return nil
			}
			fmt.Println("replica_runtime_status:")
			printReplicaPair(statusMap, "Replica_IO_Running", "Slave_IO_Running")
			printReplicaPair(statusMap, "Replica_SQL_Running", "Slave_SQL_Running")
			printReplicaPair(statusMap, "Source_Host", "Master_Host")
			printReplicaPair(statusMap, "Source_User", "Master_User")
			printReplicaPair(statusMap, "Seconds_Behind_Source", "Seconds_Behind_Master")
			printReplicaPair(statusMap, "Last_IO_Error")
			printReplicaPair(statusMap, "Last_SQL_Error")
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceID, "source", "", "主库来源 ID")
	cmd.Flags().StringVar(&replicaID, "replica", "", "从库主机 ID（可选）")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 10, "远程执行超时秒数")
	cmd.Flags().StringVar(&mysqlUser, "mysql-user", "root", "从库本地执行 mysql 的账号")
	cmd.Flags().StringVar(&mysqlPass, "mysql-pass", "", "从库本地 mysql 账号密码（留空表示无密码）")
	return cmd
}

func buildShowReplicaStatusCmd(mysqlUser, mysqlPass string) string {
	base := "mysql -u " + shellEscapeStatus(mysqlUser)
	if mysqlPass != "" {
		base += " -p" + shellEscapeStatus(mysqlPass)
	}
	return base + " -e " + shellEscapeStatus("SHOW REPLICA STATUS\\G")
}

func parseReplicaStatus(raw string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		result[key] = val
	}
	return result
}

func printReplicaPair(m map[string]string, primary string, fallback ...string) {
	keys := append([]string{primary}, fallback...)
	for _, key := range keys {
		if val, ok := m[key]; ok {
			fmt.Printf("  %s=%s\n", primary, val)
			return
		}
	}
}

func shellEscapeStatus(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
