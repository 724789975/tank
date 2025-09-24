using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using fxnetlib.dllimport;
using UnityEditor.PackageManager;


public class NetServer : MonoBehaviour
{
	// Start is called before the first frame update
	void Start()
    {
		DLLImport.CreateSessionMake(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);
		DLLImport.TcpListen("0.0.0.0", TankManager.Instance.cfg.port);
		DLLImport.UdpListen("0.0.0.0", TankManager.Instance.cfg.port);
	}

	// Update is called once per frame
	void Update()
    {
		DLLImport.ProcessIOModule();
	}

	void OnRecvCallback(IntPtr pConnector, string pData, uint nLen)
	{ }
	void OnConnectedCallback(IntPtr pConnector)
	{
		Debug.LogFormat("{0} connected", pConnector);
	}
	void OnErrorCallback(IntPtr pConnector, IntPtr nLen)
	{
		Debug.LogFormat("connector destroy {0}", pConnector);
	}

	void OnCloseCallback(IntPtr pConnector)
	{
		Debug.Log("connector destroy");
		DLLImport.DestroyConnector(pConnector);
	}


}
