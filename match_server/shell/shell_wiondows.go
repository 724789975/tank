package shell

import (
	"os"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
)

func StartCmd(cmd string) {
	cmdPath := "C:\\Windows\\System32\\cmd.exe"
	if _, err := os.StartProcess(cmdPath, []string{cmdPath, "/k", cmd}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{HideWindow: false},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.Errorf("start cmd %s failed, err: %v", cmd, err)
	}

	klog.Infof("start cmd %s success", cmd)
}
