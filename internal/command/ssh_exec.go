package command

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/yclenove/replpilot/internal/config"
)

func runSSHCommand(host *config.Host, timeoutSec int, remoteCmd string) (string, error) {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=" + strconv.Itoa(timeoutSec),
		"-p", strconv.Itoa(host.Port),
	}
	if host.KeyPath != "" {
		args = append(args, "-i", host.KeyPath)
	}
	args = append(args, host.User+"@"+host.Address, remoteCmd)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec+2)*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	detail := strings.TrimSpace(string(out))
	if detail == "" {
		detail = "(no output)"
	}
	return detail, err
}

