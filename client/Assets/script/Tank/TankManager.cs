using System.Collections.Generic;
using UnityEngine;

public class TankManager : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
#if UNITY_SERVER && !AI_RUNNING
        Debug.Log("[TankManager][Start] running in server model");
#else
        Debug.Log("[TankManager][Start] running in client model");
#endif
		instance = this;
	}

	// Update is called once per frame
	void Update()
    {
	}

	public TankInstance AddTank(string id, string name, out bool isAdd)
	{
		if (GetTank(id))
		{
			Debug.LogWarning($"[TankManager][AddTank] tank already in map, id={id}");
			isAdd = false;
			return GetTank(id);
		}
		GameObject g = Resources.Load<GameObject>("prafab/tank");
		GameObject tankInstance = Instantiate(g);
		tankInstance.GetComponent<TankInstance>().ID = id;
		tankInstance.GetComponent<TankInstance>().Name = name;
		tankInstance.transform.position = new Vector3(0, 0, 0);

		instanceMap.Add(id, tankInstance.GetComponent<TankInstance>());
		isAdd = true;
		Debug.Log($"[TankManager][AddTank] tank created, id={id} name={name} total={instanceMap.Count}");
		return instanceMap[id];
	}

	public void RemoveTank(string id)
	{
		if (instanceMap.ContainsKey(id))
		{
			Debug.Log($"[TankManager][RemoveTank] removing tank, id={id}");
			Destroy(instanceMap[id].gameObject);
			instanceMap.Remove(id);
		}
	}

	public TankInstance GetTank(string id)
	{
		instanceMap.TryGetValue(id, out TankInstance ti);
		return ti;
	}

	public List<TankInstance> Tanks
	{
		get
		{
			return new List<TankInstance>(instanceMap.Values);
		}
	}

	public static TankManager Instance
	{
		get { return instance; }
	}

	static TankManager instance;

	public Dictionary<string, TankInstance> instanceMap = new Dictionary<string, TankInstance>();

}
