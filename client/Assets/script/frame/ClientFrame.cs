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
            points = 0f;
            differential = 0f;
            return;
        }

        points += lagBehind;
        syncRate = lagBehind / 100.0f;

        differential = lagBehind;

		//由以上信息估算syncRate

	}

	static ClientFrame instane;
    bool start = false;
	float currentTime = 0;
    float latency = 0.0f;

    float differential = 0f;
    float points = 0f;

    float updateTime = 0;


	public float syncRate = 0f; // 调整时间轮的快慢
}
