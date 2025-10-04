using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class Boom : MonoBehaviour
{
    float leftTime = 1.0f;
    // Start is called before the first frame update
    void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
        leftTime -= Time.deltaTime;
        if (leftTime <= 0)
        {
            Destroy(gameObject);
        }
    }
}
