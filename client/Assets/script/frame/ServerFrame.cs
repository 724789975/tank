using Google.Protobuf.WellKnownTypes;
using System.Collections;
using System.Collections.Generic;
using Google.Protobuf;
using UnityEngine;

public class ServerFrame : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        instane = this;
    }

    // Update is called once per frame
    void Update()
    {
	}

	public static ServerFrame Instance
	{
		get
		{
			return instane;
		}
	}

    public float CurrentTime
    {
        get
        {
            return currentTime;
        }
	}

	static ServerFrame instane;
    float currentTime = 0;

}
