using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class Config : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        // 获取屏幕左下角坐标
        Vector3 screenBottomLeft = Camera.main.ScreenToWorldPoint(new Vector3(0, 0, Camera.main.nearClipPlane));
        // 获取屏幕右上角坐标
        Vector3 screenTopRight = Camera.main.ScreenToWorldPoint(new Vector3(Screen.width, Screen.height, Camera.main.nearClipPlane));

        // // 打印屏幕左下角和右上角坐标
        // Debug.Log("Screen Bottom Left: " + screenBottomLeft);
        // Debug.Log("Screen Top Right: " + screenTopRight);

        left = screenBottomLeft.x + sceneLimit;
        right = screenTopRight.x - sceneLimit;
        top = screenTopRight.y - sceneLimit;
        bottom = screenBottomLeft.y + sceneLimit;

#if UNITY_SERVER && !AI_RUNNING
        Application.targetFrameRate = 40;
#else
        Application.targetFrameRate = 60;
#endif

        Debug.Log($"left: {left}, right: {right}, top: {top}, bottom: {bottom}");
    }

    // Update is called once per frame
    void Update()
    {
    }

    public float GetLeft()
    {
        return left;
    }

    public float GetRight()
    {
        return right;
    }

    public float GetTop()
    {
        return top;
    }

    public float GetBottom()
    {
        return bottom;
    }

    static Config instance;
	// 公共访问接口
	public static Config Instance
	{
		get
		{
			if (instance == null)
			{
				lock (Lock)
				{
					if (instance == null)
					{
						instance = FindObjectOfType<Config>();
						if (instance == null)
						{
							// 创建新的实例
							GameObject singletonObject = new GameObject();
							instance = singletonObject.AddComponent<Config>();
							singletonObject.name = typeof(Config).ToString();

							// 确保单例不会被销毁
							DontDestroyOnLoad(singletonObject);
						}
					}
				}
			}

			return instance;
		}
	}

	private float left;
    private float right;
    private float top;
    private float bottom;
    private float sceneLimit = 0.2f;

    public string serverIP = "0.0.0.0";
    public ushort port = 10085;
    public float speed = 3f;
    public int maxHp = 100;
    public float rebornProtectionTime = 5f;
	static readonly object Lock = new object();
}
