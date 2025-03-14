FROM registry.access.redhat.com/ubi8/python-311:1

USER 0
RUN yum install -y ninja-build

USER 1001
ENV PYTHONPATH=$PYTHONPATH:/usr/lib64/python3.11/site-packages
RUN pip list
