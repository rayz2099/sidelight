#!/bin/bash

# SideLight Style Comparison Script
# Quick comparison of key styles for tuning

set -e

# Color output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Key styles for comparison
KEY_STYLES=("natural" "cinematic" "teal-orange" "fuji" "film" "cyberpunk")

usage() {
    echo -e "${BLUE}SideLight Quick Style Comparison${NC}"
    echo ""
    echo "Usage: $0 <input_file> [prompt]"
    echo ""
    echo "Generates XMP and PP3 for key styles: ${KEY_STYLES[*]}"
}

if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

INPUT_FILE="$1"
PROMPT="$2"
OUTPUT_DIR="comparison_$(basename ${INPUT_FILE%.*})"

echo -e "${BLUE}🔍 Quick Style Comparison${NC}"
echo -e "Input: ${GREEN}${INPUT_FILE}${NC}"
echo -e "Output: ${GREEN}${OUTPUT_DIR}${NC}"
echo ""

mkdir -p "$OUTPUT_DIR"

for style in "${KEY_STYLES[@]}"; do
    echo -e "${YELLOW}Generating ${style}...${NC}"

    # XMP
    cmd="sidelight grade -s $style $INPUT_FILE -o $OUTPUT_DIR/$style"
    [[ -n "$PROMPT" ]] && cmd="$cmd -p \"$PROMPT\""
    eval $cmd > /dev/null 2>&1

    # PP3
    cmd="sidelight grade -s $style $INPUT_FILE -f rt -o $OUTPUT_DIR/$style"
    [[ -n "$PROMPT" ]] && cmd="$cmd -p \"$PROMPT\""
    eval $cmd > /dev/null 2>&1

    echo -e "${GREEN}✓ $style complete${NC}"
done

echo ""
echo -e "${BLUE}📊 Generated comparison files:${NC}"
ls -la "$OUTPUT_DIR"
echo ""
echo -e "${YELLOW}Compare XMP exposure settings:${NC}"
echo "grep 'crs:Exposure2012' $OUTPUT_DIR/*.xmp"
echo ""
echo -e "${YELLOW}Compare PP3 exposure settings:${NC}"
echo "grep 'Compensation=' $OUTPUT_DIR/*.pp3"