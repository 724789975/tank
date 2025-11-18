using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using UnityEngine;
public class Match : MonoBehaviour
{
	// Start is called before the first frame update
    void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
        
    }

    public void StartMatch()
    {
		AsyncWebRequest asyncWebRequest = new AsyncWebRequest();

		Login.UserChannel userChannel = new Login.UserChannel()
		{
			ver = "v1",
			exp = (long)(DateTime.UtcNow.AddSeconds(30) - new DateTime(1970, 1, 1)).TotalSeconds,
			userId = AccountInfo.Instance.Account.Openid,
		};

		string userChannelBody = JsonUtility.ToJson(userChannel);

		Dictionary<string, string> headers = new Dictionary<string, string>
		{
			{ "user-channel", userChannelBody }
		};

		Debug.Log($"userChannelBody {userChannelBody}");

		asyncWebRequest.Post("http://115.190.230.47:30080/api/1.0/public/match_server/match", "{}", headers, (ok, response) =>
		{
			if (!ok)
			{
				Debug.Log($"匹配请求失败，服务器响应异常：{response}");
			}
			else
			{
				string responseStr = Encoding.UTF8.GetString(response);
				Debug.Log($"匹配请求成功，服务器响应：{responseStr}");
				MatchProto.MatchResp rsp = MatchProto.MatchResp.Parser.ParseJson(responseStr);
			}
		});
	}

	public void PVE()
	{
		AsyncWebRequest asyncWebRequest = new AsyncWebRequest();

		Login.UserChannel userChannel = new Login.UserChannel()
		{
			ver = "v1",
			exp = (long)(DateTime.UtcNow.AddSeconds(30) - new DateTime(1970, 1, 1)).TotalSeconds,
			userId = AccountInfo.Instance.Account.Openid,
		};

		string userChannelBody = JsonUtility.ToJson(userChannel);

		Dictionary<string, string> headers = new Dictionary<string, string>
		{
			{ "user-channel", userChannelBody }
		};

		Debug.Log($"userChannelBody {userChannelBody}");

		asyncWebRequest.Post("http://115.190.230.47:30080/api/1.0/public/match_server/pve", "{}", headers, (ok, response) =>
		{
			if (!ok)
			{
				Debug.Log($"pve请求失败，服务器响应异常：{response}");
			}
			else
			{
				string responseStr = Encoding.UTF8.GetString(response);
				Debug.Log($"pve请求成功，服务器响应：{responseStr}");
				MatchProto.MatchResp rsp = MatchProto.MatchResp.Parser.ParseJson(responseStr);
			}
		});
	}
}

