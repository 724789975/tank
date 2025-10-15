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
		WSMsgProcess.Instance.RegisterHandler(WSMsg.Instance);
		string serverUrl = "ws://127.0.0.1:20002/ws";

		// 创建一个新的WebSocket实例并与指定URL建立连接
		webSocket = new WebSocket(serverUrl);

		webSocket.ConnectAsync();

		// 注册事件回调
		webSocket.OnOpen += (sender, e) =>
		{
			Debug.Log("WebSocket连接成功");
			delayConnect = 3f;
		};

		webSocket.OnError += (sender, e) =>
		{
			Debug.LogError("WebSocket连接错误：" + e.Message);
		};

		webSocket.OnClose += (sender, e) =>
		{
			Debug.Log("WebSocket连接已关闭");
		};
		webSocket.OnMessage += (sender, e) =>
		{
			Any any = Any.Parser.ParseFrom(e.RawData);
			Debug.Log("WebSocket收到消息类型：" + any.TypeUrl);
			WSMsgProcess.Instance.ProcessMessage(sender, any);
		};
	}

	// Update is called once per frame
	void Update()
    {
		if(webSocket.ReadyState == WebSocketState.Closed)
		{
			delayConnect -= Time.deltaTime;
			if(delayConnect <= 0)
			{
				webSocket.ConnectAsync();
				delayConnect = 3f;
			}
		}
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

	static readonly object Lock = new object();
	static GateWayNet instance;

	private WebSocket webSocket;

	float delayConnect = 3f;
}
