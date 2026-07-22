using System;
using System.Collections.Generic;
using UnityEngine;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using UnityWebSocket;
using FxNet;
using FxNet.Dll;

public class NetClient : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
#if CLIENT_WS
#else
		FxNetApi.StartIOModule();
		FxNetApi.SetLogCallback(OnLogCallback);
#endif
		MsgProcess.Instance.RegisterHandler(typeof(ClientMsg));
	}

	// Update is called once per frame
	void Update()
    {
#if CLIENT_WS
#else
#if SINGLE_THREAD
		FxNetInterface.ProcSingleThread();
#endif
		FxNetInterface.ProcessMessageEvents();
		int count = msgs.Count;
		for (int i = 0; i < count; i++)
		{
			try
			{
				msgs[i]();
			}
			catch (Exception e)
			{
				Debug.LogError($"[Net][Update] execute queued message failed, index={i}/{count}, exception={e.Message}\n{e.StackTrace}");
			}
		}
		msgs.Clear();
#endif
	}

	void OnApplicationQuit()
	{
#if UNITY_EDITOR && !CLIENT_WS
		FxNetInterface.CloseAllSockets();
		for (int i = 0; i < 20; i++)
		{
			FxNetInterface.ProcessMessageEvents();
		}
#endif
	}

#if CLIENT_WS
	protected void OnOpen(object sender, OpenEventArgs e)
	{
		Debug.Log($"[Net][OnOpen] client websocket connected, serverIP={Config.Instance.serverIP} port={Config.Instance.port}, start game");
		PlayerControl.Instance.StartGame();
	}

	protected void OnMessage(object sender, MessageEventArgs e)
	{
		try
		{
			Any any = Any.Parser.ParseFrom(e.RawData);
			MsgProcess.Instance.ProcessMessage(sender, any);
		}
		catch (Exception ex)
		{
			Debug.LogError($"[Net][OnMessage] parse client websocket message failed, len={(e.RawData != null ? e.RawData.Length : 0)}, exception={ex.Message}\n{ex.StackTrace}");
		}
	}

	protected void OnError(object sender, ErrorEventArgs e)
	{
		Debug.LogError($"[Net][OnError] client websocket error: {e.Message}");
	}

	protected void OnClose(object sender, CloseEventArgs e)
	{
		if(!needReconnect)
		{
			Debug.Log($"[Net][OnClose] client websocket closed, needReconnect=false, code={e.Code} reason={e.Reason}");
			return;
		}
		webSocket.CloseAsync();
		webSocket = null;
		Debug.LogWarning($"[Net][OnClose] client websocket closed unexpectedly, code={e.Code} reason={e.Reason}, will retry in 3s");
		TimerU.Instance.AddTask(3f, () =>
		{
			Reconnect();
		});
	}

	void Reconnect()
	{
		if (!needReconnect)
		{
			Debug.Log("[Net][Reconnect] skip reconnect, needReconnect=false");
			return;
		}
		if (webSocket == null || webSocket.ReadyState != WebSocketState.Open)
		{
			TimerU.Instance.AddTask(3f, Reconnect);
			Debug.LogWarning("[Net][Reconnect] client websocket disconnected, trying to reconnect");
			Instance.Create();
			Instance.Connect();
		}
	}
#else
	static void OnLogCallback(string log, int len)
	{
		Debug.Log($"[Net][OnLogCallback] fxnet: {log}");
	}

	static void OnRecvCallback(Connector pConnector, byte[] pData, int nLen)
	{
		try
		{
			Any anyMessage = Any.Parser.ParseFrom(pData, 0, nLen);
			instance.msgs.Add(delegate ()
			{
				MsgProcess.Instance.ProcessMessage(pConnector, anyMessage);
			});
		}
		catch (Exception e)
		{
			Debug.LogError($"[Net][OnRecvCallback] parse received message failed, connector={pConnector} len={nLen}, exception={e.Message}\n{e.StackTrace}");
		}
	}

	static void OnConnectedCallback(Connector pConnector)
	{
		Debug.Log($"[Net][OnConnectedCallback] connector connected: {pConnector}, invoke {instance.onConnected.Count} onConnected callbacks");
		foreach (var action in instance.onConnected)
		{
			action();
		}
	}

	static void OnErrorCallback(Connector pConnector, int error)
	{
		Debug.LogError($"[Net][OnErrorCallback] connector error, connector={pConnector} error={error}");
	}

	static void OnCloseCallback(Connector pConnector)
	{
		Debug.LogWarning($"[Net][OnCloseCallback] connector closed: {pConnector}, needReconnect={instance.needReconnect}");
		FxNetApi.DestroyConnector(pConnector);

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
				instance.connector = null;
				instance.onConnected.Clear();
			}
		}
	}
#endif

	public void Create()
	{
#if CLIENT_WS
		string serverUrl = $"ws://{Config.Instance.serverIP}:{Config.Instance.port}/game";

		// 创建一个新的WebSocket实例并与指定URL建立连接
		webSocket = new WebSocket(serverUrl);

		// 注册事件回调
		webSocket.OnOpen += OnOpen;
		webSocket.OnMessage += OnMessage;
		webSocket.OnError += OnError;
		webSocket.OnClose += OnClose;
#else
		connector = FxNetApi.CreateConnector(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);
#endif
		needReconnect = true;
		Debug.Log($"[Net][Create] connector/websocket created, serverIP={Config.Instance.serverIP} port={Config.Instance.port}");
	}

	public void Connect()
	{
#if CLIENT_WS
		if (webSocket == null)
		{
			Debug.LogError("[Net][Connect] connect failed: webSocket is null, call Create() first");
			return;
		}
		Debug.Log($"[Net][Connect] connecting websocket to {Config.Instance.serverIP}:{Config.Instance.port}");
		webSocket.ConnectAsync();
#else
		if (connector == null)
		{
			Debug.LogError("[Net][Connect] connect failed: connector is null, call Create() first");
			return;
		}
		Debug.Log($"[Net][Connect] tcp connecting to {Config.Instance.serverIP}:{Config.Instance.port}");
#if AI_RUNING
		FxNetApi.TcpConnect(connector, Config.Instance.serverIP, Config.Instance.port);
#else
		FxNetApi.TcpConnect(connector, Config.Instance.serverIP, Config.Instance.port);
#endif
#endif
	}

	public void SendMessage(Google.Protobuf.IMessage message)
	{
#if CLIENT_WS
		if (webSocket == null)
		{
			Debug.LogError($"[Net][SendMessage] send failed: webSocket is null, msgType={message?.Descriptor?.FullName}");
			return;
		}
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		webSocket.SendAsync(messageBytes);
#else
		if (connector == null)
		{
			Debug.LogError($"[Net][SendMessage] send failed: connector is null, msgType={message?.Descriptor?.FullName}");
			return;
		}
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		FxNetApi.Send(connector, messageBytes, messageBytes.Length);
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
		Debug.Log("[Net][Disconnect] disconnect requested, needReconnect set to false");
#if CLIENT_WS
		webSocket.CloseAsync();
		webSocket = null;
#else
		if(connector != null)
		{
			FxNetApi.Close(connector);
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
	Connector connector;
	delegate void P();
	List<P> msgs = new List<P>();
	List<Action> onConnected = new List<Action>();
#endif
	bool needReconnect = true;
}
