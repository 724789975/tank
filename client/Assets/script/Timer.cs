using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using UnityEngine;

public class TimerU : MonoBehaviour
{
	class Task
	{
		public float time;
		public Action callback;
	}
	// Start is called before the first frame update
	void Start()
	{

	}

	// Update is called once per frame
	void Update()
	{
		if (tasks.Count > 0)
		{
			Task task = tasks[0];
			if (task.time <= Time.time)
			{
				task.callback();
				tasks.RemoveAt(0);
			}
		}
	}

	public void AddTask(float time, Action callback)
	{
		Task task = new Task();
		task.time = time;
		task.callback = callback;
		tasks.Add(task);
		tasks.Sort((a, b) => a.time.CompareTo(b.time));
	}

	// 公共访问接口
	public static TimerU Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<TimerU>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<TimerU>();
							singletonObject.name = typeof(TimerU).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	static TimerU instance;
	static readonly object Lock = new object();

	List<Task> tasks = new List<Task>();
}