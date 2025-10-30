using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class Enemy : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        
    }
	// Update is called once per frame
	void Update()
    {
		if ((shoot -= Time.deltaTime) < 0)
        {
            shoot = 1f;

			transform.right = (target.transform.position - transform.position).normalized;
			TankInstance tankInstance = GetComponent<TankInstance>();
            tankInstance.Shoot(tankInstance.ID, tankInstance.bulletPos.transform.position, tankInstance.bulletPos.transform.rotation, Config.Instance.speed);
        }

        transform.position += transform.right * moveSpeed;
    }

    float shoot = 1f;

    public GameObject target;

    float moveSpeed = 0.01f;
}
