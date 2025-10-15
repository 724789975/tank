using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using TankGame;

public class WSMsg : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
	}

	// Update is called once per frame
	void Update()
    {
	}

	[WSHandler("gate_way.Test")]
	static void Tesst(object sender, Any anyMessage)
	{
		GateWay.Test test = anyMessage.Unpack<GateWay.Test>();
		Debug.Log(test.ToString());
	}

	static void LoginResp(object sender, Any anyMessage)
	{
		GateWay.LoginResp test = anyMessage.Unpack<GateWay.LoginResp>();
		Debug.Log(test.ToString());
	}

	public static WSMsg Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<WSMsg>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<WSMsg>();
							singletonObject.name = typeof(WSMsg).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	static WSMsg instance;
	static readonly object Lock = new object();
}
