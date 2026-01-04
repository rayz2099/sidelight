# SideLight

> **AI-Powered Professional Color Grading & Framing Tool**
>
> SideLight 是一个专为摄影师和视觉创作者设计的智能命令行工具 (CLI)。它利用 Google Gemini 的多模态视觉能力深度分析照片内容，生成电影级的调色参数（XMP Sidecar），并提供大师级的艺术边框生成功能。

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org)

---

## 📸 效果演示

**智能调色 (AI Grading)**

SideLight 能精准还原场景光影，赋予照片电影质感。

<p align="center">
  <img src="docs/images/input-1.jpg" width="45%" alt="Original RAW Preview (Flat/Log)" />
  <img src="docs/images/output-1.jpg" width="45%" alt="SideLight Processed (Cinematic)" />
</p>

> *左：原始 RAW 预览 (Flat/Log) | 右：SideLight 自动调色 (Cinematic Style)*

**艺术边框 (Art Frames)**

一键生成带 EXIF 参数的精美展示图。

<p align="center">
  <img src="docs/images/output-1_framed.jpg" width="80%" alt="Modern Glass Frame Style" />
</p>

---

## 🌟 核心功能

### 1. AI 智能调色 (`grade`)

不再依赖固定预设。SideLight 会像专业调色师一样"看懂"你的照片——识别光影、情绪、场景和主体——并据此生成独一无二的调色参数。

* **非破坏性工作流**：仅生成 Adobe 兼容的 `.xmp` 副档文件，**绝不修改**原始 RAW 文件。
* **全格式支持**：完美支持 Sony ARW, Canon CR3, Nikon NEF 等 RAW 格式，以及 JPG/PNG 标准图片（自动嵌入元数据）。
* **自然语言控制**：支持使用自然语言（如"更温暖一点"、"像Wes Anderson电影"）微调 AI 的创作。

### 2. 智能艺术边框 (`frame`)

一键生成带有 EXIF 拍摄参数的精美展示图，瞬间提升作品格调。

* **智能布局**：自动提取镜头、焦段、ISO、快门等信息并排版。
* **多样风格**：内置 25+ 种精心设计的边框风格（Leica红标、拍立得、毛玻璃、极简画廊等）。
* **批量处理**：极速并发处理整个文件夹。

---

## 🛠️ 安装指南

### 前置依赖

SideLight 依赖 **ExifTool** 进行元数据读写。请确保系统中已安装：

* **macOS**: `brew install exiftool`
* **Linux**: `sudo apt-get install libimage-exiftool-perl`
* **Windows**: 下载 `exiftool.exe` 并添加至系统 PATH。

### 编译安装

```bash
git clone https://github.com/rayz2099/sidelight.git
cd sidelight

# 编译二进制文件 (输出至 ./bin/sidelight)
go build -o bin/sidelight ./cmd/sidelight

# (可选) 将其移动到系统路径以便全局调用
sudo mv bin/sidelight /usr/local/bin/
```

### 配置 API Key

使用 AI 调色功能需要 Google Gemini API Key（相框功能不需要）。
[点击申请免费 Key](https://aistudio.google.com/)

```bash
# 临时生效
export GEMINI_API_KEY="your_api_key_here"

# 或写入配置文件 (推荐)
mkdir -p ~/.config/sidelight
echo '{"gemini_api_key": "your_api_key_here"}' > ~/.config/sidelight/config.json
```

---

## 📖 使用指南

### 🎨 AI 调色 (Grade)

分析图片并生成 Lightroom/Camera Raw 可读的 XMP 调色数据。

**基本用法**:

```bash
sidelight grade [文件或目录...] [flags]
```

**常用选项**:

* `-s, --style <name>`: 指定调色风格 (默认 "natural")。
* `-p, --prompt <text>`: 给 AI 的额外自然语言指令。
* `-j, --concurrency <int>`: 并发处理数量 (默认 4)。

**示例**:

```bash
# 1. 基础调色 (自然风格)
sidelight grade photo.ARW

# 2. 指定风格 (例如: 胶片感)
sidelight grade photo.ARW --style film

# 3. 自定义指令 (例如: 想要赛博朋克感的夜晚)
sidelight grade night_street.jpg --style cyberpunk --prompt "强调霓虹灯的反射，增加对比度"

# 4. 批量处理整个目录
sidelight grade ./vacation_photos/ --style kodak
```

> **注意 (JPG/PNG 用户)**: 对于非 RAW 格式，SideLight 会自动将 XMP 元数据**嵌入**到图片文件中，以确保 Lightroom 能正确读取。请留意文件修改时间。

👉 **[查看完整调色风格列表 (Grade Styles)](docs/grade.md)**

---

## 🖼️ 艺术相框 (Frame)

为图片添加带有 EXIF 信息的高级边框。

**基本用法**:

```bash
sidelight frame [文件或目录...] [flags]
```

**常用选项**:

* `-s, --style <name>`: 边框风格 (默认 "M1-Simple-White")。
* `-f, --format <jpg|png>`: 输出格式 (默认 jpg)。
* `-q, --quality <int>`: JPG 输出质量 (1-100, 默认 90)。
* `-o, --output <dir>`: 输出目录 (默认在原图同级目录的 `output` 文件夹)。

**示例**:

```bash
# 1. 使用默认白边框
sidelight frame photo.jpg

# 2. 使用"现代毛玻璃"风格
sidelight frame photo.jpg --style Modern-Glass

# 3. 批量处理并指定输出目录
sidelight frame ./selection/ --style F1-Polaroid-Classic --output ./framed_results/
```

👉 **[查看完整相框风格列表 (Frame Styles)](docs/frame.md)**

---

## 常见问题 (FAQ)

**Q: 为什么 Lightroom 看不到 JPG 的调色结果？**
A: Lightroom 默认忽略 JPG 的 XMP Sidecar 文件。SideLight v2.0+ 已升级为**自动嵌入元数据**模式。如果您遇到问题，请在 Lightroom 中右键图片 -> "元数据" -> "从文件读取元数据"。

**Q: 处理 RAW 文件安全吗？**
A: 绝对安全。对于 RAW 文件（ARW, CR3 等），SideLight 严格遵循**只读**原则，仅生成 `.xmp` 副档文件，绝不修改原始数据。

**Q: 支持哪些相机？**
A: 只要 ExifTool 支持的相机型号（覆盖市面 99% 机型），SideLight 均可读取并正确生成边框参数。

---

## License

MIT © 2025 linran
