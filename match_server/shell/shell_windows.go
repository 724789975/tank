package shell

import (
	common_config "match_server/config"
	"os"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
)

func StartServer(params string) {
	cmdPath := "C:\\Windows\\System32\\cmd.exe"
	if _, err := os.StartProcess(cmdPath, []string{cmdPath, "/k", common_config.Get("exe_path.windows").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{HideWindow: false},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.Errorf("start cmd %s failed, err: %v", common_config.Get("exe_path.windows").(string), err)
	}

	klog.Infof("start cmd %s success", common_config.Get("exe_path.windows").(string))
}

func StartAiClient(params string) {
	cmdPath := "C:\\Windows\\System32\\cmd.exe"
	if _, err := os.StartProcess(cmdPath, []string{cmdPath, "/k", common_config.Get("exe_path.ai_windows").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{HideWindow: false},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.Errorf("start cmd %s failed, err: %v", common_config.Get("exe_path.ai_windows").(string), err)
	}

	klog.Infof("start cmd %s success", common_config.Get("exe_path.ai_windows").(string))
}
