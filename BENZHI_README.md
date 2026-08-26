# rate-limit — Go 令牌桶与滑动窗口限流 HTTP 服务（含持久化快照与恢复）

按给定速率与突发容量判定请求放行或拒绝，算法可选令牌桶、固定窗口或滑动日志；状态写入检查点，崩溃后须从快照恢复且计数不回退。超限返回拒绝，速率或突发非法时不得启动；同一密钥在恢复前后的剩余配额必须连续。

## Build / Run / Test

```bash
go build -o rate-limit .
./rate-limit httpd -addr :8080 -rate 10 -burst 20
go test ./...
```

## Evaluation Image

Evaluation-specific files (do not overwrite project Dockerfile/README):

- `benzhi.Dockerfile`
- `build_benzhi_docker.sh`
- `BENZHI_README.md` (this file)

Build and verify in container:

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh <image-name> linux/arm64
./build_benzhi_docker.sh <image-name> linux/amd64
docker run -it <image-name>:latest
```
