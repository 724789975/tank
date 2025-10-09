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
					AccountInfo.Instance.SetAccount(task.Result);
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
	}

	// Update is called once per frame
	void Update()
    {
        
    }
}
