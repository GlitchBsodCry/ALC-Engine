# ALC Engine - AI与文件协同工作平台

## 📋 项目概述

ALC Engine (Ai Assistant Lite Cloud Engine) 是一个集成了AI智能对话与虚拟文件系统的协同工作平台。项目聚焦于两大核心模块：

- 🤖 **AI智能聊天系统** - 支持多模型、流式响应、会话管理
- 📁 **虚拟文件夹系统** - 实现文件管理、云存储集成、多人协作

## 🎯 核心功能

### 虚拟文件夹系统 

#### 文件空间

每个项目都有一个独立的根目录，可以对虚拟文件夹进行存储

虚拟文件夹之间除了文件夹上的嵌套关系，还可以是调用关系

**我推荐的虚拟文件夹关系**

- 通过调用大宗虚拟文件夹，实现对一类文件资源的快速使用和关联

![dazong](image/dazong.png)

- 通过占位虚拟文件夹，代表逻辑上未开发的文件

![placeholder](image/placeholder.png)

- 通过调用实体虚拟文件夹，实现某类资源集合的多次重复使用

![leg](image/leg.png)

- 建议虚拟文件夹多表示一类概念，并整理子概念的关系

![spider](image/spider.png)

***

### 多人协作

#### 项目内的权限内容

对于一个项目，存在owner、admin、member、viewer、ban从高到低五种权限类型

其中，member和admin要在本地进行虚拟文件夹的编辑，进行一次性的提交，再经过其他admin的批准后真正编辑到项目里。

对于云文件的上传也同样。

![morepeople1](image/morepeople1.png)



可以把文件挂载到某虚拟文件夹上
通过虚拟文件夹的自带的逻辑性质代表此文件在开发中的逻辑性质

通过文件的上传、更新、同步等，实现多人的逻辑同步

![cloudfile](image/cloudfile.260407.png)

### AI聊天模块

- 🤖 **多模型支持** - OpenAI、RAG、MCP、Ollama 等多种模型类型
- 💬 **会话管理** - 完整的会话创建、继续、历史记录管理
- 🖼️ **图像识别** - ONNXRuntime 图像推理与识别
- 📚 **RAG检索增强** - 文档向量化与检索增强生成
- 🔧 **MCP工具调用** - Model Context Protocol 外部工具调用能力
- 🔊 **TTS语音合成** - 百度语音合成集成
- ✅ **流式响应** - 支持实时流式输出

#### 智能文件理解

当用户提问中提到相关文件时，系统会自动根据文件类型采取相应的处理策略，从对象存储中获得文件信息，无需用户额外操作和发送文件即可获得准确的回答。

![提问](image/提问.png)

#### 虚拟文件夹结构理解

系统能够理解虚拟文件夹的嵌套关系和调用关系，通过MCP工具实现对虚拟文件夹结构的智能解析与操作。

![提问2](image/提问2.png)


## 🏗️ 技术架构

### 技术栈

| 类别   | 技术选型                         |
| ---- | ---------------------------- |
| 后端框架 | Go 1.26.0 + Gin v1.10.1      |
| AI集成 | CloudWeGo Eino v0.7.34       |
| 模型支持 | Ollama + OpenAI              |
| 向量检索 | Redis Retriever              |
| 数据库  | PostgreSQL 16 + MySQL (兼容)   |
| 对象存储 | MinIO v7                     |
| 消息队列 | RabbitMQ                     |
| 缓存   | Redis v9                     |
| 配置管理 | Viper v1.21.0                |
| 语音合成 | ONNX Runtime                 |

<br />

## ⚙️ 配置说明

主要配置文件：`config.yaml`

### 关键配置项

```yaml
# AI配置 (核心)
ai:
  model_name: "Qwen/Qwen3-32B"      # 模型名称
  base_url: "https://api.siliconflow.cn/v1"  # API地址
  api_key: "your-api-key"           # API密钥

# 云存储配置 (核心)
minio:
  endpoint: "localhost:9000"        # MinIO地址
  access_key_id: "minioadmin"       # 访问密钥
  secret_access_key: "minioadmin"   # 秘密密钥
  default_bucket: "alc-files"       # 默认存储桶

# 数据库配置
postgresql:
  host: "localhost"
  port: "5432"
  dbname: "alc_engine"

# Redis配置
redis:
  addr: "localhost:6379"
  password: ""
  db: 0

# RabbitMQ配置
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
```

## 📡 API概览

### 认证相关

- `POST /user/register` - 用户注册
- `POST /user/login` - 用户登录
- `POST /user/logincode` - 验证码登录
- `POST /user/code` - 发送登录验证码
- `POST /user/registercode` - 发送注册验证码
- `POST /user/refresh` - Token刷新
- `POST /user/reset-password-code` - 发送找回密码验证码
- `POST /user/reset-password` - 重置密码
- `GET /auth/verify` - 令牌验证
- `POST /auth/logout` - 退出登录

### AI聊天 (核心)

- `POST /chat/create` - 创建新聊天
- `POST /chat/continue` - 继续聊天
- `GET /chat/sessions` - 获取会话列表
- `POST /chat/history` - 获取聊天历史
- `POST /chat/stream/create` - 流式创建新聊天
- `POST /chat/stream/continue` - 流式继续聊天

### AI图像识别

- `POST /ai/image/recognize` - 图像识别

### AI RAG服务

- `POST /ai/rag/upload` - 上传文件并创建索引
- `POST /ai/rag/query` - 查询RAG知识库
- `DELETE /ai/rag/index` - 删除RAG索引

### AI MCP工具调用

- `POST /ai/mcp/weather` - 天气查询
- `POST /ai/mcp/call` - 通用工具调用

### AI TTS语音合成

- `POST /ai/tts/create` - 创建语音合成任务
- `POST /ai/tts/query` - 查询任务状态

### AI智能查询

- `POST /ai/smart/query` - 智能查询
- `POST /ai/smart/process` - 处理MinIO文件

### 项目管理

- `POST /project/create` - 创建新项目
- `GET /project/list` - 获取项目列表

### 项目特定操作 (`/project/:project_id/`)

- **成员管理**
  - `GET /members` - 获取项目成员
  - `POST /members` - 添加项目成员
  - `PUT /members` - 更新成员权限

- **标签管理**
  - `GET /tags` - 获取项目标签
  - `POST /tags` - 创建标签
  - `PUT /tags` - 更新标签
  - `DELETE /tags/:tag_id` - 删除标签

- **虚拟文件夹**
  - `GET /folders/root` - 获取根目录
  - `GET /folders/parent/:parent_id` - 获取子文件夹
  - `POST /folders` - 创建虚拟文件夹
  - `PUT /folders` - 更新虚拟文件夹
  - `DELETE /folders/:folder_id` - 删除文件夹
  - `PUT /folders/move` - 移动文件夹

- **文件管理**
  - `POST /files/login` - 登记文件
  - `POST /files/mount` - 挂载文件
  - `DELETE /files/unmount` - 卸载文件
  - `DELETE /files/logout` - 注销文件
  - `PUT /files/rename` - 重命名文件

- **云文件服务**
  - `GET /cloud/download/:cloud_file_id` - 获取下载URL
  - `POST /cloud/sync` - 确认文件同步
  - `GET /cloud/upload` - 获取上传URL
  - `POST /cloud/upload` - 完成上传

- **变更管理**
  - `POST /changes` - 提交变更请求
  - `GET /changes/pending` - 待审批列表
  - `POST /approve-change` - 审批变更请求

## 🔧 开发指南

### 项目结构

```
ALC Engine/
├── api/                    # API定义层
│   ├── errors/            # 错误定义
│   └── model/             # 数据模型
├── cmd/main/              # 入口点
├── image/                 # 图片资源
├── internal/              # 内部包
│   ├── ai/               # AI模块 (核心)
│   │   ├── image/        # 图像识别(ONNX)
│   │   ├── rag/          # RAG检索增强
│   │   ├── mcp/          # MCP工具调用
│   │   ├── tts/          # 语音合成
│   │   ├── factory.go    # 模型工厂
│   │   ├── manager.go    # AI管理器
│   │   └── model.go      # 基础模型
│   ├── control/          # 控制器层
│   ├── rabbitmq/         # RabbitMQ消息队列
│   ├── repository/       # 数据访问层
│   └── service/          # 业务逻辑层
├── pkg/                   # 公共包
│   ├── config/           # 配置管理
│   ├── interfacer/       # 接口定义
│   ├── logger/           # 日志系统
│   ├── middleware/       # 中间件(认证/限流/CORS)
│   ├── minio/            # MinIO对象存储
│   ├── router/           # 路由定义
│   └── utils/            # 工具函数
├── docker-compose.yml     # Docker配置
├── go.mod                 # Go依赖
└── config.yaml            # 配置文件
```

### 扩展AI模型

1. 在 `internal/ai/` 创建新模型包
2. 实现 `AIModel` 接口
3. 在 `internal/ai/factory.go` 注册模型
4. 更新配置文件支持新模型

### 添加新API

1. 在 `api/model/` 定义请求/响应结构
2. 在 `internal/control/` 创建控制器
3. 在 `internal/service/` 实现业务逻辑
4. 在 `internal/repository/` 添加数据访问
5. 在 `pkg/router/router.go` 注册路由

### 项目权限说明

项目支持五种权限等级（从高到低）：
- **owner** - 项目所有者，拥有全部权限
- **admin** - 管理员，可审批变更、管理成员
- **member** - 成员，可提交变更请求
- **viewer** - 查看者，仅可查看项目内容
- **ban** - 被封禁用户

## 📊 版本说明

### v3.0.0 (当前版本)

- ✅ 权限控制系统
- ✅ 多人协作优化
- ✅ 变更审批流程
- ✅ AI聊天系统完整实现
- ✅ AI图像识别（ONNX）
- ✅ AI RAG检索增强
- ✅ AI MCP工具调用
- ✅ AI TTS语音合成
- ✅ 虚拟文件夹核心功能
- ✅ 云文件存储集成
- ✅ 项目管理基础框架

### 后续规划

- 更好的实时性

<br />

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [CloudWeGo Eino](https://github.com/cloudwego/eino) - AI模型框架
- [Gin](https://github.com/gin-gonic/gin) - Web框架
- [GORM](https://gorm.io/) - ORM库
- [MinIO](https://min.io/) - 对象存储

***

