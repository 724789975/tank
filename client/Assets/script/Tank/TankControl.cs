using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class TankControl : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
	}

	// Update is called once per frame
	void Update()
    {
        timer += Time.deltaTime;
        if (timer < UPDATE_INTERVAL)
        {
            return;
        }

        timer -= UPDATE_INTERVAL;
        TankInstance tankInstance = TankManager.Instance.GetTank(PlayerControl.Instance.PlayerId);

        if (tankInstance != null)
        {
            Vector2 dir = Terresquall.VirtualJoystick.GetAxis(0);
            Vector2 pos = new Vector2(dir.x * speed, dir.y * speed);
            // Debug.Log("dir: " + dir + " pos: " + pos);
            tankInstance.AddPos(pos);
            if (dir.magnitude > 0.1f)
            {
                tankInstance.SetDir(dir.normalized);
            }
        }
    }

    public void Shoot()
    {
        TankInstance tankInstance = TankManager.Instance.GetTank(PlayerControl.Instance.PlayerId);
		if (tankInstance == null)
        {
            return;
        }
        tankInstance.Shoot();
    }

    public float speed;

    private float timer = 0f;
    private const float UPDATE_INTERVAL = 0.03f;
}
