using System.Collections;
using System.Collections.Generic;
using UnityEngine;

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
    }

    // Update is called once per frame
    void Update()
    {
        
    }

}
