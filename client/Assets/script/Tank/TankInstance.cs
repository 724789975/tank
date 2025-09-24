using System.Collections;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

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
        
    }

    public void SetPos(Vector2 pos)
    {
        transform.position = new Vector3(pos.x, pos.y, 0);
    }

    public void SetDir(Vector2 dir)
    {
        transform.right = dir;
    }

    public void AddPos(Vector2 pos)
    {
        transform.position = new Vector3(pos.x + transform.position.x, pos.y + transform.position.y, 0);
    }

    public void Shoot()
    {
        GameObject bullet = Instantiate(bulletPrefab, bulletPos.transform.position, Quaternion.identity);
        bullet.GetComponent<bullet>().ownerId = ID;
        bullet.GetComponent<bullet>().SetDirection(transform.right);
    }

    /// <summary>
    /// 坦克血量
    /// </summary>
    public int HP;

	/// <summary>
	/// 坦克ID
	/// </summary>
	public string ID;

    public GameObject bulletPos;

    public GameObject bulletPrefab;

    public TextMeshPro idText;
}
