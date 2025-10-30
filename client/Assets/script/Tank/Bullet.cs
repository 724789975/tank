using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using fxnetlib.dllimport;
using Google.Protobuf;

public class Bullet : MonoBehaviour
{

	// Start is called before the first frame update
	void Start()
	{
	}

	// Update is called once per frame
	void Update()
	{
#if !AI_TRAIN
        // 移动子弹
        Move(Time.deltaTime);
		// 检查是否超出屏幕边界
		CheckScreenBounds();
#endif
	}

	public void Move(float dt)
    {
        transform.Translate(Vector3.right * speed * dt);
    }

    
	// 检查屏幕边界
	public bool CheckScreenBounds()
    {
        Vector3 position = transform.position;
        
        // 获取屏幕边界
        float left = Config.Instance.GetLeft();
        float right = Config.Instance.GetRight();
        float top = Config.Instance.GetTop();
        float bottom = Config.Instance.GetBottom();
        
        // 如果子弹超出屏幕边界，销毁子弹
        if (position.x < left || position.x > right || position.y < bottom || position.y > top)
        {
            BulletManager.Instance.RemoveBullet(bulletId);
            return true;
        }
        return false;
    }

    // 碰撞检测
	void OnCollisionEnter(Collision collision)
	{
#if UNITY_SERVER
        // 检测是否碰撞到坦克
        if (collision.gameObject.CompareTag("Tank"))
        {
            TankInstance tankInstance = collision.gameObject.GetComponent<TankInstance>();
            if (tankInstance == null)
            {
                Debug.LogWarning("find tankinstance error");
            }
            else
            {
                if (tankInstance.rebornTime > 0)
                {
                    // 坦克处于重生保护时间，不受伤害
                    Debug.Log($"Tank {tankInstance.ID} is in reborn protection.");
                    return;
				}
				tankInstance.HP -= 10;
                TankGame.TankHpSyncNtf ntf = new TankGame.TankHpSyncNtf() { Id = tankInstance.ID, Hp = tankInstance.HP};
                byte[] hpBytes = Any.Pack(ntf).ToByteArray();

				PlayerManager.Instance.ForEach((p) =>
                {
                    if (p.session != IntPtr.Zero)
                    {
                        DLLImport.Send(p.session, hpBytes, (uint)hpBytes.Length);
                    }
                });

                if (tankInstance.HP <= 0)
                {
                    // 坦克死亡处理
                    Debug.Log($"Tank {tankInstance.ID} is destroyed!");
                    TankGame.PlayerDieNtf dieNtf = new TankGame.PlayerDieNtf() { KilledId = tankInstance.ID, KillerId = ownerId};
                    dieNtf.RebornProtectTime = ServerFrame.Instance.CurrentTime + Config.Instance.rebornProtectionTime;
                    byte[] dieBytes = Any.Pack(dieNtf).ToByteArray();
                    PlayerManager.Instance.ForEach((p) =>
                    {
                        DLLImport.Send(p.session, dieBytes, (uint)dieBytes.Length);
                    });
                    tankInstance.HP = Config.Instance.maxHp;
                    TankGame.TankHpSyncNtf tankHpSyncNtf = new TankGame.TankHpSyncNtf() { Id = tankInstance.ID, Hp = tankInstance.HP };
                    byte[] tankHpBytes = Any.Pack(tankHpSyncNtf).ToByteArray();
                    PlayerManager.Instance.ForEach((p) =>
                    {
                        if (p.session != IntPtr.Zero)
                        {
                            DLLImport.Send(p.session, tankHpBytes, (uint)tankHpBytes.Length);
                        }
                    });
                    tankInstance.rebornTime = Config.Instance.rebornProtectionTime;
                    Debug.Log($"Tank {tankInstance.ID} will be in reborn protection until {tankInstance.rebornTime}");
                    tankInstance.GetComponent<BoxCollider>().enabled = false;
				}

            }
            Debug.Log($"Bullet {ownerId}:{bulletId} hit tank {tankInstance.ID}, HP:{tankInstance.HP}");
			//Debug.Log($"Bullet {bulletId} collided with {collision.gameObject.name} tag {collision.gameObject.tag}, destroying bullet.");
			TankGame.BulletDestoryNtf bulletDestoryNtf = new TankGame.BulletDestoryNtf();
			bulletDestoryNtf.Id = bulletId;
			Vector3 pos = collision.collider.ClosestPoint(collision.transform.position);

			bulletDestoryNtf.Pos = new TankCommon.Vector3() {X = pos.x, Y = pos.y, Z = pos.z};

			byte[] messageBytes = Any.Pack(bulletDestoryNtf).ToByteArray();
			PlayerManager.Instance.ForEach((p) =>
			{
				if (p.session != IntPtr.Zero)
				{
					DLLImport.Send(p.session, messageBytes, (uint)messageBytes.Length);
				}
			});

			BulletManager.Instance.RemoveBullet(bulletId);
            // 销毁子弹
            Destroy(gameObject);
        }

#endif
    }

    // 子弹速度
    public float speed;

    public string ownerId;

    public UInt32 bulletId;

}

