using System.Collections;
using System.Collections.Generic;
using System;
using UnityEngine;

public class GameStart : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
	}

	// Update is called once per frame
	void Update()
    {
        
    }

	public static GameStart Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<GameStart>();
						if (instance == null)
						{
							// �����µ�ʵ��
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GameStart>();
							singletonObject.name = typeof(GameStart).ToString();

							// ȷ���������ᱻ����
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	public static void Name(string name)
	{
		AccountInfo.Instance.Account.Name = name;
	}

	static readonly object Lock = new object();
	static GameStart instance;
}

