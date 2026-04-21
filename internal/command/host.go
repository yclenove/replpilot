package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/config"
)

func newHostCmd() *cobra.Command {
	hostCmd := &cobra.Command{
		Use:   "host",
		Short: "主机配置管理",
	}
	hostCmd.AddCommand(newHostAddCmd(), newHostListCmd(), newHostRemoveCmd())
	return hostCmd
}

func newHostAddCmd() *cobra.Command {
	var id, address, user, authType, keyPath string
	var port int

	cmd := &cobra.Command{
		Use:   "add",
		Short: "新增主机配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" || address == "" || user == "" {
				return fmt.Errorf("--id --address --user 为必填参数")
			}
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			for _, item := range cfg.Hosts {
				if item.ID == id {
					return fmt.Errorf("主机 ID 已存在: %s", id)
				}
			}
			cfg.Hosts = append(cfg.Hosts, config.Host{
				ID:       id,
				Address:  address,
				Port:     port,
				User:     user,
				AuthType: authType,
				KeyPath:  keyPath,
			})
			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Printf("主机已添加: %s (%s:%d)\n", id, address, port)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "主机 ID")
	cmd.Flags().StringVar(&address, "address", "", "主机地址")
	cmd.Flags().IntVar(&port, "port", 22, "SSH 端口")
	cmd.Flags().StringVar(&user, "user", "", "SSH 用户")
	cmd.Flags().StringVar(&authType, "auth-type", "key", "认证方式: key|password")
	cmd.Flags().StringVar(&keyPath, "key-path", "", "私钥路径（auth-type=key 时建议填写）")
	return cmd
}

func newHostListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出主机配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			if len(cfg.Hosts) == 0 {
				fmt.Println("暂无主机配置")
				return nil
			}
			for _, item := range cfg.Hosts {
				fmt.Printf("- id=%s address=%s:%d user=%s auth=%s\n", item.ID, item.Address, item.Port, item.User, item.AuthType)
			}
			return nil
		},
	}
}

func newHostRemoveCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "删除主机配置",
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
			next := make([]config.Host, 0, len(cfg.Hosts))
			removed := false
			for _, item := range cfg.Hosts {
				if item.ID == id {
					removed = true
					continue
				}
				next = append(next, item)
			}
			if !removed {
				return fmt.Errorf("主机不存在: %s", id)
			}
			cfg.Hosts = next
			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Printf("主机已删除: %s\n", id)
			return nil
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&id, "id", "", "主机 ID")
	return cmd
}
