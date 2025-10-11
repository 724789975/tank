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
		// �������� ��ϸ������ [TapTapSDK]
		TapTapSdkOptions coreOptions = new TapTapSdkOptions()
		{  // �ͻ��� ID�������ߺ�̨��ȡ
			clientId = "oy5mqqrghbgdlrfmf4",
			// �ͻ������ƣ������ߺ�̨��ȡ
			clientToken = "bs8IwOTzg5mbjcl08nOkkLgSbh8f1kEGM6EOHoMs",
			// ������CN Ϊ���ڣ�Overseas Ϊ����
			region = TapTapRegionType.CN,
			// ���ԣ�Ĭ��Ϊ Auto��Ĭ������£�����Ϊ zh_Hans������Ϊ en
			preferredLanguage = TapTapLanguageType.zh_Hans,
			// �Ƿ�����־��Release �汾������Ϊ false
			enableLog = true
		};
		// TapSDK ��ʼ��
		TapTapSDK.Init(coreOptions);

		try
		{
			// ������Ȩ��Χ
			List<string> scopes = new List<string>
			{
				TapTapLogin.TAP_LOGIN_SCOPE_PUBLIC_PROFILE
			};
			// ���� Tap ��¼
			TapTapLogin.Instance.LoginWithScopes(scopes.ToArray()).ContinueWith(task =>
			{
				if (task.IsFaulted)
				{
					Debug.Log($"��¼ʧ�ܣ������쳣��{task.Exception}");
				}
				else
				{
					Debug.Log($"��¼�ɹ�����ǰ�û� ID��{task.Result.ToJson()}");

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
											Debug.Log($"��¼ʧ�ܣ���������Ӧ�쳣��{response}");
										}
										else
										{
											string responseStr = Encoding.UTF8.GetString(response);
											Debug.Log($"��¼�ɹ�����������Ӧ��{responseStr}");
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
			Debug.Log("�û�ȡ����¼");
		}
		catch (System.Exception exception)
		{
			Debug.Log($"��¼ʧ�ܣ������쳣��{exception}");
		}
#else
		AccountInfo.Instance.SetAccount(new UserCenter.TapInfo() { Avatar = "https://img3.tapimg.com/default_avatars/aba00206f8642b0bbef01ef8f271e9da.jpg?imageMogr2/auto-orient/strip/thumbnail/!270x270r/gravity/Center/crop/270x270/format/jpg/interlace/1/quality/80", Gender = "", Name = "��ˮ����", Openid = "mzw0536knQSO+bhbdL6dtw==", Unionid = "SnwhJ5s2EURKCKt0LBsDLw==" });

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
