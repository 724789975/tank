using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using fxnetlib.dllimport;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using TankGame;
using AOT;
using UnityWebSocket;

public class NetClient : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
#if CLIENT_WS
#else
		DLLImport.StartIOModule();
		DLLImport.SetLogCallback(OnLogCallback);
#endif
		MsgProcess.Instance.RegisterHandler(typeof(ClientMsg));
	}

	// Update is called once per frame
	void Update()
    {
#if CLIENT_WS
#else
		DLLImport.ProcessIOModule();
		int count = msgs.Count;
		for (int i = 0; i < count; i++)
		{
			try
			{
				msgs[i]();
			}
			catch (Exception e)
			{
				Debug.LogError($"error on update: {e.Message}\n{e.StackTrace}");
			}
		}
		msgs.Clear();
#endif
	}

	void OnApplicationQuit()
	{
#if UNITY_EDITOR && !CLIENT_WS
		DLLImport.StopAllSockets();
		for (int i = 0; i < 20; i++)
		{
			DLLImport.ProcessIOModule();
		}
#endif
	}

#if CLIENT_WS
#else
	[MonoPInvokeCallback(typeof(fxnetlib.dllimport.DLLImport.OnLogCallback))]
	static void OnLogCallback (byte[] pData, int dwLen)
	{
		string logMessage = System.Text.Encoding.UTF8.GetString(pData, 0, dwLen);
		Debug.Log(logMessage);
	}

	[MonoPInvokeCallback(typeof(fxnetlib.dllimport.DLLImport.OnRecvCallback))]
	static void OnRecvCallback(IntPtr pConnector, byte[] pData, uint nLen)
	{
		try
		{
			Any anyMessage = Any.Parser.ParseFrom(pData, 0, (int)nLen);
			instance.msgs.Add(delegate ()
			{
				MsgProcess.Instance.ProcessMessage(pConnector, anyMessage);
			});
		}
		catch (Exception e)
		{
			Debug.LogError($"Failed to parse message: {e.Message}\n{e.StackTrace}");
		}
	}

	[MonoPInvokeCallback(typeof(fxnetlib.dllimport.DLLImport.OnConnectedCallback))]
	static void OnConnectedCallback(IntPtr pConnector)
	{
		Debug.LogFormat("{0} connected", pConnector);
		foreach (var action in instance.onConnected)
		{
			action();
		}
	}

	[MonoPInvokeCallback(typeof(fxnetlib.dllimport.DLLImport.OnErrorCallback))]
	static void OnErrorCallback(IntPtr pConnector, IntPtr nLen)
	{
		Debug.LogFormat("connector destroy {0}", pConnector);
	}

	[MonoPInvokeCallback(typeof(fxnetlib.dllimport.DLLImport.OnCloseCallback))]
	static void OnCloseCallback(IntPtr pConnector)
	{
		Debug.Log("connector destroy");
		DLLImport.DestroyConnector(pConnector);

		if (instance.needReconnect)
		{
			TimerU.Instance.AddTask(3f, () =>
			{
				instance.Connect();
			});
		}
		else
		{
			if (instance.connector == pConnector)
			{
				instance.connector = IntPtr.Zero;
				instance.onConnected.Clear();
			}
		}
	}
#endif

	public void Connect()
	{
#if CLIENT_WS
		string serverUrl = $"ws://{Config.Instance.serverIP}:{Config.Instance.port}/game";

		// 创建一个新的WebSocket实例并与指定URL建立连接
		webSocket = new WebSocket(serverUrl);

		webSocket.ConnectAsync();

		// 注册事件回调
		webSocket.OnOpen += (sender, e) =>
		{
			Debug.Log("Client WebSocket连接成功");
		};

		webSocket.OnError += (sender, e) =>
		{
			Debug.LogError("Client WebSocket连接错误：" + e.Message);
		};

		webSocket.OnClose += (sender, e) =>
		{
			Debug.Log("Client WebSocket连接已关闭");
			if(needReconnect)
			{
				TimerU.Instance.AddTask(3f, () =>
				{
					webSocket.ConnectAsync();
				});
			}
		};

		webSocket.OnMessage += (sender, e) =>
		{
			Any any = Any.Parser.ParseFrom(e.RawData);
			//Debug.Log("client WebSocket收到消息类型：" + any.TypeUrl);
			MsgProcess.Instance.ProcessMessage(sender, any);
		};
#else
		connector = DLLImport.CreateConnector(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);
		DLLImport.TcpConnect(connector, Config.Instance.serverIP, Config.Instance.port);
#endif
	}

	public void SendMessage(Google.Protobuf.IMessage message)
	{
#if CLIENT_WS
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		webSocket.SendAsync(messageBytes);
#else
		if (connector == IntPtr.Zero)
		{
			Debug.LogError("connector is null");
			return;
		}
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		DLLImport.Send(connector, messageBytes, (uint)messageBytes.Length);
#endif
	}

	public void AddOnConnected(Action action)
	{
#if CLIENT_WS
		webSocket.OnOpen += (sender, e) =>
		{
			if (needReconnect)
			{
				action();
			}
		};
#else
		onConnected.Add(action);
#endif
		}

	public void OnConnected()
	{
	}

	public void Disconnect()
	{
#if CLIENT_WS
		webSocket.CloseAsync();
		webSocket = null;
#else
		if(connector != IntPtr.Zero)
		{
			DLLImport.Close(connector);
		}
#endif
		needReconnect = false;
	}

	public static NetClient Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<NetClient>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<NetClient>();
							singletonObject.name = typeof(NetClient).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}


	static NetClient instance;
	static readonly object Lock = new object();
#if CLIENT_WS
	WebSocket webSocket;
#else
	IntPtr connector = IntPtr.Zero;
	List<P> msgs = new List<P>();
	delegate void P();
	List<Action> onConnected = new List<Action>();
#endif
	bool needReconnect = true;
}
