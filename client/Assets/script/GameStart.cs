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
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GameStart>();
							singletonObject.name = typeof(GameStart).ToString();

							// 确保单例不会被销毁
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

