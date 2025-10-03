using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class Bullet : MonoBehaviour
{
    // 子弹速度
    public float speed;

    public string ownerId;


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
	private void CheckScreenBounds()
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
            Destroy(gameObject);
        }
    }

    // 碰撞检测
    private void OnTriggerEnter(Collider other)
    {
#if UNITY_SERVER
        // 检测是否碰撞到坦克
        if (other.CompareTag("Tank"))
        {
            // 销毁子弹
            Destroy(gameObject);
            
            // 这里可以添加对坦克的伤害逻辑
            // 例如：other.GetComponent<Tank>().TakeDamage(damage);
        }
#endif
    }

}

