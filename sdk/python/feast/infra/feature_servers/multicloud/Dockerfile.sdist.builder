FROM registry.access.redhat.com/ubi8/python-311:1

#USER 0
#RUN yum install -y python3.11-wheel-wheel.noarch
# python3.11-numpy

#USER 1001
#ENV PYTHONPATH=$PYTHONPATH:/usr/lib64/python3.11/site-packages
RUN pip install setuptools_scm>=6.2 wheel pybindgen==0.22.0 sphinx!=4.0.0 flit_core
RUN pip list
