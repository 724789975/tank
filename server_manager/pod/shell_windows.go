package pod

import (
	"context"
	"os"
	common_config "server_manager/config"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
)

func StartGameServer(ctx context.Context, id int64, params string) (err error, tcpPort int32, udpPort int32) {
	cmdPath := "C:\\Windows\\System32\\cmd.exe"
	if _, err := os.StartProcess(cmdPath, []string{cmdPath, "/k", common_config.Get("exe_path.windows").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{HideWindow: false},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.CtxErrorf(ctx, "start cmd %s failed, err: %v", common_config.Get("exe_path.windows").(string), err)
	}

	klog.CtxInfof(ctx, "start cmd %s success", common_config.Get("exe_path.windows").(string))

	return nil, 0, 0
}

func StartAiClient(ctx context.Context, id int64, params string) (err error, tcpPort int32, udpPort int32) {
	cmdPath := "C:\\Windows\\System32\\cmd.exe"
	if _, err := os.StartProcess(cmdPath, []string{cmdPath, "/k", common_config.Get("exe_path.ai_windows").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{HideWindow: false},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.CtxErrorf(ctx, "start cmd %s failed, err: %v", common_config.Get("exe_path.ai_windows").(string), err)
	}

	klog.CtxInfof(ctx, "start cmd %s success", common_config.Get("exe_path.ai_windows").(string))
	return nil, 0, 0
}
