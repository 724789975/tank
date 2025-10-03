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
		MsgProcess.Instance.RegisterHandler(this);
	}

	// Update is called once per frame
	void Update()
    {
        
    }


	[RpcHandler("tank_game.Ping")]
	static void Ping(IntPtr pConnector, Any anyMessage)
	{
		TankGame.Ping ping = anyMessage.Unpack<TankGame.Ping>();
		Debug.Log($"OnPing {ping.Ts}");

		// 回复 Ping 消息
		TankGame.Pong pongMessage = new TankGame.Pong();
		pongMessage.Ts = DateTime.Now.Ticks;
		byte[] messageBytes = Any.Pack(pongMessage).ToByteArray();
		DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);
	}

	[RpcHandler("tank_game.LoginReq")]
	static void LoginReq(IntPtr pConnector, Any anyMessage)
	{
#if UNITY_SERVER
		TankGame.LoginReq loginReq = anyMessage.Unpack<TankGame.LoginReq>();
		Debug.Log($"OnLoginReq {loginReq.Name} {loginReq.Id}");

		bool bRemovePlayer = PlayerManager.Instance.RemovePlayer(loginReq.Id);

		// 回复 LoginReq 消息
		TankGame.LoginRsp loginRspMessage = new TankGame.LoginRsp();
		if (PlayerManager.Instance.AddPlayer(loginReq.Id, new ServerPlayer() { Id = loginReq.Id, Name = loginReq.Name,
			session = pConnector,
		}))
		{
			Debug.Log("into players");
			loginRspMessage.Code = 0;
			loginRspMessage.Msg = "Login successful";
			byte[] messageBytes = Any.Pack(loginRspMessage).ToByteArray();
			DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);

			TankInstance tankInstance = TankManager.Instance.AddTank(loginReq.Id);

			TankGame.PlayerApperanceNtf playerApperanceNtf = new TankGame.PlayerApperanceNtf();
			playerApperanceNtf.Id = loginReq.Id;
			playerApperanceNtf.Name = loginReq.Name;
			playerApperanceNtf.Transform = new TankCommon.Transform();
			playerApperanceNtf.Transform.Position = new TankCommon.Vector3() { X = tankInstance.transform.position.x, Y = tankInstance.transform.position.y, Z = tankInstance.transform.position.z};
			playerApperanceNtf.Transform.Rotation = new TankCommon.Quaternion() { X = tankInstance.transform.rotation.x, Y = tankInstance.transform.rotation.y, Z = tankInstance.transform.rotation.z, W = tankInstance.transform.rotation.w};

			byte[] messageBytes2 = Any.Pack(playerApperanceNtf).ToByteArray();
			DLLImport.Send(pConnector, messageBytes2, (uint)messageBytes2.Length);
			Debug.Log("send appearance");
			PlayerManager.Instance.ForEach((playerData) =>
				{
					if (playerData.Id != loginReq.Id)
					{
						if (!bRemovePlayer)
						{ 
							DLLImport.Send(((ServerPlayer)playerData).session, messageBytes2, (uint)messageBytes2.Length);
						}
						TankGame.PlayerApperanceNtf playerJoinNtf = new TankGame.PlayerApperanceNtf();
						playerJoinNtf.Id = playerData.Id;
						playerJoinNtf.Name = playerData.Name;
						TankInstance otherTankInstance = TankManager.Instance.GetTank(playerData.Id);
						if (otherTankInstance != null)
						{
							playerJoinNtf.Transform = new TankCommon.Transform();
							playerJoinNtf.Transform.Position = new TankCommon.Vector3() { X = otherTankInstance.transform.position.x, Y = otherTankInstance.transform.position.y, Z = otherTankInstance.transform.position.z };
							playerJoinNtf.Transform.Rotation = new TankCommon.Quaternion() { X = otherTankInstance.transform.rotation.x, Y = otherTankInstance.transform.rotation.y, Z = otherTankInstance.transform.rotation.z, W = otherTankInstance.transform.rotation.w };
						}
						byte[] messageBytes3 = Any.Pack(playerJoinNtf).ToByteArray();
						DLLImport.Send(pConnector, messageBytes3, (uint)messageBytes3.Length);
					}
				});

		}
		else
		{
			loginRspMessage.Code = TankGame.ErrorCode.Failed;
			loginRspMessage.Msg = "Duplicate login";
			byte[] messageBytes = Any.Pack(loginRspMessage).ToByteArray();
			DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);
		}
#endif
	}

	[RpcHandler("tank_game.PlayerStateSyncReq")]
	static void PlayerStateSyncReq(IntPtr pConnector, Any anyMessage)
	{
#if UNITY_SERVER
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
				playerData.speedCheckDelate += playerStateSyncReq.DeltaMs;
				Vector3 np = new Vector3(playerStateSyncReq.Transform.Position.X, playerStateSyncReq.Transform.Position.Y, playerStateSyncReq.Transform.Position.Z);
				if (Vector3.Distance(playerData.lastPos, np) > tankInstance.speed * playerData.speedCheckDelate / 990)
				{
					Debug.LogWarning($"Position jump too large from {playerData.lastPos} to {np}, distance {Vector3.Distance(playerData.lastPos, np)}, max {tankInstance.speed * playerData.speedCheckDelate / 1000}");
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

		playerData.SyncTime += playerStateSyncReq.DeltaMs;

		TankGame.PlayerStateNtf playerStateNtf = new TankGame.PlayerStateNtf();
		playerStateNtf.Id = playerData.Id;
		playerStateNtf.Transform = playerStateSyncReq.Transform;
		playerStateNtf.SyncTime = playerData.SyncTime;
		byte[] messageBytes = Any.Pack(playerStateNtf).ToByteArray();

		PlayerManager.Instance.ForEach((pd) =>
		{
			if (pd.Id != playerData.Id)
			{
				if (((ServerPlayer)pd).session != IntPtr.Zero)
				{
					DLLImport.Send(((ServerPlayer)pd).session, messageBytes, (uint)messageBytes.Length);
				}
			}
		});
		playerData.SyncTime = 0;
#endif
	}

	[RpcHandler("tank_game.PlayerShootReq")]
	static void PlayerShootReq(IntPtr pConnector, Any anyMessage)
	{
#if UNITY_SERVER
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
		tankInstance.Shoot(new Vector3(playerShootReq.Transform.Position.X, playerShootReq.Transform.Position.Y, playerShootReq.Transform.Position.Z), new Quaternion(playerShootReq.Transform.Rotation.X, playerShootReq.Transform.Rotation.Y, playerShootReq.Transform.Rotation.Z, playerShootReq.Transform.Rotation.W), Config.Instance.speed);
		TankGame.PlayerShootNtf playerShootNtf = new TankGame.PlayerShootNtf();
		playerShootNtf.Id = playerData.Id;
		playerShootNtf.Speed = Config.Instance.speed;
		playerShootNtf.Transform = playerShootReq.Transform;

		byte[] messageBytes = Any.Pack(playerShootNtf).ToByteArray();
		PlayerManager.Instance.ForEach((pd) =>
		{
			if (pd.Id != playerData.Id)
			{
				if (((ServerPlayer)pd).session != IntPtr.Zero)
				{
					DLLImport.Send(((ServerPlayer)pd).session, messageBytes, (uint)messageBytes.Length);
				}
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
