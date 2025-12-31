# SideLight 💡

> **AI-Powered RAW Image Color Grading Tool**
>
> SideLight 是一个专为摄影师打造的智能命令行工具。它利用 Google Gemini 的视觉能力分析 RAW 照片，生成专业级的调色参数，并输出为 Adobe 兼容的 XMP Sidecar 文件。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/linran/sidelight)](https://goreportcard.com/report/github.com/linran/sidelight)

## 📸 效果演示

**Before (Raw) vs After (SideLight AI)**

<p align="center">
  <img src="docs/images/input-1.jpg" width="45%" alt="Original RAW Preview" />
  <img src="docs/images/output-1.jpg" width="45%" alt="SideLight Processed" />
</p>

> *左：原始 RAW 预览 (Flat/Log) | 右：SideLight 自动调色 (Cinematic Style)*

## ✨ 核心特性

- 🛡️ **非破坏性编辑**：仅生成 `.xmp` 文件，**绝不修改**原始 RAW 文件。
- 🎨 **风格化调色**：内置多种风格预设（胶片、黑白、电影感等），并支持自然语言微调。
- 🖼️ **艺术相框**：一键生成带 EXIF 参数的精美边框图，支持20+种大师级设计。
- 📷 **广泛支持**：兼容 Sony ARW, Nikon NEF, Canon CR3, Fuji RAF 等所有主流 RAW 格式。
- ⚡ **极速处理**：并发架构 + 智能预览提取，无需上传庞大的 RAW 文件。
- 🔧 **工作流友好**：生成的 XMP 可被 Lightroom / Camera Raw 自动识别读取。

## 🛠️ 安装

### 依赖

请确保系统已安装以下工具：

1. **ExifTool** (必须): 用于提取 RAW 预览图和元数据。
    * macOS: `brew install exiftool`
    * Linux: `sudo apt-get install libimage-exiftool-perl`
2. **Just** (可选): 方便的命令运行工具。
    * macOS: `brew install just`

### 从源码编译

```bash
git clone https://github.com/rayz2099/sidelight.git
cd sidelight

# 编译 (产物在 bin/sidelight)
just build

# 安装到 $GOPATH/bin
just install
```

## 🚀 快速开始

### 1. 配置 API Key

SideLight 需要 Google Gemini API Key 才能进行 **AI 调色**（加相框不需要）。[点击这里申请免费 Key](https://aistudio.google.com/)。

```bash
export GEMINI_API_KEY="你的_API_KEY_粘贴在这里"
```

### 2. AI 智能调色 (`grade`)

处理单个文件或整个文件夹：

```bash
# 默认自然风格
sidelight grade images/raw/DSC_001.ARW

# 指定电影感风格
sidelight grade images/raw/ -s teal-orange
```

### 3. 生成艺术相框 (`frame`)

无需 API Key，瞬间为你的照片加上带有参数的高级边框。支持 RAW 和 JPG。

```bash
# 默认经典白边
sidelight frame images/raw/photo.jpg

# 使用大师级风格 (例如: 现代毛玻璃)
sidelight frame images/raw/photo.jpg --style Modern-Glass

# 导出为无损 PNG
sidelight frame images/raw/photo.jpg -f png
```

### 4. 快速导出预览 (`export`)

从 RAW 文件中极速提取全尺寸 JPG 预览图。

```bash
sidelight export images/raw/ -q 100
```

---

## 🎨 调色风格指南 (Grading Styles)

使用 `-s` 或 `--style` 参数指定。

| 分类        | 风格代码 (Code)    | 效果特点                     |
|:----------|:---------------|:-------------------------|
| **基础通用**  | `natural`      | **自然 (默认)** 还原肉眼所见，色彩准确。 |
|           | `vivid`        | **鲜艳** 高饱和、高对比，画面通透。     |
|           | `flat`         | **灰片/Log** 极低对比度，适合后期。   |
| **黑白艺术**  | `bw`           | **经典黑白** 标准黑白转换。         |
|           | `bw-contrast`  | **高反差黑白** 森山大道风格，冲击力强。   |
| **胶片模拟**  | `kodak`        | **柯达风格** 经典的金黄色调，肤色讨喜。   |
|           | `fuji`         | **富士风格** 偏冷绿/洋红，适合风光人文。  |
| **电影/艺术** | `teal-orange`  | **青橙色调** 商业大片标配。         |
|           | `cyberpunk`    | **赛博朋克** 霓虹感，适合夜景。       |
|           | `wes-anderson` | **韦斯·安德森** 糖果配色，对称感。     |

*(更多风格请参考源码 `internal/ai/gemini.go`)*

---

## 🖼️ 相框设计廊 (Frame Gallery)

使用 `sidelight frame --style [StyleID]` 指定。

### M 系列：极简主义 (Minimalist)

> *Less is More. 高屏占比，聚焦画面本身。*

| Style ID          | Name                | 描述               |
|:------------------|:--------------------|:-----------------|
| `M1-Simple-White` | **Minimal White**   | 窄白边，经典画廊布局，极简参数。 |
| `M2-Simple-Black` | **Minimal Black**   | 窄黑边，沉浸式暗色模式。     |
| `M3-Pure-Frame`   | **Pure Frame**      | 纯白框，无任何文字干扰。     |
| `M4-Grey-Matte`   | **Museum Grey**     | 博物馆级灰底，优雅中性。     |
| `M5-Bottom-Heavy` | **Bottom Weighted** | 底部略宽，视觉重心下沉。     |

### F 系列：胶片情怀 (Film & Vintage)

> *致敬经典胶片时代，特殊的比例与质感。*

| Style ID              | Name                    | 描述                    |
|:----------------------|:------------------------|:----------------------|
| `F1-Polaroid-Classic` | **Polaroid Classic**    | 经典的暖白宝丽来风格，底部宽大。      |
| `F2-Film-Dark`        | **Dark Slide**          | 模拟底片扫描边框，黄色 KODAK 字体。 |
| `F3-Fuji-Green`       | **Fuji Style**          | 淡绿底色，致敬富士胶片包装。        |
| `F4-Cinema-Wide`      | **Cinematic Letterbox** | 电影宽银幕遮幅，上下黑边。         |
| `F5-Square-Crop`      | **Square Instax**       | 拍立得方形构图感。             |

### G 系列：氛围毛玻璃 (Glass & Blur)

> *现代科技感，背景模糊处理，延展视觉边界。*

| Style ID           | Name            | 描述              |
|:-------------------|:----------------|:----------------|
| `G1-Glass-Deep`    | **Deep Blur**   | 深度模糊背景，主体悬浮感强。  |
| `G2-Glass-Light`   | **Light Blur**  | 轻微模糊，保留环境色块，清新。 |
| `G3-Frost`         | **Frosty**      | 磨砂玻璃质感，高亮高调。    |
| `G4-Vivid-Glass`   | **Vivid Glass** | 鲜艳背景，适合色彩丰富的照片。 |
| `G5-Subtle-Border` | **Subtle Blur** | 极窄的模糊边框，精致细腻。   |

### E 系列：杂志排版 (Editorial)

> *像时尚杂志内页一样展示你的作品。*

| Style ID            | Name                | 描述                 |
|:--------------------|:--------------------|:-------------------|
| `E1-Vogue`          | **Vogue Style**     | 大标题排版，时尚大片感。       |
| `E2-Tech-Spec`      | **Technical Specs** | 详细列出 ISO、光圈、焦段等参数。 |
| `E3-Clean-Date`     | **Clean Date**      | 仅显示日期，日记风格。        |
| `E4-Vertical-Stack` | **Vertical Stack**  | 垂直堆叠信息，现代排版。       |
| `E5-Corner-Data`    | **Corner Data**     | 四角显示信息，取景器风格。      |

### C 系列：创意撞色 (Creative)

> *大胆的配色与设计。*

| Style ID          | Name             | 描述                 |
|:------------------|:-----------------|:-------------------|
| `C1-Leica-Red`    | **Leica Red**    | 黑底红字，致敬可乐标。        |
| `C2-Cyber-Neon`   | **Cyber Neon**   | 赛博朋克霓虹配色 (青/洋红)。   |
| `C3-Gold-Elegant` | **Gold Elegant** | 黑金配色，奢华感。          |
| `C4-Orange-Sony`  | **Alpha Orange** | 索尼橙配色，致敬 Alpha 系列。 |
| `C5-Blueprint`    | **Blueprint**    | 蓝晒图风格，工程美学。        |

## 📄 License

MIT © 2025 linran
