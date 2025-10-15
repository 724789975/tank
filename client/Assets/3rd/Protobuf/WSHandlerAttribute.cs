/****************************************************
* Unity版本：2022.3.17f1
* 作者：ZD
* 日期：2025/03/21 16:50:31
* 功能：
*****************************************************/

using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

[AttributeUsage(AttributeTargets.Method)]
public class WSHandlerAttribute : Attribute
{
    private string _name;
    public string Name
    {
        get { return _name; }
    }
    public WSHandlerAttribute(string name)
    {
        _name = name;
    }
}
