using System.Collections;
using System.Collections.Generic;
using UnityEngine;

using TapSDK.Core;
using System.Threading.Tasks;
using TapSDK.Login;

public class Login : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
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
					AccountInfo.Instance.SetAccount(task.Result);
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
	}

	// Update is called once per frame
	void Update()
    {
        
    }
}
