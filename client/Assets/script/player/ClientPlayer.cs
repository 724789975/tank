using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

#if UNITY_SERVER && !AI_RUNNING
#else
using PLAYERDATA = ClientPlayer;
#endif

[Serializable]
public class ClientPlayer : PlayerData
{
	public List<TankGame.PlayerStateNtf> syncs = new List<TankGame.PlayerStateNtf>();
}

