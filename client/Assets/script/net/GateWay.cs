using System;
using System.Collections;
using System.Collections.Generic;
using UnityWebSocket;
using UnityEngine;

public class GateWay : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		string serverUrl = "ws://yourserveraddress";

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
			Debug.Log("WebSocket�յ���Ϣ��" + e.Data);
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

	private GateWay() { }

	public static GateWay Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<GateWay>();
						if (instance == null)
						{
							// �����µ�ʵ��
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GateWay>();
							singletonObject.name = typeof(GateWay).ToString();

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
	static GateWay instance;

	private WebSocket webSocket;

	float delayConnect = 3f;
}
