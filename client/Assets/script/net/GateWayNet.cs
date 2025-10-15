using System;
using System.Collections;
using System.Collections.Generic;
using UnityWebSocket;
using UnityEngine;

public class GateWayNet : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		string serverUrl = "ws://127.0.0.1:20002/ws";

		// ����һ���µ�WebSocketʵ������ָ��URL��������
		webSocket = new WebSocket(serverUrl);

		webSocket.ConnectAsync();

		// ע���¼��ص�
		webSocket.OnOpen += (sender, e) =>
		{
			Debug.Log("WebSocket���ӳɹ�");
			delayConnect = 3f;
		};

		webSocket.OnError += (sender, e) =>
		{
			Debug.LogError("WebSocket���Ӵ���" + e.Message);
		};

		webSocket.OnClose += (sender, e) =>
		{
			Debug.Log("WebSocket�����ѹر�");
		};
		webSocket.OnMessage += (sender, e) =>
		{
			Debug.Log("WebSocket�յ���Ϣ��" + e.RawData);
		};
	}

	// Update is called once per frame
	void Update()
    {
		if(webSocket.ReadyState == WebSocketState.Closed)
		{
			delayConnect -= Time.deltaTime;
			if(delayConnect <= 0)
			{
				webSocket.ConnectAsync();
				delayConnect = 3f;
			}
		}
	}

	private GateWayNet() { }

	public static GateWayNet Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<GateWayNet>();
						if (instance == null)
						{
							// �����µ�ʵ��
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GateWayNet>();
							singletonObject.name = typeof(GateWayNet).ToString();

							// ȷ���������ᱻ����
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	public void SendMessage(byte[] message)
	{
		webSocket.SendAsync(message);
	}

	static readonly object Lock = new object();
	static GateWayNet instance;

	private WebSocket webSocket;

	float delayConnect = 3f;
}
