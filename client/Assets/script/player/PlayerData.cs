using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;


[Serializable]
public class PlayerData
{
    public string Id = "";
    public string Name = "";
#if UNITY_SERVER
    public IntPtr session = IntPtr.Zero;
#endif
}
