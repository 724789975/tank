using System.Collections;
using System.Collections.Generic;
using System;
using UnityEngine;
using Dirichlet.Mediation;
using System.Net.Sockets;
using System.Net;

public class GameStart : MonoBehaviour
{
	[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterAssembliesLoaded)]
	static void OnRuntimeMethodLoad()
	{
#if UNITY_SERVER && !AI_RUNNING
        Application.targetFrameRate = 40;
#else
		Application.targetFrameRate = 60;
#endif

		Resolution reslution = Screen.currentResolution;

		int standard_width = reslution.width;
		int standard_height = ((standard_width * 9) / 16);
		if (standard_height > reslution.height)
		{
			standard_height = reslution.height;
			standard_width = ((standard_height * 16) / 9);
		}

#if PLATFORM_STANDALONE_WIN || PLATFORM_STANDALONE_LINUX || PLATFORM_STANDALONE_OSX
		standard_width = 960;
		standard_height = 540;
#endif
		Debug.LogFormat("[GameStart][OnRuntimeMethodLoad] set screen resolution to {0}x{1}", standard_width, standard_height);
		Screen.SetResolution(Convert.ToInt32(standard_width), Convert.ToInt32(standard_height), false);

		// Use commandline options passed to the application
		var text = System.Environment.CommandLine + "\n";

		// Load the commandline file content.
		// You need to adjust the path to where the file is located in your project.
		// 根据 DEPLOYENV 环境变量读取不同的命令行参数文件(添加环境后缀), 例如 test2 -> CommandLine-test2.txt;
		// 未设置环境变量或对应环境文件不存在时, 回退到默认的 CommandLine.txt
		var commandLineFile = "CommandLine.txt";
		var deployEnv = System.Environment.GetEnvironmentVariable("DEPLOYENV");
		if (!string.IsNullOrEmpty(deployEnv))
		{
			var envFile = string.Format("CommandLine-{0}.txt", deployEnv);
			var envFilePath = System.IO.Path.Combine(Application.streamingAssetsPath, envFile);
			if (System.IO.File.Exists(envFilePath))
			{
				commandLineFile = envFile;
			}
			else
			{
#if UNITY_EDITOR || DEVELOPMENT_BUILD
				Debug.LogWarningFormat("[GameStart][OnRuntimeMethodLoad] env commandline file '{0}' not found, fallback to '{1}'", envFilePath, commandLineFile);
#endif
			}
		}

		var path = System.IO.Path.Combine(Application.streamingAssetsPath, commandLineFile);
		Debug.LogFormat("[GameStart][OnRuntimeMethodLoad] DEPLOYENV='{0}', using commandline file '{1}'", deployEnv, path);
		if (System.IO.File.Exists(path))
		{
			text += System.IO.File.ReadAllText(path);
		}
		else
		{
#if UNITY_EDITOR || DEVELOPMENT_BUILD
			Debug.LogWarningFormat("[GameStart][OnRuntimeMethodLoad] could not find commandline file '{0}'", path);
#endif
		}

		// Initialize the CommandLine
		Debug.Log("[GameStart][OnRuntimeMethodLoad] CommandLine: " + text);
		Oddworm.Framework.CommandLine.Init(text);

		GameStart.Instance.ToString();
		TimerU.Instance.ToString();
		EtcdUtil.Instance.ToString();
		Config.Instance.ToString();
	}

	[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
	static void BeforeSceneLoad()
	{
		AccountInfo.Instance.Account.Name = AccountInfo.Instance.Account.Name == "" ? Oddworm.Framework.CommandLine.GetString("-name", "冷水泡面") : AccountInfo.Instance.Account.Name;
		AccountInfo.Instance.Account.Avatar = AccountInfo.Instance.Account.Avatar == "" ? Oddworm.Framework.CommandLine.GetString("-avatar", "https://img3.tapimg.com/default_avatars/aba00206f8642b0bbef01ef8f271e9da.jpg?imageMogr2/auto-orient/strip/thumbnail/!270x270r/gravity/Center/crop/270x270/format/jpg/interlace/1/quality/80") : AccountInfo.Instance.Account.Avatar;
		AccountInfo.Instance.Account.Openid = AccountInfo.Instance.Account.Openid == "" ? Oddworm.Framework.CommandLine.GetString("-openid", "mzw0536knQSO+bhbdL6dtw==") : AccountInfo.Instance.Account.Openid;
		AccountInfo.Instance.Account.Unionid = AccountInfo.Instance.Account.Unionid == "" ? Oddworm.Framework.CommandLine.GetString("-unionid", "SnwhJ5s2EURKCKt0LBsDLw==") : AccountInfo.Instance.Account.Unionid;

#if UNITY_EDITOR && !AI_RUNNING
		// test2 本地调试: 编辑器内真人玩家改用 test_openid 账号, 避免与容器 ai-client 的默认账号
		// (mzw0536knQSO+bhbdL6dtw==) 撞号——两者 openid 相同会在 game-server 端被 kick old player 互踢下线.
		// 仅 UNITY_EDITOR 生效: ai-client(AI_RUNNING) 与 android 正式包不受影响. test_openid 已存在于 user-center redis.
		if (AccountInfo.Instance.Account.Openid == "mzw0536knQSO+bhbdL6dtw==")
		{
			AccountInfo.Instance.Account.Openid = "test_openid";
			AccountInfo.Instance.Account.Name = "test_name";
			AccountInfo.Instance.Account.Unionid = "test_unionid";
			Debug.Log("[GameStart][BeforeSceneLoad] editor player account overridden to test_openid to avoid collision with ai-client");
		}
#endif

		Config.Instance.serviceName = Oddworm.Framework.CommandLine.GetString("-service_name", "123456");

		Config.Instance.serverIP = Oddworm.Framework.CommandLine.GetString("-server_ip", "127.0.0.1");
		Debug.Log($"[GameStart][BeforeSceneLoad] serverIP={Config.Instance.serverIP}");

		// 网关登录地址按环境注入(见 CommandLine-<env>.txt), 缺省回退到本地 test2 网关
		Config.Instance.gatewayUrl = Oddworm.Framework.CommandLine.GetString("-gateway_url", Config.Instance.gatewayUrl);
		Debug.Log($"[GameStart][BeforeSceneLoad] gatewayUrl={Config.Instance.gatewayUrl}");

		// 公共 HTTP API 基地址按环境注入(登录/匹配等)
		Config.Instance.httpBaseUrl = Oddworm.Framework.CommandLine.GetString("-http_base_url", Config.Instance.httpBaseUrl);
		Debug.Log($"[GameStart][BeforeSceneLoad] httpBaseUrl={Config.Instance.httpBaseUrl}");

		EtcdUtil.Instance.etcdAddr = Oddworm.Framework.CommandLine.GetString("-etcd_addr", "");
		EtcdUtil.Instance.etcdUserName = Oddworm.Framework.CommandLine.GetString("-etcd_user_name", "");
		EtcdUtil.Instance.etcdPassword = Oddworm.Framework.CommandLine.GetString("-etcd_password", "");

		var host = Dns.GetHostName();
		var ipEntry = Dns.GetHostEntry(host);
		foreach (var ipAddr in ipEntry.AddressList)
		{
			if (ipAddr.AddressFamily == AddressFamily.InterNetwork)
			{
				Config.Instance.localIp = ipAddr.ToString();
				break;
			}
		}

		Debug.Log($"[GameStart][BeforeSceneLoad] localIp={Config.Instance.localIp}");

		GameStart.Instance.ToString();
#if UNITY_SERVER && !AI_RUNNING
		//EtcdUtil.Instance.Keys();
		RpcService.Instance.ToString();
		GameStart.Instance.register();
		UserCenterClient.Instance.ToString();
#else
		NetClient.Instance.ToString();
		WSMsgProcess.Instance.ToString();
#endif

	}

	void register()
	{
		TimerU.Instance.AddTask(80, () =>
		{
			register();
		});
		EtcdUtil.Instance.Put($"/{Config.Instance.serviceName}/{Config.Instance.localIp}:{Config.Instance.rpcPort}", $"{{ \"network\":\"tcp\",\"address\":\"{Config.Instance.localIp}:{Config.Instance.port}\",\"weight\":10,\"tags\":null}}", 100);

		//EtcdUtil.Instance.Get("", (result, succeed) =>
		//{
		//	if (!succeed)
		//	{
		//		Debug.LogError("get etcd keys failed");
		//		return;
		//	}
		//	foreach (var item in result)
		//	{
		//		Debug.Log(item.Key + " " + item.Value);
		//	}
		//});
	}

	// Start is called before the first frame update
	void Start()
    {
#if AI_RUNNING
		AIStart.Instance.ToString();
#endif
		Application.runInBackground = true;
#if UNITY_EDITOR || UNITY_ANDROID
		DirichletSdk.RequestPermissionIfNecessary();
#endif
	}

	// Update is called once per frame
	void Update()
    {
	}

	public static GameStart Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<GameStart>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GameStart>();
							singletonObject.name = typeof(GameStart).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	static readonly object Lock = new object();
	static GameStart instance;
}

