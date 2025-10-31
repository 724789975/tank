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
        nameText.text = Name;
#if UNITY_SERVER && !AI_RUNNING
#else
        GetComponent<BoxCollider>().enabled = false;
#endif
    }

    // Update is called once per frame
    void Update()
    {
#if !AI_TRAIN
#if UNITY_SERVER && !AI_RUNNING
        if (rebornTime > 0)
        {
            rebornTime -= Time.deltaTime;
            isDead = true;
        }
        else
        {
            if (isDead)
            {
				rebornTime = 0;
				GetComponent<BoxCollider>().enabled = true;
				Debug.Log($"Tank {ID} reborn protection ended.");
                isDead = false;
			}
        }

        ServerPlayer playerData = PlayerManager.Instance.GetPlayer(ID);
        if (playerData == null)
        {
            Debug.LogWarning($"get player error {ID}");
            return;
		}
        if (playerData.session == System.IntPtr.Zero)
        {
            offLineTime += Time.deltaTime;
        }
#else
		// 客户端模式下的同步
		ClientPlayer clientPlayer = PlayerManager.Instance.GetPlayer(ID);
		if (clientPlayer == null)
		{
			Debug.LogWarning($"get player error {ID}");
			return;
		}

		if (rebornTime > 0)
		{
			isDead = true;
			rebornTime -= Time.deltaTime;
			gameObject.GetComponent<MeshRenderer>().enabled = (int)(rebornTime * 10) % 2 == 0 ? false : true;
			gameObject.GetComponentInChildren<MeshRenderer>().enabled = (int)(rebornTime * 10) % 2 == 0 ? false : true;
		}
		else
		{
			if (isDead)
			{
				isDead = false;
				rebornTime = 0;
				gameObject.GetComponent<MeshRenderer>().enabled = true;
				gameObject.GetComponentInChildren<MeshRenderer>().enabled = true;
				Debug.Log($"Tank {ID} reborn protection ended.");
			}
		}

		if (clientPlayer.syncs.Count == 0)
		{
			return;
		}
		while (clientPlayer.syncs.Count > 1 && clientPlayer.syncs[1].SyncTime <= ClientFrame.Instance.CurrentTime)
		{
			clientPlayer.syncs.RemoveAt(0);
		}

		if (clientPlayer.syncs.Count > 1)
		{
			TankGame.PlayerStateNtf sync1 = clientPlayer.syncs[0];
			TankGame.PlayerStateNtf sync2 = clientPlayer.syncs[1];

			float s = Mathf.Clamp((ClientFrame.Instance.CurrentTime - sync1.SyncTime) / (sync2.SyncTime - sync1.SyncTime), 0, 1);

			transform.position = Vector3.Lerp(new Vector3(sync1.Transform.Position.X, sync1.Transform.Position.Y, sync1.Transform.Position.Z), new Vector3(sync2.Transform.Position.X, sync2.Transform.Position.Y, sync2.Transform.Position.Z), s);

			transform.rotation = Quaternion.Slerp(new Quaternion(sync1.Transform.Rotation.X, sync1.Transform.Rotation.Y, sync1.Transform.Rotation.Z, sync1.Transform.Rotation.W), new Quaternion(sync2.Transform.Rotation.X, sync2.Transform.Rotation.Y, sync2.Transform.Rotation.Z, sync2.Transform.Rotation.W), s);
		}
		else
		{
			TankGame.PlayerStateNtf sync = clientPlayer.syncs[0];
			transform.position = new Vector3(sync.Transform.Position.X, sync.Transform.Position.Y, sync.Transform.Position.Z);
			transform.rotation = new Quaternion(sync.Transform.Rotation.X, sync.Transform.Rotation.Y, sync.Transform.Rotation.Z, sync.Transform.Rotation.W);
		}
#endif
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
        get { return hp; }
        set
        {
            hp = value;
			hpSlider.value = hp / 100.0f;
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

    public string Name;

    public GameObject bulletPos;

    public GameObject bulletPrefab;

    public TextMeshPro nameText;
    public float speed;

    public Slider hpSlider;

    public float rebornTime = 0f;
    bool isDead = false;
    int hp = 0;

#if UNITY_SERVER && !AI_RUNNING
    public float offLineTime;
#endif
}
