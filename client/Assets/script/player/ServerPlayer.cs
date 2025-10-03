using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

#if UNITY_SERVER
using PLAYERDATA = ServerPlayer;
#endif
[Serializable]
public class ServerPlayer : PlayerData
{
	public IntPtr session = IntPtr.Zero;
	public int SyncTime = 0;
	public Vector3 lastPos = Vector3.zero;
	public int speedCheckDelate = 0;
}
