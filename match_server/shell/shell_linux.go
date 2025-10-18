package shell

import (
	common_config "match_server/config"
	"os"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
)

func StartCmd(params string) {
	cmdPath := "/bin/bash"
	if _, err := os.StartProcess(cmdPath, []string{"pwd;\n", common_config.Get("exe_path.linux").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.Errorf("start cmd %s failed, err: %v", common_config.Get("exe_path.linux").(string), err)
	}

	klog.Infof("start cmd success, %s %s", cmdPath, params)
}
