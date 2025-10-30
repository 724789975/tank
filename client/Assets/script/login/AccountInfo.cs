using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class AccountInfo : MonoBehaviour
{
	// �������ʽӿ�
	public static AccountInfo Instance
	{
		get
		{
			if (_instance == null)
			{
				lock (Lock)
				{
					if (_instance == null)
					{
						_instance = FindObjectOfType<AccountInfo>();
						if (_instance == null )
						{
							// �����µ�ʵ��
							GameObject singletonObject = new GameObject();
							_instance = singletonObject.AddComponent<AccountInfo>();
							singletonObject.name = typeof(AccountInfo).ToString();

							// ȷ���������ᱻ����
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return _instance;
		}
	}

	// ��ֹͨ�����캯������ʵ��
	private AccountInfo() { }
	
	// Start is called before the first frame update
	void Start()
    {
	}

	// Update is called once per frame
	void Update()
    {
        
    }

    public UserCenter.TapInfo Account
    {
        get
        {
#if !USE_TAP_LOGIN
			if (account == null)
			{
				account = new UserCenter.TapInfo()
				{
					Avatar = "https://img3.tapimg.com/default_avatars/aba00206f8642b0bbef01ef8f271e9da.jpg?imageMogr2/auto-orient/strip/thumbnail/!270x270r/gravity/Center/crop/270x270/format/jpg/interlace/1/quality/80",
					Gender = "",
					Name = "��ˮ����",
					Openid = "mzw0536knQSO+bhbdL6dtw==",
					Unionid = "SnwhJ5s2EURKCKt0LBsDLw=="
				};
			}
#endif
			return account;
        }
    }

    public void SetAccount(UserCenter.TapInfo account)
    {
        this.account = account;
    }

	// ����ʵ��
	static AccountInfo _instance;
	// �߳�����ȷ���̰߳�ȫ
	static readonly object Lock = new object();
	UserCenter.TapInfo account = new UserCenter.TapInfo();
}
