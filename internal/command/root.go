package command

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "replpilot",
		Short: "MySQL 主从一键化工具（跨端控制，Linux 目标机）",
		Long:  "replpilot 通过 SSH 远程编排 MySQL 主从初始化与诊断，支持 Linux/Windows/macOS 作为控制端。",
	}

	rootCmd.AddCommand(
		newInitCmd(),
		newHostCmd(),
		newSourceCmd(),
		newPreflightCmd(),
		newBootstrapCmd(),
		newStatusCmd(),
		newDiagnoseCmd(),
	)

	return rootCmd
}
