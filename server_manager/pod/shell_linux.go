package pod

import (
	"context"
	"fmt"
	"os"
	common_config "server_manager/config"
	common_redis "server_manager/redis"
	"strings"

	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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

	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		time.Minute*30,
		informers.WithNamespace("tank"), // 指定监听的命名空间（改为""监听所有）
	)

	podInformer := factory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			klog.CtxInfof(context.Background(), "Pod %s added.\n", pod.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*corev1.Pod)

			pod_name_prefix := ""
			if strings.HasPrefix(pod.Name, common_config.Get("pod.game_server.name").(string)) {
				pod_name_prefix = common_config.Get("pod.game_server.name").(string)
			} else if strings.HasPrefix(pod.Name, common_config.Get("pod.ai_client.name").(string)) {
				pod_name_prefix = common_config.Get("pod.ai_client.name").(string)
			} else {
				return
			}

			redis_key := fmt.Sprintf(pod_redis_key, pod_name_prefix)

			switch pod.Status.Phase {
			case corev1.PodRunning:
				klog.CtxInfof(context.Background(), "Pod %s running.\n", pod.Name)
			case corev1.PodSucceeded:
			case corev1.PodFailed:
				length, err := common_redis.GetRedis().LLen(context.Background(), redis_key).Result()
				if err != nil {
					klog.CtxErrorf(context.Background(), "get redis len failed, err: %v", err)
					return
				}
				if length < 10 {
					common_redis.GetRedis().LPush(context.Background(), redis_key, pod.Name)
				} else {
					uninstall_pod(context.Background(), pod.Name, pod.Namespace)
				}
			default:
				klog.CtxInfof(context.Background(), "Pod %s updated.\n", pod.Name)
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			klog.CtxInfof(context.Background(), "Pod %s deleted.\n", pod.Name)
		},
	})

	ch := make(chan struct{})
	defer close(ch)

	factory.Start(ch)
	if !cache.WaitForCacheSync(ch, podInformer.HasSynced) {
		panic("wait for cache sync failed")
	}
}

func create_pod(ctx context.Context, podName string, namespace string, image string, params string) (err error, tcpPort int32, udpPort int32) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 {
				replicas := int32(1)
				return &replicas
			}(),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": podName,
				},
			},
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
		},
	}, metav1.CreateOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "create deployment %s failed, err: %v", deployment.Name, err)
		return err, 0, 0
	}
	klog.CtxInfof(ctx, "Deployment %v created successfully.\n", deployment)

	svc, err := clientset.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": podName,
			},
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

func delete_pod(ctx context.Context, podName string, namespace string) (err error, tcpPort int32, udpPort int32) {
	err = clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "delete pod %s failed, err: %v", podName, err)
		return err, 0, 0
	}
	klog.CtxInfof(ctx, "Pod %v deleted successfully.\n", podName)
	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "get service %s failed, err: %v", podName, err)
		return err, 0, 0
	}
	klog.CtxInfof(ctx, "Service %v deleted successfully.\n", svc)
	return nil, svc.Spec.Ports[0].NodePort, svc.Spec.Ports[1].NodePort
}

func uninstall_pod(ctx context.Context, podName string, namespace string) (err error) {
	err = clientset.AppsV1().Deployments(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "delete deployment %s failed, err: %v", podName, err)
		return err
	}
	klog.CtxInfof(ctx, "Deployment %v deleted successfully.\n", podName)

	err = clientset.CoreV1().Services(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		klog.CtxErrorf(ctx, "delete service %s failed, err: %v", podName, err)
		return err
	}
	klog.CtxInfof(ctx, "Service %v deleted successfully.\n", podName)
	return nil
}

func StartGameServer(ctx context.Context, id int64, params string) (err error, tcpPort int32, udpPort int32) {
	podName := fmt.Sprintf("%s-%d", common_config.Get("pod.game_server.name").(string), id)
	image := common_config.Get("pod.game_server.image").(string)

	redis_key := fmt.Sprintf(pod_redis_key, common_config.Get("pod.game_server.namespace").(string))
	if common_redis.GetRedis().LLen(context.Background(), redis_key).Val() > 0 {
		podName = common_redis.GetRedis().RPop(context.Background(), redis_key).Val()

		err, tcpPort, udpPort = delete_pod(ctx, podName, common_config.Get("pod.game_server.namespace").(string))
		if err != nil {
			klog.CtxErrorf(ctx, "delete pod %s failed, err: %v", podName, err)
			return err, 0, 0
		}

		return err, tcpPort, udpPort
	}

	err, tcpPort, udpPort = create_pod(ctx, podName, common_config.Get("pod.game_server.namespace").(string), image, params)
	if err != nil {
		klog.CtxErrorf(ctx, "create pod %s failed, err: %v", podName, err)
		return err, 0, 0
	}

	return nil, tcpPort, udpPort
}
