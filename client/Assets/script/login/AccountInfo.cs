using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class AccountInfo : MonoBehaviour
{
	// 公共访问接口
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
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							_instance = singletonObject.AddComponent<AccountInfo>();
							singletonObject.name = typeof(AccountInfo).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return _instance;
		}
	}

	// 防止通过构造函数创建实例
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
					Name = "冷水泡面",
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

	// 单例实例
	static AccountInfo _instance;
	// 线程锁，确保线程安全
	static readonly object Lock = new object();
	UserCenter.TapInfo account = new UserCenter.TapInfo();
}
