using System.Collections;
using System.Collections.Generic;
using Unity.MLAgents.Actuators;
using Unity.MLAgents.Sensors;
using UnityEngine;

public class AIControl : Unity.MLAgents.Agent
{
	// Start is called before the first frame update
	void Start()
    {
        
    }

    // Update is called once per frame
    void Update()
    {
		tank = TankManager.Instance.GetTank(AccountInfo.Instance.Account.Openid);
		enemy = null;
		foreach (TankInstance t in TankManager.Instance.Tanks)
		{
			if(t.ID != tank.ID)
			{
				enemy = t;
				break;
			}
		}

		if(tank && enemy)
		{
			if (Time.time - lastShootTime > 1f)
			{
				lastShootTime = Time.time;
				canShoot = true;
			}
			RequestDecision();
		}
    }

	public override void CollectObservations(VectorSensor sensor)
	{
		if (Time.time - lastShootTime > 1f)
		{
			canShoot = true;
		}
		if (useVecObs)
		{
			sensor.AddObservation(toAIVecotr(tank.transform.position));
			sensor.AddObservation(tank.transform.right);
			sensor.AddObservation(canShoot);
			sensor.AddObservation(toAIVecotr(enemy.transform.position));
			sensor.AddObservation(enemy.transform.right);

			BulletManager.Instance.GetNearbyBullets(tank.transform.position, obsBulletNum, out List<Bullet> nearbyBullets);

			for (int i = 0; i < obsBulletNum; i++)
			{
				if (i < nearbyBullets.Count)
				{
					sensor.AddObservation(toAIVecotr(nearbyBullets[i].transform.position));
					sensor.AddObservation(nearbyBullets[i].transform.right);
				}
				else
				{
					sensor.AddObservation(Vector2.zero);
					sensor.AddObservation(Vector3.zero);
				}
			}
		}
	}

	public override void OnActionReceived(ActionBuffers actionBuffers)
	{
		float actionX = (actionBuffers.ContinuousActions[0] + 1) / 2f * (Config.Instance.GetRight() - Config.Instance.GetLeft()) + Config.Instance.GetLeft();
		float actionY = (actionBuffers.ContinuousActions[1] + 1) / 2f * (Config.Instance.GetTop() - Config.Instance.GetBottom()) + Config.Instance.GetBottom();

		bool shoot = actionBuffers.DiscreteActions[0] != 0f;

		Vector3 dir = new Vector3(actionX, actionY, 0);

		if (shoot && canShoot)
		{
			dir = (enemy.transform.position - tank.transform.position).normalized;

			tank.SetDir(dir.normalized);
			tank.Shoot(tank.ID, tank.bulletPos.transform.position, tank.bulletPos.transform.rotation, Config.Instance.speed);

			lastShootTime = Time.time;
			canShoot = false;
		}

		if (dir.magnitude > 1f)
		{
			dir.Normalize();
		}

		tank.SetDir(dir.normalized);
		tank.AddPos(dir * tank.speed * Time.deltaTime);
	}

	Vector2 toAIVecotr(Vector3 vec)
	{
		return new Vector2((vec.x - Config.Instance.GetLeft()) / (Config.Instance.GetRight() - Config.Instance.GetLeft()), (vec.y - Config.Instance.GetBottom()) / (Config.Instance.GetTop() - Config.Instance.GetBottom()));
	}

	TankInstance enemy;
	TankInstance tank;
	public bool useVecObs;

	public int obsBulletNum;
	bool canShoot = false;
	float lastShootTime = 0f;
}

