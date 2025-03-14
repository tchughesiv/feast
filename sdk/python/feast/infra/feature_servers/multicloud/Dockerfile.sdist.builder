FROM registry.access.redhat.com/ubi8/python-311:1

USER 0
RUN yum install -y python3.11-setuptools-wheel python3.11-numpy

USER 1001
RUN pip install setuptools_scm wheel
