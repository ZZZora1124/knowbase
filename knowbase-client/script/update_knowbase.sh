#!/bin/bash
# -*- coding: utf-8 -*-
# update knowbase script

# 脚本使用说明
# 依赖环境
# 1. bcecmd安装，配置好key
# 2. 脚本机器: work@bjyz-agile-web01.bjyz.baidu.com
# 3. 脚本目录: /home/work/hk/knowledge/upgrade
#
# 1. 更新knowbase测试服务
#    使用测试版本升级knowbase测试服务
#    bash update_knowbase.sh --operate=upgrade --service=knowbase --version=123456
#    使用正式版本升级knowbase测试服务
#    bash update_knowbase.sh --operate=upgrade --service=knowbase --version=1.0.0.0
#
# 2. 更新knowbase线上服务
#    使用正式版本升级knowbase线上服务
#    bash update_knowbase.sh --operate=upgrade --service=knowbase --version=1.0.0.0 --release


set -e

# Global configuration
IREPO_TOKEN="xxx"
OPERATE_LIST=("upgrade")
SERVICE_LIST=("knowbase")

# Variables
OPERATE=""
SERVICE=""
VERSION=""
FORCE=0
RELEASE=0

# Function: print usage
usage() {
    echo "update knowbase"
    echo "Usage: bash update_knowbase.sh <option>"
    echo "Available options:"
    echo "  --operate  :  support < upgrade >"
    echo "  --service  :  service name, support < knowbase >"
    echo "  --version  :  the artifact version"
    echo "  --force    :  force to prepare service artifact package"
    echo "  --release  :  whether upgrade online service"
    echo ""
    echo "  ===> upgrade knowbase:"
    echo "  1. upgrade knowbase test service with test version:"
    echo "     bash update_knowbase.sh --operate=upgrade --service=knowbase --version=123456"
    echo "  2. upgrade knowbase test service with released version:"
    echo "     bash update_knowbase.sh --operate=upgrade --service=knowbase --version=1.0.0.0"
    echo "  3. upgrade knowbase online service with released version:"
    echo "     bash update_knowbase.sh --operate=upgrade --service=knowbase --version=1.0.0.0 --release"
}

# Function: get file md5
get_file_md5() {
    local file_path="$1"
    if [[ ! -f "$file_path" ]]; then
        echo "-1"
        return
    fi

    if [[ "$OSTYPE" == "darwin"* ]]; then
        md5 -q "$file_path"
    else
        md5sum "$file_path" | awk '{print $1}'
    fi
}

# Function: get artifact url
get_artifact_url() {
    local repo="$1"
    local version="$2"
    local artifact_url=""

    if [[ "$version" == *"."* ]]; then
        artifact_url="https://irepo.baidu-int.com/rest/prod/v3/${repo}/releases/${version}/files"
    else
        artifact_url="https://irepo.baidu-int.com/rest/prod/v3/${repo}/nodes/${version}/files"
    fi

    echo "$artifact_url"
}

# Function: check artifact validity
check_artifact_validity() {
    local service="$1"
    local version="$2"
    local token=""
    local repo=""

    if [[ "${service^^}" == "KNOWBASE" ]]; then
        token="$IREPO_TOKEN"
        repo="baidu/devops-ai/knowbase-client"
    else
        echo "[ERROR] does not support service[$service]"
        return 1
    fi

    local artifact_url=$(get_artifact_url "$repo" "$version")
    # Remove /files suffix
    artifact_url="${artifact_url%/files}"

    # Check whether artifact exists
    local check_cmd="curl -k -L -H 'IREPO-TOKEN:${token}' ${artifact_url}"
    echo "$check_cmd"

    local result=$(eval "$check_cmd" 2>&1)
    if [[ $? -ne 0 ]] || [[ ! "$result" =~ "$version" ]]; then
        echo "[ERROR] Not found artifact with command $check_cmd, error:"
        echo "$result"
        return 1
    fi

    return 0
}

# Function: prepare knowbase package
prepare_knowbase_package() {
    local version="$1"
    local release="$2"

    echo "===> Start to download artifact ..."
    local artifact_url=$(get_artifact_url "baidu/devops-ai/knowbase-client" "$version")

    local dst_dir="knowbase_prod"
    local download_cmd="rm -rf ${dst_dir} && mkdir -p ${dst_dir} && cd ${dst_dir} && wget -nv -O output.tar.gz --no-check-certificate --header \"IREPO-TOKEN:${IREPO_TOKEN}\" ${artifact_url}"

    echo "$download_cmd"
    if ! eval "$download_cmd"; then
        echo "[ERROR] Fail to download artifact"
        return 1
    fi

    echo "===> Start to decompress artifact ..."
    local decompress_cmd="tar -zxf output.tar.gz"
    if ! eval "$decompress_cmd"; then
        echo "[ERROR] Fail to decompress artifact"
        return 1
    fi

    local output_dir="$./output"
    if [[ ! -d "$output_dir" ]]; then
        echo "[ERROR] output directory not found: $output_dir"
        return 1
    fi

    # Define os-arch combinations
    local platforms=(
        "linux:amd64"
        "linux:arm64"
        "darwin:amd64"
        "darwin:arm64"
        "windows:amd64"
        "windows:arm64"
    )

    for platform in "${platforms[@]}"; do
        local os_name="${platform%%:*}"
        local arch="${platform##*:}"

        echo "===> Start to generate knowbase-${os_name}-${arch}-${version}.tar.bz2 ..."

        local tmp_dir="${dst_dir}/tmp_${os_name}_${arch}"
        local pkg_name="knowbase-${os_name}-${arch}-${version}.tar.bz2"

        # Create temporary directory structure
        rm -rf "$tmp_dir"
        mkdir -p "$tmp_dir"

        # Copy bin directory with the appropriate binary
        local bin_src="${output_dir}/bin/knowbase-${os_name}-${arch}"
        if [[ ! -f "$bin_src" ]]; then
            echo "[WARNING] Binary not found: $bin_src, skipping this platform"
            continue
        fi

        local bin_dst="${tmp_dir}/bin"
        mkdir -p "$bin_dst"
        cp "$bin_src" "${bin_dst}/knowbase"

        # Test mode: create .test file
        if [[ "$release" -eq 0 ]]; then
            touch "${tmp_dir}/.test"
        fi

        # Create version file
        echo "$version" > "${tmp_dir}/version"

        # Create tar.bz2 package
        rm -f "$pkg_name"
        tar -cjf "$pkg_name" -C "$tmp_dir" .

        if [[ $? -ne 0 ]]; then
            echo "[ERROR] Fail to generate $pkg_name"
            return 1
        fi

        echo "===> Successfully generated $pkg_name"
    done

    return 0
}

# Function: update knowbase
update_knowbase() {
    local version="$1"
    local release="$2"
    local force="$3"

    if [[ "$release" -eq 1 ]]; then
        echo "===> Start to get current version of knowbase (online service)..."
    else
        echo "===> Start to get current version of knowbase (test service)..."
    fi

    local version_file="knowbase_latest_version.txt"
    local package_root_url=""
    if [[ "$release" -eq 1 ]]; then
        package_root_url="https://knowbase-client.bj.bcebos.com/knowbase"
    else
        package_root_url="https://knowbase-client.bj.bcebos.com/knowbase-test"
    fi

    local cmd="wget -O ${version_file} ${package_root_url}/${version_file} --no-check-certificate"
    local current_version=""
    if ! eval "$cmd" 2>/dev/null; then
        echo "[WARNING] Fail to download knowbase version file (maybe first deployment)"
    else
        current_version=$(head -n 1 "$version_file")
    fi

    echo "current version: ${current_version:-N/A}"
    echo "target  version: $version"

    if [[ "$version" == "$current_version" ]]; then
        echo "Current version is the same as target version, it needn't to update"
        return 0
    fi

    local str_check=""
    if [[ "$force" -eq 0 ]]; then
        read -p "Are you sure to update knowbase version from [${current_version:-N/A}] to [$version] ? (y/yes/n/no): " str_check
    else
        str_check="yes"
    fi

    if [[ "${str_check^^}" == "YES" ]] || [[ "${str_check^^}" == "Y" ]]; then
        if ! prepare_knowbase_package "$version" "$release"; then
            return 1
        fi

        echo "===> Start to generate $version_file ..."
        echo "$version" > "$version_file"

        # Collect all package files
        local platforms=(
            "linux:amd64"
            "linux:arm64"
            "darwin:amd64"
            "darwin:arm64"
            "windows:amd64"
            "windows:arm64"
        )

        local upload_file_list=()
        for platform in "${platforms[@]}"; do
            local os_name="${platform%%:*}"
            local arch="${platform##*:}"
            local pkg_name="knowbase-${os_name}-${arch}-${version}.tar.bz2"
            if [[ -f "$pkg_name" ]]; then
                upload_file_list+=("$pkg_name")
            fi
        done
        upload_file_list+=("$version_file")

        echo "===> Start to upload knowbase package to bos ... "
        echo "**********************************************************************"
        echo "The following file will be uploaded:"
        for item in "${upload_file_list[@]}"; do
            echo "  $item"
        done
        echo "**********************************************************************"

        local bos_dir=""
        if [[ "$release" -eq 1 ]]; then
            bos_dir="knowbase"
        else
            bos_dir="knowbase-test"
        fi

        # Upload files to BOS using bcecmd
        local bucket="knowbase-client"
        for upload_file in "${upload_file_list[@]}"; do
            if [[ ! -f "$upload_file" ]]; then
                echo "[WARNING] $upload_file doesn't exist"
                continue
            fi

            local bos_path="bos://${bucket}/${bos_dir}/$(basename "$upload_file")"
            echo "Uploading $upload_file to $bos_path ..."

            if ! bcecmd bos cp "$upload_file" "$bos_path"; then
                echo "[ERROR] Failed to upload $upload_file to BOS"
                return 1
            fi

            # Verify MD5 (optional, bcecmd might handle this automatically)
            echo "Successfully uploaded $upload_file"
        done

        if [[ "$release" -eq 1 ]]; then
            echo "===> Update [online service] knowbase version from [${current_version:-N/A}] to [$version] successfully ^_^"
        else
            echo "===> Update [test service] knowbase version from [${current_version:-N/A}] to [$version] successfully ^_^"
        fi
        return 0
    else
        echo "===> you give up updating!"
        return 1
    fi
}

# Parse command line arguments
parse_opts() {
    for arg in "$@"; do
        case "$arg" in
            --operate=*)
                OPERATE="${arg#*=}"
                ;;
            --service=*)
                SERVICE="${arg#*=}"
                ;;
            --version=*)
                VERSION="${arg#*=}"
                ;;
            --force)
                FORCE=1
                ;;
            --release)
                RELEASE=1
                ;;
            -h|--help|help)
                usage
                exit 0
                ;;
            *)
                echo "[ERROR] Unknown option: $arg"
                usage
                exit 1
                ;;
        esac
    done

    # Validate options
    if [[ ! " ${OPERATE_LIST[@]} " =~ " ${OPERATE} " ]]; then
        echo "[ERROR] option --operate [$OPERATE] is illegal, just support ${OPERATE_LIST[*]}"
        return 1
    fi

    if [[ ! " ${SERVICE_LIST[@]} " =~ " ${SERVICE} " ]]; then
        echo "[ERROR] option --service [$SERVICE] is illegal, just support ${SERVICE_LIST[*]}"
        return 1
    fi

    if [[ -z "$VERSION" ]]; then
        echo "[ERROR] --version couldn't be empty"
        return 1
    fi

    return 0
}

# Main
main() {
    if [[ $# -lt 1 ]]; then
        usage
        exit 1
    fi

    if ! parse_opts "$@"; then
        exit 1
    fi

    local ret=1
    if [[ "$OPERATE" == "upgrade" ]] && [[ "$SERVICE" == "knowbase" ]]; then
        if check_artifact_validity "$SERVICE" "$VERSION"; then
            update_knowbase "$VERSION" "$RELEASE" "$FORCE"
            ret=$?
        fi
    fi

    exit $ret
}

# Run main function
main "$@"
