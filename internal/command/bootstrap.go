package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/config"
	"github.com/yclenove/replpilot/internal/state"
)

func newBootstrapCmd() *cobra.Command {
	var sourceID string
	var replicaID string
	var mode string
	var dryRun bool
	var timeoutSec int
	var mysqlUser string
	var mysqlPass string

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "执行主从一键初始化",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sourceID == "" || replicaID == "" {
				return fmt.Errorf("--source 和 --replica 为必填参数")
			}
			if mode == "" {
				mode = "auto"
			}
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			if _, ok := cfg.FindSource(sourceID); !ok {
				return fmt.Errorf("未找到 source 配置: %s", sourceID)
			}
			if _, ok := cfg.FindHost(replicaID); !ok {
				return fmt.Errorf("未找到 replica host 配置: %s", replicaID)
			}

			now := time.Now()
			task := state.Task{
				ID:        fmt.Sprintf("task-%d", now.UnixNano()),
				SourceID:  sourceID,
				ReplicaID: replicaID,
				Mode:      mode,
				DryRun:    dryRun,
				Status:    "success",
				Steps: []string{
					"载入配置",
					"执行 preflight 建议检查",
					"准备初始化模式",
					"写入复制参数",
					"启动复制并校验状态",
				},
				Message:   "初始化编排已完成（当前为执行骨架，真实复制步骤待接入）",
				CreatedAt: now,
				UpdatedAt: now,
			}
			if !dryRun {
				replica, _ := cfg.FindHost(replicaID)
				source, _ := cfg.FindSource(sourceID)
				remoteSQL := buildReplicationSQL(source)
				remoteCmd := buildReplicaMySQLCommand(mysqlUser, mysqlPass, remoteSQL)
				out, execErr := runSSHCommand(replica, timeoutSec, remoteCmd)
				if execErr != nil {
					task.Status = "failed"
					task.Message = "复制命令执行失败: " + out
				} else {
					task.Status = "success"
					task.Message = "复制命令执行成功，已尝试 STOP/CHANGE/START REPLICA。输出: " + out
				}
			}
			if err := state.AppendTask(task); err != nil {
				return err
			}
			fmt.Printf("[初始化] source=%s replica=%s mode=%s dry_run=%v\n", sourceID, replicaID, mode, dryRun)
			fmt.Printf("任务已创建: %s\n", task.ID)
			for idx, step := range task.Steps {
				fmt.Printf("  %d. %s\n", idx+1, step)
			}
			fmt.Println(task.Message)
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceID, "source", "", "主库来源 ID")
	cmd.Flags().StringVar(&replicaID, "replica", "", "从库主机 ID")
	cmd.Flags().StringVar(&mode, "mode", "auto", "初始化模式：auto|physical|logical")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "仅模拟执行流程，不进行真实变更")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 12, "远程执行超时秒数")
	cmd.Flags().StringVar(&mysqlUser, "mysql-user", "root", "从库本地执行 mysql 的账号")
	cmd.Flags().StringVar(&mysqlPass, "mysql-pass", "", "从库本地 mysql 账号密码（留空表示无密码）")

	return cmd
}

func buildReplicationSQL(source *config.Source) string {
	escapedHost := escapeSQLString(source.MasterHost)
	escapedUser := escapeSQLString(source.ReplUser)
	escapedPass := escapeSQLString(source.ReplPass)
	return fmt.Sprintf(
		"STOP REPLICA; CHANGE REPLICATION SOURCE TO SOURCE_HOST='%s', SOURCE_PORT=%d, SOURCE_USER='%s', SOURCE_PASSWORD='%s', SOURCE_AUTO_POSITION=1; START REPLICA;",
		escapedHost, source.MasterPort, escapedUser, escapedPass,
	)
}

func buildReplicaMySQLCommand(mysqlUser, mysqlPass, sql string) string {
	base := "mysql -u " + shellEscape(mysqlUser)
	if mysqlPass != "" {
		base += " -p" + shellEscape(mysqlPass)
	}
	return base + " -e " + shellEscape(sql)
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
