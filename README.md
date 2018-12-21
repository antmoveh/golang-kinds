# 拉取谷歌镜像代理服务
-  查询谷歌镜像和他的tag
  https://console.cloud.google.com/gcr/images/google-containers?project=google-containers 
- github仓库目录中创建makefile
    ```
    FROM k8s.gcr.io/kube-cross:v1.10.1-1
    MAINTAINER antmove
    ```
- docker hub 
  - 创建个人镜像仓库
  - 关联github账户
  - 创建自动构建流程
  - 配置github上的dockerfile目录地址
  - 执行构建拉取镜像
- 拉取成功后服务器上拉取个人镜像仓库中的镜像
  - docker pull antmove/kube-cross:v1.10.1-1
  - docker tag antmove/kube-cross:v1.10.1-1 k8s.gcr.io/kube-cross:v1.10.1-1
- 至此拉取完成