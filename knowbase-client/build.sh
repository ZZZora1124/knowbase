#!/bin/bash

#================================================================
# Build Script for knowbase-client
# Description: Cross-platform build script supporting multiple architectures
# Reference: https://ppmoon.github.io/post/golang/golang-cross-compile/
#================================================================

set -e  # Exit on error
set -o pipefail  # Exit on pipe failure

# Error handler
error_exit() {
    echo ""
    echo "  [ERROR] Build failed at line $1"
    echo ""
    exit 1
}

# Trap errors
trap 'error_exit $LINENO' ERR

#================================================================
# Configuration
#================================================================

# Build configuration
readonly PROJECT_NAME="knowbase"
readonly OUTPUT_DIR="output"
readonly BIN_DIR="${OUTPUT_DIR}/bin"
readonly LDFLAGS="-s -w"

# Target platforms
readonly PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
)

#================================================================
# Functions
#================================================================

# Print section header
print_header() {
    echo ""
    echo "========================================="
    echo "$1"
    echo "========================================="
}

# Print info message
print_info() {
    echo "  [INFO] $1"
}

# Print success message
print_success() {
    echo "  [SUCCESS] $1"
}

# Print error message
print_error() {
    echo "  [ERROR] $1" >&2
}

# Configure Go environment
configure_go_env() {
    print_header "Configuring Go Environment"

    # Force use local Go toolchain, disable auto-download
    export GOTOOLCHAIN=local

    # Check if go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go first."
        return 1
    fi

    print_info "Go version: $(go version)"

    go env -w GO111MODULE=on || return 1
    go env -w GONOPROXY=\*\*.baidu.com\*\* || return 1
    go env -w GOPROXY=https://goproxy.baidu-int.com || return 1
    go env -w GONOSUMDB=\* || return 1
    unset GOCACHE
    go env -w GOCACHE=$HOME/.cache/go-build || return 1

    print_info "GOPATH      : $(go env GOPATH)"
    print_info "GOCACHE     : $(go env GOCACHE)"
    print_info "GO111MODULE : $(go env GO111MODULE)"
    print_info "GOPROXY     : $(go env GOPROXY)"
    print_info "GOTOOLCHAIN : $(go env GOTOOLCHAIN)"

    print_success "Go environment configured"
}

# Prepare output directory
prepare_output() {
    print_header "Preparing Output Directory"

    if [ -d "${OUTPUT_DIR}" ]; then
        print_info "Cleaning existing output directory..."
        rm -rf "${OUTPUT_DIR}" || {
            print_error "Failed to remove existing output directory"
            return 1
        }
    fi

    mkdir -p "${BIN_DIR}" || {
        print_error "Failed to create directory: ${BIN_DIR}"
        return 1
    }
    print_info "Created directory: ${BIN_DIR}"

    print_success "Output directory prepared"
}

# Build for a specific platform
build_platform() {
    local goos=$1
    local goarch=$2
    local output_name="${PROJECT_NAME}-${goos}-${goarch}"

    # # Add .exe suffix for Windows
    # if [ "$goos" = "windows" ]; then
    #     output_name="${output_name}.exe"
    # fi

    local output_path="${BIN_DIR}/${output_name}"

    print_info "Building ${goos}/${goarch}..."

    if CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -ldflags "${LDFLAGS}" \
        -o "${output_path}"; then

        local file_size=$(du -h "${output_path}" | cut -f1)
        print_success "Built ${output_name} (${file_size})"
        return 0
    else
        print_error "Failed to build ${goos}/${goarch}"
        return 1
    fi
}

# Build all platforms
build_all() {
    print_header "Building for Multiple Architectures"

    local success_count=0
    local fail_count=0
    local start_time=$(date +%s)

    for platform in "${PLATFORMS[@]}"; do
        local goos=${platform%/*}
        local goarch=${platform#*/}

        if build_platform "$goos" "$goarch"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    print_header "Build Summary"
    print_info "Total platforms: ${#PLATFORMS[@]}"
    print_info "Success: ${success_count}"
    print_info "Failed: ${fail_count}"
    print_info "Duration: ${duration}s"

    if [ $fail_count -gt 0 ]; then
        print_error "Some builds failed!"
        return 1
    fi

    print_success "All builds completed successfully!"
}

# Show build results
show_results() {
    print_header "Build Results"

    if [ ! -d "${OUTPUT_DIR}" ]; then
        print_error "Output directory not found"
        return 1
    fi

    # Use tree command if available
    if command -v tree &> /dev/null; then
        tree -L 2 -h "${OUTPUT_DIR}"
    else
        print_info "tree command not available, using ls instead"
        echo ""
        find "${OUTPUT_DIR}" -type f -exec ls -lh {} \; | awk '{print $9, "("$5")"}'
    fi
}

#================================================================
# Main
#================================================================

main() {
    configure_go_env || exit 1
    prepare_output || exit 1
    build_all || exit 1
    show_results || exit 1

    echo ""
    print_success "Build process completed!"
    echo ""
}

# Run main function
main "$@"

