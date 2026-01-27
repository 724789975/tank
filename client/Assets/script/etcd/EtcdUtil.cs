using Google.Protobuf.WellKnownTypes;
using Newtonsoft.Json;
using System.Collections;
using System.Collections.Generic;
using System.Drawing.Drawing2D;
using System.Text;
using UnityEngine;

public class EtcdUtil : MonoBehaviour
{
	class AuthResp
	{
		public string token;
		public object header;
	}
    // Start is called before the first frame update
    void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
        
    }

	public delegate void etcd_operator(string token);
	public void EtcdOperator(etcd_operator op)
	{
		if (isLogin)
		{
			ops.Add(op);
			return;
		}
		isLogin = true;
		ops.Add(op);
		string AuthUrl = string.Format("http://{0}/v3/auth/authenticate", etcdAddr);
		AsyncWebRequest asyncWebRequest = new AsyncWebRequest();

		Dictionary<string, string> AuthData = new Dictionary<string, string>
		{
			{ "name", etcdUserName },
			{ "password", etcdPassword },
		};

		string body = JsonConvert.SerializeObject(AuthData);

		Debug.Log($"AuthData {body}");

		asyncWebRequest.Post(AuthUrl, body, new Dictionary<string, string> { }, (ok, response) =>
		{
			isLogin = false;
			if (!ok)
			{
				Debug.Log($"登录etcd失败，服务器响应异常：{response}");
			}
			else
			{
				string responseStr = System.Text.Encoding.UTF8.GetString(response);
				Debug.Log($"登录etcd成功，服务器响应：{responseStr}");
				AuthResp authResp = JsonUtility.FromJson<AuthResp>(responseStr);
				
				foreach (etcd_operator op in ops)
				{
					try
					{
						op(authResp.token);
					}
					catch(System.Exception e)
					{
						Debug.LogException(e);
					}
				}
			}
			ops.Clear();
		});
	}

	public void Keys()
	{
		EtcdOperator((token) =>
		{
			string url = string.Format("http://{0}/v3/kv/range", etcdAddr);
			Dictionary<string, string> header = new Dictionary<string, string>
			{
				{ "Authorization", token },
			};

			Dictionary<string, string> body = new Dictionary<string, string>
			{
				{ "key", System.Convert.ToBase64String(Encoding.UTF8.GetBytes("\0")) },
				{ "range_end", System.Convert.ToBase64String(Encoding.UTF8.GetBytes("\0")) },
			};

			string pbody = JsonConvert.SerializeObject(body);
			Debug.Log($"请求etcd，{url}，请求参数：{pbody}");

			AsyncWebRequest asyncWebRequest = new AsyncWebRequest();
			asyncWebRequest.Post(url, pbody, header, (ok, response) =>
			{
				if (!ok)
				{
					Debug.Log($"请求etcd失败，{url}， 服务器响应异常：{response}");
				}
				else
				{
					string responseStr = System.Text.Encoding.UTF8.GetString(response);
					Debug.Log($"请求etcd成功，{url}， 服务器响应：{responseStr}");
				}
			});
		});
	}

	public string etcdAddr;
	public string etcdUserName;
	public string etcdPassword;
	bool isLogin = false;
	List<etcd_operator> ops = new List<etcd_operator>();

	static EtcdUtil instance;
	// 公共访问接口
	public static EtcdUtil Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<EtcdUtil>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<EtcdUtil>();
							singletonObject.name = typeof(EtcdUtil).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	static readonly object Lock = new object();
}
