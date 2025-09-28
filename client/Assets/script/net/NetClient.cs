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
		DLLImport.StartIOModule();
		DLLImport.SetLogCallback(delegate (byte[] pData, int dwLen)
			{
				string logMessage = System.Text.Encoding.UTF8.GetString(pData, 0, dwLen);
				Debug.Log(logMessage);
			}
		);
		connector = DLLImport.CreateConnector(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);

		DLLImport.TcpConnect(connector, TankManager.Instance.cfg.serverIP, TankManager.Instance.cfg.port);
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
			// ���� pData �Ǿ�������� protobuf ��Ϣ
			// �������ʵ�ʵ���Ϣ��ʽ���н�����ʾ���м����� Ping ��Ϣ
			Any anyMessage = Any.Parser.ParseFrom(pData, 0, (int)nLen);
			
			// ���������ﴦ��ת����� Any ������Ϣ
			Debug.Log("Received message converted to Any type: " + anyMessage.TypeUrl);
		}
		catch (Exception e)
		{
			Debug.LogError("Failed to parse message: " + e.Message);
		}
	}
	static void OnConnectedCallback(IntPtr pConnector)
	{
		Debug.LogFormat("{0} connected", pConnector);

		// ���ӳɹ��󣬿����������������Ϣ
		// �������Ҫ����һ�� Ping ��Ϣ
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

	IntPtr connector;
}
