using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class BulletManager : MonoBehaviour
{
	void Start()
	{
		instance = this;
	}

	// Update is called once per frame
	void Update()
	{
	}

	public Bullet AddBullet(Vector3 position, Quaternion rotation, string ownerId, float speed)
	{
		UInt32 id = nextId++;
		GameObject bulletInstance = Instantiate(bulletPrefab);
		bulletInstance.transform.position = position;
		bulletInstance.transform.rotation = rotation;
		Bullet bullet = bulletInstance.GetComponent<Bullet>();
		bullet.speed = speed;
		bullet.bulletId = id;
		bullet.ownerId = ownerId;
		bullets.Add(id, bullet);
		return bullet;
	}

	public bool RemoveBullet(UInt32 id)
	{
		if (bullets.ContainsKey(id))
		{
			Destroy(bullets[id].gameObject);
			bullets.Remove(id);
			return true;
		}
		return false;
	}

	/// <summary>
	/// 获取与指定位置最近的num个子弹
	/// </summary>
	/// <param name="position"></param>
	/// <param name="num"></param>
	/// <param name="nearestBullet"></param>
	public void GetNearbyBullets(Vector3 position, int num, out List<Bullet> nearestBullet)
	{
		nearestBullet = new List<Bullet>();
		foreach (var bullet in bullets.Values)
		{
			nearestBullet.Add(bullet);
		}
		nearestBullet.Sort((a, b) => Vector3.Distance(position, a.transform.position).CompareTo(Vector3.Distance(position, b.transform.position)));
		if (nearestBullet.Count > num)
		{
			nearestBullet.RemoveRange(num, nearestBullet.Count - num);
		}
	}

	public void ResetBullets()
	{
		foreach (var bullet in bullets.Values)
		{
			Destroy(bullet.gameObject);
		}
		bullets.Clear();
	}

	public List<Bullet> Bullets
	{
		get {
			return new List<Bullet>(bullets.Values);
		}
	}

	public static BulletManager Instance
	{
		get { 
			return instance;
		}
	}

	static BulletManager instance;

	UInt32 nextId = 1;
	Dictionary<UInt32, Bullet> bullets = new Dictionary<UInt32, Bullet>();
	public GameObject bulletPrefab;
}

