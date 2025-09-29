using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class ClientMsg : MonoBehaviour
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
        if (PlayerManager.Instance.AddPlayer(playerApperanceNtf.Id, new PlayerData() { Id = playerApperanceNtf.Id, Name = playerApperanceNtf.Name }))
        {
            Debug.Log($"添加玩家 {playerApperanceNtf.Id} {playerApperanceNtf.Name} 成功");
            TankManager.Instance.AddTank(playerApperanceNtf.Id);
        }
        else
        {
            Debug.LogWarning($"添加玩家 {playerApperanceNtf.Id} {playerApperanceNtf.Name} 失败，ID已存在");
        }
    }
}
