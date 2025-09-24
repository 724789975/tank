using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class ModelSelect : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
#if UNITY_SERVER
        foreach (GameObject model in serverModels)
        {
            model.SetActive(true);
        }
#else
        foreach (GameObject model in clientModels)
        {
            model.SetActive(true);
        }
#endif
    }

    // Update is called once per frame
    void Update()
    {
        
    }

	public GameObject[] serverModels;
	public GameObject[] clientModels;

}
