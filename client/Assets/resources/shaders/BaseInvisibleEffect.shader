// Upgrade NOTE: replaced 'mul(UNITY_MATRIX_MVP,*)' with 'UnityObjectToClipPos(*)'

Shader "Demo/BaseInvisibleEffect" 
{
	Properties
	{
		_MainTex("MainTex(RGB)", 2D) = "white" {}                   // 人物模型纹理
//		_DissolveVector("DissolveVector", Vector) = (0,0,0,0)        // 消失阈值
		_InvisibleX("从头到尾", Range(-2, 0.8)) = -0.8            // 人物从头到尾方向
		_InvisibleY("从前到后", Range(-0.5, 0.15)) = -0.2           //人物从前到后方向
		_InvisibleZ("从右到左", Range(-0.9, 0.55)) = -0.6           //人物从右到左方向
	}
	
	CGINCLUDE
	#include "Lighting.cginc"
	uniform sampler2D _MainTex;
	uniform float4 _MainTex_ST;
	//uniform float4 _DissolveVector;
	uniform float _InvisibleX;
	uniform float _InvisibleY;
	uniform float _InvisibleZ;

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
			//不满足条件的discard
//		clip(i.objPos.xyz - _DissolveVector.xyz);
		clip(i.objPos.x - _InvisibleX);
		clip(i.objPos.y - _InvisibleY);
		clip(i.objPos.z - _InvisibleZ);

		half3 normal = normalize(i.worldNormal);
		half3 light = normalize(i.worldLight);
		fixed diff = max(0, dot(normal, light));
		fixed4 albedo = tex2D(_MainTex, i.uv);

		fixed4 c;
		c.rgb = diff * albedo;
		c.a = 1;
		return c;
	}
	ENDCG

	SubShader
	{
		
		Pass
		{
			Tags{ "RenderType" = "Opaque" }
			
			CGPROGRAM
			#pragma vertex vert
			#pragma fragment frag
			ENDCG	
		}
	}
	FallBack "Diffuse"
}
