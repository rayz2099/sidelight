# SideLight 💡

> **AI-Powered RAW Image Color Grading Tool**
>
> SideLight 是一个专为摄影师打造的智能命令行工具。它利用 Google Gemini 的视觉能力分析 RAW 照片，生成专业级的调色参数，并输出为 Adobe 兼容的 XMP Sidecar 文件。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/linran/sidelight)](https://goreportcard.com/report/github.com/linran/sidelight)

## ✨ 核心特性

- 🛡️ **非破坏性编辑**：仅生成 `.xmp` 文件，**绝不修改**原始 RAW 文件。
- 🎨 **风格化调色**：内置多种风格预设（胶片、黑白、电影感等），并支持自然语言微调。
- 📷 **广泛支持**：兼容 Sony ARW, Nikon NEF, Canon CR3, Fuji RAF 等所有主流 RAW 格式。
- ⚡ **极速处理**：并发架构 + 智能预览提取，无需上传庞大的 RAW 文件。
- 🔧 **工作流友好**：生成的 XMP 可被 Lightroom / Camera Raw 自动识别读取。

## 🛠️ 安装

### 依赖

请确保系统已安装以下工具：

1.  **ExifTool** (必须): 用于提取 RAW 预览图。
    *   macOS: `brew install exiftool`
    *   Linux: `sudo apt-get install libimage-exiftool-perl`
2.  **Just** (可选): 方便的命令运行工具。
    *   macOS: `brew install just`

### 从源码编译

```bash
git clone https://github.com/linran/sidelight.git
cd sidelight

# 编译 (产物在 bin/sidelight)
just build

# 安装到 $GOPATH/bin
just install
```

## 🚀 快速开始

### 1. 配置 API Key

SideLight 需要 Google Gemini API Key 才能工作。[点击这里申请免费 Key](https://aistudio.google.com/)。

```bash
export GEMINI_API_KEY="你的_API_KEY_粘贴在这里"
```

### 2. 基础用法

处理单个文件或整个文件夹：

```bash
sidelight images/raw/DSC_001.ARW
# 或者
sidelight images/raw/
```

### 3. 进阶调色 (Styles & Prompts)

SideLight 不仅仅是自动曝光，你还可以告诉 AI 你想要的风格：

**使用预设风格 (`--style` / `-s`)：**

可选值：`natural` (默认), `cinematic`, `film`, `bw` (黑白), `portrait`.

```bash
# 电影感
sidelight -s cinematic images/raw/
```

**自然语言微调 (`--prompt` / `-p`)：**

你可以用自然语言进一步描述你的意图：

```bash
# 胶片感，但希望更暖一些
sidelight -s film -p "Make it warmer, golden hour vibe" images/raw/

# 黑白，高对比度
sidelight -s bw -p "High contrast, dramatic shadows" images/raw/
```

**并发控制 (`--concurrency` / `-j`)：**

```bash
# 同时处理 8 张照片
sidelight -j 8 images/raw/
```

##  workflow: Lightroom 配合指南

1.  运行 `sidelight` 处理你的 RAW 文件夹。
2.  **场景 A (未导入)**: 直接将文件夹导入 Lightroom，调色会自动应用。
3.  **场景 B (已导入)**:
    *   在 Lightroom 选中照片。
    *   右键 -> **元数据** -> **从文件中读取元数据**。
    *   或者使用快捷键: `Cmd + Option + Shift + R` (Mac)。

## 📝 参数列表

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--style` | `-s` | 调色风格 (natural, cinematic, film, bw, portrait) | `natural` |
| `--prompt` | `-p` | 自定义微调指令 (英文描述效果最佳) | `""` |
| `--concurrency` | `-j` | 并发处理线程数 | `4` |
| `--api-key` | | Gemini API Key (推荐使用环境变量) | |

## 🏗️ 架构

*   **Language**: Go 1.22+
*   **CLI**: Cobra + Viper
*   **Imaging**: ExifTool (Wrapper)
*   **AI**: Google Gemini Pro Vision

## 📄 License

MIT © 2025 linran