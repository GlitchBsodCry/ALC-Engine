# ALC Engine - AI与文件协同工作平台

## 📋 项目概述

ALC Engine (Ai Assistant Lite Cloud Engine) 是一个集成了AI智能对话与虚拟文件系统的协同工作平台。项目聚焦于两大核心模块：

- 🤖 **AI智能聊天系统** - 支持多模型、流式响应、会话管理
- 📁 **虚拟文件夹系统** - 实现文件管理、云存储集成、多人协作

## 🎯 核心功能

### AI聊天模块 

- AI 对话引擎集成（多会话 + 流式输出）
- AI 图像模型推理（ONNXRuntime 部署 + 推理加速）
- 正在实现中

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

## 🏗️ 技术架构

### 技术栈

| 类别   | 技术选型                         |
| ---- | ---------------------------- |
| 后端框架 | Go 1.26 + Gin                |
| AI集成 | CloudWeGo Eino + SiliconFlow |
| 数据库  | PostgreSQL (主) + MySQL (兼容)  |
| 对象存储 | MinIO                        |
| 消息队列 | RabbitMQ (基础集成)              |
| 缓存   | Redis (令牌黑名单)                |
| 配置管理 | Viper (YAML配置)               |

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
```

## 📡 API概览

### 认证相关

- `POST /user/login` - 用户登录
- `POST /user/register` - 用户注册
- `POST /auth/verify` - 令牌验证

### AI聊天 (核心)

- `POST /chat/create` - 创建新聊天会话
- `POST /chat/continue` - 继续现有会话
- `POST /chat/stream/create` - 流式创建会话
- `POST /chat/stream/continue` - 流式继续会话
- `GET /chat/sessions` - 获取用户会话列表

### 文件管理 (核心)

- `POST /file/loginfile` - 登记文件（批量）
- `POST /file/newmount` - 文件挂载到文件夹
- `DELETE /file/deletemount` - 解除挂载关系
- `PUT /file/newrename` - 重命名文件
- `DELETE /file/logoutfile` - 登出文件

### 云文件服务

- `GET /file/cloud/upload` - 获取上传URL
- `POST /file/cloud/upload` - 确认上传完成
- `GET /file/cloud/download/:id` - 获取下载URL
- `POST /file/cloud/sync` - 确认同步完成

### 虚拟文件夹

- `POST /virtual/folder` - 创建虚拟文件夹
- `PUT /virtual/folder` - 修改文件夹
- `DELETE /virtual/folder/:id` - 删除文件夹
- `PUT /virtual/folder/move` - 移动文件夹
- `GET /virtual/folder/root/:id` - 获取根目录文件夹
- `GET /virtual/folder/parent/:id` - 获取子文件夹

### 变更管理

- `POST /changes` - 提交批量变更申请
- `GET /changes/pending` - 获取待审批变更列表
- `PUT /changes/approve/:id` - 批准变更申请
- `PUT /changes/reject/:id` - 拒绝变更申请

### 项目成员管理

- `POST /project/:id/member` - 添加项目成员
- `PUT /project/:id/member/:userid` - 更新成员权限
- `DELETE /project/:id/member/:userid` - 移除项目成员
- `GET /project/:id/members` - 获取项目成员列表

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
│   ├── control/          # 控制器层
│   ├── rabbitmq/         # RabbitMQ消息队列
│   ├── repository/       # 数据访问层
│   └── service/          # 业务逻辑层
├── pkg/                   # 公共包
│   ├── config/           # 配置管理
│   ├── interfacer/       # 接口定义
│   ├── logger/           # 日志系统
│   ├── middleware/       # 中间件
│   ├── minio/            # MinIO对象存储
│   ├── router/           # 路由定义
│   └── utils/            # 工具函数
└── config.yaml           # 配置文件
```

### 扩展AI模型

1. 在 `internal/ai/` 添加新模型实现
2. 在 `factory.go` 注册模型创建器
3. 更新配置支持新模型类型

### 添加新API

1. 在 `pkg/router/router.go` 定义路由
2. 在 `internal/control/` 创建控制器
3. 在 `internal/service/` 实现业务逻辑
4. 在 `internal/repository/` 添加数据访问

## 📊 版本说明

### v2.0.0 (当前版本)

- ✅ 权限控制系统
- ✅ 多人协作优化
- ✅ 变更审批流程
- ✅ AI聊天系统完整实现
- ✅ 虚拟文件夹核心功能
- ✅ 云文件存储集成
- ✅ 项目管理基础框架

### 后续规划

- 更丰富的AI工具集成（V3）
- 增强项目实时性（V4）

<br />

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [CloudWeGo Eino](https://github.com/cloudwego/eino) - AI模型框架
- [Gin](https://github.com/gin-gonic/gin) - Web框架
- [GORM](https://gorm.io/) - ORM库
- [MinIO](https://min.io/) - 对象存储

***

