package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yclenove/replpilot/internal/config"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "初始化本地配置目录",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			fmt.Printf("配置初始化完成: %s\n", path)
			return nil
		},
	}
}
