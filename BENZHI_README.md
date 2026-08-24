# ConfigCenter

分布式配置中心示例：版本化配置存储、发布/回滚、灰度、watch 推送与
客户端缓存。启动后访问 / 打开控制台，/healthz 提供健康检查，
/api/pull、/api/publish、/api/rollback、/api/status 提供运维接口。

## 构建与运行

```sh
go build -mod=vendor ./...
./configd -addr 127.0.0.1:8080
```

容器：

```sh
sh build_benzhi_docker.sh
docker run --rm -p 8080:8080 confighub:latest
```
