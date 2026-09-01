# Restic Backup Dashboard

[![CI](https://github.com/fengcyborg/restic-backup-dashboard/actions/workflows/ci.yml/badge.svg)](https://github.com/fengcyborg/restic-backup-dashboard/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md)

一个面向 Restic 备份链路的安全、只读状态面板。它把成功标记、systemd 状态、脱敏事件日志、ZFS 容量、异地同步进度和恢复验证结果汇总到一个响应式 Web 页面中。

## 为什么开发这个项目

`restic snapshots` 可以证明快照能够被列出，却无法让运维人员一眼确认：最新本地备份代次是否已经同步到异地存储、调度器是否仍在运行，以及最近的恢复测试是否满足策略要求。本项目用于补齐这一可观测性缺口，同时确保 Web 进程无法接触备份凭据。

后端使用 Go 编写。NAS 上长期运行的小服务适合采用单个静态二进制：内存占用可预测、启动快，并可内嵌前端资源。项目支持 Go 1.26 及以上版本。

## 演示

```bash
go run ./cmd/restic-backup-dashboard serve --demo --listen 127.0.0.1:8080
```

访问 <http://127.0.0.1:8080/?demo=1>。界面会跟随浏览器语言，也可手动切换中英文。

## 安全模型

采集器与 Web 服务被有意划分为两个独立的信任区域：

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

- 采集器在备份宿主机上运行，可以读取运行状态。
- 生成的 JSON 不包含仓库地址、源路径、命令行、密码或服务商凭据。
- Web 进程只读取该 JSON，不需要 Docker socket、systemd socket、Restic 密码或仓库挂载。
- 页面不提供备份、恢复、清理、遗忘或删除操作。
- 服务默认监听 `127.0.0.1`，并设置严格的 CSP、防嵌套和 `nosniff` 等安全响应头。

完整边界说明见[架构与威胁模型](docs/ARCHITECTURE.zh-CN.md)。

## 可观测内容

- 增量备份、异地同步和恢复审计的新鲜度
- systemd 服务/定时器状态及安全的退出状态展示
- 本地与异地备份代次的精确比较
- rclone 或结构化 JSON 格式的传输进度
- Restic JSON 摘要总量和短快照 ID
- 恢复审计策略、保留规则、抽检字节数及数据集阈值
- ZFS 数据集容量和存储池健康状态（可选）
- 显式文件或命令依赖检查
- Prometheus 指标、存活检查以及感知状态新鲜度的就绪检查

它不会执行或调度 Restic 备份。现有自动化仍是事实来源，并为采集器输出少量标记和事件。

## 生产环境部署

构建并安装二进制文件：

```bash
go build -trimpath -o restic-backup-dashboard ./cmd/restic-backup-dashboard
sudo install -m 0755 restic-backup-dashboard /usr/local/bin/restic-backup-dashboard
sudo install -d -m 0755 /etc/restic-backup-dashboard /var/lib/restic-backup-dashboard
sudo install -m 0644 configs/config.example.json /etc/restic-backup-dashboard/config.json
```

根据实际任务名称和标记路径修改示例配置，然后先校验配置并执行一次采集：

```bash
restic-backup-dashboard validate-config --config /etc/restic-backup-dashboard/config.json
sudo restic-backup-dashboard collect --config /etc/restic-backup-dashboard/config.json
restic-backup-dashboard serve --listen 127.0.0.1:8080 \
  --status-file /var/lib/restic-backup-dashboard/status.json
```

项目提供的 [systemd 单元](deploy/systemd/) 每分钟执行一次采集器，并可将 Web 服务作为经过加固的动态用户运行。安装到宿主机前，请根据实际环境检查路径和加固选项。

完整的标记与事件格式见[配置参考](docs/CONFIGURATION.zh-CN.md)。

## 容器部署

只有 Web 服务适合放入容器。请在宿主机运行 `collect`，确保生成目录可读，再以只读方式挂载该目录：

```bash
mkdir -p runtime
cp /var/lib/restic-backup-dashboard/status.json runtime/status.json
docker compose up -d
```

示例默认只绑定本机回环地址。如需远程访问，请在服务前部署带身份认证的反向代理。

## HTTP 接口

| 接口 | 用途 |
|---|---|
| `/` | 内嵌的响应式状态面板 |
| `/api/v1/status` | 经过脱敏且带版本号的 JSON |
| `/healthz` | 进程存活检查 |
| `/readyz` | 检查状态文件是否存在、合法且足够新鲜 |
| `/metrics` | Prometheus 文本格式指标 |

## CLI

```text
restic-backup-dashboard serve
restic-backup-dashboard collect
restic-backup-dashboard validate-config
restic-backup-dashboard healthcheck
restic-backup-dashboard version
```

使用 `-h` 查看各命令的参数。

## 文档

- [配置参考](docs/CONFIGURATION.zh-CN.md)（[English](docs/CONFIGURATION.md)）
- [架构与威胁模型](docs/ARCHITECTURE.zh-CN.md)（[English](docs/ARCHITECTURE.md)）
- [贡献指南](CONTRIBUTING.zh-CN.md)（[English](CONTRIBUTING.md)）
- [安全策略](SECURITY.zh-CN.md)（[English](SECURITY.md)）

## 项目状态

本项目处于早期公开发布阶段。状态结构带有版本号，配置会拒绝未知字段，CI 会测试受支持的 Go 版本。在将绿色状态视为运行保障之前，请根据自己的恢复策略检查配置和恢复审计语义。

欢迎参与贡献，详见[贡献指南](CONTRIBUTING.zh-CN.md)。安全问题请按照[安全策略](SECURITY.zh-CN.md)报告。

## 许可证

[MIT](LICENSE)
