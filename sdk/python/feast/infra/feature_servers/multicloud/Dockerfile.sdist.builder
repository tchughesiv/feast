FROM registry.access.redhat.com/ubi8/python-311:1
USER 0
RUN yum install -y ninja-build dnf-plugins-core
#RUN yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
RUN yum config-manager --set-enabled powertools --set-enabled epel-release
RUN yum install -y libarrow-devel
USER 1001
