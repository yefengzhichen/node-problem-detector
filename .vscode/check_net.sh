#!/bin/bash

readonly OK=0
readonly NONOK=1
readonly UNKNOWN=2

# 检查管理网接口状态
check_mgmt() {
    if ip link show br-mgmt | grep -q "UP"; then
        echo "br-mgmt: Interface is UP"
    else
        echo "br-mgmt: Interface is DOWN"
        return 1
    fi
}

# 检查存储网接口状态
check_storagepub() {
    if ip link show br-storagepub | grep -q "UP"; then
        echo "br-storagepub: Interface is UP"
    else
        echo "br-storagepub: Interface is DOWN"
        return 1
    fi
}

error_flag=$OK  # 初始化错误标志

if check_mgmt; then
    echo "Management network is active."
else
    echo "Management network is disconnected!"
    error_flag=$NONOK
fi

if check_storagepub; then
    echo "Storagepub network is active."
else
    echo "Storagepub network is disconnected!"
    error_flag=$NONOK
fi

exit $error_flag