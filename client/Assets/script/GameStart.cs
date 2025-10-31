using System.Collections;
using System.Collections.Generic;
using System;
using UnityEngine;

public class GameStart : MonoBehaviour
{
	string c;
	[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterAssembliesLoaded)]
	static void OnRuntimeMethodLoad()
	{
		Debug.Log("After Scene is loaded and game is running");

		// Use commandline options passed to the application
		var text = System.Environment.CommandLine + "\n";

		// Load the commandline file content.
		// You need to adjust the path to where the file is located in your project.
		var path = System.IO.Path.Combine(Application.streamingAssetsPath, "CommandLine.txt");
		if (System.IO.File.Exists(path))
		{
			text += System.IO.File.ReadAllText(path);
		}
		else
		{
#if UNITY_EDITOR || DEVELOPMENT_BUILD
			Debug.LogErrorFormat("Could not find commandline file '{0}'.", path);
#endif
		}

		GameStart.Instance.c = text;
		// Initialize the CommandLine
		Oddworm.Framework.CommandLine.Init(text);
	}

	[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
	static void BeforeSceneLoad()
	{
		AccountInfo.Instance.Account.Name = AccountInfo.Instance.Account.Name == "" ? Oddworm.Framework.CommandLine.GetString("-name", "冷水泡面") : AccountInfo.Instance.Account.Name;
		AccountInfo.Instance.Account.Avatar = AccountInfo.Instance.Account.Avatar == "" ? Oddworm.Framework.CommandLine.GetString("-avatar", "https://img3.tapimg.com/default_avatars/aba00206f8642b0bbef01ef8f271e9da.jpg?imageMogr2/auto-orient/strip/thumbnail/!270x270r/gravity/Center/crop/270x270/format/jpg/interlace/1/quality/80") : AccountInfo.Instance.Account.Avatar;
		AccountInfo.Instance.Account.Openid = AccountInfo.Instance.Account.Openid == "" ? Oddworm.Framework.CommandLine.GetString("-openid", "mzw0536knQSO+bhbdL6dtw==") : AccountInfo.Instance.Account.Openid;
		AccountInfo.Instance.Account.Unionid = AccountInfo.Instance.Account.Unionid == "" ? Oddworm.Framework.CommandLine.GetString("-unionid", "SnwhJ5s2EURKCKt0LBsDLw==") : AccountInfo.Instance.Account.Unionid;

		Config.Instance.serverIP = Config.Instance.serverIP == "0.0.0.0" ? Oddworm.Framework.CommandLine.GetString("-server_ip", "114.132.124.13") : Config.Instance.serverIP;
		Debug.Log(Config.Instance.serverIP);

		GameStart.Instance.ToString();
	}

	// Start is called before the first frame update
	void Start()
    {
#if AI_RUNNING
		AIStart.Instance.ToString();
#endif

		Debug.Log($"GameStart Start {c} {AccountInfo.Instance.Account.Name} {Config.Instance.serverIP}");
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

