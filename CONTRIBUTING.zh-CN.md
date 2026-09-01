# 贡献指南

[English](CONTRIBUTING.md)

感谢你帮助改进 Restic Backup Dashboard。

## 开发

环境要求：Go 1.26 或更高版本。本项目有意不引入第三方 Go 依赖或前端运行时依赖。

```bash
make check
make test
make demo
```

启动演示后，访问 <http://127.0.0.1:8080/?demo=1>。

## Pull Request

- 保持采集器与 Web 服务之间的权限隔离。
- 不要把仓库地址、源路径、命令输出、凭据或原始日志文本加入公开状态结构。
- 为解析逻辑、状态判断和 HTTP 行为添加聚焦的测试。
- 提交前运行 `gofmt`、`go vet`、`go test -race ./...` 和 `node --check web/static/app.js`。
- 同时更新面向用户的英文和中文字符串。

如果改动涉及较大的状态结构或架构调整，请先创建 issue 讨论，再投入实现工作。

## 文档

面向用户的 Markdown 采用英文文件和配对的 `.zh-CN.md` 文件维护。请在同一个 Pull Request 中更新两种语言，并保持标题、示例和链接的结构一致。命令名称、JSON 字段、状态值、文件路径和指标名称在翻译中保持不变。

代码和测试是运行行为的事实来源。行为发生变化时，请同步更新相关 README、配置、架构、安全说明和示例内容的中英文版本。

## 安全问题

不要为漏洞或意外泄露的秘密信息创建公开 issue。请遵循[安全策略](SECURITY.zh-CN.md)（[English](SECURITY.md)）。
