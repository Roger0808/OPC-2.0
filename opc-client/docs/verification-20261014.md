# 2026-10-14 验证记录

本次工作区为 `opc` 分支，复制基线 `8dc7f014b`。这是账号密码登录阶段的实现验证，SSO 按用户选择留到下一步。

## 已完成

| 检查 | 当前结果与证据 |
| --- | --- |
| 原生源码保护 | `scripts/check-upstream.py` 校验复制来源的 914 个文件 SHA-256 全部一致。Git 状态仅新增 `opc-client/`，原生文件没有修改。 |
| 独立业务依赖 | `go list -deps ./desktop` 的 WeKnora 路径仅有 `opc-client` 自己的 cloud、securestore、desktop；没有 Gin、GORM、SQLite 或根业务 container/handler。 |
| Go 检查 | `go test -race ./...` 与 `go vet ./...` 通过。 |
| 传输契约测试 | 固定目标地址和路径验证、Host/Origin/能力校验、CORS 原生来源及 Embed 请求头、四类受保护文件路径、上游状态码/响应头保留、multipart/Embed 上传、SSE 首段即时刷新与取消、WS ticket/能力剥离与关闭、禁止外部下载重定向、批量 POST ZIP 下载专用路径均通过。 |
| Mac 凭据存储 | 显式 `OPC_TEST_KEYCHAIN=1 go test ./internal/securestore -v` 通过，实际创建、读取、清除、重新保存独立 QA 钥匙串项目，最后删除。未读取真实账号凭据。 |
| 前端检查 | `npm run type-check`、`npm run test:opc`、`npm run build` 通过；针对客户端/登录刷新/四类文件权限的 22 项测试通过。构建仍有原生前端的大 chunk 提示，不影响构建成功。未声称运行复制前端的全部历史测试。 |
| 浏览器账号密码冒烟 | 使用已有私有本地测试账号、现有 `127.0.0.1:8080` 后端和新开发入口 `5174`，完成账号密码登录、auth/me、知识库页面、连接检测。观察到 23 次 API 请求全部经过新转发器，页面脚本错误 0；没有 auto-setup，localStorage 没有明文 access/refresh token。开发浏览器刷新后回到登录页（仅内存凭据）。没有创建知识库、对话或远端演示数据。 |
| Mac ARM64 构建 | Wails 生产构建、打包和本地自签名成功；程序为 Mach-O ARM64，前端嵌入应用。 |
| Mac 原生窗口 | 实际启动最终 `.app`，显示账号密码登录入口和连接配置；默认 `https://admin.tokendance.sh`，原生窗口检测明确显示“当前服务端连接正常”。发现并修复 Mac `wails://wails` 的 CORS 来源差异后重新验证通过。未使用测试服真实账号登录。 |
| Windows AMD64 构建 | 独立 Go module 跨编译成功；Wails Windows GUI EXE 和 NSIS 安装包均构建成功。应用 EXE 为 PE32+ x86-64；NSIS 自解压启动壳为正常的 PE32，安装内容为 AMD64 应用。 |
| 本地业务数据库 | 原生测试进程仅监听随机 `127.0.0.1` 端口，对外连接公网 HTTPS；本客户端配置目录没有业务 SQLite 或业务后端配置。WebView 自身的浏览器存储不属于业务数据库。 |

## 程序包

- `desktop/build/bin/OPC Client.app`：Mac ARM64 程序。
- `desktop/build/bin/OPC Client.exe`：Windows AMD64 程序。
- `desktop/build/bin/OPC Client-amd64-installer.exe`：Windows AMD64 安装包。

这些是本地验证构建，目录已被本客户端的 `.gitignore` 忽略。没有发布到 release、阿里云或测试服务器，也没有进行正式签名、公证或客户端自动更新发布。

## 未执行和后续验收

- Windows 实机安装、WebView2、账号登录恢复、DPAPI、系统文件保存、SSE/WS 运行验证。本机没有 Windows 运行环境，跨编译成功不能替代这些检查。
- 测试服真实账号的完整知识库上传、聊天、共享文件、远程沙箱、MCP OAuth、浏览器扩展配对端到端操作。公网连接已验证，功能传输和权限保留有契约测试与本地登录验证；没有把这些证据写成所有云端业务功能均已通过。
- Mac 原生的真实账号密码登录恢复和系统保存对话框完整操作。密码登录 UI 与 API 流程在浏览器开发入口通过，钥匙串在实际 OS 集成测试通过，两者尚未使用真实测试服账号联合验收。
- 企业微信/其他 SSO 回桌面协议、本机命令/目录/Ollama/MCP 执行、正式签名和 OPC 更新源，均不在本期实现范围。

没有修改或部署服务器服务；没有提交或推送 Git 改动。实现、构建和说明均保持在 `opc-client` 内。
