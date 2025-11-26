#!/usr/bin/env bash
# Waffle build script for cross-platform compilation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="waffle"
MAIN_PATH="./cmd/waffle"
BUILD_DIR="build"
DIST_DIR="dist"

# Version information
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "none")}
DATE=${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -s -w"

# Platforms to build
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Functions
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo "=========================================="
    echo "$1"
    echo "=========================================="
    echo ""
}

build_current_platform() {
    print_header "Building for Current Platform"
    
    print_info "Version: ${VERSION}"
    print_info "Commit:  ${COMMIT}"
    print_info "Date:    ${DATE}"
    echo ""
    
    mkdir -p "${BUILD_DIR}"
    
    print_info "Building ${BINARY_NAME}..."
    go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}" "${MAIN_PATH}"
    
    if [ $? -eq 0 ]; then
        print_info "Build successful: ${BUILD_DIR}/${BINARY_NAME}"
        
        # Display binary info
        if [ -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
            SIZE=$(du -h "${BUILD_DIR}/${BINARY_NAME}" | cut -f1)
            print_info "Binary size: ${SIZE}"
        fi
    else
        print_error "Build failed"
        exit 1
    fi
}

build_all_platforms() {
    print_header "Building for All Platforms"
    
    print_info "Version: ${VERSION}"
    print_info "Commit:  ${COMMIT}"
    print_info "Date:    ${DATE}"
    echo ""
    
    mkdir -p "${DIST_DIR}"
    
    local success_count=0
    local fail_count=0
    
    for platform in "${PLATFORMS[@]}"; do
        IFS='/' read -r -a parts <<< "$platform"
        GOOS="${parts[0]}"
        GOARCH="${parts[1]}"
        
        output_name="${DIST_DIR}/${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}"
        
        if [ "$GOOS" = "windows" ]; then
            output_name="${output_name}.exe"
        fi
        
        print_info "Building for ${GOOS}/${GOARCH}..."
        
        if GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "${LDFLAGS}" -o "${output_name}" "${MAIN_PATH}"; then
            if [ "$GOOS" != "windows" ]; then
                chmod +x "${output_name}"
            fi
            
            SIZE=$(du -h "${output_name}" | cut -f1)
            print_info "  ✓ ${output_name} (${SIZE})"
            ((success_count++))
        else
            print_error "  ✗ Failed to build for ${GOOS}/${GOARCH}"
            ((fail_count++))
        fi
    done
    
    echo ""
    print_info "Build summary: ${success_count} successful, ${fail_count} failed"
    
    if [ $fail_count -gt 0 ]; then
        exit 1
    fi
}

create_archives() {
    print_header "Creating Release Archives"
    
    if [ ! -d "${DIST_DIR}" ]; then
        print_error "Distribution directory not found. Run build first."
        exit 1
    fi
    
    mkdir -p "${DIST_DIR}/archives"
    
    for binary in "${DIST_DIR}/${BINARY_NAME}"-*; do
        if [ -f "$binary" ]; then
            basename=$(basename "$binary")
            
            if [[ "$basename" == *"windows"* ]]; then
                # Create ZIP for Windows
                archive="${DIST_DIR}/archives/${basename}.zip"
                print_info "Creating ${archive}..."
                zip -j "$archive" "$binary" README.md config.example.yaml > /dev/null
            else
                # Create tar.gz for Unix-like systems
                archive="${DIST_DIR}/archives/${basename}.tar.gz"
                print_info "Creating ${archive}..."
                tar -czf "$archive" -C "${DIST_DIR}" "$(basename $binary)" -C .. README.md config.example.yaml
            fi
            
            if [ $? -eq 0 ]; then
                SIZE=$(du -h "$archive" | cut -f1)
                print_info "  ✓ Created (${SIZE})"
            else
                print_error "  ✗ Failed to create archive"
            fi
        fi
    done
}

generate_checksums() {
    print_header "Generating Checksums"
    
    if [ ! -d "${DIST_DIR}/archives" ]; then
        print_error "Archives directory not found. Run create_archives first."
        exit 1
    fi
    
    cd "${DIST_DIR}/archives"
    
    print_info "Generating SHA256 checksums..."
    sha256sum * > SHA256SUMS 2>/dev/null || shasum -a 256 * > SHA256SUMS
    
    if [ $? -eq 0 ]; then
        print_info "Checksums saved to ${DIST_DIR}/archives/SHA256SUMS"
        echo ""
        cat SHA256SUMS
    else
        print_error "Failed to generate checksums"
        exit 1
    fi
    
    cd - > /dev/null
}

clean() {
    print_header "Cleaning Build Artifacts"
    
    if [ -d "${BUILD_DIR}" ]; then
        print_info "Removing ${BUILD_DIR}/"
        rm -rf "${BUILD_DIR}"
    fi
    
    if [ -d "${DIST_DIR}" ]; then
        print_info "Removing ${DIST_DIR}/"
        rm -rf "${DIST_DIR}"
    fi
    
    if [ -f "${BINARY_NAME}" ]; then
        print_info "Removing ${BINARY_NAME}"
        rm -f "${BINARY_NAME}"
    fi
    
    print_info "Clean complete"
}

show_help() {
    cat << EOF
Waffle Build Script

Usage: $0 [command]

Commands:
    build           Build for current platform (default)
    all             Build for all platforms
    release         Build all platforms and create archives
    archives        Create release archives from existing binaries
    checksums       Generate checksums for archives
    clean           Remove build artifacts
    help            Show this help message

Environment Variables:
    VERSION         Version string (default: git describe)
    COMMIT          Git commit hash (default: git rev-parse)
    DATE            Build date (default: current UTC time)

Examples:
    $0 build
    $0 all
    VERSION=1.0.0 $0 release
    $0 clean

EOF
}

# Main
main() {
    local command=${1:-build}
    
    case "$command" in
        build)
            build_current_platform
            ;;
        all)
            build_all_platforms
            ;;
        release)
            clean
            build_all_platforms
            create_archives
            generate_checksums
            ;;
        archives)
            create_archives
            ;;
        checksums)
            generate_checksums
            ;;
        clean)
            clean
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main
main "$@"
