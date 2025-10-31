// Upgrade NOTE: replaced 'mul(UNITY_MATRIX_MVP,*)' with 'UnityObjectToClipPos(*)'

Shader "Demo/DissolveInvisibleEffect" 
{
	Properties
	{
		_MainTex("MainTex(RGB)", 2D) = "white" {}                   // 人物模型纹理
		_DissolveMap("_DissolveMap", 2D) = "white" {}               //噪声纹理
//		_DissolveVector("DissolveVector", Vector) = (0,0,0,0)        // 消失阈值
		_InvisibleX("从头到尾", Range(-0.82, 0.8)) = -0.8            // 人物从头到尾方向
		_InvisibleY("从前到后", Range(-0.2, 0.15)) = -0.2           //人物从前到后方向
		_InvisibleZ("从右到左", Range(-0.6, 0.55)) = -0.6           //人物从右到左方向
		_DissolveThreshold("DissolveThreshold", Range(0, 1)) = 0  //消失阈值
		_DissolveSpeedFactor("DissolveSpeed", Range(0, 5)) = 2  // 消失速度
		_DissolveColor("Dissolve Color", Color) = (1, 1, 1, 1)       // 边缘颜色
		 _ColorFactor("ColorFactor", Range(0,1)) = 0.7              // 边缘颜色带（值越大,颜色带越宽）
	}
	
	CGINCLUDE
	#include "Lighting.cginc"
	uniform sampler2D _MainTex;
	uniform float4 _MainTex_ST;
	uniform sampler2D _DissolveMap;
	//uniform float4 _DissolveVector;
	uniform float _InvisibleX;
	uniform float _InvisibleY;
	uniform float _InvisibleZ;
	uniform float _DissolveThreshold;
	uniform float _DissolveSpeedFactor;
	uniform fixed4 _DissolveColor;
	uniform float _ColorFactor;

	struct v2f 
	{
		float4 pos : SV_POSITION;
		float3 worldNormal : NORMAL;
		float2 uv : TEXCOORD0;
		float3 worldLight : TEXCOORD1;
		float4 objPos : TEXCOORD2;
	};

	v2f vert(appdata_base v)
	{
		v2f o;
		//模型的范围是尝试出来的，不同的模型对应不同的范围
//		v.vertex.x += 0.8;    （-0.8, 0.78）      Y 方向 （人物从头到尾方向）
//		v.vertex.y += 0.2;    （-0.2, 0.12 ）     Z 方向 （人物从前到后方向）
//		v.vertex.z += 0.6;    （-0.6, 0.55 ）     X 方向 （人物从右到左方向）
		o.pos = UnityObjectToClipPos(v.vertex);
		o.uv = TRANSFORM_TEX(v.texcoord, _MainTex);
		//顶点转化到世界空间
		o.objPos = v.vertex;
		o.worldNormal = UnityObjectToWorldNormal(v.normal);
		o.worldLight = UnityObjectToWorldDir(_WorldSpaceLightPos0.xyz);
		return o;
	}
			
	fixed4 frag(v2f i) : SV_Target
	{
		 fixed4 dissolve = tex2D(_DissolveMap, i.uv);  
			//不满足条件的discard
//		clip(i.objPos.xyz - _DissolveVector.xyz);
		float factor = i.objPos.x - _InvisibleX; 
		// 由于abs(factor)由人物模型从头到尾方向逐渐变小，所以模型仍是从头到尾逐渐消失
        //clip(x) 当x<0时，才进行剔除
		float dissolveFactor = _DissolveThreshold - dissolve.r * abs(factor) * _DissolveSpeedFactor;
		 if(factor < 0)    // 可以让这个条件一直成立
        {  
            clip(dissolveFactor);  
        }  
		clip(i.objPos.y - _InvisibleY);
		clip(i.objPos.z - _InvisibleZ);

		half3 normal = normalize(i.worldNormal);
		half3 light = normalize(i.worldLight);
		fixed diff = max(0, dot(normal, light));
		fixed4 albedo = tex2D(_MainTex, i.uv);

		fixed3 color;
		color = diff * albedo;

		 fixed lerpFactor = saturate(sign(_ColorFactor - dissolveFactor));    

        return lerpFactor * _DissolveColor + (1 - lerpFactor) * fixed4(color, 1);  

        return fixed4(color, 1);
	}
	ENDCG

	SubShader
	{
		
		Pass
		{
			Cull Off
			Tags{ "RenderType" = "Opaque" }
			
			CGPROGRAM
			#pragma vertex vert
			#pragma fragment frag
			ENDCG	
		}
	}
	FallBack "Diffuse"
}
