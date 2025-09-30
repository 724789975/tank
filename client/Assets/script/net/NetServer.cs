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
		DLLImport.SetLogCallback(delegate (byte[] pData, int dwLen)
			{
				string logMessage = System.Text.Encoding.UTF8.GetString(pData, 0, dwLen);
				Debug.Log(logMessage);
			}
		);
		DLLImport.CreateSessionMake(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);
		DLLImport.TcpListen("0.0.0.0", TankManager.Instance.cfg.port);
		DLLImport.UdpListen("0.0.0.0", TankManager.Instance.cfg.port);
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
	}
	static void OnErrorCallback(IntPtr pConnector, IntPtr nLen)
	{
		Debug.LogFormat("connector destroy {0}", pConnector);
	}

	static void OnCloseCallback(IntPtr pConnector)
	{
		Debug.Log("connector destroy");
		DLLImport.DestroyConnector(pConnector);
		PlayerManager.Instance.AfterCloseCallback(pConnector);
	}

	public static NetServer Instance
	{
		get
		{
			return instance;
		}
	}


	static NetServer instance;
}
