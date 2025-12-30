# SideLight - 智能 RAW 照片调色助手

SideLight 是一个基于 AI 的命令行工具，专为摄影师设计。它能够自动分析 RAW 格式照片（如 .ARW, .NEF, .CR3 等），利用 Google Gemini 的视觉能力生成调色参数，并输出为 Adobe 兼容的 XMP Sidecar 文件。

这意味着你可以在 Lightroom 或 Camera Raw 中直接应用 AI 生成的调色，而无需修改原始 RAW 文件。

## ✨ 功能特点

*   **非破坏性编辑**：只生成 `.xmp` 文件，绝不修改原始 RAW 文件。
*   **广泛的格式支持**：支持 Sony ARW, Nikon NEF, Canon CR2/CR3, Fuji RAF 等所有 ExifTool 支持的格式。
*   **AI 智能调色**：利用 Gemini Pro/Flash Vision 模型分析画面内容（曝光、白平衡、风格），生成自然的调色参数。
*   **批量处理**：支持并发处理整个文件夹的 RAW 文件，并在终端显示进度条。
*   **高效**：仅提取嵌入的 JPEG 预览图进行上传分析，大幅节省流量和时间。

## 🛠️ 准备工作

在开始之前，请确保你的系统已安装以下依赖：

1.  **ExifTool**: 用于从 RAW 文件中提取预览图。
    *   **macOS**: `brew install exiftool`
    *   **Windows**: 下载并安装 [ExifTool](https://exiftool.org/)，确保其在系统 PATH 中。
    *   **Linux**: `sudo apt-get install libimage-exiftool-perl`

2.  **Google Gemini API Key**: 需要一个有效的 API Key。
    *   可在 [Google AI Studio](https://aistudio.google.com/) 免费申请。

## 🚀 安装与构建

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/sidelight.git
cd sidelight
```

### 2. 编译

```bash
go build -o sidelight ./cmd/sidelight
```

## 📖 使用指南

### 1. 设置 API Key

你可以通过环境变量设置 API Key（推荐）：

```bash
export GEMINI_API_KEY="你的_API_KEY_粘贴在这里"
```

或者在运行时通过参数传递：

```bash
./sidelight --api-key "你的_API_KEY" ...
```

### 2. 处理照片

**处理单个文件：**

```bash
./sidelight images/raw/DSC_001.ARW
```

**处理整个文件夹：**

```bash
./sidelight images/raw/
```

**调整并发数量（默认为 4）：**
如果你想加快速度或减少 API 请求频率，可以使用 `-j` 参数：

```bash
./sidelight -j 8 images/raw/
```

### 3. 在 Lightroom 中查看结果

1.  处理完成后，你会发现每个 RAW 文件旁边都有一个同名的 `.xmp` 文件。
2.  打开 Adobe Lightroom Classic 或 Photoshop Camera Raw。
3.  导入该 RAW 文件（或者如果已经在库中，右键 -> 元数据 -> 从文件读取元数据）。
4.  你会看到“基本”面板中的曝光、对比度、高光、阴影等参数已被自动调整。

## ⚙️ 参数说明

| 参数 | 简写 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `--api-key` | 无 | Gemini API Key | 无 (必须) |
| `--concurrency` | `-j` | 并发处理的 worker 数量 | 4 |

## 🏗️ 技术架构

*   **语言**: Go (Golang)
*   **CLI 框架**: Cobra
*   **图像提取**: ExifTool Wrapper
*   **AI 引擎**: Google Gemini (via `google-generative-ai-go`)
*   **并发模型**: Worker Pool pattern

## ⚠️ 免责声明

虽然本工具不会修改您的原始 RAW 文件，但建议在批量操作前备份您的数据。AI 生成的调色结果仅供参考，旨在提供一个良好的修图起点。
