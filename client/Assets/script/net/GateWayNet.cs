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
		Debug.Log($"[Gateway][Create] creating websocket, url={serverUrl}");

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
		Debug.Log("[Gateway][Connect] connecting to gateway");
		webSocket.ConnectAsync();
	}

	protected void OnOpen(object sender, OpenEventArgs e)
	{
		Debug.Log($"[Gateway][OnOpen] connected, send login request, openid={AccountInfo.Instance.Account.Openid}");

		GateWay.LoginRequest loginRequest = new GateWay.LoginRequest();
		loginRequest.Id = AccountInfo.Instance.Account.Openid;

		SendGW(Any.Pack(loginRequest).ToByteArray());
	}

	protected void OnMessage(object sender, MessageEventArgs e)
	{
		try
		{
			Any any = Any.Parser.ParseFrom(e.RawData);
			Debug.Log($"[Gateway][OnMessage] recv message, typeUrl={any.TypeUrl}");
			WSMsgProcess.Instance.ProcessMessage(sender, any);
		}
		catch (System.Exception ex)
		{
			Debug.LogError($"[Gateway][OnMessage] parse gateway message failed, len={(e.RawData != null ? e.RawData.Length : 0)}, exception={ex.Message}\n{ex.StackTrace}");
		}
	}

	protected void OnError(object sender, ErrorEventArgs e)
	{
		Debug.LogError($"[Gateway][OnError] websocket error: {e.Message}\n{(e.Exception != null ? e.Exception.StackTrace : string.Empty)}");
	}

	protected void OnClose(object sender, CloseEventArgs e)
	{
		webSocket.CloseAsync();
		webSocket = null;
		Debug.LogWarning($"[Gateway][OnClose] websocket closed, code={e.Code} reason={e.Reason}, will retry in 3s");
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
			Debug.LogWarning("[Gateway][Reconnect] websocket disconnected, trying to reconnect");
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
		if (webSocket == null)
		{
			Debug.LogError($"[Gateway][SendGW] send failed: webSocket is null, len={(message != null ? message.Length : 0)}");
			return;
		}
		webSocket.SendAsync(message);
	}

	public void Close()
	{
		Debug.Log("[Gateway][Close] close gateway websocket");
		webSocket.CloseAsync();
	}

	static readonly object Lock = new object();
	static GateWayNet instance;

	private WebSocket webSocket;
	// bool neetReconnect = false;
}


