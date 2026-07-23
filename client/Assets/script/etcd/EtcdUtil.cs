using Newtonsoft.Json;
using System;
using System.Collections;
using System.Collections.Generic;
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

	public void EtcdOperator(Action<string, bool> op)
	{
		// etcd 用户名为空时(如 test2 未开启认证的本地 etcd), 跳过认证,
		// 直接以空 token 执行回调; 否则无认证 etcd 会对认证请求返回 400
		if (string.IsNullOrEmpty(etcdUserName))
		{
			try
			{
				op("", true);
			}
			catch (System.Exception e)
			{
				Debug.LogError($"[Etcd][EtcdOperator] etcd operator callback failed, exception={e.Message}\n{e.StackTrace}");
			}
			return;
		}
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

		Debug.Log($"[Etcd][EtcdOperator] authenticating, addr={etcdAddr} user={etcdUserName}");

		asyncWebRequest.Post(AuthUrl, body, new Dictionary<string, string> { }, (ok, response) =>
		{
			isLogin = false;
			if (!ok)
			{
				Debug.LogError($"[Etcd][EtcdOperator] etcd auth failed, response={(response != null ? System.Text.Encoding.UTF8.GetString(response) : "null")}");
			}
			else
			{
				string responseStr = System.Text.Encoding.UTF8.GetString(response);
				Debug.Log($"[Etcd][EtcdOperator] etcd auth success, response={responseStr}");
				AuthResp authResp = JsonUtility.FromJson<AuthResp>(responseStr);
				
				foreach (Action<string, bool> op in ops)
				{
					try
					{
						op(authResp.token, true);
					}
					catch(System.Exception e)
					{
						Debug.LogError($"[Etcd][EtcdOperator] etcd operator callback failed, exception={e.Message}\n{e.StackTrace}");
					}
				}
			}
			ops.Clear();
		});
	}

	public void Keys()
	{
		EtcdOperator((token, succeed) =>
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
			Debug.Log($"[Etcd] request, url={url} body={pbody}");

			if (!succeed)
			{
				Debug.LogError($"[Etcd] get token failed, url={url} body={pbody}");
				return;
			}
			AsyncWebRequest asyncWebRequest = new AsyncWebRequest();
			asyncWebRequest.Post(url, pbody, header, (ok, response) =>
			{
				if (!ok)
				{
					Debug.LogError($"[Etcd] request failed, url={url} response={(response != null ? System.Text.Encoding.UTF8.GetString(response) : "null")}");
				}
				else
				{
					string responseStr = System.Text.Encoding.UTF8.GetString(response);
					Debug.Log($"[Etcd] request success, url={url} response={responseStr}");
				}
			});
		});
	}

	public void Get(string prefix, Action<Dictionary<string, string>, bool> callback)
	{
		EtcdOperator((token, succeed) =>
		{
			string url = string.Format("http://{0}/v3/kv/range", etcdAddr);
			Dictionary<string, string> header = new Dictionary<string, string>
			{
				{ "Authorization", token },
			};

			Dictionary<string, string> body = new Dictionary<string, string>();
			if (string.IsNullOrEmpty(prefix))
			{
				body["key"] = System.Convert.ToBase64String(Encoding.UTF8.GetBytes("\0"));
				body["range_end"] = System.Convert.ToBase64String(Encoding.UTF8.GetBytes("\0"));
			}else
			{
				body["key"] = System.Convert.ToBase64String(Encoding.UTF8.GetBytes(prefix));
				body["range_end"] = System.Convert.ToBase64String(Encoding.UTF8.GetBytes(prefix + "\xff"));
			};

			string pbody = JsonConvert.SerializeObject(body);
			Debug.Log($"[Etcd] request, url={url} body={pbody}");

			if (!succeed)
			{
				Debug.LogError($"[Etcd] get token failed, url={url} body={pbody}");
				callback(null, false);
				return;
			}

			AsyncWebRequest asyncWebRequest = new AsyncWebRequest();
			asyncWebRequest.Post(url, pbody, header, (ok, response) =>
			{
				if (!ok)
				{
					Debug.LogError($"[Etcd][Get] request failed, url={url} body={pbody} response={(response != null ? System.Text.Encoding.UTF8.GetString(response) : "null")}");
					callback(null, false);
				}
				else
				{
					string responseStr = System.Text.Encoding.UTF8.GetString(response);
					Debug.Log($"[Etcd] request success, url={url} response={responseStr}");
					Dictionary<string, object> ret = JsonConvert.DeserializeObject<Dictionary<string, object>>(responseStr);
					Dictionary<string, string> keyValues = new Dictionary<string, string>();
					if (ret.TryGetValue("kvs", out object kvs))
					{
						//Debug.Log(kvs.GetType());
						ArrayList arrayList = JsonConvert.DeserializeObject<ArrayList>(kvs.ToString());
						foreach (var item in (Newtonsoft.Json.Linq.JArray)kvs)
						{
							string key = System.Text.Encoding.UTF8.GetString(System.Convert.FromBase64String(item["key"].ToString()));
							string value = System.Text.Encoding.UTF8.GetString(System.Convert.FromBase64String(item["value"].ToString()));
							keyValues.Add(key, value);
						}
					}
					
					callback(keyValues, true);
				}
			});
		});
	}

	public void Put(string key, string value, int ttl)
	{
		Action<string, string> a = (token, lease) =>
		{
			string url = string.Format("http://{0}/v3/kv/put", etcdAddr);
			Dictionary<string, string> header = new Dictionary<string, string>
			{
				{ "Authorization", token },
			};

			Dictionary<string, object> body = new Dictionary<string, object>
			{
				{ "key", System.Convert.ToBase64String(Encoding.UTF8.GetBytes(key)) },
				{"value", System.Convert.ToBase64String(Encoding.UTF8.GetBytes(value))  },
				{"lease", lease }
			};

			string pbody = JsonConvert.SerializeObject(body);
			Debug.Log($"[Etcd] request, url={url} body={pbody}");

			AsyncWebRequest asyncWebRequest = new AsyncWebRequest();
			asyncWebRequest.Post(url, pbody, header, (ok, response) =>
			{
				if (!ok)
				{
					Debug.LogError($"[Etcd] request failed, url={url} response={(response != null ? System.Text.Encoding.UTF8.GetString(response) : "null")}");
				}
				else
				{
					string responseStr = System.Text.Encoding.UTF8.GetString(response);
					Debug.Log($"[Etcd] request success, url={url} response={responseStr}");
				}
			});
		};
		EtcdOperator((token, succeed) =>
		{
			string url = string.Format("http://{0}/v3/lease/grant", etcdAddr);

			Dictionary<string, string> header = new Dictionary<string, string>
			{
				{ "Authorization", token },
			};

			Dictionary<string, int> body = new Dictionary<string, int>
			{
				{ "TTL", ttl },
			};

			string pbody = JsonConvert.SerializeObject(body);
			Debug.Log($"[Etcd] request, url={url} body={pbody}");

			if (!succeed)
			{
				Debug.LogError($"[Etcd] get token failed, url={url} body={pbody}");
				return;
			}

			AsyncWebRequest asyncWebRequest = new AsyncWebRequest();
			asyncWebRequest.Post(url, pbody, header, (ok, response) =>
			{
				if (!ok)
				{
					Debug.LogError($"[Etcd] request failed, url={url} response={(response != null ? System.Text.Encoding.UTF8.GetString(response) : "null")}");
				}
				else
				{
					string responseStr = System.Text.Encoding.UTF8.GetString(response);
					Debug.Log($"[Etcd] request success, url={url} response={responseStr}");

					Dictionary<string, object> ret = JsonConvert.DeserializeObject<Dictionary<string, object>>(responseStr);
					a(token, ret["ID"].ToString());
				}
			});
		});
	}

	public void Put(string key, string value)
	{
		EtcdOperator((token, succeed) =>
		{
			string url = string.Format("http://{0}/v3/kv/put", etcdAddr);
			Dictionary<string, string> header = new Dictionary<string, string>
			{
				{ "Authorization", token },
			};

			Dictionary<string, object> body = new Dictionary<string, object>
			{
				{ "key", System.Convert.ToBase64String(Encoding.UTF8.GetBytes(key)) },
				{"value", System.Convert.ToBase64String(Encoding.UTF8.GetBytes(value))  },
			};

			string pbody = JsonConvert.SerializeObject(body);
			Debug.Log($"[Etcd] request, url={url} body={pbody}");

			if (!succeed)
			{
				Debug.LogError($"[Etcd] get token failed, url={url} body={pbody}");
				return;
			}

			AsyncWebRequest asyncWebRequest = new AsyncWebRequest();
			asyncWebRequest.Post(url, pbody, header, (ok, response) =>
			{
				if (!ok)
				{
					Debug.LogError($"[Etcd] request failed, url={url} response={(response != null ? System.Text.Encoding.UTF8.GetString(response) : "null")}");
				}
				else
				{
					string responseStr = System.Text.Encoding.UTF8.GetString(response);
					Debug.Log($"[Etcd] request success, url={url} response={responseStr}");
				}
			});
		});
	}

	public string etcdAddr;
	public string etcdUserName;
	public string etcdPassword;
	bool isLogin = false;
	List<Action<string, bool>> ops = new List<Action<string, bool>>();

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

