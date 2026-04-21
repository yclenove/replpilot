package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/config"
)

func newSourceCmd() *cobra.Command {
	sourceCmd := &cobra.Command{
		Use:   "source",
		Short: "主库来源管理",
	}
	sourceCmd.AddCommand(newSourceAddCmd(), newSourceListCmd(), newSourceRemoveCmd())
	return sourceCmd
}

func newSourceAddCmd() *cobra.Command {
	var id, masterHost, replUser, replPass string
	var masterPort int

	cmd := &cobra.Command{
		Use:   "add",
		Short: "新增主库来源配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" || masterHost == "" || replUser == "" {
				return fmt.Errorf("--id --master-host --repl-user 为必填参数")
			}
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			for _, item := range cfg.Sources {
				if item.ID == id {
					return fmt.Errorf("来源 ID 已存在: %s", id)
				}
			}
			cfg.Sources = append(cfg.Sources, config.Source{
				ID:         id,
				MasterHost: masterHost,
				MasterPort: masterPort,
				ReplUser:   replUser,
				ReplPass:   replPass,
			})
			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Printf("来源已添加: %s (%s:%d)\n", id, masterHost, masterPort)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "来源 ID")
	cmd.Flags().StringVar(&masterHost, "master-host", "", "主库地址")
	cmd.Flags().IntVar(&masterPort, "master-port", 3306, "主库端口")
	cmd.Flags().StringVar(&replUser, "repl-user", "", "复制账号")
	cmd.Flags().StringVar(&replPass, "repl-pass", "", "复制账号密码（注意: 当前为明文存储）")
	return cmd
}

func newSourceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出主库来源",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			if len(cfg.Sources) == 0 {
				fmt.Println("暂无来源配置")
				return nil
			}
			for _, item := range cfg.Sources {
				fmt.Printf("- id=%s master=%s:%d repl_user=%s has_pass=%v\n", item.ID, item.MasterHost, item.MasterPort, item.ReplUser, item.ReplPass != "")
			}
			return nil
		},
	}
}

func newSourceRemoveCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "删除主库来源配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id 为必填参数")
			}
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			next := make([]config.Source, 0, len(cfg.Sources))
			removed := false
			for _, item := range cfg.Sources {
				if item.ID == id {
					removed = true
					continue
				}
				next = append(next, item)
			}
			if !removed {
				return fmt.Errorf("来源不存在: %s", id)
			}
			cfg.Sources = next
			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Printf("来源已删除: %s\n", id)
			return nil
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&id, "id", "", "来源 ID")
	return cmd
}
