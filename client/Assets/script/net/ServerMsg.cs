using Google.Protobuf.WellKnownTypes;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using Google.Protobuf;
using fxnetlib.dllimport;
using System;

public class ServerMsg : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		instance = this;
	}

	// Update is called once per frame
	void Update()
    {
        
    }

	[RpcHandler("tank_game.Ping")]
	static void Ping(object pConnector, Any anyMessage)
	{
		TankGame.Ping ping = anyMessage.Unpack<TankGame.Ping>();

		// 回复 Ping 消息
		TankGame.Pong pongMessage = new TankGame.Pong();
		pongMessage.Ts = ping.Ts;
		pongMessage.CurrentTime = ServerFrame.Instance.CurrentTime;
		//Debug.Log($"OnPing {ping.ToString()}, {pongMessage.ToString()}");
		NetServer.Instance.SendMessage(pConnector, pongMessage);
	}

	[RpcHandler("tank_game.LoginReq")]
	static void LoginReq(object pConnector, Any anyMessage)
	{
#if UNITY_SERVER && !AI_RUNNING
		TankGame.LoginReq loginReq = anyMessage.Unpack<TankGame.LoginReq>();
		Debug.Log($"OnLoginReq {loginReq.Name} {loginReq.Id}");

		bool bRemovePlayer = false;

		// 回复 LoginReq 消息
		TankGame.LoginRsp loginRspMessage = new TankGame.LoginRsp();
		ServerPlayer playerData = PlayerManager.Instance.GetPlayer(loginReq.Id);
		if (playerData != null)
		{
			//踢出老用户
			PlayerManager.Instance.RemovePlayer(playerData.Id);
			Debug.Log("kick old player");
			bRemovePlayer = true;
		}
		if (PlayerManager.Instance.AddPlayer(loginReq.Id, new ServerPlayer() { Id = loginReq.Id, Name = loginReq.Name,
			session = pConnector,
		}))
		{
			Debug.Log("into players");
			loginRspMessage.Code = 0;
			loginRspMessage.Msg = "Login successful";
			NetServer.Instance.SendMessage(pConnector, loginRspMessage);

			TankInstance tankInstance = TankManager.Instance.AddTank(loginReq.Id, loginReq.Name, out bool isAdd);
			tankInstance.name = "tank:" + loginReq.Id;

			TankGame.PlayerApperanceNtf playerApperanceNtf = new TankGame.PlayerApperanceNtf();
			playerApperanceNtf.Id = loginReq.Id;
			playerApperanceNtf.Name = loginReq.Name;
			if (isAdd)
			{
				tankInstance.HP = Config.Instance.maxHp;
				tankInstance.transform.position = new Vector3(UnityEngine.Random.Range(Config.Instance.GetLeft() + 1, Config.Instance.GetRight() - 1), UnityEngine.Random.Range(Config.Instance.GetTop() - 1, Config.Instance.GetBottom() + 1), 0);
				tankInstance.transform.right = new Vector3(UnityEngine.Random.Range(-1f, 1f), UnityEngine.Random.Range(-1f, 1f), 0).normalized;
				tankInstance.rebornTime = Config.Instance.rebornProtectionTime;
			}
			playerApperanceNtf.Hp = tankInstance.HP;
			playerApperanceNtf.Transform = new TankCommon.Transform();
			playerApperanceNtf.Transform.Position = new TankCommon.Vector3() { X = tankInstance.transform.position.x, Y = tankInstance.transform.position.y, Z = tankInstance.transform.position.z};
			playerApperanceNtf.Transform.Rotation = new TankCommon.Quaternion() { X = tankInstance.transform.rotation.x, Y = tankInstance.transform.rotation.y, Z = tankInstance.transform.rotation.z, W = tankInstance.transform.rotation.w};
			playerApperanceNtf.RebornProtectTime = tankInstance.rebornTime + ServerFrame.Instance.CurrentTime;

			byte[] messageBytes2 = Any.Pack(playerApperanceNtf).ToByteArray();
			NetServer.Instance.SendMessage(pConnector, messageBytes2);
			Debug.Log("send appearance");

			TankGame.GameStateNtf state_ntf = new TankGame.GameStateNtf();
			state_ntf.State = Status.Instance.status;
			state_ntf.Time = Status.Instance.stateTime;

			NetServer.Instance.SendMessage(pConnector, state_ntf);

			PlayerManager.Instance.ForEach((playerData) =>
				{
					if (playerData.Id != loginReq.Id)
					{
						if (!bRemovePlayer)
						{
							NetServer.Instance.SendMessage(playerData.session, messageBytes2);
						}
						TankGame.PlayerApperanceNtf playerJoinNtf = new TankGame.PlayerApperanceNtf();
						playerJoinNtf.Id = playerData.Id;
						playerJoinNtf.Name = playerData.Name;
						TankInstance otherTankInstance = TankManager.Instance.GetTank(playerData.Id);
						if (otherTankInstance != null)
						{
							playerJoinNtf.Hp = otherTankInstance.HP;
							playerJoinNtf.Transform = new TankCommon.Transform();
							playerJoinNtf.Transform.Position = new TankCommon.Vector3() { X = otherTankInstance.transform.position.x, Y = otherTankInstance.transform.position.y, Z = otherTankInstance.transform.position.z };
							playerJoinNtf.Transform.Rotation = new TankCommon.Quaternion() { X = otherTankInstance.transform.rotation.x, Y = otherTankInstance.transform.rotation.y, Z = otherTankInstance.transform.rotation.z, W = otherTankInstance.transform.rotation.w };
						}
						NetServer.Instance.SendMessage(pConnector, playerJoinNtf);
					}
				});

		}
		else
		{
			loginRspMessage.Code = Common.ErrorCode.Failed;
			loginRspMessage.Msg = "Duplicate login";
			NetServer.Instance.SendMessage(pConnector, loginRspMessage);
		}
#endif
	}

	[RpcHandler("tank_game.PlayerStateSyncReq")]
	static void PlayerStateSyncReq(object pConnector, Any anyMessage)
	{
#if UNITY_SERVER && !AI_RUNNING
		TankGame.PlayerStateSyncReq playerStateSyncReq = anyMessage.Unpack<TankGame.PlayerStateSyncReq>();
		ServerPlayer playerData = PlayerManager.Instance.GetPlayerBySession(pConnector);
		if (playerData == null) {
			Debug.LogWarning($"Player data not found: {pConnector}");
			return;
		}

		TankManager.Instance.GetTank(playerData.Id);
		TankInstance tankInstance = TankManager.Instance.GetTank(playerData.Id);
		if (tankInstance == null)
		{
			Debug.LogWarning($"Tank instance not found: {playerData.Id}");
			return;
		}

		if (playerStateSyncReq.Transform != null)
		{
			if (playerStateSyncReq.Transform.Position != null)
			{
				playerData.speedCheckDelate += playerStateSyncReq.SyncTime - playerData.SyncTime;
				Vector3 np = new Vector3(playerStateSyncReq.Transform.Position.X, playerStateSyncReq.Transform.Position.Y, playerStateSyncReq.Transform.Position.Z);
				if (Vector3.Distance(playerData.lastPos, np) > tankInstance.speed * playerData.speedCheckDelate * 1.01f)
				{
					Debug.LogWarning($"Position jump too large from {playerData.lastPos} to {np}, distance {Vector3.Distance(playerData.lastPos, np)}, max {tankInstance.speed * playerData.speedCheckDelate * 1.01f}");
				}
				else
				{
					playerData.speedCheckDelate = 0;
					playerData.lastPos = np;
				}
				tankInstance.transform.position = np;
			}
			if (playerStateSyncReq.Transform.Rotation != null)
			{
				tankInstance.transform.rotation = new Quaternion(playerStateSyncReq.Transform.Rotation.X, playerStateSyncReq.Transform.Rotation.Y, playerStateSyncReq.Transform.Rotation.Z, playerStateSyncReq.Transform.Rotation.W);
			}
		}

		playerData.SyncTime = playerStateSyncReq.SyncTime;

		TankGame.PlayerStateNtf playerStateNtf = new TankGame.PlayerStateNtf();
		playerStateNtf.Id = playerData.Id;
		playerStateNtf.Transform = playerStateSyncReq.Transform;
		playerStateNtf.SyncTime = playerData.SyncTime;
		byte[] messageBytes = Any.Pack(playerStateNtf).ToByteArray();

		PlayerManager.Instance.ForEach((pd) =>
		{
			if (pd.Id != playerData.Id)
			{
				if (((ServerPlayer)pd).session != null)
				{
					NetServer.Instance.SendMessage(((ServerPlayer)pd).session, messageBytes);
				}
			}
		});
		playerData.SyncTime = 0;
#endif
	}

	[RpcHandler("tank_game.PlayerShootReq")]
	static void PlayerShootReq(object pConnector, Any anyMessage)
	{
#if UNITY_SERVER && !AI_RUNNING
		TankGame.PlayerShootReq playerShootReq = anyMessage.Unpack<TankGame.PlayerShootReq>();
		ServerPlayer playerData = PlayerManager.Instance.GetPlayerBySession(pConnector);
		if (playerData == null)
		{
			Debug.LogWarning($"Player data not found: {pConnector}");
			return;
		}
		TankManager.Instance.GetTank(playerData.Id);
		TankInstance tankInstance = TankManager.Instance.GetTank(playerData.Id);
		if (tankInstance == null)
		{
			Debug.LogWarning($"Tank instance not found: {playerData.Id}");
			return;
		}
		tankInstance.Shoot(playerData.Id, new Vector3(playerShootReq.Transform.Position.X, playerShootReq.Transform.Position.Y, playerShootReq.Transform.Position.Z), new Quaternion(playerShootReq.Transform.Rotation.X, playerShootReq.Transform.Rotation.Y, playerShootReq.Transform.Rotation.Z, playerShootReq.Transform.Rotation.W), Config.Instance.speed);
		TankGame.PlayerShootNtf playerShootNtf = new TankGame.PlayerShootNtf();
		playerShootNtf.Id = playerData.Id;
		playerShootNtf.Speed = Config.Instance.speed;
		playerShootNtf.Transform = playerShootReq.Transform;

		byte[] messageBytes = Any.Pack(playerShootNtf).ToByteArray();
		PlayerManager.Instance.ForEach((pd) =>
		{
			if (pd.Id != playerData.Id)
			{
				NetServer.Instance.SendMessage(((ServerPlayer)pd).session, messageBytes);
			}
		});
#endif
	}

	static ServerMsg instance;
	public static ServerMsg Instance
	{
		get
		{
			return instance;
		}
	}
}
