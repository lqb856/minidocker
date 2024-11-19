#!/bin/bash

NEWROOT="/home/lqb/go-project/minidocker/busybox"

# 创建必要的目录结构
mkdir -p "$NEWROOT"/{proc,sys,tmp,dev,oldroot}

# 启动隔离环境
unshare --mount --pid --fork -- bash -c "
    # 确保挂载隔离
    mount --make-rprivate /

    # 挂载新的根文件系统
    mount --bind $NEWROOT $NEWROOT

    # 切换到新的根文件系统
    mkdir -p $NEWmouROOT/oldroot
    pivot_root $NEWROOT $NEWROOT/oldroot

    # 清理旧根文件系统
    cd /
    umount -l /oldroot
    rmdir /oldroot

    # # 挂载虚拟文件系统
    # mount -t proc none /proc
    # mount -t sysfs none /sys
    # mount -t tmpfs none /tmp

    # # 挂载独立的 /dev
    # mount -t devtmpfs none /dev
    # mount -t devpts none /dev/pts
    # mount -t tmpfs none /dev/shm

    # 启动隔离环境中的程序
    exec /bin/sh
"
