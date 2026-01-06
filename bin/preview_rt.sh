#!/bin/bash
set -e

# Config
RT_CLI="/Users/linran/Downloads/RawTherapee_macOS_15.4_Universal_5.12_folder/rawtherapee-cli"
SIDELIGHT_BIN="./bin/sidelight"

# Args
IMAGE=$1
STYLE=${2:-natural}
PROMPT=${3:-}

if [ -z "$IMAGE" ]; then
  echo "Usage: $0 <image_path> [style] [prompt]"
  exit 1
fi

if [ ! -f "$IMAGE" ]; then
  echo "Error: Image '$IMAGE' not found."
  exit 1
fi

if [ ! -f "$SIDELIGHT_BIN" ]; then
  echo "Error: sidelight binary not found. Run 'just build' first."
  exit 1
fi

echo "🎨 Grading '$IMAGE' with style '$STYLE' (Format: PP3)..."
if [ -n "$PROMPT" ]; then
  echo "📝 Custom prompt: $PROMPT"
  $SIDELIGHT_BIN grade "$IMAGE" --style "$STYLE" --format pp3 --prompt "$PROMPT"
else
  $SIDELIGHT_BIN grade "$IMAGE" --style "$STYLE" --format pp3
fi

echo "📸 Rendering with RawTherapee CLI..."
# Construct output filename: [path/to/filename]_[style]_rt.jpg
BASE_NAME="${IMAGE%.*}"
OUTPUT="${BASE_NAME}_${STYLE}_rt.jpg"

# -j100 = JPEG quality 100%, -Y = overwrite, -s = use sidecar pp3
"$RT_CLI" -o "$OUTPUT" -j100 -s -Y -c "$IMAGE"

if [ -f "$OUTPUT" ]; then
    echo "✅ Done! Preview saved to: $OUTPUT"
    # Try to open on macOS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        open "$OUTPUT"
    fi
else
    echo "❌ Error: Output file was not generated."
    exit 1
fi
