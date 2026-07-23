using UnityEngine;

public class PlayerControl : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        instance = this;
		NetClient.Instance.Create();
#if CLIENT_WS
		// WS 模式下 StartGame 由 NetClient.OnOpen 触发
#else
		// FxNet 模式下连接成功回调(OnConnectedCallback)只遍历 onConnected 列表,
		// 不会自动触发登录; 需显式注册 StartGame, 否则 Ping/LoginReq 永不发送
		NetClient.Instance.AddOnConnected(StartGame);
#endif
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
        GameObject g = Resources.Load<GameObject>("prafab/notice");
        if (g == null)
        {
            return;
        }
        GameObject go = Instantiate(g, g.transform.parent);
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
}
