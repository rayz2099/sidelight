#!/bin/bash
set -e

IMAGE=$1
if [ -z "$IMAGE" ]; then
  IMAGE="images/output/output-1.jpg"
fi

if [ ! -f "$IMAGE" ]; then
  echo "Error: Image file '$IMAGE' not found."
  exit 1
fi

# Ensure sidelight binary exists
if [ ! -f ./bin/sidelight ]; then
  echo "Error: bin/sidelight not found. Run 'just build' first."
  exit 1
fi

echo "🎨 Generating previews for $IMAGE..."

# Get directory and base name
DIR=$(dirname "$IMAGE")
FILENAME=$(basename "$IMAGE")
NAME="${FILENAME%.*}"

# Iterate over all style json files in assets/styles
for style_file in assets/styles/*.json; do
  # Extract style ID (filename without path and extension)
  style_id=$(basename "$style_file" .json)
  
  printf "  Processing style: %-20s ... " "$style_id"
  
  # Run sidelight frame
  # This generates [NAME]_framed.jpg by default
  ./bin/sidelight frame "$IMAGE" --style "$style_id" > /dev/null 2>&1
  
  # Rename the output file to [NAME]_[StyleID].jpg
  # Note: sidelight frame defaults to jpg output unless -f is specified.
  GENERATED_FILE="${DIR}/${NAME}_framed.jpg"
  TARGET_FILE="${DIR}/${NAME}_${style_id}.jpg"
  
  if [ -f "$GENERATED_FILE" ]; then
    mv "$GENERATED_FILE" "$TARGET_FILE"
    echo "✅ Saved to $(basename "$TARGET_FILE")"
  else
    echo "❌ Failed"
  fi
done

echo ""
echo "✨ All done! Check $DIR folder for the results."
