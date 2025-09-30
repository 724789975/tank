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
		if (PlayerManager.Instance.AddPlayer(loginReq.Id, new PlayerData() { Id = loginReq.Id, Name = loginReq.Name,
			session = pConnector,
		}))
		{
			loginRspMessage.Code = 0;
			loginRspMessage.Msg = "登录成功";
			byte[] messageBytes = Any.Pack(loginRspMessage).ToByteArray();
			DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);

			TankGame.PlayerApperanceNtf playerApperanceNtf = new TankGame.PlayerApperanceNtf();
			playerApperanceNtf.Id = loginReq.Id;
			playerApperanceNtf.Name = loginReq.Name;

			byte[] messageBytes2 = Any.Pack(playerApperanceNtf).ToByteArray();
			DLLImport.Send(pConnector, messageBytes2, (uint)messageBytes2.Length);
			PlayerManager.Instance.ForEach((playerData) =>
				{
					if (playerData.Id != loginReq.Id)
					{
						if (!bRemovePlayer)
						{ 
							DLLImport.Send(playerData.session, messageBytes2, (uint)messageBytes2.Length);
						}
						TankGame.PlayerApperanceNtf playerJoinNtf = new TankGame.PlayerApperanceNtf();
						playerJoinNtf.Id = playerData.Id;
						playerJoinNtf.Name = playerData.Name;
						byte[] messageBytes3 = Any.Pack(playerJoinNtf).ToByteArray();
						DLLImport.Send(pConnector, messageBytes3, (uint)messageBytes3.Length);
					}
				});

			TankManager.Instance.AddTank(loginReq.Id);
			if (!bRemovePlayer)
			{ 
			}
		}
		else
		{
			loginRspMessage.Code = TankGame.ErrorCode.Failed;
			loginRspMessage.Msg = "重复登录";
			byte[] messageBytes = Any.Pack(loginRspMessage).ToByteArray();
			DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);
		}
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
