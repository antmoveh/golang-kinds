# ��ȡ�ȸ辵��������
-  ��ѯ�ȸ辵�������tag
  https://console.cloud.google.com/gcr/images/google-containers?project=google-containers 
- github�ֿ�Ŀ¼�д���makefile
    ```
    FROM k8s.gcr.io/kube-cross:v1.10.1-1
    MAINTAINER antmove
    ```
- docker hub 
  - �������˾���ֿ�
  - ����github�˻�
  - �����Զ���������
  - ����github�ϵ�dockerfileĿ¼��ַ
  - ִ�й�����ȡ����
- ��ȡ�ɹ������������ȡ���˾���ֿ��еľ���
  - docker pull antmove/kube-cross:v1.10.1-1
  - docker tag antmove/kube-cross:v1.10.1-1 k8s.gcr.io/kube-cross:v1.10.1-1
- ������ȡ���