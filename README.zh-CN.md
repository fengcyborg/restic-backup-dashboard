# Restic Backup Dashboard

一个面向 Restic 备份链路的安全、只读状态面板。它把成功标记、systemd 状态、脱敏事件、ZFS 容量、异地同步进度和恢复抽检结果汇总到一个响应式页面中。

## 为什么后端选择 Go

NAS 上长期运行的小服务更适合单文件部署。Go 可以把 API、采集器和前端资源编译成一个二进制，不依赖 Python/Node 运行环境，启动快、内存占用可控，也方便在普通 Linux、TrueNAS SCALE 虚拟机或容器环境中分发。项目最低支持 Go 1.26。

## 先看演示

```bash
go run ./cmd/restic-backup-dashboard serve --demo --listen 127.0.0.1:8080
```

访问 <http://127.0.0.1:8080/?demo=1>。界面会跟随浏览器语言，也可手动切换中英文。

## 安全边界

```text
Restic / rclone / systemd / ZFS
              │ 只读采集
              ▼
          宿主机采集器
              │ 原子写入脱敏 status.json
              ▼
      非特权 Web 进程或容器
              │ 只读 API
              ▼
             浏览器
```

- 只有宿主机采集器读取运行状态。
- `status.json` 不包含仓库地址、源文件路径、命令行、密码或网盘凭据。
- Web 进程不挂载 Restic 密钥、仓库、Docker socket 或 systemd socket。
- 页面没有备份、恢复、清理、删除等写操作。
- 默认只监听 `127.0.0.1`；如需远程访问，应放在带身份认证的反向代理之后。

更完整的说明见 [架构与威胁模型](docs/ARCHITECTURE.md)。

## 它能展示什么

- 本地增量、异地同步、恢复抽检三段链路
- systemd 服务与定时器状态
- 本地成功代次和异地已上传代次是否一致
- rclone/结构化 JSON 实时传输进度
- Restic JSON 摘要、短快照 ID、数据量变化
- 恢复抽检新鲜度、抽检比例、数据组最低容量阈值
- ZFS 数据集容量和存储池健康（可选）
- 依赖项状态、最近事件、Prometheus 指标

它不直接执行或调度备份。现有的 Restic/rclone 自动化继续负责备份、同步和抽检，只需输出很小的成功标记和脱敏事件。

## 部署

```bash
go build -trimpath -o restic-backup-dashboard ./cmd/restic-backup-dashboard
sudo install -m 0755 restic-backup-dashboard /usr/local/bin/restic-backup-dashboard
sudo install -d -m 0755 /etc/restic-backup-dashboard /var/lib/restic-backup-dashboard
sudo install -m 0644 configs/config.example.json /etc/restic-backup-dashboard/config.json
```

根据实际任务名称和路径修改配置，然后先校验、采集一次：

```bash
restic-backup-dashboard validate-config --config /etc/restic-backup-dashboard/config.json
sudo restic-backup-dashboard collect --config /etc/restic-backup-dashboard/config.json
restic-backup-dashboard serve --listen 127.0.0.1:8080 \
  --status-file /var/lib/restic-backup-dashboard/status.json
```

[deploy/systemd](deploy/systemd/) 提供每分钟采集以及非特权 Web 服务模板。标记文件、事件格式和恢复抽检约定见 [配置说明](docs/CONFIGURATION.md)。

容器只运行 Web 部分，宿主机仍负责采集。示例 Compose 将状态目录以只读方式挂入，并默认仅映射到本机回环地址。

## 接口

| 路径 | 用途 |
|---|---|
| `/` | 内嵌响应式面板 |
| `/api/v1/status` | 脱敏且带版本的状态 JSON |
| `/healthz` | 进程存活检查 |
| `/readyz` | 状态文件存在、合法且足够新鲜 |
| `/metrics` | Prometheus 指标 |

## 重要说明

这是早期公开版本。“绿色”表示采集到的信号满足你配置的策略，不替代人工灾难恢复演练。上线前应按自己的 RPO/RTO、保留策略和恢复标准审核配置。

项目采用 [MIT License](LICENSE)。贡献方式见 [CONTRIBUTING.md](CONTRIBUTING.md)，安全问题请按 [SECURITY.md](SECURITY.md) 私下报告。
