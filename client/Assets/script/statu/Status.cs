using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class Status : MonoBehaviour
{
    void Awake()
    {
        instance = this;
    }
    // Start is called before the first frame update
    void Start()
    {
        status = TankGame.GameState.None;
#if UNITY_SERVER && !AI_RUNNING
        status = TankGame.GameState.Ready;
        Debug.Log("Server is Ready");
        stateTime = TimerU.Instance.AddTask(10, () => {
            status = TankGame.GameState.Fight;
            Debug.Log("Server is Fight");

            stateTime = TimerU.Instance.AddTask(3 * 60, () => {
                status = TankGame.GameState.End;
				Debug.Log("Server will be shutdown in 10 seconds");
                stateTime = TimerU.Instance.AddTask(10, () =>
                {
                    status = TankGame.GameState.Destory;
                    Debug.Log("Server is Destory");
                }).time;
                OnStatusChange?.Invoke(status, stateTime);
            }).time;
            OnStatusChange?.Invoke(status, stateTime);
        }).time;
        OnStatusChange?.Invoke(status, stateTime);
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

    public TankGame.GameState status =  TankGame.GameState.None;
    public float stateTime = 0;

#if UNITY_SERVER && !AI_RUNNING
	public delegate void StatusChangeHandler(TankGame.GameState statusType, float stateTime);
    public StatusChangeHandler OnStatusChange;
#endif
}

