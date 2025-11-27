using AOT;
using fxnetlib.dllimport;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class NetServer : MonoBehaviour
{

	// Start is called before the first frame update
	void Start()
    {
		instance = this;
#if CLIENT_WS
		wsServer = new WebSocketSharp.Server.WebSocketServer(Config.Instance.port);
		wsServer.AddWebSocketService<Laputa>("/game");
#else
		DLLImport.StartIOModule();
		DLLImport.SetLogCallback(OnLogCallback);
		DLLImport.CreateSessionMake(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);
		DLLImport.TcpListen("0.0.0.0", Config.Instance.port);
		DLLImport.UdpListen("0.0.0.0", Config.Instance.port);
#endif
	}

	// Update is called once per frame
	void Update()
    {
#if CLIENT_WS
#else
		DLLImport.ProcessIOModule();
		// 使用 for 循环遍历 msgs 列表并执行其中的委托
		int count = msgs.Count;
		for (int i = 0; i < count; i++)
		{
			try
			{
				msgs[i]();
			}
			catch (Exception e)
			{
				Debug.LogError($"Error executing message processing delegate: {e.Message}\n{e.StackTrace}");
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
		Debug.Log($"OnApplicationQuit");
#endif
	}

#if CLIENT_WS
	public class Laputa : WebSocketSharp.Server.WebSocketBehavior
	{
		protected override void OnOpen()
		{
			Debug.Log("Laputa opened");
		}

		protected override void OnClose(WebSocketSharp.CloseEventArgs e)
		{
			Debug.Log($"Laputa closed {e.Reason}");
		}

		protected override void OnError(WebSocketSharp.ErrorEventArgs e)
		{
			Debug.LogError("Laputa error: " + e.Message);
		}

		protected override void OnMessage(WebSocketSharp.MessageEventArgs evnt)
		{
			try
			{
				Any anyMessage = Any.Parser.ParseFrom(evnt.RawData, 0, evnt.RawData.Length);
				MsgProcess.Instance.ProcessMessage(this, anyMessage);
			}
			catch (Exception e)
			{
				Debug.LogError($"Failed to parse message: {e.Message}\n{e.StackTrace}");
			}
		}

		public new void Send(byte[] messageBytes)
		{
			base.Send(messageBytes);
		}
	}

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
			//MsgProcess.Instance.ProcessMessage(pConnector, anyMessage);
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
		((Laputa)pSession).Send(messageBytes);
#else
		if ((IntPtr)pSession == IntPtr.Zero)
		{
			Debug.LogError("connector is null");
			return;
		}
		DLLImport.Send((IntPtr)pSession, messageBytes, (uint)messageBytes.Length);
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
#else
	List<P> msgs = new List<P>();
	delegate void P();
#endif

}
