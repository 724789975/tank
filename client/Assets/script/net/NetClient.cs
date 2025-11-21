using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using fxnetlib.dllimport;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using TankGame;
using AOT;

public class NetClient : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
#if !UNITY_EDITOR
		DLLImport.StartIOModule();
#endif
		DLLImport.SetLogCallback(OnLogCallback);
	}

	// Update is called once per frame
	void Update()
    {
		DLLImport.ProcessIOModule();
		int count = msgs.Count;
		for (int i = 0; i < count; i++)
		{
			try
			{
				msgs[i]();
			}
			catch (Exception e)
			{
				Debug.LogError($"error on update: {e.Message}\n{e.StackTrace}");
			}
		}
		msgs.Clear();
	}

	void OnApplicationQuit()
	{
#if UNITY_EDITOR
		DLLImport.StopAllSockets();
		for (int i = 0; i < 20; i++)
		{
			DLLImport.ProcessIOModule();
		}
#endif
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
		foreach (var action in instance.onConnected)
		{
			action();
		}
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

		if (instance.needReconnect)
		{
			Timer.Instance.AddTask(3f, () =>
			{
				instance.Connect();
			});
		}
		else
		{
			if (instance.connector == pConnector)
			{
				instance.connector = IntPtr.Zero;
				instance.onConnected.Clear();
			}
		}
	}

	public void Connect()
	{
		connector = DLLImport.CreateConnector(OnRecvCallback, OnConnectedCallback, OnErrorCallback, OnCloseCallback);

		DLLImport.TcpConnect(connector, Config.Instance.serverIP, Config.Instance.port);
	}

	public void SendMessage(Google.Protobuf.IMessage message)
	{
		if (connector == IntPtr.Zero)
		{
			Debug.LogError("connector is null");
			return;
		}
		byte[] messageBytes = Any.Pack(message).ToByteArray();
		DLLImport.Send(connector, messageBytes, (uint)messageBytes.Length);
	}

	public void AddOnConnected(Action action)
	{
		onConnected.Add(action);
	}

	public void OnConnected()
	{
	}

	public void Disconnect()
	{
		if(connector != IntPtr.Zero)
		{
			DLLImport.Close(connector);
		}
		needReconnect = false;
	}

	public static NetClient Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<NetClient>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<NetClient>();
							singletonObject.name = typeof(NetClient).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	IntPtr connector = IntPtr.Zero;

	static NetClient instance;
	static readonly object Lock = new object();
	List<P> msgs = new List<P>();
	delegate void P();
	List<Action> onConnected = new List<Action>();
	bool needReconnect = true;
}
