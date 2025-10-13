using System.Collections;
using System.Collections.Generic;
using TMPro;
using UnityEngine;

public class Notice : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
    }

    // Update is called once per frame
    void Update()
    {
        destroyTime -= Time.deltaTime;
        if (destroyTime <= 0)
        {
            Destroy(gameObject);
        }
        transform.position += Vector3.up * upSpeed * Time.deltaTime;
    }

    public float destroyTime = 5.0f;
    public float upSpeed = 1.0f;
	public TextMeshPro text;

 }

