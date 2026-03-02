# Gemini Image Web App

基于 Gemini 模型的 AI 图片生成 Web 可视化界面，Go 后端 + 纯 HTML/CSS/JS 前端，无需任何前端框架。

## 目录结构

```
webapp/
├── main.go              # Go 后端服务器
├── go.mod               # Go 模块文件
├── go.sum               # 依赖校验文件
├── gemini-webapp.exe    # 编译后的可执行文件（Windows）
├── start.ps1            # Windows PowerShell 启动脚本
├── README.md            # 本文件
└── static/
    ├── index.html       # 前端页面（三栏布局）
    └── outputs/         # 图片自动保存目录（按日期子目录）
        └── YYYYMMDD/
            └── *.png
```

## 快速开始

### 1. 设置 API Key（必须）

API Key **仅**从环境变量读取，不存储在任何配置文件中。

**Windows cmd.exe（不加引号）：**
```cmd
set GEMINI_API_KEY=your-api-key-here
```

**Windows PowerShell：**
```powershell
$env:GEMINI_API_KEY = "your-api-key-here"
```

**Linux / macOS：**
```bash
export GEMINI_API_KEY="your-api-key-here"
```

### 2. 启动服务

#### Windows

**方式一：直接运行 exe（推荐）**
```cmd
set GEMINI_API_KEY=your-api-key
gemini-webapp.exe
```

**方式二：使用 PowerShell 启动脚本**
```powershell
.\start.ps1
```
若未设置环境变量，脚本会提示手动输入。

**方式三：从源码运行**
```powershell
$env:GEMINI_API_KEY = "your-api-key"
go run main.go
```

#### Linux / macOS

**方式一：编译后运行**
```bash
cd /path/to/webapp
go build -o gemini-webapp .
export GEMINI_API_KEY="your-api-key"
./gemini-webapp
```

**方式二：直接 go run**
```bash
export GEMINI_API_KEY="your-api-key"
go run main.go
```

**方式三：一行启动（临时设置环境变量）**
```bash
GEMINI_API_KEY="your-api-key" ./gemini-webapp
```

**方式四：添加到 shell 配置（永久生效）**
```bash
# 写入 ~/.bashrc 或 ~/.zshrc
echo 'export GEMINI_API_KEY="your-api-key"' >> ~/.bashrc
source ~/.bashrc
./gemini-webapp
```

**后台运行（可选）：**
```bash
export GEMINI_API_KEY="your-api-key"
nohup ./gemini-webapp > webapp.log 2>&1 &
echo "服务已在后台启动，日志输出到 webapp.log"
# 停止服务
# kill $(lsof -ti:8080)
```

### 3. 访问界面

打开浏览器访问：**http://localhost:8080**

---

## 界面说明

界面分为三栏：

```
┌─────────┬──────────────────┬──────────────────────────┐
│ 📁 图片库 │   配置 / 生成     │      预览区               │
│         │                  │                          │
│ 按日期   │ Prompt           │  图片全尺寸自适应显示        │
│ 分组缩略图│ 模型 / 宽高比    │  点击 → Lightbox 全屏      │
│         │ 尺寸 / 前缀       │                          │
│ 单击预览 │ [🎨 生成图片]     │  工具栏: 全屏预览 / 下载    │
│ 双击全屏 │ 状态 / 计时       │                          │
└─────────┴──────────────────┴──────────────────────────┘
```

### 左侧图片库
- 自动按日期分组显示所有历史图片缩略图
- **单击**缩略图：在右侧预览区显示
- **双击**缩略图：打开 Lightbox 全屏查看
- 点击 ↻ 按钮手动刷新图片列表

### 中间配置面板

| 设置项 | 说明 |
|--------|------|
| **Prompt** | 图片描述文字，支持多行 |
| **API Base URL** | 兼容 OpenAI 协议的 API Endpoint，页面加载时自动填入默认值，可随时修改切换到其他服务商 |
| **Model** | 选择 Gemini 图片生成模型 |
| **Aspect Ratio** | 点击按钮选择宽高比 |
| **Image Size** | 选择分辨率 1K / 2K / 4K |
| **文件名前缀** | 自定义保存文件名前缀 |

快捷键：`Ctrl+Enter` 快速提交生成

### 右侧预览区
- 全尺寸自适应显示当前图片
- 点击图片或工具栏「⛶ 全屏预览」打开 Lightbox
- 工具栏「⬇ 下载」直接下载当前图片

### Lightbox 全屏查看
- `←` `→` 方向键或屏幕箭头翻页浏览所有图片
- `Esc` 或点击背景关闭
- 底部显示文件名、大小、当前序号

---

## 图片保存规则

图片自动保存到 `static/outputs/YYYYMMDD/` 目录，无需手动配置保存路径。

**文件名格式：**
```
{prefix}_{model_short}_{aspect_ratio}_{image_size}_{timestamp}.png

示例：
gemini_generated_g31f_9x16_2K_20260302_153000.png
```

**模型简称对照：**

| 模型 | 简称 |
|------|------|
| `gemini-3.1-flash-image-preview` | g31f（默认） |
| `gemini-2.5-flash-image` | g25f |
| `gemini-3-pro-image-preview` | g3p |

---

## 支持的参数

**宽高比：**
- 标准（所有模型）：`1:1` `2:3` `3:2` `3:4` `4:3` `4:5` `5:4` `9:16` `16:9` `21:9`
- 扩展（仅 gemini-3.1-flash-image-preview）：`1:4` `1:8` `4:1` `8:1`

**图片尺寸：** `1K` `2K` `4K`

---

## API 接口

### `POST /api/generate` — 生成图片

请求体（JSON）：
```json
{
  "prompt": "A beautiful sunset over mountains",
  "model": "gemini-3.1-flash-image-preview",
  "aspect_ratio": "16:9",
  "image_size": "2K",
  "prefix": "gemini_generated"
}
```

响应体（JSON）：
```json
{
  "success": true,
  "message": "图片生成并保存成功",
  "image_url": "/outputs/20260302/gemini_generated_g31f_16x9_2K_20260302_153000.png",
  "file_name": "gemini_generated_g31f_16x9_2K_20260302_153000.png",
  "file_path": "static\\outputs\\20260302\\gemini_generated_g31f_16x9_2K_20260302_153000.png"
}
```

### `GET /api/gallery` — 获取图片库列表

响应体（JSON，按日期降序分组）：
```json
[
  {
    "date": "20260302",
    "images": [
      {
        "url": "/outputs/20260302/gemini_generated_g31f_16x9_2K_20260302_153000.png",
        "file_name": "gemini_generated_g31f_16x9_2K_20260302_153000.png",
        "date": "20260302",
        "size": 2048576
      }
    ]
  }
]
```

### `GET /api/config` — 获取配置信息

返回支持的模型、比例、尺寸等。

---

