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
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.CtxErrorf(context.Background(), "[POD-INIT-001] failed to build kubernetes config, kubeconfig: %s, error: %v", kubeconfig, err)
		return
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		klog.CtxErrorf(context.Background(), "[POD-INIT-002] failed to create kubernetes clientset, error: %v", err)
		panic(err)
	}
	klog.CtxInfof(context.Background(), "[POD-INIT-003] successfully initialized kubernetes clientset")
}

func create_pod(ctx context.Context, podName string, namespace string, image string, params string) (err error, tcpPort int32, udpPort int32) {
	klog.CtxInfof(ctx, "[POD-CREATE-004] starting pod creation, podName: %s, namespace: %s, image: %s", podName, namespace, image)
	
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
		klog.CtxErrorf(ctx, "[POD-CREATE-005] failed to create job, podName: %s, namespace: %s, error: %v", podName, namespace, err)
		return err, 0, 0
	}
	klog.CtxInfof(ctx, "[POD-CREATE-006] successfully created job, podName: %s, jobName: %s", podName, job.Name)

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
				"auto-clean": "true",
				"created-at": fmt.Sprintf("%d", time.Now().Unix()),
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
		klog.CtxErrorf(ctx, "[POD-CREATE-007] failed to create service, podName: %s, namespace: %s, error: %v", podName, namespace, err)
		return err, 0, 0
	}
	klog.CtxInfof(ctx, "[POD-CREATE-008] successfully created service, podName: %s, serviceName: %s, tcpPort: %d, udpPort: %d", 
		podName, svc.Name, svc.Spec.Ports[0].NodePort, svc.Spec.Ports[1].NodePort)
	
	return nil, svc.Spec.Ports[0].NodePort, svc.Spec.Ports[1].NodePort
}

func StartGameServer(ctx context.Context, id int64, params string) (err error, tcpPort int32, udpPort int32) {
	podName := fmt.Sprintf("%s-%d", common_config.Get("pod.game_server.name").(string), id)
	image := common_config.Get("pod.game_server.image").(string)

	klog.CtxInfof(ctx, "[POD-START-009] starting game server, id: %d, podName: %s, image: %s, params: %s", id, podName, image, params)

	err, tcpPort, udpPort = create_pod(ctx, podName, common_config.Get("pod.game_server.namespace").(string), image, params)
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-START-010] failed to create pod for game server, id: %d, podName: %s, error: %v", id, podName, err)
		return err, 0, 0
	}

	klog.CtxInfof(ctx, "[POD-START-011] successfully started game server, id: %d, podName: %s, tcpPort: %d, udpPort: %d", 
		id, podName, tcpPort, udpPort)
	return nil, tcpPort, udpPort
}

func StartAiClient(ctx context.Context, id int64, params string) (err error, tcpPort int32, udpPort int32) {
	podName := fmt.Sprintf("%s-%d", common_config.Get("pod.ai_client.name").(string), id)
	image := common_config.Get("pod.ai_client.image").(string)

	klog.CtxInfof(ctx, "[POD-START-012] starting ai client, id: %d, podName: %s, image: %s, params: %s", id, podName, image, params)

	err, tcpPort, udpPort = create_pod(ctx, podName, common_config.Get("pod.ai_client.namespace").(string), image, params)
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-START-013] failed to create pod for ai client, id: %d, podName: %s, error: %v", id, podName, err)
		return err, 0, 0
	}

	klog.CtxInfof(ctx, "[POD-START-014] successfully started ai client, id: %d, podName: %s, tcpPort: %d, udpPort: %d", 
		id, podName, tcpPort, udpPort)
	return nil, tcpPort, udpPort
}