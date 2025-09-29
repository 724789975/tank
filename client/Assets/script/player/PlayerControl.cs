using Google.Protobuf.WellKnownTypes;
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
    }

    // Update is called once per frame
    void Update()
    {
        
    }

    public void StartGame()
    {
		TankGame.LoginReq req = new TankGame.LoginReq();
        req.Name = nameText.text;
        req.Id = idText.text;
		NetClient.Instance.SendMessage(req);
    }

    public static PlayerControl Instance
    {
        get
        {
            return instance;
        }
    }

    public string PlayerName
    {
        get
        {
            return nameText.text;
        }
    }

	public string PlayerId
    {
        get
        {
            return idText.text;
        }
    }

	static PlayerControl instance;

	public TextMeshProUGUI nameText;
	public TextMeshProUGUI idText;
}
