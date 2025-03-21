# to run -> source ./infra/scripts/cachi2.sh
# requires uv, docker, git

# Get Feast project repository root directory
export PROJECT_ROOT_DIR=$(git rev-parse --show-toplevel)
cd ${PROJECT_ROOT_DIR}
rm -rf ./cachi2-output ./cachi2.env ./milvus-lite # ./arrow

# yum builder
docker build \
  --tag yum-builder \
  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.builder.yum \
  --load sdk/python/feast/infra/feature_servers/multicloud

git clone --branch apache-arrow-17.0.0 https://github.com/apache/arrow
./arrow/cpp/thirdparty/download_dependencies.sh ${PROJECT_ROOT_DIR}/arrow/cpp/arrow-thirdparty
# arrow builder - version 17.0.0
docker build \
  --volume "$(realpath ./arrow)":/tmp/arrow:Z \
  --network none \
  --tag arrow-builder \
  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.builder.arrow \
  --load sdk/python/feast/infra/feature_servers/multicloud

# pip builder
docker build \
  --tag pip-builder \
  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.builder.pip \
  --load .

############################

alias cachi2='docker run --rm -ti -v "$PWD:$PWD:z" -w "$PWD" quay.io/konflux-ci/cachi2:f7a61b067f4446e4982d0e3b9545ce4aa0d8284f'
#git clone --branch v0.8.11 https://github.com/tkaitchuck/ahash
#rm -rf ./cachi2-output
#cd ahash
#cargo generate-lockfile
#cd ${PROJECT_ROOT_DIR}
#cachi2 fetch-deps cargo --source ahash
#cachi2 inject-files --for-output-dir /tmp/cachi2-output cachi2-output
## ahash builder w/ cargo install
#docker build \
#  --network none \
#  --tag ahash-builder \
#  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.builder.ahash \
#  --load ahash

#rm -rf ./cachi2-output
#git clone --branch v1.0.4 https://github.com/samuelcolvin/watchfiles
#cachi2 fetch-deps cargo --source watchfiles
#cachi2 inject-files --for-output-dir /tmp/cachi2-output cachi2-output
## watchfiles builder w/ cargo install
#docker build \
#  --volume "$(realpath ./cachi2-output)":/tmp/cachi2-output:Z \
#  --network none \
#  --tag watchfiles-builder \
#  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.builder.watchfiles \
#  --load watchfiles

#rm -rf ./cachi2-output
#git clone --branch v2.27.2 https://github.com/pydantic/pydantic-core
#cachi2 fetch-deps cargo --source pydantic-core
#uv export --all-groups --format requirements-txt -o pydantic-core/requirements.txt --project pydantic-core -p 3.11
#cachi2 inject-files --for-output-dir /tmp/cachi2-output cachi2-output
rm -rf ./cachi2-output ./cachi2.env ./milvus-lite
cachi2 fetch-deps \
  --output ./cachi2-output \
  '{
  "type": "pip",
  "path": ".",
  "requirements_files": [
"sdk/python/feast/infra/feature_servers/multicloud/requirements.txt"
],
  "requirements_build_files": [
"sdk/python/requirements/py3.11-sdist-requirements.txt",
"sdk/python/requirements/py3.11-pandas-requirements.txt",
"sdk/python/requirements/py3.11-python-dateutil-requirements.txt"
],
  "allow_binary": "false"
}'
#"pydantic-core/requirements.txt",
cachi2 generate-env ./cachi2-output -o ./cachi2.env --for-output-dir /tmp/cachi2-output

## pydantic-core builder w/ pip install
#docker build \
#  --volume "$(realpath ./cachi2-output)":/tmp/cachi2-output:Z \
#  --volume "$(realpath ./cachi2.env)":/tmp/cachi2.env:Z \
#  --network none \
#  --tag feast-sdist-builder \
#  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.builder.pydantic-core \
#  --load pydantic-core

git clone --branch v2.4.12 --recurse-submodules https://github.com/milvus-io/milvus-lite
# feast builder
docker build \
  --volume "$(realpath ./cachi2-output)":/tmp/cachi2-output:Z \
  --volume "$(realpath ./cachi2.env)":/tmp/cachi2.env:Z \
  --network none \
  --tag feast:build \
  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.sdist \
  --load .

docker run --rm -ti feast:build feast version
