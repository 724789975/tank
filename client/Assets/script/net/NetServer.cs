using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections.Generic;
using System.Linq;
using UnityEngine;
using FxNet;
using FxNet.Dll;

#if UNITY_SERVER && !AI_RUNNING
using PLAYERDATA = ServerPlayer;
#else
using PLAYERDATA = ClientPlayer;
#endif

public class NetServer : MonoBehaviour
{

	// Start is called before the first frame update
	void Start()
    {
		instance = this;
#if CLIENT_WS
		wsServer = new WebSocketSharp.Server.WebSocketServer(Config.Instance.port);
		wsServer.AddWebSocketService<Laputa>("/game");
		wsServer.Start();
		Debug.Log($"[NetSvr][Start] websocket server started at {wsServer.Address}, port={wsServer.Port}, path={wsServer.WebSocketServices.Paths.ElementAt(0)}");
#else
		FxNetApi.StartIOModule();
		FxNetApi.SetLogCallback(OnLogCallback);
		FxNetApi.CreateSessionMaker(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);
		FxNetApi.TcpListen("0.0.0.0", Config.Instance.port);
		FxNetApi.UdpListen("0.0.0.0", Config.Instance.port);
		Debug.Log($"[NetSvr][Start] fxnet server listening on 0.0.0.0:{Config.Instance.port} (tcp+udp)");
#endif
		MsgProcess.Instance.RegisterHandler(typeof(ServerMsg));
	}

	// Update is called once per frame
	void Update()
    {
		List<P> _msg = new List<P>();
#if CLIENT_WS
		lock (Lock)
		{
#else
#if SINGLE_THREAD
		FxNetInterface.ProcSingleThread();
#endif
		FxNetInterface.ProcessMessageEvents();
#endif
			_msg.AddRange(msgs);
		msgs.Clear();
#if CLIENT_WS
		}
#endif
		// 使用 for 循环遍历 msgs 列表并执行其中的委托
		int count = _msg.Count;
		for (int i = 0; i < count; i++)
		{
			try
			{
				_msg[i]();
			}
			catch (Exception e)
			{
				Debug.LogError($"[NetSvr][Update] execute queued message failed, index={i}/{count}, exception={e.Message}\n{e.StackTrace}");
			}
		}
	}

	void OnApplicationQuit()
	{
#if UNITY_EDITOR && !CLIENT_WS
		FxNetInterface.CloseAllSockets();
		for (int i = 0; i < 20; i++)
		{
			FxNetInterface.ProcessMessageEvents();
		}
		Debug.Log($"OnApplicationQuit");
#endif
	}

#if CLIENT_WS
	public class Laputa : WebSocketSharp.Server.WebSocketBehavior
	{
		protected override void OnOpen()
		{
			Debug.Log($"[NetSvr][OnOpen] session opened, id={ID}");
		}

		protected override void OnClose(WebSocketSharp.CloseEventArgs e)
		{
			Debug.Log($"[NetSvr][OnClose] session closed, id={ID} reason={e.Reason} code={e.Code}");
#if UNITY_SERVER && !AI_RUNNING
			PLAYERDATA playerData = PlayerManager.Instance.GetPlayerBySession(this);
			if (playerData != null)
			{
				playerData.session = null;
			}
#endif
		}

		protected override void OnError(WebSocketSharp.ErrorEventArgs e)
		{
			Debug.LogError($"[NetSvr][OnError] session error, id={ID} message={e.Message}\n{e.Exception}");
		}

		protected override void OnMessage(WebSocketSharp.MessageEventArgs evnt)
		{
			try
			{
				Any anyMessage = Any.Parser.ParseFrom(evnt.RawData, 0, evnt.RawData.Length);
				lock (NetServer.Instance.Lock)
				{
					instance.msgs.Add(delegate ()
					{
						MsgProcess.Instance.ProcessMessage(this, anyMessage);
					});
				}
			}
			catch (Exception e)
			{
				Debug.LogError($"[NetSvr][OnMessage] parse session message failed, id={ID} len={(evnt.RawData != null ? evnt.RawData.Length : 0)}, exception={e.Message}\n{e.StackTrace}");
			}
		}

		public new void Send(byte[] messageBytes)
		{
			base.Send(messageBytes);
		}

		public new void Close()
		{
			base.Close();
		}
	}

#else
	static void OnLogCallback(string log, int len)
	{
		Debug.Log($"[NetSvr][OnLogCallback] fxnet: {log}");
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
			//MsgProcess.Instance.ProcessMessage(pConnector, anyMessage);
		}
		catch (Exception e)
		{
			Debug.LogError($"[NetSvr][OnRecvCallback] parse received message failed, connector={pConnector} len={nLen}, exception={e.Message}\n{e.StackTrace}");
		}
	}

	static void OnConnectedCallback(Connector pConnector)
	{
		Debug.Log($"[NetSvr][OnConnectedCallback] connector connected: {pConnector}");
	}

	static void OnErrorCallback(Connector pConnector, int error)
	{
		Debug.LogError($"[NetSvr][OnErrorCallback] connector error, connector={pConnector} error={error}");
	}

	static void OnCloseCallback(Connector pConnector)
	{
		Debug.LogWarning($"[NetSvr][OnCloseCallback] connector closed: {pConnector}");
		FxNetApi.DestroyConnector(pConnector);
		PlayerManager.Instance.AfterCloseCallback(pConnector);
	}
#endif

	public void SendMessage(object pSession, Google.Protobuf.IMessage message)
	{
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		SendMessage(pSession, messageBytes);
	}

	public void SendMessage(object pSession, byte[] messageBytes)
	{
#if CLIENT_WS
		if (pSession == null)
		{
			Debug.LogError($"[NetSvr][SendMessage] send failed: session is null, len={(messageBytes != null ? messageBytes.Length : 0)}");
			return;
		}
		((Laputa)pSession).Send(messageBytes);
#else
		if (pSession == null)
		{
			Debug.LogError($"[NetSvr][SendMessage] send failed: connector is null, len={(messageBytes != null ? messageBytes.Length : 0)}");
			return;
		}
		FxNetApi.Send((Connector)pSession, messageBytes, messageBytes.Length);
#endif
	}

	public void CloseSession(object pSession)
	{
#if CLIENT_WS
		((Laputa)pSession).Close();

#else
		if (pSession == null)
		{
			Debug.LogError("[NetSvr][CloseSession] close failed: connector is null");
			return;
		}
		FxNetApi.Close((Connector)pSession);
#endif
	}

	public static NetServer Instance
	{
		get
		{
			return instance;
		}
	}

	
	static NetServer instance;

#if CLIENT_WS
	WebSocketSharp.Server.WebSocketServer wsServer;
	readonly object Lock = new object();
#else
#endif
	List<P> msgs = new List<P>();
	delegate void P();

}
