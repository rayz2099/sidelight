#!/bin/bash

# SideLight Style Generator Script
# Generates XMP and PP3 files for all available styles

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Available styles from the codebase
STYLES=(
    "natural"
    "cinematic"
    "teal-orange"
    "cyberpunk"
    "film"
    "kodak"
    "fuji"
    "polaroid"
    "retro-70s"
    "matte"
    "dreamy"
    "wes-anderson"
    "vivid"
    "bw"
    "bw-contrast"
    "moody"
    "bright"
    "warm"
    "cool"
    "vintage"
    "modern"
    "soft"
    "dramatic"
    "pastel"
    "neon"
)

# Function to display usage
usage() {
    echo -e "${BLUE}SideLight Style Generator${NC}"
    echo ""
    echo "Usage: $0 <input_file> [output_directory] [custom_prompt]"
    echo ""
    echo "Arguments:"
    echo "  input_file        Input RAW/JPG/PNG file"
    echo "  output_directory  Directory to save generated files (default: styles_output)"
    echo "  custom_prompt     Optional custom prompt for AI analysis"
    echo ""
    echo "Examples:"
    echo "  $0 test.arw"
    echo "  $0 test.arw ./my_styles"
    echo "  $0 test.arw ./my_styles 'warm golden hour lighting'"
    echo ""
    echo -e "${GREEN}Available styles (${#STYLES[@]} total):${NC}"
    printf '  %s\n' "${STYLES[@]}"
}

# Function to generate style files
generate_style() {
    local style="$1"
    local input_file="$2"
    local output_dir="$3"
    local prompt="$4"
    local base_name="${input_file%.*}"

    echo -e "${YELLOW}Generating ${style} style...${NC}"

    # Generate XMP file
    echo -e "  ${BLUE}→${NC} Creating XMP configuration"
    local xmp_cmd="sidelight grade -s ${style} ${input_file}"
    if [[ -n "$prompt" ]]; then
        xmp_cmd="${xmp_cmd} -p \"${prompt}\""
    fi
    xmp_cmd="${xmp_cmd} -o ${output_dir}/${style}/${base_name}-${style}"

    eval $xmp_cmd > /dev/null 2>&1

    # Generate PP3 file
    echo -e "  ${BLUE}→${NC} Creating PP3 configuration"
    local pp3_cmd="sidelight grade -s ${style} ${input_file} -f rt"
    if [[ -n "$prompt" ]]; then
        pp3_cmd="${pp3_cmd} -p \"${prompt}\""
    fi
    pp3_cmd="${pp3_cmd} -o ${output_dir}/${style}/${base_name}-${style}"

    eval $pp3_cmd > /dev/null 2>&1

    echo -e "  ${GREEN}✓${NC} Generated: ${style}.{xmp,pp3}"
}

# Main script
main() {
    # Check arguments
    if [[ $# -lt 1 ]]; then
        usage
        exit 1
    fi

    local input_file="$1"
    local output_dir="${2:-styles_output}"
    local custom_prompt="$3"

    # Check if input file exists
    if [[ ! -f "$input_file" ]]; then
        echo -e "${RED}Error: Input file '$input_file' not found${NC}"
        exit 1
    fi

    # Check if sidelight binary exists
    local sidelight_cmd
    if [[ -f "./sidelight" ]]; then
        sidelight_cmd="./sidelight"
    elif command -v sidelight &> /dev/null; then
        sidelight_cmd="sidelight"
    else
        echo -e "${RED}Error: sidelight binary not found${NC}"
        echo "Make sure it's in current directory or PATH"
        exit 1
    fi

    echo -e "${BLUE}🎨 SideLight Style Generator${NC}"
    echo -e "${BLUE}=========================================${NC}"
    echo -e "Input file: ${GREEN}${input_file}${NC}"
    echo -e "Output directory: ${GREEN}${output_dir}${NC}"
    [[ -n "$custom_prompt" ]] && echo -e "Custom prompt: ${GREEN}${custom_prompt}${NC}"
    echo -e "Styles to generate: ${GREEN}${#STYLES[@]}${NC}"
    echo ""

    # Create output directory structure
    echo -e "${YELLOW}Creating output directories...${NC}"
    for style in "${STYLES[@]}"; do
        mkdir -p "${output_dir}/${style}"
    done
    echo -e "${GREEN}✓${NC} Created ${#STYLES[@]} style directories"
    echo ""

    # Generate files for each style
    local start_time=$(date +%s)
    local current=0
    local total=${#STYLES[@]}

    for style in "${STYLES[@]}"; do
        current=$((current + 1))
        echo -e "${BLUE}[${current}/${total}]${NC} Processing style: ${style}"

        if generate_style "$style" "$input_file" "$output_dir" "$custom_prompt"; then
            echo -e "${GREEN}✓${NC} Completed: $style"
        else
            echo -e "${RED}✗${NC} Failed: $style"
        fi
        echo ""
    done

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Summary
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${GREEN}🎉 Generation Complete!${NC}"
    echo -e "Time taken: ${duration} seconds"
    echo -e "Generated files in: ${GREEN}${output_dir}${NC}"
    echo ""
    echo -e "${YELLOW}Output structure:${NC}"
    echo -e "${output_dir}/"
    local count=0
    for style in "${STYLES[@]}"; do
        if [[ $count -lt 3 ]]; then
            echo -e "├── ${style}/"
            echo -e "│   ├── $(basename ${input_file%.*})-${style}.xmp"
            echo -e "│   └── $(basename ${input_file%.*})-${style}.pp3"
            count=$((count + 1))
        else
            break
        fi
    done
    [[ ${#STYLES[@]} -gt 3 ]] && echo -e "└── ... (${#STYLES[@]} styles total)"
    echo ""

    # Comparison commands
    echo -e "${YELLOW}💡 Useful commands for comparison:${NC}"
    echo -e "# Compare XMP files:"
    echo -e "  diff ${output_dir}/natural/*.xmp ${output_dir}/cinematic/*.xmp"
    echo ""
    echo -e "# Compare PP3 files:"
    echo -e "  diff ${output_dir}/natural/*.pp3 ${output_dir}/cinematic/*.pp3"
    echo ""
    echo -e "# View specific style:"
    echo -e "  cat ${output_dir}/fuji/*.pp3"
}

# Handle Ctrl+C
trap 'echo -e "\n${RED}Generation interrupted${NC}"; exit 1' INT

# Run main function
main "$@"