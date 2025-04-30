#!/bin/bash

readonly OK=0
readonly NONOK=1
readonly UNKNOWN=2

check_net() {
    if ip link show eth0 | grep -q "UP"; then
        echo "eth0: Interface is UP"
    else
        echo "eth0: Interface is DOWN"
        return 1
    fi
}

error_flag=$OK  # 初始化错误标志

if check_net; then
    echo "eth0 network is active."
else
    echo "eth0 network is disconnected!"
    error_flag=$NONOK
fi

exit $error_flag
