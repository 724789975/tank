using UnityEngine;

public class RpcService : MonoBehaviour
{

	// Start is called before the first frame update
	void Start()
    {
		Grpc.Core.Server server = new Grpc.Core.Server
		{
			//服务端接口消息处理类
			Services = { TankGameService.TankGameService.BindService(new ServicerImpl()) },
			//服务端ip、端口、保密类型
			Ports = { new Grpc.Core.ServerPort("0.0.0.0", Config.Instance.rpcPort, Grpc.Core.ServerCredentials.Insecure) }
		};
		//开始侦听
		server.Start();
		Debug.Log("RpcService Start");
	}

    // Update is called once per frame
    void Update()
    {
        
    }


	static RpcService instance;
	// 公共访问接口
	public static RpcService Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<RpcService>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<RpcService>();
							singletonObject.name = typeof(RpcService).ToString();

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
