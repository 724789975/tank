using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

#if UNITY_SERVER && !AI_RUNNING
using PLAYERDATA = ServerPlayer;
#endif
[Serializable]
public class ServerPlayer : PlayerData
{
	public object session = IntPtr.Zero;
	public float SyncTime = 0;
	public Vector3 lastPos = Vector3.zero;
	public float speedCheckDelate = 0;
}
