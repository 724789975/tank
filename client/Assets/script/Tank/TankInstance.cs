using System.Collections;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using static UnityEngine.Tilemaps.Tilemap;

public class TankInstance : MonoBehaviour
{
    // Start is called before the first frame update
    void Start()
    {
        HP = 100;
        idText.text = ID;
    }

    // Update is called once per frame
    void Update()
    {
#if !UNITY_SERVER
		// 客户端模式下的同步
        ClientPlayer clientPlayer = PlayerManager.Instance.GetPlayer(ID);
        if (clientPlayer == null)
        {
            Debug.LogWarning($"get player error {ID}");
            return;
        }

        if (clientPlayer.syncs.Count == 0)
        {
            return;
        }
        while(clientPlayer.syncs.Count > 1 && clientPlayer.syncs[1].SyncTime <= syncTime)
        {
            clientPlayer.syncs.RemoveAt(0);
        }

        if (clientPlayer.syncs.Count > 1)
        {
            TankGame.PlayerStateNtf sync1 = clientPlayer.syncs[0];
            TankGame.PlayerStateNtf sync2 = clientPlayer.syncs[1];

			float s = Mathf.Clamp((syncTime - sync1.SyncTime) / (sync2.SyncTime - sync1.SyncTime), 0, 1);

            transform.position = Vector3.Lerp(new Vector3(sync1.Transform.Position.X, sync1.Transform.Position.Y, sync1.Transform.Position.Z), new Vector3(sync2.Transform.Position.X, sync2.Transform.Position.Y, sync2.Transform.Position.Z), s);

            transform.rotation = Quaternion.Slerp(new Quaternion(sync1.Transform.Rotation.X, sync1.Transform.Rotation.Y, sync1.Transform.Rotation.Z, sync1.Transform.Rotation.W), new Quaternion(sync2.Transform.Rotation.X, sync2.Transform.Rotation.Y, sync2.Transform.Rotation.Z, sync2.Transform.Rotation.W), s);
			syncTime += Time.deltaTime * 1000;
        }
        else
        {
            TankGame.PlayerStateNtf sync = clientPlayer.syncs[0];
            transform.position = new Vector3(sync.Transform.Position.X, sync.Transform.Position.Y, sync.Transform.Position.Z);
            transform.rotation = new Quaternion(sync.Transform.Rotation.X, sync.Transform.Rotation.Y, sync.Transform.Rotation.Z, sync.Transform.Rotation.W);

            syncTime = sync.SyncTime;
        }

        rebornTime -= Time.deltaTime;

#endif
	}

    public void SetDir(Vector3 dir)
    {
        transform.right = dir;
    }

    public void AddPos(Vector2 pos)
    {
        transform.position = new Vector3(pos.x + transform.position.x, pos.y + transform.position.y, 0);
        Vector3 p = transform.position;
        float left = Config.Instance.GetLeft();
        float right = Config.Instance.GetRight();
        float top = Config.Instance.GetTop();
        float bottom = Config.Instance.GetBottom();
        p.x = Mathf.Clamp(p.x, left, right);
        p.y = Mathf.Clamp(p.y, bottom, top);
        transform.position = p;
    }

    public void Shoot(string Id, Vector3 pos, Quaternion rot, float speed)
    {
		Bullet bullet = BulletManager.Instance.AddBullet(pos, rot, Id, speed);
	}

    public void Shoot()
    {
        Bullet bullet = BulletManager.Instance.AddBullet(bulletPos.transform.position, bulletPos.transform.rotation, ID, Config.Instance.speed);

        TankGame.PlayerShootReq playerShootReq = new TankGame.PlayerShootReq();
        playerShootReq.Transform = new TankCommon.Transform();
        playerShootReq.Transform.Position = new TankCommon.Vector3() { X = bullet.transform.position.x, Y = bullet.transform.position.y, Z = bullet.transform.position.z };
        playerShootReq.Transform.Rotation = new TankCommon.Quaternion() { X = bullet.transform.rotation.x, Y = bullet.transform.rotation.y, Z = bullet.transform.rotation.z, W = bullet.transform.rotation.w };
        NetClient.Instance.SendMessage(playerShootReq);
    }



    /// <summary>
    /// 坦克血量
    /// </summary>
    public int HP
    {
        get { return (int)(hpSlider.value * 100); }
        set
        {
			hpSlider.value = value / 100.0f;
            float colorValue = hpSlider.value - 0.2f;
            if (colorValue < 0) colorValue = 0;
            colorValue = colorValue / 0.8f;
            hpSlider.fillRect.GetComponent<Image>().color = Color.Lerp(Color.red, Color.green, colorValue);
		}
	}

	/// <summary>
	/// 坦克ID
	/// </summary>
	public string ID;

    public GameObject bulletPos;

    public GameObject bulletPrefab;

    public TextMeshPro idText;
    public float speed;

    public Slider hpSlider;

    public float rebornTime = 3.0f;

#if !UNITY_SERVER
	float syncTime = 0;
#endif
}
