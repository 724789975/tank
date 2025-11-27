using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using fxnetlib.dllimport;

public class ServerControl : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
#if UNITY_SERVER && !AI_RUNNING
        updateTime += Time.deltaTime;
        if (updateTime < 1)
        {
            return;
        }
        updateTime = 0;
        List<string> waitDestroy = new List<string>();
        PlayerManager.Instance.ForEach((data =>
        {
            if (data.session == null)
            {
                TankInstance tankInstance = TankManager.Instance.GetTank(data.Id);
                if (tankInstance != null && tankInstance.offLineTime > 10)
                {
                    waitDestroy.Add(data.Id);
                }
			}
		}));

        foreach (string id in waitDestroy)
        {
            PlayerManager.Instance.RemovePlayer(id);
            TankManager.Instance.RemoveTank(id);
            TankGame.PlayerDisappearNtf playerDisappearNtf = new TankGame.PlayerDisappearNtf { Id = id };

            byte[] buffer = Any.Pack(playerDisappearNtf).ToByteArray();
            PlayerManager.Instance.ForEach((data =>
            {
				NetServer.Instance.SendMessage(data.session, buffer);
            }));
		}
#endif
	}

#if UNITY_SERVER && !AI_RUNNING
    float updateTime = 0;
#endif
}
