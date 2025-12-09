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
		webSocket.OnOpen += Instance.OnOpen;
		webSocket.OnMessage += Instance.OnMessage;
		webSocket.OnError += Instance.OnError;
		webSocket.OnClose += Instance.OnClose;
	}

	public void Connect()
	{
		webSocket.ConnectAsync();
	}

	protected void OnOpen(object sender, OpenEventArgs e)
	{
		Debug.Log("WebSocket连接成功");

		GateWay.LoginRequest loginRequest = new GateWay.LoginRequest();
		loginRequest.Id = AccountInfo.Instance.Account.Openid;

		SendGW(Any.Pack(loginRequest).ToByteArray());
	}

	protected void OnMessage(object sender, MessageEventArgs e)
	{
		Any any = Any.Parser.ParseFrom(e.RawData);
		Debug.Log("WebSocket收到消息类型：" + any.TypeUrl);
		WSMsgProcess.Instance.ProcessMessage(sender, any);
	}

	protected void OnError(object sender, ErrorEventArgs e)
	{
		Debug.LogError($"WebSocket连接错误：{e.Message}\n {(e.Exception != null ? e.Exception.StackTrace : string.Empty)}");
	}

	protected void OnClose(object sender, CloseEventArgs e)
	{
		webSocket.CloseAsync();
		webSocket = null;
		Debug.Log("WebSocket连接已关闭");
		TimerU.Instance.AddTask(3f, () => 
		{
			Reconnect();
		});
	}

	void Reconnect()
	{
		if(webSocket == null || webSocket.ReadyState != WebSocketState.Open)
		{
			TimerU.Instance.AddTask(3f, Reconnect);
			Debug.Log("WebSocket连接断开，尝试重新连接");
			Instance.Create();
			Instance.Connect();
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

	public void Close()
	{
		webSocket.CloseAsync();
	}

	static readonly object Lock = new object();
	static GateWayNet instance;

	private WebSocket webSocket;
	bool neetReconnect = false;
}


