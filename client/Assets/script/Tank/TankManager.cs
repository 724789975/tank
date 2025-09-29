using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

using fxnetlib.dllimport;

public class TankManager : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
#if UNITY_SERVER
        Debug.Log("server model");
#else
        Debug.Log("client model");
#endif
		instance = this;


	}

	// Update is called once per frame
	void Update()
    {
	}

	public void AddTank(string id)
	{
		GameObject tankInstance = Instantiate(tankPrefab);
		tankInstance.GetComponent<TankInstance>().ID = id;
		tankInstance.transform.position = new Vector3(0, 0, 0);

		instanceMap.Add(id, tankInstance.GetComponent<TankInstance>());
	}

	public static TankManager Instance
	{
		get { return instance; }
	}

	static TankManager instance;

	public Config cfg;

	public GameObject tankPrefab;
	public Dictionary<string, TankInstance> instanceMap = new Dictionary<string, TankInstance>();

}
