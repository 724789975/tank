using System.Collections;
using System.Collections.Generic;
using UnityEngine;

using TapTap;

public class AccountInfo : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		DontDestroyOnLoad(gameObject);
	}

	// Update is called once per frame
	void Update()
    {
        
    }

    public TapSDK.Login.TapTapAccount Account
    {
        get
        {
            return account;
        }
    }

    public void SetAccount(TapSDK.Login.TapTapAccount account)
    {
        this.account = account;
    }

	public static AccountInfo Instance
    {
        get
        {
            return instance;
        }
    }
    static AccountInfo instance;
	TapSDK.Login.TapTapAccount account;
}
