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
		// 移动子弹
		transform.Translate(Vector3.right * speed * Time.deltaTime);

		// 检查是否超出屏幕边界
		CheckScreenBounds();
	}
    
	// 检查屏幕边界
	void CheckScreenBounds()
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
        }
    }

    // 碰撞检测
	void OnCollisionEnter(Collision collision)
	{
#if UNITY_SERVER
        TankGame.BulletDestoryNtf bulletDestoryNtf = new TankGame.BulletDestoryNtf();
        bulletDestoryNtf.Id = bulletId;
        bulletDestoryNtf.Pos = new TankCommon.Vector3() {X = collision.transform.position.x, Y = collision.transform.position.y, Z = collision.transform.position.z};
		byte[] messageBytes = Any.Pack(bulletDestoryNtf).ToByteArray();
		PlayerManager.Instance.ForEach((p) =>
        {
            if (p.session != IntPtr.Zero)
            {
                DLLImport.Send(p.session, messageBytes, (uint)messageBytes.Length);
            }
        });
        // 检测是否碰撞到坦克
        if (collision.gameObject.CompareTag("Tank"))
        {
            Debug.Log($"Bullet hit tank {collision.gameObject.name} ");
            // 销毁子弹
            Destroy(gameObject);
        }

        BulletManager.Instance.RemoveBullet(bulletId);
#endif
    }

    // 子弹速度
    public float speed;

    public string ownerId;

    public UInt32 bulletId;

}

