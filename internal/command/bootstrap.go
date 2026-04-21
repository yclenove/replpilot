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
	var force bool

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
				if !force {
					return fmt.Errorf("高风险操作：真实写入复制配置前请显式传入 --force")
				}
				replica, _ := cfg.FindHost(replicaID)
				source, _ := cfg.FindSource(sourceID)
				preOut, _ := runSSHCommand(replica, timeoutSec, buildShowReplicaStatusCmd(mysqlUser, mysqlPass))
				task.PreStatus = preOut
				remoteSQL := buildReplicationSQL(source)
				remoteCmd := buildReplicaMySQLCommand(mysqlUser, mysqlPass, remoteSQL)
				out, execErr := runSSHCommand(replica, timeoutSec, remoteCmd)
				if execErr != nil {
					task.Status = "failed"
					task.Message = "复制命令执行失败: " + out
					task.RollbackHint = buildRollbackHintFromStatus(preOut)
				} else {
					task.Status = "success"
					task.Message = "复制命令执行成功，已尝试 STOP/CHANGE/START REPLICA。输出: " + out
					postOut, _ := runSSHCommand(replica, timeoutSec, buildShowReplicaStatusCmd(mysqlUser, mysqlPass))
					task.PostStatus = postOut
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
			if task.RollbackHint != "" {
				fmt.Println("回滚建议SQL:")
				fmt.Println(task.RollbackHint)
			}
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
	cmd.Flags().BoolVar(&force, "force", false, "确认执行真实复制变更（仅 dry-run=false 时生效）")

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

func buildRollbackHintFromStatus(preStatus string) string {
	m := parseReplicaStatus(preStatus)
	if len(m) == 0 {
		return "未能从执行前状态生成回滚 SQL，请人工核对后执行 CHANGE REPLICATION SOURCE TO。"
	}
	host := chooseField(m, "Source_Host", "Master_Host")
	user := chooseField(m, "Source_User", "Master_User")
	logFile := chooseField(m, "Source_Log_File", "Master_Log_File")
	logPos := chooseField(m, "Read_Source_Log_Pos", "Read_Master_Log_Pos")
	autoPos := chooseField(m, "Auto_Position")
	if host == "" || user == "" {
		return "执行前未检测到完整复制源信息，请人工回滚。"
	}
	if autoPos == "1" {
		return fmt.Sprintf("STOP REPLICA; CHANGE REPLICATION SOURCE TO SOURCE_HOST='%s', SOURCE_USER='%s', SOURCE_AUTO_POSITION=1; START REPLICA;", escapeSQLString(host), escapeSQLString(user))
	}
	if logFile != "" && logPos != "" {
		return fmt.Sprintf("STOP REPLICA; CHANGE REPLICATION SOURCE TO SOURCE_HOST='%s', SOURCE_USER='%s', SOURCE_LOG_FILE='%s', SOURCE_LOG_POS=%s; START REPLICA;", escapeSQLString(host), escapeSQLString(user), escapeSQLString(logFile), logPos)
	}
	return "执行前状态缺少日志位点，建议根据历史配置手工回滚 CHANGE REPLICATION SOURCE TO。"
}

func chooseField(m map[string]string, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}
