package pod

import (
	"context"
	"fmt"
	"os"
	common_config "server_manager/config"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset     *kubernetes.Clientset
	pod_redis_key = "game_server_wait_use:%s"
)

func init() {
	kubeconfig := os.Getenv("KUBECONFIG")                         // 获取kubeconfig路径，通常在集群外部运行时需要设置此环境变量。
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig) // 构建配置对象。如果是在集群内部，则可以省略kubeconfig。
	if err != nil {
		klog.CtxErrorf(context.Background(), "build config from flags failed, err: %v", err)
		return
	}
	clientset, err = kubernetes.NewForConfig(config) // 创建客户端对象。
	if err != nil {
		panic(err)
	}
}

func create_pod(ctx context.Context, podName string, namespace string, image string, params string) (err error, tcpPort int32, udpPort int32) {
	job, err := clientset.BatchV1().Jobs(namespace).Create(ctx, &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			// Selector: &metav1.LabelSelector{
			// 	MatchLabels: map[string]string{
			// 		"app": podName,
			// 	},
			// },
			TTLSecondsAfterFinished: func() *int32 {
				ttlSecondsAfterFinished := int32(2)
				return &ttlSecondsAfterFinished
			}(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": podName,
					},
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
							Command: []string{"./tank.x86_64", params},
							// Env:     []corev1.EnvVar{},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: func() *int32 {
				backoffLimit := int32(0)
				return &backoffLimit
			}(),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "create job %s failed, err: %v", job.Name, err)
		return err, 0, 0
	}
	klog.CtxInfof(ctx, "Job %v created successfully.\n", job)

	svc, err := clientset.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        podName,
				"auto-clean": "true",                               // 标记为需要自动清理
				"created-at": fmt.Sprintf("%d", time.Now().Unix()), // 创建时间戳
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: job.Spec.Selector.MatchLabels,
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp",
					Protocol: corev1.ProtocolTCP,
					Port:     10085,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 10085,
					},
				},
				{
					Name:     "udp",
					Protocol: corev1.ProtocolUDP,
					Port:     10085,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 10085,
					},
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "create service %s failed, err: %v", svc.Name, err)
		return err, 0, 0
	}
	klog.CtxInfof(ctx, "Service %v created successfully.\n", svc)
	return nil, svc.Spec.Ports[0].NodePort, svc.Spec.Ports[1].NodePort
}

func StartGameServer(ctx context.Context, id int64, params string) (err error, tcpPort int32, udpPort int32) {
	podName := fmt.Sprintf("%s-%d", common_config.Get("pod.game_server.name").(string), id)
	image := common_config.Get("pod.game_server.image").(string)

	err, tcpPort, udpPort = create_pod(ctx, podName, common_config.Get("pod.game_server.namespace").(string), image, params)
	if err != nil {
		klog.CtxErrorf(ctx, "create pod %s failed, err: %v", podName, err)
		return err, 0, 0
	}

	return nil, tcpPort, udpPort
}
