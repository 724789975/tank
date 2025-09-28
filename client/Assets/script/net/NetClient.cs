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
			// 假设 pData 是经过编码的 protobuf 消息
			// 这里根据实际的消息格式进行解析，示例中假设是 Ping 消息
			Any anyMessage = Any.Parser.ParseFrom(pData, 0, (int)nLen);
			
			// 可以在这里处理转换后的 Any 类型消息
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

		// 连接成功后，可以向服务器发送消息
		// 这里假设要发送一个 Ping 消息
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
