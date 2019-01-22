FROM registry.svc.ci.openshift.org/openshift/origin-v3.11:base

RUN yum -y install git golang && yum clean all
