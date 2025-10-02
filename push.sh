#!/bin/bash
# build-and-push.sh

set -e

USERNAME="q1285514609"
APP_NAME="new-api"
VERSION=$(date +%Y%m%d-%H%M%S)

echo "🔨 构建镜像..."
docker build --no-cache -t $USERNAME/$APP_NAME:latest -f Dockerfile ..
docker tag $USERNAME/$APP_NAME:latest $USERNAME/$APP_NAME:$VERSION

echo "📤 推送到 Docker Hub..."
docker push $USERNAME/$APP_NAME:latest
docker push $USERNAME/$APP_NAME:$VERSION

echo "✅ 推送完成！"
echo "🕐 等待服务器自动更新（约1-2分钟）..."
