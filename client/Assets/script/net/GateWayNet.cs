using System;
using System.Collections;
using System.Collections.Generic;
using UnityWebSocket;
using UnityEngine;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;

public class GateWayNet : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		Create();
	}

	// Update is called once per frame
	void Update()
    {
	}

	void Create()
	{
		string serverUrl = "ws://115.190.230.47:32001/ws";

		// 创建一个新的WebSocket实例并与指定URL建立连接
		webSocket = new WebSocket(serverUrl);

		// 注册事件回调
		webSocket.OnOpen += (sender, e) =>
		{
			Debug.Log("WebSocket连接成功");

			GateWay.LoginRequest loginRequest = new GateWay.LoginRequest();
			loginRequest.Id = AccountInfo.Instance.Account.Openid;

			SendGW(Any.Pack(loginRequest).ToByteArray());
		};

		webSocket.OnError += (sender, e) =>
		{
			Debug.LogError($"WebSocket连接错误：{e.Message}\n {(e.Exception != null ? e.Exception.StackTrace : string.Empty)}");
		};

		webSocket.OnClose += (sender, e) =>
		{
			webSocket = null;
			Create();
			Debug.Log("WebSocket连接已关闭");
			Action action = null;
			int retryNum = 0;
			action = () =>
			{
				switch(instance.webSocket.ReadyState)
				{
					case WebSocketState.Open:
						Debug.Log("WebSocket已重新连接");
						break;
					case WebSocketState.Closed:
						TimerU.Instance.AddTask(1f, action);
						Connect();
						Debug.Log("WebSocket尝试重新连接");
						break;
					case WebSocketState.Closing:
						TimerU.Instance.AddTask(1f, action);
						Debug.Log("WebSocket正在关闭");
						break;
					case WebSocketState.Connecting:
						TimerU.Instance.AddTask(3f, action);
						if(retryNum ++ % 10 == 0)
						{
							instance.webSocket.CloseAsync();
						}
						Debug.Log("WebSocket正在连接");
						break;
					default:
						Debug.LogError("WebSocket状态异常");
						break;
				}
			};
			TimerU.Instance.AddTask(3f, action);
		};

		webSocket.OnMessage += (sender, e) =>
		{
			Any any = Any.Parser.ParseFrom(e.RawData);
			Debug.Log("WebSocket收到消息类型：" + any.TypeUrl);
			WSMsgProcess.Instance.ProcessMessage(sender, any);
		};
	}

	public void Connect()
	{
		webSocket.ConnectAsync();
	}

	private GateWayNet() { }

	public static GateWayNet Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<GateWayNet>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GateWayNet>();
							singletonObject.name = typeof(GateWayNet).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	public void SendGW(byte[] message)
	{
		webSocket.SendAsync(message);
	}

	public void Close()
	{
		webSocket.CloseAsync();
	}

	static readonly object Lock = new object();
	static GateWayNet instance;

	private WebSocket webSocket;
}


