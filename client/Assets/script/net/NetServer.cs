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
		DLLImport.StartIOModule();
		DLLImport.SetLogCallback(OnLogCallback);
		DLLImport.CreateSessionMake(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);
		DLLImport.TcpListen("0.0.0.0", Config.Instance.port);
		DLLImport.UdpListen("0.0.0.0", Config.Instance.port);
	}

	// Update is called once per frame
	void Update()
    {
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
	}

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

	public void SendMessage(IntPtr pSession, Google.Protobuf.IMessage message)
	{
		if (pSession == IntPtr.Zero)
		{
			Debug.LogError("connector is null");
			return;
		}
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		DLLImport.Send(pSession, messageBytes, (uint)messageBytes.Length);
	}

	public void SendMessage(IntPtr pSession, byte[] messageBytes)
	{
		if (pSession == IntPtr.Zero)
		{
			Debug.LogError("connector is null");
			return;
		}
		DLLImport.Send(pSession, messageBytes, (uint)messageBytes.Length);
	}

	public static NetServer Instance
	{
		get
		{
			return instance;
		}
	}

	
	static NetServer instance;
	List<P> msgs = new List<P>();
	delegate void P();
}
