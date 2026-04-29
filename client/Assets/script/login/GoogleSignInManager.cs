using UnityEngine;
using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Text;
using System.Threading.Tasks;
using UnityEngine.Networking;

/// <summary>
/// Google 登录管理类
/// 功能：实现 Google OAuth 2.0 登录流程，获取用户信息
/// 支持平台：Windows、Android、iOS 等 Unity 支持的平台
/// </summary>
public class GoogleSignInManager : MonoBehaviour
{
    // 单例实例
    private static GoogleSignInManager instance;
    public static GoogleSignInManager Instance
    {
        get
        {
            if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<GoogleSignInManager>();
						if (instance == null)
						{
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<GoogleSignInManager>();
							singletonObject.name = typeof(GoogleSignInManager).ToString();

							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}
            return instance;
        }
    }

    // Google OAuth 配置
    [Header("Google OAuth 配置")]
    [SerializeField] private string clientId = "1";
    [SerializeField] private string cS= "G";
    [SerializeField] private string redirectUri = "http://quchifan.wang:30080/api/1.0/get/user_server/google_oauth_callback";
    [SerializeField] private string scope = "openid https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email";
    [SerializeField] private string tokenEndpoint = "https://oauth2.googleapis.com/token";
    [SerializeField] private string userInfoEndpoint = "https://www.googleapis.com/oauth2/v1/userinfo";

    // 凭据存储路径
    private string tokenFilePath;

    // 登录状态事件
    public event Action<bool> OnLoginStatusChanged;
    public event Action<GoogleUserInfo> OnUserInfoReceived;
    public event Action<string> OnErrorOccurred;

    // 登录状态
    public bool IsLoggedIn { get; private set; } = false;
    public GoogleUserInfo CurrentUser { get; private set; } = null;

    // 访问令牌
    private string accessToken;
    private string refreshToken;
    private DateTime tokenExpiry;
    private string codeVerifier;
    static readonly object Lock = new object();

    /// <summary>
    /// 开始登录流程
    /// </summary>
    public void StartLogin()
    {
        StartCoroutine(LoginCoroutine());
    }

    /// <summary>
    /// 登录协程
    /// </summary>
    private IEnumerator LoginCoroutine()
    {
        // 检查是否有有效的访问令牌
        if (IsTokenValid())
        {
            Debug.Log("使用现有令牌登录");
            yield return GetUserInfo();
            yield break;
        }

        // 检查是否有刷新令牌
        if (!string.IsNullOrEmpty(refreshToken))
        {
            Debug.Log("使用刷新令牌获取新令牌");
            yield return RefreshAccessToken();
            if (IsTokenValid())
            {
                yield return GetUserInfo();
                yield break;
            }
        }

        try
        {
            // 生成授权 URL
            string authUrl = GenerateAuthorizationUrl();
            Debug.Log("授权 URL: " + authUrl);

            // 打开浏览器进行授权
            Application.OpenURL(authUrl);

            // 等待用户输入授权码
            // 注意：在实际应用中，这里应该实现一个 UI 界面让用户输入授权码
            // 这里为了演示，我们使用一个简单的输入框
            Debug.Log("请在浏览器中授权，然后输入授权码");

            // 提示：在实际项目中，你需要创建一个 UI 来接收用户输入的授权码
            // 这里我们假设用户会通过某种方式输入授权码
            // 例如：
            // string authorizationCode = "用户输入的授权码";
            // yield return ExchangeCodeForToken(authorizationCode);

        }
        catch (Exception e)
        {
            Debug.LogError("登录过程中发生错误: " + e.Message);
            OnErrorOccurred?.Invoke("登录过程中发生错误: " + e.Message);
        }
    }

    /// <summary>
    /// 生成授权 URL
    /// </summary>
    /// <returns>授权 URL</returns>
    private string GenerateAuthorizationUrl()
    {
        string state = Guid.NewGuid().ToString();
        // 生成 code_verifier
        codeVerifier = GenerateRandomString(32);
        // 生成 code_challenge (使用 SHA-256 哈希)
        string codeChallenge = GenerateCodeChallenge(codeVerifier);
        
        string url = "https://accounts.google.com/o/oauth2/auth" +
            "?response_type=code"
            + "&client_id=" + Uri.EscapeDataString(clientId)
            + "&redirect_uri=" + Uri.EscapeDataString(redirectUri)
            + "&scope=" + Uri.EscapeDataString(scope)
            + "&state=" + Uri.EscapeDataString(state)
            + "&code_challenge=" + Uri.EscapeDataString(codeChallenge)
            + "&code_challenge_method=S256";

        return url;
    }

    /// <summary>
    /// 交换授权码获取访问令牌
    /// </summary>
    /// <param name="code">授权码</param>
    public IEnumerator ExchangeCodeForToken(string code)
    {
        try
        {
            Debug.Log("交换授权码获取令牌...");

            // 准备请求数据
            var exchangeReq = new UserCenter.GoogleOAuthExchangeReq
            {
                code = code,
                codeVerifier = codeVerifier
            };

            string jsonBody = exchangeReq.ToString();

            AsyncWebRequest asyncWebRequest = new AsyncWebRequest();

            // 设置用户 channel
            var userChannel = new Login.UserChannel
            {
                ver = "v1",
                exp = (long)(DateTime.UtcNow.AddSeconds(30) - new DateTime(1970, 1, 1)).TotalSeconds,
                userId = Guid.NewGuid().ToString()
            };
            string userChannelBody = JsonUtility.ToJson(userChannel);

            var headers = new Dictionary<string, string>
            {
                { "user-channel", userChannelBody }
            };

            bool requestComplete = false;
            bool requestSuccess = false;
            byte[] responseBytes = null;

            // 发送请求到我们自己的服务器
            asyncWebRequest.Post("http://quchifan.wang:30080/api/1.0/public/user_server/google_oauth_exchange", jsonBody, headers, (ok, response) =>
            {
                requestComplete = true;
                requestSuccess = ok;
                responseBytes = response;
            });

            // 等待请求完成
            while (!requestComplete)
            {
                yield return null;
            }

            if (requestSuccess && responseBytes != null)
            {
                // 解析响应
                string responseStr = System.Text.Encoding.UTF8.GetString(responseBytes);
                Debug.Log("令牌响应: " + responseStr);

                // 解析 protobuf
                UserCenter.GoogleOAuthExchangeRsp rsp = UserCenter.GoogleOAuthExchangeRsp.Parser.ParseJson(responseStr);
                
                if (rsp.Code == Common.ErrorCode.Ok)
                {
                    // 保存 token
                    accessToken = rsp.data.token;
                    tokenExpiry = DateTime.Now.AddHours(1);

                    // 设置用户信息
                    CurrentUser = new GoogleUserInfo
                    {
                        id = rsp.data.tapInfo.openid,
                        name = rsp.data.tapInfo.name,
                        picture = rsp.data.tapInfo.avatar
                    };

                    // 保存令牌
                    SaveToken();

                    // 设置登录状态
                    IsLoggedIn = true;

                    // 触发事件
                    OnLoginStatusChanged?.Invoke(true);
                    OnUserInfoReceived?.Invoke(CurrentUser);

                    Debug.Log("登录成功！用户: " + CurrentUser.name);
                }
                else
                {
                    Debug.LogError("登录失败: " + rsp.Msg);
                    OnErrorOccurred?.Invoke("登录失败: " + rsp.Msg);
                }
            }
            else
            {
                Debug.LogError("获取令牌失败");
                OnErrorOccurred?.Invoke("获取令牌失败");
            }
        }
        catch (Exception e)
        {
            Debug.LogError("交换授权码失败: " + e.Message);
            OnErrorOccurred?.Invoke("交换授权码失败: " + e.Message);
            yield break;
        }
    }

    /// <summary>
    /// 刷新访问令牌
    /// </summary>
    private IEnumerator RefreshAccessToken()
    {
        WWWForm form = null;
        UnityWebRequest request = null;

        try
        {
            Debug.Log("刷新访问令牌...");

            // 准备请求数据
            form = new WWWForm();
            form.AddField("refresh_token", refreshToken);
            form.AddField("client_id", clientId);
            form.AddField("client_secret", cS);
            form.AddField("grant_type", "refresh_token");

            // 发送请求
            request = UnityWebRequest.Post(tokenEndpoint, form);
        }
        catch (Exception e)
        {
            Debug.LogError("刷新令牌失败: " + e.Message);
            OnErrorOccurred?.Invoke("刷新令牌失败: " + e.Message);
            yield break;
        }

        // 发送请求
        yield return request.SendWebRequest();

        if (request.result == UnityWebRequest.Result.Success)
        {
            // 解析响应
            string response = request.downloadHandler.text;
            Debug.Log("刷新令牌响应: " + response);

            // 解析 JSON
            TokenResponse tokenResponse = JsonUtility.FromJson<TokenResponse>(response);
            accessToken = tokenResponse.access_token;
            tokenExpiry = DateTime.Now.AddSeconds(tokenResponse.expires_in);

            // 保存令牌
            SaveToken();
        }
        else
        {
            Debug.LogError("刷新令牌失败: " + request.error);
            OnErrorOccurred?.Invoke("刷新令牌失败: " + request.error);
        }
    }

    /// <summary>
    /// 获取用户信息
    /// </summary>
    private IEnumerator GetUserInfo()
    {
        UnityWebRequest request = null;

        try
        {
            Debug.Log("获取用户信息...");

            // 发送请求
            request = UnityWebRequest.Get(userInfoEndpoint);
            request.SetRequestHeader("Authorization", "Bearer " + accessToken);
        }
        catch (Exception e)
        {
            Debug.LogError("获取用户信息失败: " + e.Message);
            OnErrorOccurred?.Invoke("获取用户信息失败: " + e.Message);
            yield break;
        }

        // 发送请求
        yield return request.SendWebRequest();

        if (request.result == UnityWebRequest.Result.Success)
        {
            // 解析响应
            string response = request.downloadHandler.text;
            Debug.Log("用户信息响应: " + response);

            // 解析 JSON
            CurrentUser = JsonUtility.FromJson<GoogleUserInfo>(response);
            IsLoggedIn = true;

            // 触发事件
            OnLoginStatusChanged?.Invoke(true);
            OnUserInfoReceived?.Invoke(CurrentUser);

            Debug.Log("登录成功！用户: " + CurrentUser.name);
        }
        else
        {
            Debug.LogError("获取用户信息失败: " + request.error);
            OnErrorOccurred?.Invoke("获取用户信息失败: " + request.error);
        }
    }

    /// <summary>
    /// 退出登录
    /// </summary>
    public void SignOut()
    {
        // 清除令牌
        accessToken = null;
        refreshToken = null;
        tokenExpiry = DateTime.MinValue;
        CurrentUser = null;
        IsLoggedIn = false;

        // 删除保存的令牌文件
        if (File.Exists(tokenFilePath))
        {
            try
            {
                File.Delete(tokenFilePath);
                Debug.Log("已删除令牌文件");
            }
            catch (Exception e)
            {
                Debug.LogError("删除令牌文件失败: " + e.Message);
            }
        }

        // 触发事件
        OnLoginStatusChanged?.Invoke(false);
        Debug.Log("已退出登录");
    }

    /// <summary>
    /// 检查令牌是否有效
    /// </summary>
    /// <returns>令牌是否有效</returns>
    private bool IsTokenValid()
    {
        return !string.IsNullOrEmpty(accessToken) && DateTime.Now < tokenExpiry;
    }

    /// <summary>
    /// 保存令牌到文件
    /// </summary>
    private void SaveToken()
    {
        try
        {
            TokenData tokenData = new TokenData
            {
                access_token = accessToken,
                refresh_token = refreshToken,
                expiry = tokenExpiry.ToString("o")
            };

            string json = JsonUtility.ToJson(tokenData);
            File.WriteAllText(tokenFilePath, json);
            Debug.Log("令牌已保存到: " + tokenFilePath);
        }
        catch (Exception e)
        {
            Debug.LogError("保存令牌失败: " + e.Message);
        }
    }

    /// <summary>
    /// 从文件加载令牌
    /// </summary>
    private void LoadToken()
    {
        try
        {
            if (File.Exists(tokenFilePath))
            {
                string json = File.ReadAllText(tokenFilePath);
                TokenData tokenData = JsonUtility.FromJson<TokenData>(json);

                accessToken = tokenData.access_token;
                refreshToken = tokenData.refresh_token;
                tokenExpiry = DateTime.Parse(tokenData.expiry);

                Debug.Log("已从文件加载令牌");

                // 检查令牌是否有效
                if (IsTokenValid())
                {
                    StartCoroutine(GetUserInfo());
                }
            }
        }
        catch (Exception e)
        {
            Debug.LogError("加载令牌失败: " + e.Message);
        }
    }

    /// <summary>
    /// 生成随机字符串
    /// </summary>
    /// <param name="length">字符串长度</param>
    /// <returns>随机字符串</returns>
    private string GenerateRandomString(int length)
    {
        const string chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
        StringBuilder sb = new StringBuilder(length);
        System.Random random = new System.Random();

        for (int i = 0; i < length; i++)
        {
            sb.Append(chars[random.Next(chars.Length)]);
        }

        return sb.ToString();
    }

    /// <summary>
    /// 生成 code_challenge
    /// </summary>
    /// <param name="codeVerifier">code_verifier</param>
    /// <returns>code_challenge</returns>
    private string GenerateCodeChallenge(string codeVerifier)
    {
        using (var sha256 = System.Security.Cryptography.SHA256.Create())
        {
            var bytes = sha256.ComputeHash(System.Text.Encoding.UTF8.GetBytes(codeVerifier));
            // 进行 base64url 编码
            return System.Convert.ToBase64String(bytes)
                .Replace('+', '-')
                .Replace('/', '_')
                .TrimEnd('=');
        }
    }

    // 令牌响应类
    [Serializable]
    private class TokenResponse
    {
        public string access_token;
        public string refresh_token;
        public int expires_in;
        public string token_type;
        public string id_token;
        public string scope;
    }

    // 令牌数据类
    [Serializable]
    private class TokenData
    {
        public string access_token;
        public string refresh_token;
        public string expiry;
    }
}

/// <summary>
/// Google 用户信息类
/// </summary>
[Serializable]
public class GoogleUserInfo
{
    public string id;
    public string email;
    public bool verified_email;
    public string name;
    public string given_name;
    public string family_name;
    public string picture;
    public string locale;
}
