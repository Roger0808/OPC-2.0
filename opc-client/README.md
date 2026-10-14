# OPC Client

这是 `opc` 分支上的独立云端桌面客户端。首次复制基线为 `8dc7f014b`；原生 `cmd/desktop` 和 `frontend` 保持原样。复制来源和每个文件的 SHA-256 保存在 `upstream.json`，所有客户端修改都在本目录内。

## 当前行为

- 沿用原生 Wails 桌面窗口、拖拽、主题和 Vue 界面，前端随应用打包。默认服务端为 `https://admin.tokendance.sh`。
- 知识库、文档、聊天、用户、空间、模型、技能与远程沙箱继续使用服务端 API；不启动本机 Gin 业务服务、不初始化 SQLite、管理员、JWT 签名密钥、MinIO 或数据库迁移。
- 本机仅运行随机端口的 `127.0.0.1` 转发器，承担 HTTP、SSE、WebSocket 传输；不接受 LAN 连接，不提供任意地址代理。每次启动重新生成访问能力，校验页面来源和 Host，能力不会传给云端。
- 支持现有账号密码登录、刷新登录状态、空间请求头和受保护文件鉴权。企业微信/其他网页登录回桌面的 SSO **本期不接入**，客户端不展示该登录入口。服务器自己的网页登录和 SSO 不受影响。
- 登录凭据不写 WebView localStorage：Mac 使用系统钥匙串，Windows 使用当前用户 DPAPI 加密文件；系统存储失败会提示，本次会话保留内存登录状态，不降级为明文保存。退出清除写入无敏感信息的标记，避免旧凭据在存储故障后被恢复。
- 主题、字体、语言、面板偏好继续本地保存；WebView 数据按服务端隔离，已有用户偏好再按用户 ID 隔离。切换服务器需要重开客户端，避免旧请求流入新服务器。
- 文档、产物、FAQ 导出通过本机保存对话框下载；小型生成文件走原生保存，大文件优先由 Go 流式下载。保留四类文件作用域，禁止带凭据跳到其他服务器。
- 浏览器扩展配对使用云端公开地址。MCP OAuth 使用现有服务端授权状态轮询，不依赖系统浏览器返回的 popup 对象。
- 本期不实现云端调用本机命令、工作目录、本机 Ollama 或本机 MCP 执行。远程沙箱继续取决于服务端能力。本机目录选择入口会明确拒绝，不能把客户端存在误认为服务端具备本机沙箱能力。
- OPC 更新源尚未配置，自动检查为空操作，手动检查显示说明；不会访问原生 WeKnora 的发布源。

## 目录

| 路径 | 用途 |
| --- | --- |
| `desktop/` | 从原生桌面复制的 Wails 壳、独立启动、窗口、连接配置和文件保存 |
| `frontend/` | 从原生前端复制的本地 UI；`src/opc/` 为新增运行配置、传输、凭据适配与连接设置 |
| `internal/cloud/` | 地址验证、本机转发、SSE/WS、下载 |
| `internal/securestore/` | Mac Keychain / Windows DPAPI；不依赖业务数据库 |
| `cmd/dev-relay/` | 浏览器开发和冒烟测试专用转发器 |
| `scripts/` | 构建、嵌入前端、原生源码保护检查和同步补丁 |
| `docs/` | 架构、范围和验证记录 |

## 构建

需要 Go 1.26、Node/npm、Wails 2.12.0；Windows 安装包额外需要 NSIS。Mac 构建须在 Mac 上运行。所有命令在本目录执行。

```bash
bash scripts/build.sh darwin/arm64
bash scripts/build.sh windows/amd64
```

脚本先在本目录安装锁定的前端依赖，构建并嵌入资源，再用独立 Go module 构建。可通过 `WAILS_BIN` 指定 Wails CLI；构建产物位于 `desktop/build/bin`，不提交生成资源、依赖或程序包。Windows 和 Mac 客户端 CPU 架构与云端容器架构相互独立。

默认链接使用 HTTPS。界面右下角或应用菜单的“服务端连接”可以检测当前连接、保存下次启动的服务器地址。这里应填写公开前端/API 共用的**域名根地址**，不带 `/api/v1`。连接地址不会变更任何云端配置。

本地开发示例（仅显式开发允许私有 IP/loopback HTTP）：

```bash
# 终端 1：本目录
go run ./cmd/dev-relay -origin http://127.0.0.1:8080 -allow-http -config .runtime.json
# 终端 2：本目录
cd frontend && npm run dev -- --port 5174 --strictPort
```

浏览器开发入口使用内存凭据，刷新页面需要重新登录；这与原生版的系统凭据恢复有意不同。`.runtime.json` 含开发转发能力，已忽略，停止开发转发器时会删除。不要把该文件或能力发送给其他人。

原生开发可在启动进程环境中设置 `OPC_CLIENT_SERVER_URL`；HTTP 另需 `OPC_CLIENT_ALLOW_HTTP=1`。`OPC_CLIENT_CONFIG_DIR` 用于隔离 QA 配置；`OPC_CLIENT_DEV_ORIGIN` 只在需要显式允许开发浏览器来源时使用。正式分发不设置这些开发选项。

## 验证

```bash
go test -race ./...
go vet ./...
cd frontend
npm run type-check
npm run test:opc
npm run build
```

Mac 系统凭据集成检查可显式运行 `OPC_TEST_KEYCHAIN=1 go test ./internal/securestore -v`，创建并删除独立的 QA 项目，不读取真实账号凭据。浏览器登录检查见 `scripts/frontend-smoke.cjs`，需要通过 `OPC_SMOKE_ACCOUNT_FILE` 指定私有本地测试账号文件，不能提交该文件。

## 同步原生更新

```bash
python3 scripts/check-upstream.py
python3 scripts/upstream-patch.py <已存在的原生分支或提交>
```

第二条命令只生成映射到 `opc-client` 的补丁，不自动应用。检查 `sync-patches/` 中的补丁，用 `git apply --check` 检查冲突，再人工合并。桌面启动、凭据、传输和更新源是有意保留的 OPC 差异，不能直接用原生版本覆盖。同步后，将 `upstream.json` 更新为所选原生提交和对应原生文件哈希；不要把 OPC 修改后的哈希记成原生基线。

`THIRD_PARTY_NOTICES.md` 和 `licenses/` 保留复制基线的声明，其中部分声明覆盖原生完整应用的额外依赖，不表示本云端客户端会启动那些组件。
