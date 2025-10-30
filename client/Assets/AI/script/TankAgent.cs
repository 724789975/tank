using UnityEngine;
using Unity.MLAgents;
using Unity.MLAgents.Actuators;
using Unity.MLAgents.Sensors;
using Random = UnityEngine.Random;
using System.Collections.Generic;
using UnityEngine.SceneManagement;
using System;

public class TankAgent : Agent
{
    [Header("Specific to Enemy")]
    public GameObject enemy;
    [Header("Specific to Tank")]
    public GameObject tank;
    public bool useVecObs;
    EnvironmentParameters m_ResetParams;

	public override void Initialize()
    {
        m_ResetParams = Academy.Instance.EnvironmentParameters;
        Debug.Log("TankAgent Initialized");
    }

    public override void CollectObservations(VectorSensor sensor)
    {
        if(Time.time - lastShootTime > 1f)
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
        float delta = Time.time - lastUpdateTime;
        lastUpdateTime = Time.time;

        Vector3 dir = new Vector3(actionX, actionY, 0);
        TankInstance instance = tank.GetComponent<TankInstance>();

        if (shoot)
        {
            if(canShoot)
            {
                SetReward(.5f);
            }
            else
            {
                SetReward(-.1f);
            }
        }
        if (shoot && canShoot)
        {
            dir = (enemy.transform.position - tank.transform.position).normalized;

			instance.SetDir(dir.normalized);
            instance.Shoot(instance.ID, instance.bulletPos.transform.position, instance.bulletPos.transform.rotation, Config.Instance.speed);

            lastShootTime = Time.time;
            canShoot = false;
        }

        if(dir.magnitude > 1f)
        {
            dir.Normalize();
        }

		instance.SetDir(dir.normalized);
        instance.AddPos(dir * instance.speed * delta);
        foreach (Bullet bullet in BulletManager.Instance.Bullets)
        {
            bullet.Move(delta);
            if (Vector3.Distance(bullet.transform.position, tank.transform.position) < 1f)
            {
				if (bullet.ownerId == tank.GetComponent<TankInstance>().ID) { continue; }
				SetReward(-1f);
                DiffAction(() =>
                {
					EndEpisode();
					Debug.Log("TankAgent EndEpisode");
					SetResetParameters();
                    return;
				});
               
            }

			if (Vector3.Distance(bullet.transform.position, enemy.transform.position) < 1f)
			{
                if (bullet.ownerId == enemy.GetComponent<TankInstance>().ID) { continue; }
				SetReward(10f);
                DiffAction(() =>
                {
					EndEpisode();
					Debug.Log("TankAgent EndEpisode 22222");
					SetResetParameters();
                    return;
                });
            }

			if (bullet.CheckScreenBounds())
            {
                SetReward(0.1f);
            }
        }
    }

    public override void OnEpisodeBegin()
    {
        Debug.Log("TankAgent OnEpisodeBegin");
        SetResetParameters();
    }

    public override void Heuristic(in ActionBuffers actionsOut)
    {
        var continuousActionsOut = actionsOut.ContinuousActions;
        continuousActionsOut[0] = -Input.GetAxis("Horizontal");
        continuousActionsOut[1] = Input.GetAxis("Vertical");
        continuousActionsOut[2] = Input.GetButton("Fire1")? 1f : 0f;
    }

    public void SetResetParameters()
    {
		tank.transform.position = new Vector3(UnityEngine.Random.Range(Config.Instance.GetLeft() + .5f, Config.Instance.GetRight() - .5f), UnityEngine.Random.Range(Config.Instance.GetTop() - .5f, Config.Instance.GetBottom() + .5f), 0);

		tank.transform.right = new Vector3(UnityEngine.Random.Range(-1f, 1f), UnityEngine.Random.Range(-1f, 1f), 0).normalized;
		tank.name = "ai";
		TankInstance instance = tank.GetComponent<TankInstance>();
		instance.Name = "ai";
		instance.ID = "ai";

		enemy.transform.position = new Vector3(UnityEngine.Random.Range(Config.Instance.GetLeft() + .5f, Config.Instance.GetRight() - .5f), UnityEngine.Random.Range(Config.Instance.GetTop() - .5f, Config.Instance.GetBottom() + .5f), 0);
		enemy.transform.right = (tank.transform.position - enemy.transform.position).normalized;
		enemy.name = "enemy";
		TankInstance enemyInstance = enemy.GetComponent<TankInstance>();
		enemyInstance.Name = "enemy";
		enemyInstance.ID = "enemy";
        Enemy enemyScript = enemy.GetComponent<Enemy>();
        if (enemyScript == null)
        {
            enemyScript = enemy.AddComponent<Enemy>();
        }
		enemyScript.target = tank;

		BulletManager.Instance.ResetBullets();
	}

    Vector2 toAIVecotr(Vector3 vec)
    {
        return new Vector2((vec.x - Config.Instance.GetLeft()) / (Config.Instance.GetRight() - Config.Instance.GetLeft()), (vec.y - Config.Instance.GetBottom()) / (Config.Instance.GetTop() - Config.Instance.GetBottom()));
    }

    Vector2 toAIVecotr2(Vector3 vec)
    {
        float magnitude = Mathf.Min(Mathf.Abs(Config.Instance.GetRight()), Mathf.Abs(Config.Instance.GetTop()));
        if (magnitude < vec.magnitude)
        {
            vec = vec.normalized * magnitude;
        }

        return new Vector2(vec.x / magnitude, vec.y / magnitude);
    }

    void DiffAction(Action a)
    {
#if AI_TRAIN
        a();
#endif
    }

	public void Update()
	{
		WaitTimeInference();
	}

	void WaitTimeInference()
	{
#if AI_TRAIN
        if (Academy.Instance.IsCommunicatorOn)
        {
        	RequestDecision();
        }
#else
        RequestDecision();
#endif
    }

	public int obsBulletNum;
    bool canShoot = false;
    float lastShootTime = 0f;
    float lastUpdateTime = 0f;
}
