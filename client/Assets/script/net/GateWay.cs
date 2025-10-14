using System;
using System.Collections;
using System.Collections.Generic;
using UnityWebSocket;
using UnityEngine;

public class GateWay : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		string serverUrl = "ws://yourserveraddress";

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
			Debug.Log("WebSocket收到消息：" + e.Data);
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

	private GateWay() { }

	public static GateWay Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<GateWay>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GateWay>();
							singletonObject.name = typeof(GateWay).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	public void SendMessage(byte[] message)
	{
		webSocket.SendAsync(message);
	}

	static readonly object Lock = new object();
	static GateWay instance;

	private WebSocket webSocket;

	float delayConnect = 3f;
}
