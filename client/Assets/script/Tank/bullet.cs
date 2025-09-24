using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class bullet : MonoBehaviour
{
    // 子弹速度
    public float speed;
    // 子弹方向
    private Vector3 direction;

    public string ownerId;

    // 设置子弹方向的方法
    public void SetDirection(Vector3 dir)
    {
        direction = dir.normalized; // 归一化方向向量
    }

    // Start is called before the first frame update
    void Start()
    {
        // 默认方向为向上
        if (direction == Vector3.zero)
        {
            direction = Vector3.up;
        }
    }

    // Update is called once per frame
    void Update()
    {
        // 移动子弹
        transform.Translate(direction * speed * Time.deltaTime);
        
        // 检查是否超出屏幕边界
        CheckScreenBounds();
    }

    // 检查屏幕边界
    private void CheckScreenBounds()
    {
        Vector3 position = transform.position;
        
        // 获取屏幕边界
        float left = Config.GetLeft();
        float right = Config.GetRight();
        float top = Config.GetTop();
        float bottom = Config.GetBottom();
        
        // 如果子弹超出屏幕边界，销毁子弹
        if (position.x < left || position.x > right || position.y < bottom || position.y > top)
        {
            Destroy(gameObject);
        }
    }

    // 碰撞检测
    private void OnTriggerEnter(Collider other)
    {
        // 检测是否碰撞到坦克
        if (other.CompareTag("Tank"))
        {
            // 销毁子弹
            Destroy(gameObject);
            
            // 这里可以添加对坦克的伤害逻辑
            // 例如：other.GetComponent<Tank>().TakeDamage(damage);
        }
    }
}
