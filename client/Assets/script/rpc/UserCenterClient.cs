using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class UserCenterClient : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
        
    }


	static UserCenterClient instance;
	// 公共访问接口
	public static UserCenterClient Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<UserCenterClient>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<UserCenterClient>();
							singletonObject.name = typeof(UserCenterClient).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}
	static readonly object Lock = new object();
}
