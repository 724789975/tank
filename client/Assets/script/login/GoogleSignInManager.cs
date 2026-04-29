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
using UnityEngine.UI;

/// <summary>
/// Google 登录管理类
/// 功能：实现 Google OAuth 2.0 登录流程，获取用户信息
/// 支持平台：Windows、Android、iOS 等 Unity 支持的平台
/// </summary>
public class GoogleSignInManager : MonoBehaviour
{
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

    [Header("Google OAuth 配置")]
    [SerializeField] private string clientId = "1";
    [SerializeField] private string cS= "G";
    [SerializeField] private string redirectUri = "http://quchifan.wang:30080/api/1.0/get/user_server/google_oauth_callback";
    [SerializeField] private string scope = "openid https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email";
    [SerializeField] private string tokenEndpoint = "https://oauth2.googleapis.com/token";
    [SerializeField] private string userInfoEndpoint = "https://www.googleapis.com/oauth2/v1/userinfo";

    private string tokenFilePath;

    public event Action<bool> OnLoginStatusChanged;
    public event Action<GoogleUserInfo> OnUserInfoReceived;
    public event Action<string> OnErrorOccurred;

    public bool IsLoggedIn { get; private set; } = false;
    public GoogleUserInfo CurrentUser { get; private set; } = null;

    private string accessToken;
    private string refreshToken;
    private DateTime tokenExpiry;
    private string codeVerifier;
    static readonly object Lock = new object();

    private GameObject loginUIRoot;
    private InputField codeInputField;
    private Button confirmButton;
    private Text statusText;
    private bool isLoading = false;

    public void StartLogin()
    {
        StartCoroutine(LoginCoroutine());
    }

    private IEnumerator LoginCoroutine()
    {
        CreateLoginUI();

        while (!IsLoggedIn && loginUIRoot != null)
        {
            yield return null;
        }
    }

    private void CreateLoginUI()
    {
        loginUIRoot = new GameObject("GoogleLoginUI");
        loginUIRoot.transform.SetParent(transform);

        Canvas canvas = loginUIRoot.AddComponent<Canvas>();
        canvas.renderMode = RenderMode.ScreenSpaceOverlay;
        canvas.sortingOrder = 1000;
        loginUIRoot.AddComponent<CanvasScaler>();
        loginUIRoot.AddComponent<GraphicRaycaster>();

        GameObject panelObj = new GameObject("Panel");
        panelObj.transform.SetParent(loginUIRoot.transform);
        Image panelImage = panelObj.AddComponent<Image>();
        panelImage.color = new Color(0, 0, 0, 0.7f);

        RectTransform panelRect = panelObj.GetComponent<RectTransform>();
        panelRect.anchorMin = Vector2.zero;
        panelRect.anchorMax = Vector2.one;
        panelRect.sizeDelta = Vector2.zero;

        GameObject titleObj = new GameObject("Title");
        titleObj.transform.SetParent(panelObj.transform);
        Text titleText = titleObj.AddComponent<Text>();
        titleText.text = "Google 授权登录";
        titleText.fontSize = 24;
        titleText.alignment = TextAnchor.MiddleCenter;
        titleText.color = Color.white;
        RectTransform titleRect = titleObj.GetComponent<RectTransform>();
        titleRect.anchorMin = new Vector2(0.2f, 0.7f);
        titleRect.anchorMax = new Vector2(0.8f, 0.85f);
        titleRect.sizeDelta = Vector2.zero;

        GameObject codeLabelObj = new GameObject("CodeLabel");
        codeLabelObj.transform.SetParent(panelObj.transform);
        Text codeLabelText = codeLabelObj.AddComponent<Text>();
        codeLabelText.text = "请输入授权码:";
        codeLabelText.fontSize = 16;
        codeLabelText.color = Color.white;
        RectTransform codeLabelRect = codeLabelObj.GetComponent<RectTransform>();
        codeLabelRect.anchorMin = new Vector2(0.1f, 0.45f);
        codeLabelRect.anchorMax = new Vector2(0.9f, 0.55f);
        codeLabelRect.sizeDelta = Vector2.zero;

        GameObject inputObj = new GameObject("CodeInputField");
        inputObj.transform.SetParent(panelObj.transform);
        codeInputField = inputObj.AddComponent<InputField>();
        codeInputField.characterLimit = 500;
        codeInputField.contentType = InputField.ContentType.Standard;
        codeInputField.interactable = true;

        Image inputImage = inputObj.AddComponent<Image>();
        inputImage.color = Color.white;
        codeInputField.targetGraphic = inputImage;

        Text inputPlaceholder = CreateTextObject("Placeholder", inputObj.transform);
        inputPlaceholder.text = "请粘贴授权码...";
        inputPlaceholder.color = Color.gray;
        codeInputField.placeholder = inputPlaceholder;

        Text inputText = CreateTextObject("InputText", inputObj.transform);
        inputText.fontSize = 14;
        inputText.color = Color.black;
        codeInputField.textComponent = inputText;

        RectTransform inputRect = inputObj.GetComponent<RectTransform>();
        inputRect.anchorMin = new Vector2(0.1f, 0.3f);
        inputRect.anchorMax = new Vector2(0.9f, 0.42f);
        inputRect.sizeDelta = Vector2.zero;

        GameObject buttonObj = new GameObject("ConfirmButton");
        buttonObj.transform.SetParent(panelObj.transform);
        confirmButton = buttonObj.AddComponent<Button>();
        confirmButton.interactable = true;

        Image buttonImage = buttonObj.AddComponent<Image>();
        buttonImage.color = new Color(0.26f, 0.42f, 0.95f);
        confirmButton.targetGraphic = buttonImage;

        Text buttonText = CreateTextObject("ButtonText", buttonObj.transform);
        buttonText.text = "确认";
        buttonText.fontSize = 18;
        buttonText.color = Color.white;
        buttonText.alignment = TextAnchor.MiddleCenter;

        RectTransform buttonRect = buttonObj.GetComponent<RectTransform>();
        buttonRect.anchorMin = new Vector2(0.3f, 0.1f);
        buttonRect.anchorMax = new Vector2(0.7f, 0.22f);
        buttonRect.sizeDelta = Vector2.zero;

        confirmButton.onClick.AddListener(OnConfirmButtonClicked);

        GameObject statusObj = new GameObject("StatusText");
        statusObj.transform.SetParent(panelObj.transform);
        statusText = statusObj.AddComponent<Text>();
        statusText.text = "请在浏览器中完成授权后，粘贴授权码";
        statusText.fontSize = 14;
        statusText.color = Color.yellow;
        statusText.alignment = TextAnchor.MiddleCenter;
        RectTransform statusRect = statusObj.GetComponent<RectTransform>();
        statusRect.anchorMin = new Vector2(0.1f, 0.02f);
        statusRect.anchorMax = new Vector2(0.9f, 0.08f);
        statusRect.sizeDelta = Vector2.zero;

        UpdateURLText();
    }

    private Text CreateTextObject(string name, Transform parent)
    {
        GameObject obj = new GameObject(name);
        obj.transform.SetParent(parent);
        Text text = obj.AddComponent<Text>();
        text.font = Resources.GetBuiltinResource<Font>("Arial.ttf");
        return text;
    }

    private void UpdateURLText()
    {
        if (string.IsNullOrEmpty(codeVerifier))
        {
            codeVerifier = GenerateRandomString(32);
        }
        string codeChallenge = GenerateCodeChallenge(codeVerifier);

        string authUrl = "https://accounts.google.com/o/oauth2/auth" +
            "?response_type=code"
            + "&client_id=" + Uri.EscapeDataString(clientId)
            + "&redirect_uri=" + Uri.EscapeDataString(redirectUri)
            + "&scope=" + Uri.EscapeDataString(scope)
            + "&state=" + Uri.EscapeDataString(Guid.NewGuid().ToString())
            + "&code_challenge=" + Uri.EscapeDataString(codeChallenge)
            + "&code_challenge_method=S256";

        Debug.Log("授权 URL: " + authUrl);
        Application.OpenURL(authUrl);

        if (statusText != null)
        {
            statusText.text = "请在浏览器中完成授权后，粘贴授权码到上方输入框";
        }
    }

    private void OnConfirmButtonClicked()
    {
        if (isLoading)
        {
            return;
        }

        string code = codeInputField != null ? codeInputField.text.Trim() : "";

        if (string.IsNullOrEmpty(code))
        {
            ShowError("授权码不能为空");
            return;
        }

        if (code.Length < 10)
        {
            ShowError("授权码格式不正确");
            return;
        }

        StartCoroutine(ExchangeCodeCoroutine(code));
    }

    private IEnumerator ExchangeCodeCoroutine(string code)
    {
        isLoading = true;
        UpdateLoadingUI(true);

        string url = "http://quchifan.wang:30080/api/1.0/get/user_server/google_oauth_exchange"
            + "?code=" + Uri.EscapeDataString(code)
            + "&codeVerifier=" + Uri.EscapeDataString(codeVerifier);

        Debug.Log("请求 URL: " + url);

        UnityWebRequest request = UnityWebRequest.Get(url);
        request.timeout = 30;

        yield return request.SendWebRequest();

        isLoading = false;
        UpdateLoadingUI(false);

        if (request.result == UnityWebRequest.Result.Success)
        {
            string responseText = request.downloadHandler.text;
            Debug.Log("响应: " + responseText);

            try
            {
                UserCenter.GoogleOAuthExchangeRsp rsp = UserCenter.GoogleOAuthExchangeRsp.Parser.ParseJson(responseText);

                if (rsp.Code == Common.ErrorCode.Ok)
                {
                    accessToken = rsp.Data.Token;
                    tokenExpiry = DateTime.Now.AddHours(1);

                    CurrentUser = new GoogleUserInfo
                    {
                        id = rsp.Data.TapInfo.Openid,
                        name = rsp.Data.TapInfo.Name,
                        picture = rsp.Data.TapInfo.Avatar
                    };

                    IsLoggedIn = true;

                    OnLoginStatusChanged?.Invoke(true);
                    OnUserInfoReceived?.Invoke(CurrentUser);

                    Debug.Log("登录成功！用户: " + CurrentUser.name);
                    DestroyLoginUI();
                }
                else
                {
                    ShowError("登录失败: " + rsp.Msg);
                    OnErrorOccurred?.Invoke(rsp.Msg);
                }
            }
            catch (Exception e)
            {
                ShowError("解析响应失败: " + e.Message);
                OnErrorOccurred?.Invoke("解析响应失败: " + e.Message);
            }
        }
        else
        {
            string errorMsg = "请求失败: " + request.error;
            ShowError(errorMsg);
            OnErrorOccurred?.Invoke(errorMsg);
            Debug.LogError(errorMsg);
        }
    }

    private void UpdateLoadingUI(bool loading)
    {
        if (confirmButton != null)
        {
            confirmButton.interactable = !loading;
        }

        if (codeInputField != null)
        {
            codeInputField.interactable = !loading;
        }

        if (statusText != null)
        {
            statusText.text = loading ? "正在验证授权码..." : "请在浏览器中完成授权后，粘贴授权码到上方输入框";
            statusText.color = loading ? Color.cyan : Color.yellow;
        }
    }

    private void ShowError(string message)
    {
        if (statusText != null)
        {
            statusText.text = message;
            statusText.color = Color.red;
        }
        Debug.LogError(message);
    }

    private void DestroyLoginUI()
    {
        if (loginUIRoot != null)
        {
            Destroy(loginUIRoot);
            loginUIRoot = null;
            codeInputField = null;
            confirmButton = null;
            statusText = null;
        }
    }

    private string GenerateAuthorizationUrl()
    {
        string state = Guid.NewGuid().ToString();
        codeVerifier = GenerateRandomString(32);
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

    private string GenerateCodeChallenge(string codeVerifier)
    {
        using (var sha256 = System.Security.Cryptography.SHA256.Create())
        {
            var bytes = sha256.ComputeHash(System.Text.Encoding.UTF8.GetBytes(codeVerifier));
            return System.Convert.ToBase64String(bytes)
                .Replace('+', '-')
                .Replace('/', '_')
                .TrimEnd('=');
        }
    }

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