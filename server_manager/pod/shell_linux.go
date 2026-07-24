package pod

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	common_config "server_manager/config"
	"strings"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// gameServerPort 与 k8s Job/Service 中暴露的游戏服务端口保持一致
	gameServerPort = 10085
	// deployEnvTest2 表示 test2 环境, 该环境使用 Docker 容器化部署而非 k8s
	deployEnvTest2 = "test2"
	// defaultDockerNetwork docker-compose 默认网络名(未配置 pod.docker.network 时使用)
	defaultDockerNetwork = "tank-net"
)

var (
	clientset *kubernetes.Clientset
)

// isDockerDeploy 判断当前是否为 Docker 容器化部署环境.
// 当 DEPLOYENV 环境变量为 "test2" 时, 使用 Docker 启动 game-server / ai-client,
// 其它环境仍然使用 k8s API 部署.
func isDockerDeploy() bool {
	return os.Getenv("DEPLOYENV") == deployEnvTest2
}

// dockerNetwork 返回容器需要加入的 docker 网络, 保证 game-server / ai-client
// 可以访问 etcd / redis 等基础设施服务.
func dockerNetwork() string {
	if v, ok := common_config.Get("pod.docker.network").(string); ok && v != "" {
		return v
	}
	return defaultDockerNetwork
}

// allocateHostPort 在宿主机上申请一个空闲 TCP 端口, 用于将容器内的
// gameServerPort 映射到宿主机, 类似 k8s NodePort 的作用.
func allocateHostPort() (int32, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return int32(l.Addr().(*net.TCPAddr).Port), nil
}

func init() {
	// test2 环境通过 Docker 部署, 无需初始化 k8s 客户端
	if isDockerDeploy() {
		klog.CtxInfof(context.Background(), "[POD-INIT-000] docker deploy mode (DEPLOYENV=test2), skip kubernetes clientset init")
		return
	}

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

func create_job(ctx context.Context, podName string, namespace string, image string, params []string) (err error, job *batchv1.Job) {
	klog.CtxInfof(ctx, "[POD-CREATE-004] starting pod creation, podName: %s, namespace: %s, image: %s", podName, namespace, image)

	job, err = clientset.BatchV1().Jobs(namespace).Create(ctx, &batchv1.Job{
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
			ActiveDeadlineSeconds: func() *int64 {
				activeDeadlineSeconds := int64(15 * 60)
				return &activeDeadlineSeconds
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
							Command: func() []string {
								ret := []string{"./tank.x86_64"}
								return append(ret, params...)
							}(),
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
	}
	klog.CtxInfof(ctx, "[POD-CREATE-006] successfully created job, podName: %s, jobName: %s", podName, job.Name)

	return err, job
}

func create_svc(ctx context.Context, job *batchv1.Job) (err error, svc *corev1.Service) {
	namespace := job.Namespace
	podName := job.Name
	svc, err = clientset.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
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
		return err, svc
	}

	// K8s 默认为 TCP / UDP 端口各分配不同的随机 NodePort, 但客户端只会拿到一个 GamePort(TCP NodePort).
	// 客户端已改用 UDP 连接, 若仍按 TCP 的 NodePort 下发, UDP 报文会打到错误端口而连不上.
	// 这里把 UDP 端口的 NodePort 对齐到 TCP 端口(同一 Service 内不同协议允许共享同一 NodePort),
	// 使 TCP/UDP 对外发布在同一端口号, 客户端无论用哪种协议都能用同一 GamePort 连接.
	if len(svc.Spec.Ports) >= 2 && svc.Spec.Ports[0].NodePort != svc.Spec.Ports[1].NodePort {
		tcpNodePort := svc.Spec.Ports[0].NodePort
		svc.Spec.Ports[1].NodePort = tcpNodePort
		updated, uerr := clientset.CoreV1().Services(namespace).Update(ctx, svc, metav1.UpdateOptions{})
		if uerr != nil {
			klog.CtxErrorf(ctx, "[POD-CREATE-009] failed to align udp nodePort to tcp nodePort, podName: %s, tcpNodePort: %d, error: %v", podName, tcpNodePort, uerr)
			// Update 失败时 Service 已创建成功, 若不清理会残留并占用 NodePort,
			// 还会让同名 podName 的后续 Create 因名称冲突而持续失败, 故 best-effort 删除.
			if derr := clientset.CoreV1().Services(namespace).Delete(ctx, svc.Name, metav1.DeleteOptions{}); derr != nil {
				klog.CtxErrorf(ctx, "[POD-CREATE-010] failed to cleanup service after update failure, podName: %s, error: %v", podName, derr)
			} else {
				klog.CtxInfof(ctx, "[POD-CREATE-011] cleaned up service after update failure, podName: %s", podName)
			}
			return uerr, nil
		}
		svc = updated
	}

	klog.CtxInfof(ctx, "[POD-CREATE-008] successfully created service, podName: %s, serviceName: %s, clusterIP: %s, tcpPort: %d, udpPort: %d",
		podName, svc.Name, svc.Spec.ClusterIP, svc.Spec.Ports[0].NodePort, svc.Spec.Ports[1].NodePort)

	return err, svc
}

// startGameServerDocker 在 test2 环境下通过 Docker 启动 game-server 容器实例.
// 将容器内 gameServerPort 同时以 TCP/UDP 发布到宿主机的同一空闲端口,
// 供真人客户端(宿主机原生)通过 127.0.0.1:hostPort 访问.
// 返回的 addr 仅用于 ai-client(同处 docker 网络的容器), 故返回容器名 podName,
// 让 ai-client 通过 docker 内嵌 DNS 解析容器名, 直连 game-server 容器内部端口.
func startGameServerDocker(ctx context.Context, podName string, image string, params []string) (err error, addr string, tcpPort int32, udpPort int32) {
	hostPort, err := allocateHostPort()
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-DOCKER-001] failed to allocate host port for game server, podName: %s, error: %v", podName, err)
		return err, "", 0, 0
	}

	args := []string{
		"run", "-d",
		"--name", podName,
		"--network", dockerNetwork(),
		"--label", "auto-clean=true",
		// 传递 DEPLOYENV, 使容器内 Unity server 选择对应的 CommandLine-<env>.txt 参数文件
		"-e", "DEPLOYENV=" + os.Getenv("DEPLOYENV"),
		"-p", fmt.Sprintf("%d:%d/tcp", hostPort, gameServerPort),
		"-p", fmt.Sprintf("%d:%d/udp", hostPort, gameServerPort),
		image,
		"./tank.x86_64",
	}
	args = append(args, params...)

	klog.CtxInfof(ctx, "[POD-DOCKER-002] starting game server via docker, podName: %s, image: %s, hostPort: %d, args: %v", podName, image, hostPort, args)

	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-DOCKER-003] failed to start game server container, podName: %s, error: %v, output: %s", podName, err, string(out))
		return err, "", 0, 0
	}

	// addr 仅供 ai-client(同一 docker 网络内的容器)使用, 返回 game-server 容器名,
	// ai-client 通过 docker 内嵌 DNS 解析容器名并连接容器内部端口(gameServerPort);
	// 真人客户端不使用此 addr(其地址来自 match-server 的 game.addr 配置).
	addr = podName
	klog.CtxInfof(ctx, "[POD-DOCKER-004] successfully started game server container, podName: %s, containerID: %s, addr(for ai-client): %s, hostPort(for player): %d",
		podName, strings.TrimSpace(string(out)), addr, hostPort)
	// tcpPort/udpPort 返回宿主机发布端口(hostPort), 供真人客户端经 127.0.0.1:hostPort 连接.
	return nil, addr, hostPort, hostPort
}

// startAiClientDocker 在 test2 环境下通过 Docker 启动 ai-client 容器实例.
// ai-client 作为客户端主动连接游戏服务, 无需对外暴露端口,
// 运行结束后自动清理(--rm).
func startAiClientDocker(ctx context.Context, podName string, image string, params []string) (err error) {
	args := []string{
		"run", "-d", "--rm",
		"--name", podName,
		"--network", dockerNetwork(),
		"--label", "auto-clean=true",
		// 传递 DEPLOYENV, 使容器内 Unity 客户端选择对应的 CommandLine-<env>.txt 参数文件
		"-e", "DEPLOYENV=" + os.Getenv("DEPLOYENV"),
		image,
		"./tank.x86_64",
	}
	args = append(args, params...)

	klog.CtxInfof(ctx, "[POD-DOCKER-005] starting ai client via docker, podName: %s, image: %s, args: %v", podName, image, args)

	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-DOCKER-006] failed to start ai client container, podName: %s, error: %v, output: %s", podName, err, string(out))
		return err
	}

	klog.CtxInfof(ctx, "[POD-DOCKER-007] successfully started ai client container, podName: %s, containerID: %s", podName, strings.TrimSpace(string(out)))
	return nil
}

func StartGameServer(ctx context.Context, id int64, params []string) (err error, clusterIP string, tcpPort int32, udpPort int32) {
	podName := fmt.Sprintf("%s-%d", common_config.Get("pod.game_server.name").(string), id)
	image := common_config.Get("pod.game_server.image").(string)

	params = append(params, "-service_name")
	params = append(params, podName)

	klog.CtxInfof(ctx, "[POD-START-009] starting game server, id: %d, podName: %s, image: %s, params: %v", id, podName, image, params)

	// test2 环境使用 Docker 容器化部署, 非 test2 仍使用 k8s API
	if isDockerDeploy() {
		klog.CtxInfof(ctx, "[POD-START-009-DOCKER] DEPLOYENV=test2, deploy game server via docker, id: %d, podName: %s", id, podName)
		return startGameServerDocker(ctx, podName, image, params)
	}

	err, job := create_job(ctx, podName, common_config.Get("pod.game_server.namespace").(string), image, params)
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-START-010] failed to create job for game server, id: %d, podName: %s, error: %v", id, podName, err)
		return err, "", 0, 0
	}

	err, svc := create_svc(ctx, job)
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-START-011] failed to create service for game server, id: %d, podName: %s, error: %v", id, podName, err)
		return err, "", 0, 0
	}

	klog.CtxInfof(ctx, "[POD-START-011] successfully started game server, id: %d, podName: %s, clusterIP: %s, tcpPort: %d, udpPort: %d",
		id, podName, svc.Spec.ClusterIP, svc.Spec.Ports[0].NodePort, svc.Spec.Ports[1].NodePort)
	return err, svc.Spec.ClusterIP, svc.Spec.Ports[0].NodePort, svc.Spec.Ports[1].NodePort
}

func StartAiClient(ctx context.Context, id int64, params []string) (err error, clusterIP string, tcpPort int32, udpPort int32) {
	podName := fmt.Sprintf("%s-%d", common_config.Get("pod.ai_client.name").(string), id)
	image := common_config.Get("pod.ai_client.image").(string)

	klog.CtxInfof(ctx, "[POD-START-012] starting ai client, id: %d, podName: %s, image: %s, params: %s", id, podName, image, params)

	// test2 环境使用 Docker 容器化部署, 非 test2 仍使用 k8s API
	if isDockerDeploy() {
		klog.CtxInfof(ctx, "[POD-START-012-DOCKER] DEPLOYENV=test2, deploy ai client via docker, id: %d, podName: %s", id, podName)
		return startAiClientDocker(ctx, podName, image, params), "", 0, 0
	}

	err, _ = create_job(ctx, podName, common_config.Get("pod.ai_client.namespace").(string), image, params)
	if err != nil {
		klog.CtxErrorf(ctx, "[POD-START-013] failed to create job for ai client, id: %d, podName: %s, error: %v", id, podName, err)
	}

	return err, clusterIP, tcpPort, udpPort
}
