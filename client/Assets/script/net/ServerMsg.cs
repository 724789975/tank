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
		TankGame.LoginReq loginReq = anyMessage.Unpack<TankGame.LoginReq>();
		Debug.Log($"OnLoginReq {loginReq.Name} {loginReq.Id}");

		// 回复 LoginReq 消息
		TankGame.LoginRsp loginRspMessage = new TankGame.LoginRsp();
		if (PlayerManager.Instance.AddPlayer(loginReq.Id, new PlayerData() { Id = loginReq.Id, Name = loginReq.Name }))
		{
			loginRspMessage.Code = 0;
			loginRspMessage.Msg = "登录成功";
			byte[] messageBytes = Any.Pack(loginRspMessage).ToByteArray();
			DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);

			TankGame.PlayerApperanceNtf playerApperanceNtf = new TankGame.PlayerApperanceNtf();
			playerApperanceNtf.Id = loginReq.Id;
			playerApperanceNtf.Name = loginReq.Name;
			messageBytes = Any.Pack(playerApperanceNtf).ToByteArray();
			DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);

			TankManager.Instance.AddTank(loginReq.Id);
		}
		else
		{
			loginRspMessage.Code = TankGame.ErrorCode.Failed;
			loginRspMessage.Msg = "重复登录";
			byte[] messageBytes = Any.Pack(loginRspMessage).ToByteArray();
			DLLImport.Send(pConnector, messageBytes, (uint)messageBytes.Length);
		}
	}

}
