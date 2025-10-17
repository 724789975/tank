using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.SceneManagement;

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
        ClientFrame.Instance.CorrectFrame(pong.CurrentTime, Time.time - pong.Ts);
        Debug.Log($"OnPong {pong.ToString()}, current_time:{ClientFrame.Instance.CurrentTime}, Latency:{ClientFrame.Instance.Latency}");
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
		TankInstance tankInstance = TankManager.Instance.AddTank(playerApperanceNtf.Id, playerApperanceNtf.Name, out bool isAdd);
        tankInstance.name = "tank:" + playerApperanceNtf.Id;
        tankInstance.ID = playerApperanceNtf.Id;
        tankInstance.HP = playerApperanceNtf.Hp;
        tankInstance.rebornTime = playerApperanceNtf.RebornProtectTime - ClientFrame.Instance.CurrentTime;
        Debug.Log($"OnPlayerApperanceNtf {playerApperanceNtf.ToString()} {tankInstance.rebornTime} {ClientFrame.Instance.CurrentTime}");
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


    [RpcHandler("tank_game.PlayerDisappearNtf")]
    static void PlayerDisappearNtf(IntPtr pConnection, Any anyMessage)
    {
#if !UNITY_SERVER
		TankGame.PlayerDisappearNtf playerDisappearNtf = anyMessage.Unpack<TankGame.PlayerDisappearNtf>();
        TankManager.Instance.RemoveTank(playerDisappearNtf.Id);
        PlayerManager.Instance.RemovePlayer(playerDisappearNtf.Id);
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
        Instantiate(instance.boomPrefab, new Vector3(bulletDestoryNtf.Pos.X, bulletDestoryNtf.Pos.Y, bulletDestoryNtf.Pos.Z - 5), Quaternion.identity);
#endif
    }

    [RpcHandler("tank_game.TankHpSyncNtf")]
    static void TankHpSyncNtf(IntPtr pConnection, Any anyMessage)
    {
#if !UNITY_SERVER
        TankGame.TankHpSyncNtf tankHpSyncNtf = anyMessage.Unpack<TankGame.TankHpSyncNtf>();
        TankInstance tankInstance = TankManager.Instance.GetTank(tankHpSyncNtf.Id);
        if (tankInstance == null)
        {
            Debug.LogWarning($"Tank instance not found: {tankHpSyncNtf.Id}");
            return;
        }
        tankInstance.HP = tankHpSyncNtf.Hp;
#endif
    }

	[RpcHandler("tank_game.PlayerDieNtf")]
	static void PlayerDieNtf(IntPtr pConnection, Any anyMessage)
    {
#if !UNITY_SERVER
        TankGame.PlayerDieNtf playerDieNtf = anyMessage.Unpack<TankGame.PlayerDieNtf>();
        TankInstance tankKilled = TankManager.Instance.GetTank(playerDieNtf.KilledId);
        if (tankKilled == null)
        {
            Debug.LogWarning($"Tank instance not found: {playerDieNtf.KilledId}");
            return;
        }
        tankKilled.rebornTime = playerDieNtf.RebornProtectTime - ClientFrame.Instance.CurrentTime;

        TankInstance tankKiller = TankManager.Instance.GetTank(playerDieNtf.KillerId);
        if (tankKiller == null)
        {
            Debug.LogWarning($"Tank instance not found: {playerDieNtf.KillerId}");
            return;
        }

        string notice = $"玩家 <color=#00FF00>{tankKilled.Name}</color> 已被 <color=#F00000>{tankKiller.Name}</color> 击毁";
        if (tankKilled.ID == tankKiller.ID)
        {
            notice = $"玩家 <color=#00FF00>{tankKilled.Name}</color> 自爆了";
            if (tankKiller.ID == AccountInfo.Instance.Account.Openid)
            {
                notice = $"<color=#00FF00>你</color> 自爆了";
            }
        }
        else
        {
            if (tankKiller.ID == AccountInfo.Instance.Account.Openid)
            {
                notice = $"<color=#00FF00>你</color> 击毁了 <color=#F00000>{tankKilled.Name}</color>";
            }
            else if (tankKilled.ID == AccountInfo.Instance.Account.Openid)
			{
				notice = $"<color=#00FF00>你</color>被 <color=#F00000>{tankKiller.Name}</color> 击毁了";
            }
        }

        PlayerControl.Instance.ShowNotice(notice);

        Debug.Log($"OnPlayerDieNtf {playerDieNtf.ToString()} {tankKilled.rebornTime} {ClientFrame.Instance.CurrentTime}");
#endif
    }


	[RpcHandler("tank_game.GameOverNtf")]
	static void GameOverNtf(IntPtr pConnection, Any anyMessage)
    {
#if !UNITY_SERVER
        TankGame.GameOverNtf gameOverNtf = anyMessage.Unpack<TankGame.GameOverNtf>();
        string notice = $"游戏结束，你获得了胜利";
		PlayerControl.Instance.ShowNotice(notice);

		// 转场
		UnityEngine.SceneManagement.SceneManager.LoadScene("match");
#endif
	}

	public GameObject boomPrefab;
}

