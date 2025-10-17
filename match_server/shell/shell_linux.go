package shell

import (
	common_config "match_server/config"
	"os"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
)

func StartCmd(params string) {
	if _, err := os.StartProcess("sh ", []string{common_config.Get("exe_path.linux").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.Errorf("start cmd %s failed, err: %v", common_config.Get("exe_path.linux").(string), err)
	}

	klog.Infof("start cmd %s success", common_config.Get("exe_path.linux").(string))
}
