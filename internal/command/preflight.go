package command

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/config"
	"github.com/yclenove/replpilot/internal/state"
)

func newPreflightCmd() *cobra.Command {
	var sourceID string
	var replicaID string
	var autoFix bool
	var timeoutSec int

	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "执行主从配置前置检查",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sourceID == "" || replicaID == "" {
				return fmt.Errorf("--source 和 --replica 为必填参数")
			}
			if autoFix {
				fmt.Println("提示: 当前版本暂未实现 --fix 自动修复，将仅输出检查结果。")
			}
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			source, ok := cfg.FindSource(sourceID)
			if !ok {
				return fmt.Errorf("未找到 source 配置: %s", sourceID)
			}
			replica, ok := cfg.FindHost(replicaID)
			if !ok {
				return fmt.Errorf("未找到 replica host 配置: %s", replicaID)
			}

			fmt.Printf("[预检查开始] source=%s replica=%s timeout=%ds\n", sourceID, replicaID, timeoutSec)
			results := []checkResult{
				checkStaticSource(source),
				checkStaticReplica(replica),
				checkRemoteCommand(replica, timeoutSec, "ssh连通", "echo ok"),
				checkRemoteCommand(replica, timeoutSec, "sudo能力", "sudo -n true"),
				checkRemoteCommand(replica, timeoutSec, "mysql客户端", "command -v mysql"),
				checkRemoteCommand(replica, timeoutSec, "mysql版本", "mysql --version"),
				checkRemoteCommand(replica, timeoutSec, "主从网络连通", buildMasterConnectivityCmd(source)),
			}

			failed := 0
			persistChecks := make([]state.PreflightCheck, 0, len(results))
			for _, item := range results {
				status := "PASS"
				if !item.OK {
					status = "FAIL"
					failed++
				}
				fmt.Printf("- [%s] %s", status, item.Name)
				if item.Detail != "" {
					fmt.Printf(" => %s", item.Detail)
				}
				fmt.Println()
				persistChecks = append(persistChecks, state.PreflightCheck{
					SourceID:  sourceID,
					ReplicaID: replicaID,
					Name:      item.Name,
					OK:        item.OK,
					Detail:    item.Detail,
					CheckedAt: time.Now(),
				})
			}
			if err := state.UpsertPreflightChecks(sourceID, replicaID, persistChecks); err != nil {
				return err
			}

			if failed > 0 {
				return fmt.Errorf("预检查未通过，失败项: %d", failed)
			}
			fmt.Println("预检查通过，可继续执行 bootstrap。")
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceID, "source", "", "主库来源 ID")
	cmd.Flags().StringVar(&replicaID, "replica", "", "从库主机 ID")
	cmd.Flags().BoolVar(&autoFix, "fix", false, "自动修复低风险问题")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 8, "单项检查超时秒数")

	return cmd
}

type checkResult struct {
	Name   string
	OK     bool
	Detail string
}

func checkStaticSource(source *config.Source) checkResult {
	if source.MasterHost == "" || source.MasterPort <= 0 || source.ReplUser == "" {
		return checkResult{Name: "source配置完整性", OK: false, Detail: "master_host/master_port/repl_user 不能为空"}
	}
	if source.ReplPass == "" {
		return checkResult{Name: "source配置完整性", OK: false, Detail: "repl_pass 不能为空（用于 bootstrap 真执行）"}
	}
	return checkResult{Name: "source配置完整性", OK: true, Detail: fmt.Sprintf("%s:%d", source.MasterHost, source.MasterPort)}
}

func checkStaticReplica(host *config.Host) checkResult {
	if host.Address == "" || host.Port <= 0 || host.User == "" {
		return checkResult{Name: "replica配置完整性", OK: false, Detail: "address/port/user 不能为空"}
	}
	if host.AuthType == "key" && host.KeyPath == "" {
		return checkResult{Name: "replica配置完整性", OK: false, Detail: "auth-type=key 时必须配置 key_path"}
	}
	return checkResult{Name: "replica配置完整性", OK: true, Detail: fmt.Sprintf("%s@%s:%d", host.User, host.Address, host.Port)}
}

func checkRemoteCommand(host *config.Host, timeoutSec int, name string, remoteCmd string) checkResult {
	detail, err := runSSHCommand(host, timeoutSec, remoteCmd)
	if err != nil {
		return checkResult{Name: name, OK: false, Detail: detail}
	}
	return checkResult{Name: name, OK: true, Detail: detail}
}

func buildMasterConnectivityCmd(source *config.Source) string {
	// 优先用 nc，若不存在再回退到 bash /dev/tcp。
	return fmt.Sprintf("if command -v nc >/dev/null 2>&1; then nc -z -w %d %s %d; else timeout %d bash -lc 'cat < /dev/null > /dev/tcp/%s/%d'; fi && echo ok",
		3, source.MasterHost, source.MasterPort, 3, source.MasterHost, source.MasterPort)
}
