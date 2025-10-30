using Google.Protobuf.WellKnownTypes;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class TankControl : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        float t = Time.time;
        lastUpdateTime = t;
        lastSyncTime = t;
	}

	// Update is called once per frame
	void Update()
    {
#if !AI_RUNNING
        float t = Time.time;
        if (t - lastUpdateTime < UPDATE_INTERVAL)
        {
            return;
        }

        lastUpdateTime = t;
        TankInstance tankInstance = TankManager.Instance.GetTank(AccountInfo.Instance.Account.Openid);
        if (tankInstance == null)
        {
            return;
		}

		Vector2 dir = Terresquall.VirtualJoystick.GetAxis(0);
		if (dir.magnitude > 0.1f)
        {
			tankInstance.AddPos(dir * tankInstance.speed * UPDATE_INTERVAL);
			tankInstance.SetDir(dir.normalized);

            syncFlag = true;
        }

        if (t - lastSyncTime > 0.125f)
        {
			TankGame.PlayerStateSyncReq playerStateSyncReq = new TankGame.PlayerStateSyncReq();
			playerStateSyncReq.SyncTime = ClientFrame.Instance.CurrentTime;
            if (syncFlag)
            {
                playerStateSyncReq.Transform = new TankCommon.Transform();
                playerStateSyncReq.Transform.Position = new TankCommon.Vector3();
                playerStateSyncReq.Transform.Position.X = tankInstance.transform.position.x;
                playerStateSyncReq.Transform.Position.Y = tankInstance.transform.position.y;
                playerStateSyncReq.Transform.Position.Z = tankInstance.transform.position.z;
                playerStateSyncReq.Transform.Rotation = new TankCommon.Quaternion();
                playerStateSyncReq.Transform.Rotation.X = tankInstance.transform.rotation.x;
                playerStateSyncReq.Transform.Rotation.Y = tankInstance.transform.rotation.y;
                playerStateSyncReq.Transform.Rotation.Z = tankInstance.transform.rotation.z;
                playerStateSyncReq.Transform.Rotation.W = tankInstance.transform.rotation.w;
                syncFlag = false;
			}
			lastSyncTime = t;
            NetClient.Instance.SendMessage(playerStateSyncReq);
		}
#endif
    }

    public void Shoot()
    {
        TankInstance tankInstance = TankManager.Instance.GetTank(AccountInfo.Instance.Account.Openid);
		if (tankInstance == null)
        {
            return;
        }
        tankInstance.Shoot();
    }

#if !AI_RUNNING
    bool syncFlag = false;
#endif
    float lastUpdateTime = 0f;
    float lastSyncTime = 0f;
    const float UPDATE_INTERVAL = 0.03f;
}
