using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class Config : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
		instance = this;
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
    public static Config Instance
    {
        get
        {
            return instance;
        }
	}

	private float left;
    private float right;
    private float top;
    private float bottom;
    private float sceneLimit = 0.2f;

    public string serverIP;
    public ushort port;
    public float speed;
    public int maxHp;
    public float rebornProtectionTime;
}
