FROM centos:7

RUN yum -y install git golang && yum clean all
COPY bin/oc /usr/bin/oc
