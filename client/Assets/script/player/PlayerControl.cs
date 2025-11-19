using fxnetlib.dllimport;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using TankGame;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class PlayerControl : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        instance = this;
        NetClient.Instance.AddOnConnected(()=> {
            StartGame();
		});
		NetClient.Instance.Connect();
        Debug.Log("PlayerControl Start");
	}

    // Update is called once per frame
    void Update()
    {
    }

    public void StartGame()
    {
		TankGame.Ping pingMessage = new TankGame.Ping();
		pingMessage.Ts = Time.time;
		NetClient.Instance.SendMessage(pingMessage);

		TankGame.LoginReq req = new TankGame.LoginReq();
        req.Name = AccountInfo.Instance.Account.Name;
        req.Id = AccountInfo.Instance.Account.Openid;
		NetClient.Instance.SendMessage(req);
	}

    public void ShowNotice(string content)
    {
#if !AI_RUNNING
        if (notice == null)
        {
            return;
        }
        GameObject go = Instantiate(notice.gameObject, notice.transform.parent);
        Notice n = go.GetComponent<Notice>();
        n.text.text = content;
#endif
    }

    public static PlayerControl Instance
    {
        get
        {
            return instance;
        }
    }


	static PlayerControl instance;

    public Notice notice;
}
