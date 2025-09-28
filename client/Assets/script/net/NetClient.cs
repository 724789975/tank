using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using fxnetlib.dllimport;

public class NetClient : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		connector = DLLImport.CreateConnector(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);

		DLLImport.TcpConnect(connector, TankManager.Instance.cfg.serverIP, TankManager.Instance.cfg.port);
	}

	// Update is called once per frame
	void Update()
    {
		DLLImport.ProcessIOModule();
	}

	static void OnRecvCallback(IntPtr pConnector, string pData, uint nLen)
	{ }
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
	}

	IntPtr connector;
}
