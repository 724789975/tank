using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using fxnetlib.dllimport;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using TankGame;

public class NetClient : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		instance = this;
		DLLImport.StartIOModule();
		DLLImport.SetLogCallback(delegate (byte[] pData, int dwLen)
			{
				string logMessage = System.Text.Encoding.UTF8.GetString(pData, 0, dwLen);
				Debug.Log(logMessage);
			}
		);

		Connect();
	}

	// Update is called once per frame
	void Update()
    {
		DLLImport.ProcessIOModule();
	}

	static void OnRecvCallback(IntPtr pConnector, byte[] pData, uint nLen)
	{
		try
		{
			Any anyMessage = Any.Parser.ParseFrom(pData, 0, (int)nLen);
			MsgProcess.Instance.ProcessMessage(pConnector, anyMessage);
		}
		catch (Exception e)
		{
			Debug.LogError("Failed to parse message: " + e.Message);
		}
	}
	static void OnConnectedCallback(IntPtr pConnector)
	{
		Debug.LogFormat("{0} connected", pConnector);

		TankGame.Ping pingMessage = new TankGame.Ping();
		pingMessage.Ts = DateTime.Now.Ticks;
		byte[] messageBytes = Any.Pack(pingMessage).ToByteArray();
		DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);
	}
	static void OnErrorCallback(IntPtr pConnector, IntPtr nLen)
	{
		Debug.LogFormat("connector destroy {0}", pConnector);
	}

	static void OnCloseCallback(IntPtr pConnector)
	{
		Debug.Log("connector destroy");
		DLLImport.DestroyConnector(pConnector);
	}

	public void Connect()
	{
		connector = DLLImport.CreateConnector(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);

		DLLImport.TcpConnect(connector, TankManager.Instance.cfg.serverIP, TankManager.Instance.cfg.port);
	}

	public void SendMessage(Any message)
	{
		if (connector == IntPtr.Zero)
		{
			Debug.LogError("connector is null");
			return;
		}
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		DLLImport.Send(connector, messageBytes, (uint)messageBytes.Length);
	}

	public static NetClient Instance
	{
		get
		{
			return instance;
		}
	}

	IntPtr connector = IntPtr.Zero;

	static NetClient instance;
}
