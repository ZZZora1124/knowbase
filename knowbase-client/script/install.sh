#!/bin/bash

SETUP_DIR="$HOME/.knowbase"
SYMLINK_PATH="$SETUP_DIR/knowbase"
BIN_DIR="$SETUP_DIR/knowbase/bin"
KNOWBASE_SCRIPT_PATH="$BIN_DIR/knowbase"
BOS_BUILDKIT_URL='http://knowbase-client.bj.bcebos.com'
RUN_SOURCE_PATH="NO"
LATEST_VERSION_FILE='knowbase_latest_version.txt'
PROFILE=$HOME/.bashrc
[ -e $HOME/.bash_profile ] && PROFILE=$HOME/.bash_profile

OK=0
ERR=1

modify_path()
{   
    # check if macos -> .zshrc
    if [ "$OS_TYPE" = "darwin" ]; then
        PROFILE=$HOME/.zshrc
    fi

    # check PATH configuration
    grep -v "^#" $PROFILE | grep "PATH=$1:\$PATH" >/dev/null 2>&1
    [ $? -eq 0 ] && return $OK

    # if PATH is not set
    echo "PATH=$1:\$PATH" >> $PROFILE
    echo "export PATH" >> $PROFILE
    RUN_SOURCE_PATH="YES"
    return $OK
}

get_system_info()
{
    # Get OS type - simplified to only 3 categories
    OS_TYPE=$(uname | tr '[:lower:]' '[:upper:]')
    case "$OS_TYPE" in
        LINUX)
            OS_TYPE="linux"
            ;;
        DARWIN)
            OS_TYPE="darwin"
            ;;
        WINDOWS* | MINGW* | MSYS* | CYGWIN*)
            OS_TYPE="windows"
            ;;
        *)
            echo "warning: unknown OS type: $OS_TYPE, using linux as default"
            OS_TYPE="linux"
            ;;
    esac

    # Get architecture - simplified to only arm64 and amd64
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64 | amd64)
            ARCH="amd64"
            ;;
        aarch64 | arm64)
            ARCH="arm64"
            ;;
        *)
            echo "warning: unknown architecture: $ARCH, using amd64 as default"
            ARCH="amd64"
            ;;
    esac

    echo "Detected system: OS=$OS_TYPE, ARCH=$ARCH"
}

install_knowbase()
{
    # Get system information first
    get_system_info

    # check $HOME/.knowbase/knowbase/bin
    if [ ! -e $BIN_DIR ]; then
        mkdir -p $BIN_DIR
        if [ $? -ne 0 ]; then
            echo "'mkdir -p $BIN_DIR' for knowbase error!"
            return $ERR
        fi
    fi

    # download knowbase package
    echo "setup knowbase ..."

    # fetch knowbase latest version
    rm -rf $LATEST_VERSION_FILE && \
    wget -q $BOS_BUILDKIT_URL/knowbase/$LATEST_VERSION_FILE >/dev/null
    if [ $? -ne 0 ]; then
        echo "Fail to download latest version file, the cmd is: wget $BOS_BUILDKIT_URL/knowbase/$LATEST_VERSION_FILE"
        return $ERR
    fi

    LATEST_VERSION=`cat $LATEST_VERSION_FILE`

    # Set package name based on OS and architecture
    KNOWBASE_PACKAGE="knowbase-${OS_TYPE}-${ARCH}-${LATEST_VERSION}.tar.bz2"
    EXTRACTED_DIR="$SETUP_DIR/knowbase-${OS_TYPE}-${ARCH}-${LATEST_VERSION}"

    KNOWBASE_URL="$BOS_BUILDKIT_URL/knowbase/$KNOWBASE_PACKAGE"
    rm -rf $LATEST_VERSION_FILE

    (cd $SETUP_DIR && rm -rf $KNOWBASE_PACKAGE && wget -q $KNOWBASE_URL > /dev/null)
    if [ ! -e $SETUP_DIR/$KNOWBASE_PACKAGE ]; then
        echo "Fail to download knowbase package, the cmd is: wget $KNOWBASE_URL"
        return $ERR
    fi

    mkdir -p "$EXTRACTED_DIR"

    if expr "$KNOWBASE_PACKAGE" : '.*\.tar\.bz2$' > /dev/null; then
        (cd $SETUP_DIR && tar jxf "$KNOWBASE_PACKAGE" -C "$EXTRACTED_DIR" && rm -rf "$KNOWBASE_PACKAGE")
    elif expr "$KNOWBASE_PACKAGE" : '.*\.tar\.gz$' > /dev/null; then
        (cd $SETUP_DIR && tar zxf "$KNOWBASE_PACKAGE" -C "$EXTRACTED_DIR" && rm -rf "$KNOWBASE_PACKAGE")
    else
        echo "Fail to unzip knowbase: unsupported format($KNOWBASE_PACKAGE)"
        return $ERR
    fi
    
    if [ $? -ne 0 ]; then
        echo "Fail to unzip knowbase!"
        return $ERR
    fi

    # remove old KNOWBASE
    if [ -L "$SYMLINK_PATH" ] || [ -d "$SYMLINK_PATH" ] || [ -f "$SYMLINK_PATH" ]; then
        rm -rf "$SYMLINK_PATH"
    fi

    ln -s "$EXTRACTED_DIR" "$SYMLINK_PATH"

    if [ $? -ne 0 ]; then
        echo "Failed to create symlink: $SYMLINK_PATH"
        return $ERR
    fi

    chmod +x $KNOWBASE_SCRIPT_PATH

    # add knowbase into PATH configuration
    modify_path $BIN_DIR

    echo "setup knowbase success"
    if [ "$RUN_SOURCE_PATH" == "YES" ]; then
        echo -e "\033[;31m===> You must run :\033[;32;4m source $PROFILE \033[24m to complete knowbase install\033[0m"
    fi
    return $OK
}

install_knowbase
