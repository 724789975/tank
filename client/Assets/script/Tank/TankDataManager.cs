using System.Collections;
using System.Collections.Generic;
using UnityEngine;

[System.Serializable]
public class TankDataManager : Singleton<TankDataManager>
{
    public Dictionary<string, TankInstance> instanceMap = new Dictionary<string, TankInstance>();
}
