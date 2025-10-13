using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class ClientFrame : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        instane = this;
    }

    // Update is called once per frame
    void Update()
    {
        if(!start)
        {
            return;
		}
		currentTime += Time.deltaTime * (1 + syncRate);
        updateTime += Time.deltaTime;

        if (updateTime > 1)
        {
            updateTime -= 1;
            TankGame.Ping ping = new TankGame.Ping();
            ping.Ts = Time.time;

            NetClient.Instance.SendMessage(ping);
        }
	}

	public static ClientFrame Instance
    {
        get
        {
            return instane;
        }
    }

	public float CurrentTime
	{
		get
		{
			return currentTime;
		}
	}

    public float Latency
    {
        get
        {
            return latency;
        }
    }

    public void ResetTime()
    {
        currentTime = 0;
	}

    public void CorrectFrame(float serverTime, float Latency)
    {
        if (!start)
        {
            start = true;
            currentTime = serverTime + latency / 2f;
            latency = Latency;
            return;
        }
        float lagBehind = serverTime - currentTime;

		if (Mathf.Abs(lagBehind) < 0.1f)
        {
            syncRate = 0.0f;
            return;
        }

        // 设计一个平滑的同步速率计算方式
        // 1. 基于落后时间的比例计算同步速率
        // 2. 限制最大调整幅度，避免时间变化过快
        float maxSyncRate = 0.9999f; // 最大调整比例 (±99.99%)
        float syncFactor = 0.1f;  // 同步系数，控制调整速度
        
        // 根据时间差计算同步速率
        syncRate = lagBehind * syncFactor;
        
        // 限制同步速率的最大值
        syncRate = Mathf.Clamp(syncRate, -maxSyncRate, maxSyncRate);
        
        // 更新延迟值
        latency = Latency;
    }

	static ClientFrame instane;
    bool start = false;
	float currentTime = 0;
    float latency = 0.0f;

    float updateTime = 0;

	public float syncRate = 0f; // 调整时间轮的快慢
}
