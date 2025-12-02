using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class Status : MonoBehaviour
{
    public enum StatusType
    {
        None = 0,
        Ready,      // 准备中
        Fight,      // 战斗中
        End,        // 战斗结束
        Destory,    // 被摧毁
    }

    // Start is called before the first frame update
    void Start()
    {
        instance = this;
        statusType = StatusType.None;
#if UNITY_SERVER && !AI_RUNNING
        statusType = StatusType.Ready;
        Debug.Log("Server is Ready");
        TimerU.Instance.AddTask(10, () => {
            statusType = StatusType.Fight;
            Debug.Log("Server is Fight");

            TimerU.Instance.AddTask(3 * 60, () => {
                statusType = StatusType.End;
                Debug.Log("Server is End");
                TimerU.Instance.AddTask(10, () =>
                {
                    statusType = StatusType.Destory;
                    Debug.Log("Server is Destory");
                });
                OnStatusChange?.Invoke(statusType);
            });
            OnStatusChange?.Invoke(statusType);
        });
        OnStatusChange?.Invoke(statusType);
#endif
    }

	// Update is called once per frame
	void Update()
    {
        
    }

    static Status instance;
    public static Status Instance
    {
        get
        {
            return instance;
        }
    }

    public StatusType statusType = StatusType.None;

    public delegate void StatusChangeHandler(StatusType statusType);
    public StatusChangeHandler OnStatusChange;
}

