using fxnetlib.dllimport;
using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
#if UNITY_SERVER && !AI_RUNNING
using PLAYERDATA = ServerPlayer;
#else
using PLAYERDATA = ClientPlayer;
#endif
public class PlayerManager : Singleton<PlayerManager>
{
    /// <summary>
    /// 添加玩家数据
    /// </summary>
    /// <param name="id">玩家ID</param>
    /// <param name="data">玩家数据</param>
    /// <returns>是否添加成功</returns>
    public bool AddPlayer(string id, PLAYERDATA data)
    {
        if (!players.ContainsKey(id))
        {
            players.Add(id, data);
        }
        else
        {
            Debug.LogWarning($"Player ID {id} already exists, cannot add again.");
        }
#if UNITY_SERVER && !AI_RUNNING
        if(data.session != IntPtr.Zero)
        {
            if (!sessions.ContainsKey(data.session))
            {
                sessions.Add(data.session, data.Id);
            }
        }
#endif
		return true;
    }

    /// <summary>
    /// 根据玩家ID获取玩家数据
    /// </summary>
    /// <param name="id">玩家ID</param>
    /// <returns>玩家数据，如果未找到则返回null</returns>
    public PLAYERDATA GetPlayer(string id)
    {
        if (players.TryGetValue(id, out PLAYERDATA data))
        {
            return data;
        }
        Debug.LogWarning($"Player data with ID {id} not found.");
        return null;
    }

#if UNITY_SERVER && !AI_RUNNING
    public PLAYERDATA GetPlayerBySession(IntPtr pConnector)
	{
        sessions.TryGetValue(pConnector, out string id);
        if (id != null)
        {
            return GetPlayer(id);
		}
        return null;
	}
#endif

	/// <summary>
	/// 移除玩家数据
	/// </summary>
	/// <param name="id">玩家ID</param>
	/// <returns>是否移除成功</returns>
	public bool RemovePlayer(string id)
    {
        Debug.Log($"Removing player data with ID {id}.");
        if (players.ContainsKey(id))
        {
#if UNITY_SERVER && !AI_RUNNING
            if (players[id].session!= IntPtr.Zero)
            {
				DLLImport.Close(players[id].session);
				players[id].session = IntPtr.Zero;
				sessions.Remove(players[id].session);
            }
#endif
            return players.Remove(id);
        }
        Debug.Log($"Player data with ID {id} not found, cannot remove.");
        return false;
    }

    public delegate void PlayerAction(PLAYERDATA data);
    public void ForEach(PlayerAction action)
    {
        foreach (PLAYERDATA data in players.Values)
        {
            action(data);
        }
    }

    public void AfterCloseCallback(IntPtr pConnector)
    {
#if UNITY_SERVER && !AI_RUNNING
        sessions.TryGetValue(pConnector, out string id);
        if (id != null)
        {
            players.TryGetValue(id, out PLAYERDATA data);
            if (data!= null)
            {
                data.session = IntPtr.Zero;
            }
        }
        sessions.Remove(pConnector);
#endif
    }

	Dictionary<string, PLAYERDATA> players = new Dictionary<string, PLAYERDATA>();
#if UNITY_SERVER && !AI_RUNNING
    Dictionary<IntPtr, string> sessions = new Dictionary<IntPtr, string>();
#endif
}
