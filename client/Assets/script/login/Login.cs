//#define USE_TAP_LOGIN

using System.Collections;
using System.Collections.Generic;
using UnityEngine;

using TapSDK.Core;
using System.Threading.Tasks;
using TapSDK.Login;
using System;
using System.Text;


public class Login : MonoBehaviour
{
	[Serializable]
	public class PostData
	{
		public string kid;
		public string macKey;
	};
	[Serializable]
	public class UserChannel
	{
		public string ver;
		public long exp;
		public string userId;
	}
    // Start is called before the first frame update
    void Start()
    {
#if USE_TAP_LOGIN
		// 核心配置 详细参数见 [TapTapSDK]
		TapTapSdkOptions coreOptions = new TapTapSdkOptions()
		{  // 客户端 ID，开发者后台获取
			clientId = "oy5mqqrghbgdlrfmf4",
			// 客户端令牌，开发者后台获取
			clientToken = "bs8IwOTzg5mbjcl08nOkkLgSbh8f1kEGM6EOHoMs",
			// 地区，CN 为国内，Overseas 为海外
			region = TapTapRegionType.CN,
			// 语言，默认为 Auto，默认情况下，国内为 zh_Hans，海外为 en
			preferredLanguage = TapTapLanguageType.zh_Hans,
			// 是否开启日志，Release 版本请设置为 false
			enableLog = true
		};
		// TapSDK 初始化
		TapTapSDK.Init(coreOptions);

		try
		{
			// 定义授权范围
			List<string> scopes = new List<string>
			{
				TapTapLogin.TAP_LOGIN_SCOPE_PUBLIC_PROFILE
			};
			// 发起 Tap 登录
			TapTapLogin.Instance.LoginWithScopes(scopes.ToArray()).ContinueWith(task =>
			{
				if (task.IsFaulted)
				{
					Debug.Log($"登录失败，出现异常：{task.Exception}");
				}
				else
				{
					Debug.Log($"登录成功，当前用户 ID：{task.Result.ToJson()}");

					lock (Lock)
					{
						loginCallback.Add(() =>
						{
							PostData postData = new PostData()
							{
								kid = task.Result.accessToken.kid,
								macKey = task.Result.accessToken.macKey,
							};
							string body = JsonUtility.ToJson(postData);

							Debug.Log($"postData {body}");
							AsyncWebRequest asyncWebRequest = new AsyncWebRequest();

							UserChannel userChannel = new UserChannel()
							{
								ver = "v1",
								exp = (long)(DateTime.UtcNow.AddSeconds(30) - new DateTime(1970, 1, 1)).TotalSeconds,
								userId = task.Result.openId,
							};

							string userChannelBody = JsonUtility.ToJson(userChannel);

							Dictionary<string, string> headers = new Dictionary<string, string>
							{
								{ "user-channel", userChannelBody }
							};

							Debug.Log($"userChannelBody {userChannelBody}");

							lock (Lock)
							{
								loginCallback.Add(() =>
								{
									asyncWebRequest.Post("http://10.0.12.176:8080/api/1.0/public/user_server/login", body, headers, (ok, response) =>
									{
										if (!ok)
										{
											Debug.Log($"登录失败，服务器响应异常：{response}");
										}
										else
										{
											string responseStr = Encoding.UTF8.GetString(response);
											Debug.Log($"登录成功，服务器响应：{responseStr}");
											UserCenter.LoginRsp rsp = UserCenter.LoginRsp.Parser.ParseJson(responseStr);
											AccountInfo.Instance.SetAccount(rsp.TapInfo);
											Config.Instance.serverIP = rsp.ServerAddr;
											Config.Instance.port = (ushort)rsp.ServerPort;
											startButton.SetActive(true);
										}
									});
								});
							}
						});
					}
				}
			});
		}
		catch (TaskCanceledException)
		{
			Debug.Log("用户取消登录");
		}
		catch (System.Exception exception)
		{
			Debug.Log($"登录失败，出现异常：{exception}");
		}
#else
		AccountInfo.Instance.SetAccount(new UserCenter.TapInfo() { Avatar = "https://img3.tapimg.com/default_avatars/aba00206f8642b0bbef01ef8f271e9da.jpg?imageMogr2/auto-orient/strip/thumbnail/!270x270r/gravity/Center/crop/270x270/format/jpg/interlace/1/quality/80", Gender = "", Name = "冷水泡面", Openid = "mzw0536knQSO+bhbdL6dtw==", Unionid = "SnwhJ5s2EURKCKt0LBsDLw==" });

		Config.Instance.serverIP = "10.0.12.176";
		Config.Instance.port = 10085;
		startButton.SetActive(true);
#endif
	}

	// Update is called once per frame
	void Update()
    {
		lock (Lock)
		{
			List<Action> callbacks = loginCallback;
			loginCallback = new List<Action>();
			foreach (Action callback in callbacks)
			{
				callback();
			}
		}
	}

	public void StartGame()
	{
        UnityEngine.SceneManagement.SceneManager.LoadScene("Demo 1");
	}

	readonly object Lock = new object();
	List<Action> loginCallback = new List<Action>();

	public GameObject startButton;
}
