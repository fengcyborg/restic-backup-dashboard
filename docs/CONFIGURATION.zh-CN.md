# 配置参考

[English](CONFIGURATION.md)

采集器接受严格的 JSON。未知字段会被拒绝，因此拼写错误会在 `validate-config` 阶段失败，而不会悄悄停用检查。请从 [`configs/config.example.json`](../configs/config.example.json) 开始配置。

每次修改配置后，都应先完成校验再采集状态：

```bash
restic-backup-dashboard validate-config --config /etc/restic-backup-dashboard/config.json
```

## 通用规则

- `base_dir` 为必填项。相对的标记、摘要、日志、依赖文件和默认输出路径都在其下解析；绝对路径保持不变。
- 持续时间是 Go `time.ParseDuration` 所接受的 JSON 字符串，例如 `30m`、`2h` 或 `168h`。它不支持 `d` 单位，多日时长请使用小时。
- 显示名称、计划说明、摘要和阶段标签是运维人员编写的可信文本，可以本地化。ID、事件 token、JSON 字段名、状态值和指标名称应保持为稳定的 ASCII 标识符。
- 空的可选部分会停用相应观测。面板绝不会启动备份、同步、恢复、保留或调度任务。

## 顶层字段

| 字段 | 必填 | 用途 |
|---|---:|---|
| `display` | 否 | 面板标题、副标题、服务商名称和仓库显示名称。空值会使用内置英文默认值。 |
| `base_dir` | 是 | 解析相对输入和输出路径时使用的基础目录。 |
| `output_file` | 否 | 脱敏状态的输出位置。默认为 `base_dir` 下的 `dashboard/status.json`；`collect --output` 可覆盖它。 |
| `hostname` | 否 | 公开的主机标签。为空时，采集器使用操作系统主机名。 |
| `tasks` | 是 | 一个或多个备份、同步或审计任务定义。 |
| `sync` | 否 | 本地/异地代次比较及活动传输进度。 |
| `recovery` | 否 | 恢复审计策略信号和 Restic 摘要投影。 |
| `storage` | 否 | 可选的 ZFS 容量与存储池健康观测。 |
| `dependencies` | 否 | 显式文件和命令探针。 |
| `events` | 否 | 经过白名单限制和脱敏的事件源。 |
| `phase_rules` | 否 | 通过进程参数子字符串匹配选择的可信标签。 |

`display` 接受 `title`、`subtitle`、`provider_name` 和 `repository_name`。这些值会复制到公开状态中，不得包含敏感的基础设施信息。

## 任务字段

`tasks` 中的每一项接受以下字段：

| 字段 | 必填 | 用途 |
|---|---:|---|
| `id` | 是 | 唯一且稳定的标识符，会被其他部分引用并导出到指标中。 |
| `kind` | 是 | `backup`、`sync` 或 `audit` 之一。 |
| `name` | 是 | 浏览器中显示的任务名称。 |
| `schedule` | 否 | 浏览器中显示的说明；它不会创建实际调度。 |
| `service` | 否 | 通过 `systemctl show` 检查的 systemd 服务单元。 |
| `timer` | 否 | 检查可用性、活动状态和下次运行时间的 systemd 定时器单元。 |
| `success_marker` | 否 | 包含最近成功代次或时间戳的文件。 |
| `duration_marker` | 否 | 包含最近一次成功耗时的文件。 |
| `healthy_for` | 是 | 得到健康结果的正数新鲜度窗口。 |
| `error_after` | 是 | 正数错误阈值，不能短于 `healthy_for`。 |
| `running_label` | 否 | 服务活动时显示的可信后备阶段标签。 |

状态按以下优先级判断：活动服务为 `running`；失败的服务结果或除 `75` 以外的非零退出状态为 `error`；已配置但不可用或未活动的定时器为 `error`；缺少成功标记为 `unknown`；否则根据标记年龄得到 `healthy`、`warning` 或 `error`。退出状态 `75` 保留给临时锁竞争，并回退到标记新鲜度判断。

## 同步字段

设置 `sync.task_id` 可启用代次比较。它必须引用一个任务；此时 `source_generation_marker`、`uploaded_generation_marker` 和正数的 `error_after` 均为必填项。

- `progress_log_glob` 在同步任务活动时定位最新的传输进度文件。
- `success_event` 指定用于查找最新远端快照数量的事件 token；设置后，该 token 必须存在于某个事件源中。
- `snapshot_count_key` 指定承载该数量的数字型事件键。

如果两个代次标记都存在，采集器会精确比较它们脱敏后的值。只有源标记时，认为同步落后。未运行的同步任务在落后期间为 `warning`，超过 `sync.error_after` 后变为 `error`。

## 恢复字段

设置 `recovery.audit_task_id` 可启用恢复就绪判断。它必须引用一个任务，并且 `success_event` 为必填项。`audit_source_id` 可将该事件限制在一个已配置事件源内。

- `sample_percent` 是 0 到 100 之间的显示元数据；真正的数据抽样仍由审计任务负责。
- `retention.keep_last` 和 `retention.keep_daily` 是非负的显示元数据。面板不会运行 `forget` 或 `prune`。
- 每个 `datasets` 条目需要唯一的 `id` 和 `name`。`summary_file` 指向 Restic JSON Lines 输出。启用恢复功能后，`audit_size_key` 和 `audit_snapshot_key` 为必填项；`minimum_bytes` 是非负的就绪阈值。
- 每个 `checks` 条目需要唯一的 `id` 和浏览器可见的 `name`。这些检查是声明式标签：只有审计事件足够新鲜，并且每个数据集都有有效快照和足够的已验证字节数时，面板才会将它们统一标记为通过。
- `local_semantic_event` 和 `local_semantic_source_id` 可选地表示已经观测到一次独立的本地语义检查。

在所引用审计任务的 `error_after` 阈值内，审计事件被视为新鲜。只有存在新鲜的成功事件并且所有已配置数据集策略均通过时，恢复状态才是 `healthy`；没有成功事件时为 `unknown`，其他情况为 `error`。

## 存储字段

- `zfs_dataset` 启用 `zfs list -Hp -o used,avail,quota`。非 ZFS 主机应留空。
- `warn_percent` 和 `error_percent` 默认分别为 85 和 95，且必须满足 `0 <= warn_percent < error_percent <= 100`。
- `pool_command` 是用于只读存储池健康探测的显式参数数组。设置了 `zfs_dataset` 且该命令为空时，默认使用 `zpool status -x`。
- `pool_healthy_contains` 与标准输出进行不区分大小写的匹配，默认为 `all pools are healthy`。

如果未设置 ZFS 配额，采集器会以 `used + available` 作为显示的总量。

## 依赖项字段

每个依赖项都需要稳定的运行标识 `id`、浏览器可见的 `name` 和一个 `kind`：

- `file` 依赖需要 `path`，只有在路径解析为非空普通文件时才健康。
- `command` 依赖需要显式 `command` 参数数组。退出状态必须为零；非空的 `expected_exact` 和 `expected_contains` 检查也必须通过。执行时不会插入 shell。

`healthy_summary` 和 `error_summary` 是浏览器可见的可信消息。原始文件内容和命令输出绝不会被导出。整个配置都属于可信输入：应由 root 所有，权限设为 `0644` 或更严格，并且绝不能允许 Web 用户或备份任务修改。

## 事件源字段

`events.max_items` 默认为 50，且不能超过 500。每个事件源都需要唯一的 `id`、显示用 `name`、`glob` 和 `event_labels` 映射。每个事件 token 对应一个可信 `title`，以及 `info`、`success`、`warning` 或 `error` 之一的严重级别。

同步或恢复配置所引用的 token 必须出现在至少一个事件源中。恢复配置所引用的事件源 ID 也必须存在。

## 阶段规则字段

每条阶段规则需要任务 `task_id`、非空的 `contains_all` 列表和可信 `label`。采集器读取 `ps -eo args=`，当某一进程行包含全部已配置子字符串时选择该标签。进程参数本身绝不会被导出。请使用足够具体的匹配词，避免意外命中。

## 标记文件契约

成功标记和代次标记文件应包含一个 RFC 3339 时间戳：

```text
2026-09-01T09:59:42+08:00
```

为了精确比较代次，本地备份应写入 `sync.source_generation_marker`，异地核对成功后再把完全相同的值复制到 `sync.uploaded_generation_marker`。标记也可以包含不透明标识符；采集器会比较单向指纹，并且只公开短指纹。符号链接形式的标记文件会被拒绝。

持续时间标记文件可以包含整数秒、Go 持续时间或 `duration_seconds=N`：

```text
83
```

只有在对应操作及其所需校验都成功后才能写入标记。备份脚本应使用原子重命名，避免采集器读到不完整的标记。

## Restic 摘要文件

`recovery.datasets[].summary_file` 接受 Restic JSON Lines 输出。采集器只投影以下摘要字段：

- `snapshot_id`（截断为最多 12 个安全字符）
- `total_bytes_processed`
- `total_files_processed`
- `files_new`
- `files_changed`

源路径和错误消息绝不会被投影。

## 脱敏事件格式

每个已配置事件日志每行记录一个事件：

```text
2026-09-01T10:03:18+08:00 offsite_sync_success duration_seconds=124 snapshots=81 rc=0
```

时间戳必须采用 RFC 3339。第二个字段必须存在于该事件源的 `event_labels` 映射中。事件标题和严重级别来自可信配置，而不是日志行。浏览器可见的详情仅限数字型键 `duration_seconds`、`snapshots` 和 `rc`。

成功的审计事件还可携带策略引擎内部使用的各数据集数值：

```text
2026-09-01T05:22:01+08:00 restore_audit_success snapshots=81 documents_bytes=202513437719 documents_snapshot=103db44fd829 photos_bytes=278158956721 photos_snapshot=135cc9f5a77a
```

只有在所有已配置恢复检查都通过后，才应发出成功事件。面板将新鲜的成功事件、每个数据集的 `minimum_bytes` 和快照 ID 共同作为恢复就绪信号。审计脚本没有测试的语义，面板无法自行证明。

## 进度输入

已配置的同步任务活动时，采集器会检查与 `progress_log_glob` 匹配的最新文件。它支持标准 rclone 传输统计行，也支持不含路径的 JSON 行：

```json
{"event":"dashboard_progress","transferred_bytes":1073741824,"total_bytes":2147483648,"percent":50,"speed_bytes_per_second":12582912,"eta_seconds":85}
```

可见阶段标签来自可信任务配置或已配置的 `phase_rule`；传输日志中的任意文本不会被公开。

## 采集频率

浏览器每 15 秒刷新一次，但只显示最近生成的 JSON。项目提供的定时器大约每分钟重新生成一次 JSON。这样既能将宿主机命令负载保持在极低水平，又能让状态变化在约 0–75 秒内可见。备份和同步调度与此相互独立。

## 输入与执行限制

- 标记和持续时间文件必须是普通、非符号链接文件，且不超过 4 KiB。
- 事件、进度和 Restic 摘要读取器只接受普通文件，最多检查末尾 16 MiB，并使用 1 MiB 的扫描行长度上限。
- 最多 5,000 条已解析事件记录参与策略判断；最多导出 `events.max_items` 条。
- 每个已配置或内置命令的超时时间为 15 秒。超过 4 MiB 的标准输出会被拒绝，标准错误会被丢弃。
- HTTP 服务接受最大 4 MiB 的状态文档，要求 `schema_version` 为 1，拒绝未知字段，并将早于 `serve --max-status-age` 的数据视为未就绪。
