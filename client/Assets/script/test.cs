using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using Terresquall;

public class test : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
        // Debug.Log(VirtualJoystick.GetAxis(0));
    }

    // 定义一个处理点击事件的方法
    public void OnClick()
    {
        Debug.Log("点击事件触发");
    }
}
