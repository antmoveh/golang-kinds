
##### 综述

- 想做一个服务，在服务启动后服务就像在docker容器中运行一样;
- 主要借鉴runc的实现，相当于将其创建容器功能摘出来

##### 步骤

- 当容器运行时，会有一套独立对文件系统，我们现在准备一套服务运行时的文件系统
```shell script
$ docker pull registry.cn-hangzhou.aliyuncs.com/antmoveh/busybox:1.32
$ mkdir -p /tmp/container/rootfs
$ cd /tmp/container
$ docker export $(docker create registry.cn-hangzhou.aliyuncs.com/antmoveh/busybox:1.32) | tar -C rootfs -xvf -

```

##### 参考
- https://learnku.com/users/42861
- <<自己动手写docker>>
