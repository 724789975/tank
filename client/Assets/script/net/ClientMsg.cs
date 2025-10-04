using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

#if UNITY_SERVER
using PLAYERDATA = ServerPlayer;
#else
using PLAYERDATA = ClientPlayer;
#endif

public class ClientMsg : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        instance = this;
        MsgProcess.Instance.RegisterHandler(this);
    }

    // Update is called once per frame
    void Update()
    {
        
    }

    static ClientMsg instance;

    [RpcHandler("tank_game.Pong")]
    static void Pong(IntPtr pConnection, Any anyMessage)
    {
        TankGame.Pong pong = anyMessage.Unpack<TankGame.Pong>();
        Debug.Log($"OnPong {pong.Ts}");
    }

    [RpcHandler("tank_game.LoginRsp")]
    static void LoginRsp(IntPtr pConnection, Any anyMessage)
    {
        TankGame.LoginRsp loginRsp = anyMessage.Unpack<TankGame.LoginRsp>();
        Debug.Log($"OnLoginRsp {loginRsp.Code} {loginRsp.Msg}");
    }

    [RpcHandler("tank_game.PlayerApperanceNtf")]
    static void PlayerApperanceNtf(IntPtr pConnection, Any anyMessage)
    {
        TankGame.PlayerApperanceNtf playerApperanceNtf = anyMessage.Unpack<TankGame.PlayerApperanceNtf>();
        Debug.Log($"OnPlayerApperanceNtf {playerApperanceNtf.Id} {playerApperanceNtf.Name}");
		TankInstance tankInstance = TankManager.Instance.AddTank(playerApperanceNtf.Id);
        tankInstance.name = "tank:" + playerApperanceNtf.Id;
        if (PlayerManager.Instance.AddPlayer(playerApperanceNtf.Id, new PLAYERDATA() { Id = playerApperanceNtf.Id, Name = playerApperanceNtf.Name }))
        {
            Debug.Log($"Player added successfully: {playerApperanceNtf.Id}");
            if (playerApperanceNtf.Transform != null)
            {
                if (playerApperanceNtf.Transform.Position != null)
                {
                    tankInstance.transform.position = new Vector3(playerApperanceNtf.Transform.Position.X, playerApperanceNtf.Transform.Position.Y, playerApperanceNtf.Transform.Position.Z);
				}
                if (playerApperanceNtf.Transform.Rotation != null)
                {
                    tankInstance.transform.rotation = new Quaternion(playerApperanceNtf.Transform.Rotation.X, playerApperanceNtf.Transform.Rotation.Y, playerApperanceNtf.Transform.Rotation.Z, playerApperanceNtf.Transform.Rotation.W);
                }
            }
        }
        else
        {
            Debug.LogWarning($"Failed to add player {playerApperanceNtf.Id} {playerApperanceNtf.Name}, ID already exists");
        }
    }

    [RpcHandler("tank_game.PlayerStateNtf")]
    static void PlayerStateNtf(IntPtr pConnection, Any anyMessage)
    {
#if !UNITY_SERVER
		TankGame.PlayerStateNtf playerStateNtf = anyMessage.Unpack<TankGame.PlayerStateNtf>();
        PLAYERDATA playerData = PlayerManager.Instance.GetPlayer(playerStateNtf.Id);
        if (playerData == null)
        {
            Debug.LogWarning($"Player data not found: {playerStateNtf.Id}");
            return;
		}

        if (playerStateNtf.Transform != null)
        {
            playerData.syncs.Add(playerStateNtf);
        }
#endif
	}

    [RpcHandler("tank_game.PlayerShootNtf")]
    static void PlayerShootNtf(IntPtr pConnection, Any anyMessage)
    {
#if !UNITY_SERVER
		TankGame.PlayerShootNtf playerStateNtf = anyMessage.Unpack<TankGame.PlayerShootNtf>();
		PLAYERDATA playerData = PlayerManager.Instance.GetPlayer(playerStateNtf.Id);
		if (playerData == null)
		{
			Debug.LogWarning($"Player data not found: {playerStateNtf.Id}");
			return;
		}

        TankInstance tankInstance = TankManager.Instance.GetTank(playerData.Id);
        if (tankInstance == null)
        {
            Debug.LogWarning($"Tank instance not found: {playerData.Id}");
            return;
		}
        tankInstance.Shoot(playerStateNtf.Id, new Vector3(playerStateNtf.Transform.Position.X, playerStateNtf.Transform.Position.Y, playerStateNtf.Transform.Position.Z), new Quaternion(playerStateNtf.Transform.Rotation.X, playerStateNtf.Transform.Rotation.Y, playerStateNtf.Transform.Rotation.Z, playerStateNtf.Transform.Rotation.W), playerStateNtf.Speed);
#endif
	}

	[RpcHandler("tank_game.BulletDestoryNtf")]
	static void BulletDestoryNtf(IntPtr pConnection, Any anyMessage)
    {
#if !UNITY_SERVER
        TankGame.BulletDestoryNtf bulletDestoryNtf = anyMessage.Unpack<TankGame.BulletDestoryNtf>();
        BulletManager.Instance.RemoveBullet(bulletDestoryNtf.Id);
        Instantiate(instance.boomPrefab, new Vector3(bulletDestoryNtf.Pos.X, bulletDestoryNtf.Pos.Y, bulletDestoryNtf.Pos.Z), Quaternion.identity);
#endif
    }
    public GameObject boomPrefab;
}

