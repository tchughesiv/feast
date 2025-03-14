FROM registry.access.redhat.com/ubi8/python-311:1

USER 0
RUN yum install -y python3.11-setuptools-wheel python3.11-numpy python3-setuptools_scm

USER 1001
ENV PYTHONPATH=$PYTHONPATH:/usr/lib64/python3.11/site-packages
