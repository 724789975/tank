package shell

import (
	"context"
	"fmt"
	common_config "match_server/config"
	"os"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func StartServer(ctx context.Context, id int64, params string) {
	cmdPath := "/bin/bash"
	if _, err := os.StartProcess(cmdPath, []string{"pwd;\n", common_config.Get("exe_path.linux").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.CtxErrorf(ctx, "start cmd %s failed, err: %v", common_config.Get("exe_path.linux").(string), err)
	}

	klog.CtxInfof(ctx, "start cmd success, %s %s", cmdPath, params)
}

func StartAiClient(ctx context.Context, id int64, params string) {
	cmdPath := "/bin/bash"
	if _, err := os.StartProcess(cmdPath, []string{"pwd;\n", common_config.Get("exe_path.ai_linux").(string), params}, &os.ProcAttr{
		Sys:   &syscall.SysProcAttr{},
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		klog.CtxErrorf(ctx, "start cmd %s failed, err: %v", common_config.Get("exe_path.ai_linux").(string), err)
	}

	klog.CtxInfof(ctx, "start cmd %s success", common_config.Get("exe_path.ai_linux").(string))
}

func StartClient(ctx context.Context, id int64, params string) {
	podName := fmt.Sprintf("%s-%d", common_config.Get("pod.game_server.name").(string), id)
	namespace := common_config.Get("pod.game_server.namespace").(string)
	image := common_config.Get("pod.game_server.image").(string)
	kubeconfig := os.Getenv("KUBECONFIG")                         // 获取kubeconfig路径，通常在集群外部运行时需要设置此环境变量。
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig) // 构建配置对象。如果是在集群内部，则可以省略kubeconfig。
	if err != nil {
		klog.CtxErrorf(ctx, "build config from flags failed, err: %v", err)
		return
	}
	clientset, err := kubernetes.NewForConfig(config) // 创建客户端对象。
	if err != nil {
		klog.CtxErrorf(ctx, "new for config failed, err: %v", err)
		return
	}

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            podName,
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 10085,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							ContainerPort: 10085,
							Protocol:      corev1.ProtocolUDP,
						},
					},
					Command: []string{"sh", "-c", "exit 1"},
					Env: []corev1.EnvVar{
						{
							Name:  "PARAMS",
							Value: params,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	_, err = clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "create pod %s failed, err: %v", pod.Name, err)
		return
	}
	klog.CtxInfof(ctx, "Pod %s created successfully.\n", podName)

}
