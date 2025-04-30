#!/bin/bash
# Copyright 2022 EasyStack, Inc.

ff=$(find . -type f -name "*.go" -print)

for f in ${ff}; do
	c=$(sed -n 1p $f |grep -ci Copyright)
	if [ $c -ne 0 ] ;then
		echo 'had add copyright :' $f
    else
		echo 'add copyright: ' $f
		if [[ "$OSTYPE" == "darwin"* ]]; then
			# macOS
			sed -i '' '1s/^/\/\/ Copyright 2025 EasyStack, Inc.\n/' $f
        else
            # Linux
            sed -i '1i \// Copyright 2025 EasyStack, Inc.' $f
        fi
	fi
done
