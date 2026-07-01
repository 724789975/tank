#!/bin/bash

set -e

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
cd "$SCRIPT_DIR"

PUB_KEY="${1:-public.pem}"
OUTPUT="${2:-jwt-auth.yaml}"
ISSUER="${ISSUER:-tank}"

if [[ ! -f "$PUB_KEY" ]]; then
    echo "错误: 找不到公钥文件 $PUB_KEY"
    echo "用法: $0 [公钥文件] [输出文件] [--apply]"
    exit 1
fi

# 检查是否需要直接 apply
APPLY=false
for arg in "$@"; do
    [[ "$arg" == "--apply" ]] && APPLY=true
done

# 从 PEM 公钥中提取 n 和 e
KEY_INFO=$(openssl rsa -pubin -in "$PUB_KEY" -text -noout 2>/dev/null)

# 提取 modulus (n) - 转为 base64url
MODULUS_HEX=$(echo "$KEY_INFO" | grep -A 100 "Modulus" | grep -v "Modulus" | grep -v "Exponent" | tr -d ' :\n')
# 去除 DER 编码的前导 00 字节（符号位），JWK 规范不需要
MODULUS_HEX="${MODULUS_HEX#00}"
MODULUS_B64=$(echo "$MODULUS_HEX" | xxd -r -p | base64 -w0 | tr -d '=' | tr '+/' '-_')

# 提取 exponent (e)
EXPONENT_DEC=$(echo "$KEY_INFO" | grep "Exponent" | grep -o '[0-9]*' | head -1)
EXPONENT_B64=$(printf '%06x' "$EXPONENT_DEC" | xxd -r -p | base64 -w0 | tr -d '=' | tr '+/' '-_')

# 生成 KID（公钥指纹）
KID=$(openssl pkey -pubin -in "$PUB_KEY" -outform DER 2>/dev/null | openssl dgst -sha256 | awk '{print $2}' | cut -c1-16)

cat > "$OUTPUT" <<EOF
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: jwt-auth
  namespace: istio-system
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  jwtRules:
  - issuer: "${ISSUER}"
    jwks: |
      {
        "keys": [
          {
            "kty": "RSA",
            "alg": "RS256",
            "use": "sig",
            "e": "${EXPONENT_B64}",
            "n": "${MODULUS_B64}",
            "kid": "${KID}"
          }
        ]
      }
    forwardOriginalToken: true
    fromHeaders:
    - name: Authorization
      prefix: "Bearer "
EOF

echo "已生成: $OUTPUT"
echo "  Issuer:  $ISSUER"
echo "  KID:     $KID"

if $APPLY; then
    echo "应用到 Kubernetes..."
    kubectl apply -f "$OUTPUT"
    echo "完成"
fi
