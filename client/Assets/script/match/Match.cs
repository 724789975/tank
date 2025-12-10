using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using UnityEngine;
using Dirichlet.Mediation;

public class Match : MonoBehaviour
{
	// Start is called before the first frame update
    void Start()
    {
#if UNITY_EDITOR || UNITY_ANDROID
		advertisement.SetActive(true);

		DirichletAdConfig config = new DirichletAdConfig.Builder()
			.WithMediaId(349065)
			.WithMediaName("pongpongpong")
			.WithMediaKey("f9DNEKTt194oz4zyr4XuIA1vbW3XIA4dV3e4qwh6eZGf6waPEPp3DF8W4pFz0Sxh")
			.EnableDebug(true)       // 默认关闭，如需查看日志请改为 true
			.ShakeEnabled(true)       // 默认启用摇一摇
			.Build();

		DirichletSdk.Init(
			config,
			onSuccess: result =>
			{
				Debug.Log($"Dirichlet 初始化成功: {result.Message}");
			},
			onFailure: error =>
			{
				Debug.LogError($"Dirichlet 初始化失败: {error.Code} {error.Message}");
			}
		);

		if (!DirichletSdk.IsInitialized)
		{
			Debug.LogWarning("Dirichlet SDK 尚未初始化完成");
		}

		string version = DirichletSdk.GetVersion();
		Debug.Log($"Dirichlet SDK Version: {version}");
#endif
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

	public void LoadRewardAd()
	{
		DirichletRewardVideoAd rewardAd;
		DirichletAdNative adNative = DirichletAdManager.CreateAdNative();
		var request = new DirichletAdRequest.Builder()
		.WithSpaceId(124)
		.WithUserId(SystemInfo.deviceUniqueIdentifier)
		.WithRewardName("金币")
		.WithRewardAmount(100)
		.WithExtra1("level-5")
		.WithExpressViewSize(1080, 1920)   // Splash/Banner 可选
		.Build();
		adNative.LoadRewardVideoAd(request,
			onLoaded: ad =>
			{
				rewardAd = ad;
				rewardAd.Shown += () => Debug.Log("激励视频展示");
				rewardAd.Clicked += () => Debug.Log("激励视频点击");
				rewardAd.RewardVerified += args =>
				{
					if (args.IsVerified)
					{
						//GrantReward(args.RewardAmount, args.RewardName);
					}
					else
					{
						Debug.LogWarning($"奖励验证失败: {args.Code} {args.Message}");
					}
				};
				rewardAd.Closed += () =>
				{
					rewardAd.Destroy();
					rewardAd = null;
				};
			},
			onFailure: error => Debug.LogError($"激励视频加载失败 {error.Code}:{error.Message}"));
	}

	public GameObject advertisement;
}

