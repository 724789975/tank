using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class AIStart : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
        
    }

	public static AIStart Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<AIStart>();
						if (instance == null)
						{
							// �����µ�ʵ��
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<AIStart>();
							singletonObject.name = typeof(AIStart).ToString();

							// ȷ���������ᱻ����
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	static readonly object Lock = new object();
	static AIStart instance;

}
