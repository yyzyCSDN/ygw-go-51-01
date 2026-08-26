# DeviceShadow

DeviceShadow 是物联网设备影子同步服务：为每台设备维护期望状态与上报状态
两份影子文档，通过版本号合并乱序上报与并发下发，支持批量下发、断线重连
同步、离线缓存回放以及订阅变更通知，并提供浏览器控制台页面。

## 构建

```bash
./build_benzhi_docker.sh deviceshadow linux/amd64
./build_benzhi_docker.sh deviceshadow linux/arm64
```

## 运行

```bash
go run ./cmd/deviceshadow -addr :8080
```

启动后打开 http://localhost:8080 即可看到设备影子控制台。

## 接口

- `GET /healthz` 健康检查
- `POST /api/v1/devices` 注册设备
- `POST /api/v1/devices/{id}/report` 设备上报状态
- `POST /api/v1/devices/{id}/desired` 下发期望状态
- `POST /api/v1/devices/{id}/desired/batch` 批量下发
- `GET /api/v1/devices/{id}/shadow` 读取影子
- `DELETE /api/v1/devices/{id}/shadow` 删除影子
- `POST /api/v1/devices/{id}/reconnect` 设备重连同步

## 容器内验证

```bash
go build ./...
go test ./...
go vet ./...
go run ./cmd/deviceshadow
```
