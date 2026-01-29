using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using System.Collections.Generic;
using UnityEngine;

public class ServerControl : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
#if UNITY_SERVER && !AI_RUNNING
		Status.Instance.OnStatusChange += delegate (TankGame.GameState status, float stateTime)
		{
            TankGame.GameStateNtf state_ntf = new TankGame.GameStateNtf();
            state_ntf.State = status;
            state_ntf.Time = stateTime;

			Any state_any = Any.Pack(state_ntf);
			byte[] state_data = state_any.ToByteArray();
			PlayerManager.Instance.ForEach((player) =>
			{
				NetServer.Instance.SendMessage(player.session, state_data);
			});

            Debug.Log("ServerControl.StatusChange: " + status);
			if (status == TankGame.GameState.End)
			{
				TankGame.GameOverNtf ntf = new TankGame.GameOverNtf();

				Any any = Any.Pack(ntf);
				byte[] data = any.ToByteArray();
				PlayerManager.Instance.ForEach((player) =>
				{
					NetServer.Instance.SendMessage(player.session, data);
				});
			}
			if (status == TankGame.GameState.Destory)
			{
				Application.Quit();
			}
		};
#endif
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
