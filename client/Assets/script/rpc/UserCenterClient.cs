using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class UserCenterClient : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		discovery();
	}

    // Update is called once per frame
    void Update()
    {
        
    }

	void discovery()
	{
		EtcdUtil.Instance.Get("/user-center", (result, succeed) =>
		{
			Dictionary<string, DiscoveryInfo> services = new Dictionary<string, DiscoveryInfo>();
			if (!succeed)
			{
				Debug.LogError("get etcd keys failed");
				return;
			}
			foreach (var item in result)
			{
				Debug.Log(item.Key + " " + item.Value);
				services[item.Key] = JsonUtility.FromJson<DiscoveryInfo>(item.Value);
			}

			if (services.Count == 0)
			{
				Debug.LogError("no user-center service found, check etcd server or try again later");
				TimerU.Instance.AddTask(80, () =>
				{
					discovery();
				});
				return;
			}
			// 根据权重随机选择一个服务
			int totalWeight = 0;
			foreach (var item in services.Values)
			{
				totalWeight += item.weight;
			}
			int randomWeight = UnityEngine.Random.Range(0, totalWeight);
			int currentWeight = 0;
			DiscoveryInfo info = null;
			foreach (var item in services.Values)
			{
				currentWeight += item.weight;
				if (randomWeight < currentWeight)
				{
					info = item;
					break;
				}
			}
			if (info == null)
			{
				Debug.LogError($"no user-center service found, random weight error, [{randomWeight}, {totalWeight}]");
				return;
			}
			Debug.Log($"discovery user-center service {info.address}");
			Grpc.Core.Channel channel = new Grpc.Core.Channel(info.address, Grpc.Core.ChannelCredentials.Insecure);
			client = new UserCenterService.UserCenterService.UserCenterServiceClient(channel);
		});
	}

	public UserCenterService.UserCenterService.UserCenterServiceClient Client
	{
		get
		{
			return client;
		}
	}

	UserCenterService.UserCenterService.UserCenterServiceClient client = null;

	static UserCenterClient instance;
	// 公共访问接口
	public static UserCenterClient Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<UserCenterClient>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<UserCenterClient>();
							singletonObject.name = typeof(UserCenterClient).ToString();

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
}
