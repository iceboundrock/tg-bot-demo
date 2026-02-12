下面是 **详细的技术规格文档**（Technical Specification）版本，用于你的技术团队参考并实现 **Telegram Bot UI 展示 Session 列表 + 分页 + Web App（Mini-App）** 的功能。内容包括架构、API 约定、交互流程、数据库设计、Bot 逻辑、前后端协作等。

---

# **📘 Telegram AI Bot — 会话列表 UI 技术规格文档（详尽版）**

---

## **1️⃣ 总体目标**

本功能模块需实现：

1. 私聊用户能够：

   * 查看其拥有的 AI 会话列表

   * 点击会话进入（切换当前聊天上下文）

   * 分页显示大量会话

   * 点击“更多”打开更完善的 UI（Mini-App）

2. 采用 Telegram Bot API 提供的 **inline keyboard + callback** 和 **Web App Mini-App** 功能实现可交互界面。

核心技术基础：

✔ Telegram Bot API + CallbackQuery

✔ InlineKeyboardMarkup 多按钮

✔ Telegram Web App （Mini App）入口按钮

📌 支持按钮点击触发会话切换和 UI 分页交互

---

## **2️⃣ 核心概念说明**

### **🎯 会话（Session）**

一个用户的聊天上下文线程，包括历史消息、状态和元数据。

---

### **📌 Telegram 交互组件**

| 组件 | 用途 |
| ----- | ----- |
| Inline Keyboard 按钮 | 展示可点击按钮、发送 callback_data |
| CallbackQuery | 用户点击按钮后 Bot 接收的事件 |
| Web App 按钮 | 打开自定义 Web UI Mini-App |
| 📌 专用于在 Telegram 客户端中展示复杂交互 UI（非消息文本） |  |

---

## **3️⃣ 数据结构设计**

### **🔹 Session 表（数据库）**

| 字段 | 类型 | 说明 |
| ----- | ----- | ----- |
| id | UUID | 主键 |
| user_id | UUID | 所属用户 |
| title | text | 会话标题 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 最近更新时间 |
| last_message | text | 最近消息片段（可选） |

---

## **4️⃣ 会话列表 API 端点定义**

基本查询：

GET /api/sessions?userId=<userId>\&offset=<offset>\&limit=<limit>

返回结构：

{  
  "sessions": [  
    { "id": "uuid1", "title": "写影视剧推荐" },  
    { "id": "uuid2", "title": "学习总结" }  
  ],  
  "total": 132  
}  
---

## **5️⃣ Bot 与用户交互设计**

---

### **5.1 👉 用户请求显示 Session 列表**

#### 用户输入：

`/sessions`

#### Bot 响应：

请选择会话👇

附带按钮：

[ 会话1 ]  
[ 会话2 ]  
...  
[ Prev ] [ Next ]  
---

### **5.2 👉 Inline Keyboard 格式（分页）**

格式示例（JSON）：

```json
{  
  "chat_id": <chatId>,  
  "text": "请选择会话👇",  
  "reply_markup": {  
    "inline_keyboard": [  
      [  
        { "text": "写影视剧推荐", "callback_data": "open_s_uuid1" }  
      ],  
      [  
        { "text": "Prev", "callback_data": "page_sessions_0" },
        { "text": "Next", "callback_data": "page_sessions_12" }  
      ]  
    ]  
  }  
}
```

注意事项：

* callback_data 用于按钮点击事件回传给 Bot

* 数据中应包含会话 ID（如 open_s_xxxx）用于分辨点击内容

---

### **5.3 👉 分页逻辑（Offset + Limit）**

* 每页显示最多 N=6 个

* 如果有上一页，显示 `Prev` 按钮

* 如果有下一页，显示 `Next` 按钮

* 分页按钮发送 callback_data：

page_sessions_<offset>

Bot 解析后加载目标 offset 对应页

---

### **5.4 👉 回调事件处理**

Bot 端收到：

CallbackQuery{ data: "open_s_uuid1" }

逻辑：

if startsWith(data, "open_s_"):  
    sessionID := trimPrefix(data)  
    setUserCurrentSession(userId, sessionID)  
    bot.answerCallbackQuery(...)  
    bot.sendMessage("已切换到会话: <title>")  
---

### **5.5 👉 分页回调处理**

收到：

CallbackQuery{ data: "page_sessions_6" }

逻辑：

1. 解析 offset=6

2. 查询该页 limit=6，并计算是否有上一页/下一页

3. 编辑当前消息按钮区，按需显示 `Prev`/`Next`

可采用 bot.editMessageReplyMarkup 更新按钮

---

## **6️⃣ Web App Mini-App 支持**

当会话数量巨大（如 > 30）或需要更复杂操作（搜索／筛选／删除／编辑）时，采用 Web App UI。

---

### **6.1 👉 Web App 按钮格式**

使用 web_app InlineKeyboardButton：

[  
  {  
    "text": "打开会话列表",  
    "web_app": {  
      "url": "https://yourdomain.com/sessions?userId=<userId>"  
    }  
  }  
]

📌 Telegram 客户端会将此按钮作为 **Web App** 打开页面 。

---

### **6.2 👉 Web App 页面内容需求**

页面应支持：

| 功能 | 说明 |
| ----- | ----- |
| 列出全部会话 | 包含标题、最后更新时间 |
| 搜索功能 | 关键字筛选 |
| 点击进入 | 触发切换当前 Session |
| 删除 / 重命名 | 支持会话管理 |
| 返回 Telegram | 可以调用 Telegram.WebApp.close() |

---

### **6.3 👉 Web App 与 Bot 通信**

两种方式：

#### **🟢 方案 A — 在 Web App 页面自己直接调用服务器 API**

Web App 访问自己的 API 后端获取数据，无需 Bot 参与

#### **🟢 方案 B — 使用** 

#### **Telegram.WebApp.sendData**

Web App JS 调用：

Telegram.WebApp.sendData("open_s_"+sessionId)

Bot 在 Web App 中收到 WebAppData 回传bot端事件，然后处理相应逻辑 。

---

## **7️⃣ UX 交互流程规格 (状态机)**

User: /sessions  
Bot: sendMessage → Inline Keyboard List (SessionL1)

User clicks → CallbackQuery  
 ├ "open_s_xxx" → open that session  
 └ "page_sessions_offset" → paginate

optional:  
Bot sends button:  
 [ 打开完整列表 (WebApp) ]  
User clicks → opens Web App UI  
---

## **8️⃣ 错误处理 & 限制**

### **❗ callback_data 限制**

* 最大长度限制 \~64 bytes

* 不可超长数据 → 只传会话 ID 部分

---

### **❗ Web App 只能在私聊中使用**

如在群聊中发送 Web App 按钮常会失败（Bot API 限制） 。

---

## **9️⃣ 安全与验证**

* 所有 callback_data 操作必须验证 session 属于当前用户

* 防止恶意用户拼接 deep link 访问他人会话

---

## **🔟 Bot API 要点（核心方法）**

| 功能 | API 方法 |
| ----- | ----- |
| 发送文本+按钮 | sendMessage |
| 编辑按钮 | editMessageReplyMarkup |
| 回答回调 | answerCallbackQuery |
| 发送图片等 | sendPhoto |

基础 Bot API 是 HTTP 接口，团队须配置 webhook 或 long-polling 来接收 updates.

---

## **📌 Sample Inline Keyboard Contract**

{  
  "inline_keyboard": [  
    [ { "text": "写影视剧推荐", "callback_data": "open_s_uuid1" } ],  
    [ { "text": "学习总结", "callback_data": "open_s_uuid2" } ],  
    [ { "text": "Prev", "callback_data": "page_sessions_0" }, { "text": "Next", "callback_data": "page_sessions_12" } ],  
    [ { "text": "打开完整列表", "web_app": { "url": "https://yourdomain.com/sessions?userId=xxx" } } ]  
  ]  
}  
---

## **📋 开发/集成 Checklist**

### **✔ 后端**

✅ 会话分页 API

✅ callback_data 解析器

✅ sessionID 权限校验

---

### **✔ Bot 端**

☑ Inline keyboard 构造模块

☑ Callback handler 模块

☑ Web App 按钮支持

---

### **✔ 前端 (Web App)**

☑ 会话列表页面

☑ 搜索筛选 UI

☑ 点击开会话逻辑

☑ sendData / close 控制

---

## **📎 参考链接**

🔗 Telegram Web App Mini-App docs — 说明如何通过按钮打开应用

🔗 Python example shows Web App button usage pattern

🔗 callback_query 回调机制说明（标准 Telegram API）

---

如果团队需要，我也可以提供额外细化的内容：

✅ Go 语言实现示例代码（基于 go-telegram/bot）

✅ 前端 Web App 样板（React/Vue）

✅ 高级会话 UX 规范设计文档

需要哪个补充？
