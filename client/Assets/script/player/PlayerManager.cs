using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class PlayerManager : Singleton<PlayerManager>
{
    /// <summary>
    /// 添加玩家数据
    /// </summary>
    /// <param name="id">玩家ID</param>
    /// <param name="data">玩家数据</param>
    /// <returns>是否添加成功</returns>
    public bool AddPlayer(string id, PlayerData data)
    {
        if (!players.ContainsKey(id))
        {
            players.Add(id, data);
            return true;
        }
        else
        {
            Debug.LogWarning($"玩家ID {id} 已存在，无法重复添加。");
        }
        return false;
    }

    /// <summary>
    /// 根据玩家ID获取玩家数据
    /// </summary>
    /// <param name="id">玩家ID</param>
    /// <returns>玩家数据，如果未找到则返回null</returns>
    public PlayerData GetPlayer(string id)
    {
        if (players.TryGetValue(id, out PlayerData data))
        {
            return data;
        }
        Debug.LogWarning($"未找到玩家ID {id} 的数据。");
        return null;
    }

    /// <summary>
    /// 移除玩家数据
    /// </summary>
    /// <param name="id">玩家ID</param>
    /// <returns>是否移除成功</returns>
    public bool RemovePlayer(string id)
    {
        if (players.ContainsKey(id))
        {
            return players.Remove(id);
        }
        Debug.LogWarning($"未找到玩家ID {id} 的数据，无法移除。");
        return false;
    }
	Dictionary<string, PlayerData> players = new Dictionary<string, PlayerData>();
}
