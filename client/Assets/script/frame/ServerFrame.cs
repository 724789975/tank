using Google.Protobuf.WellKnownTypes;
using System.Collections;
using System.Collections.Generic;
using Google.Protobuf;
using UnityEngine;

public class ServerFrame : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        instane = this;
    }

    // Update is called once per frame
    void Update()
    {
#if UNITY_SERVER
        currentTime = Time.time;
        if (currentTime > 600)
        { 
			if (!isEnd)
            {
                isEnd = true;
                TankGame.GameOverNtf ntf = new TankGame.GameOverNtf();

                Any any = Any.Pack(ntf);
                byte[] data = any.ToByteArray();
                PlayerManager.Instance.ForEach((player) => {
                    NetServer.Instance.SendMessage(player.session, data);
				});

				Debug.Log("Server will be shutdown in 10 seconds");
            }
        }
        if (currentTime > 610)
        {
            Debug.Log("Server will be shutdown");
            Application.Quit();
        }
#endif
	}

	public static ServerFrame Instance
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

	static ServerFrame instane;
    float currentTime = 0;
#if UNITY_SERVER
    bool isEnd = false;
#endif
}
