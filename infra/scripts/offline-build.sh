# to run -> source ./infra/scripts/offline-build.sh
# on the build host... requires docker/podman, git, wget

APACHE_ARROW_VERSION="17.0.0"
SUBSTRAIT_VERSION="0.44.0"

# Get Feast project repository root directory
PROJECT_ROOT_DIR=$(git rev-parse --show-toplevel)
OFFLINE_BUILD_DIR=${PROJECT_ROOT_DIR}/offline_build
cd ${PROJECT_ROOT_DIR}

rm -rf ./offline_build
mkdir offline_build

# yum builder
docker build \
  --tag yum-builder \
  -f sdk/python/feast/infra/feature_servers/multicloud/offline/Dockerfile.builder.yum \
  --load sdk/python/feast/infra/feature_servers/multicloud/offline

git clone --branch apache-arrow-${APACHE_ARROW_VERSION} https://github.com/apache/arrow ${OFFLINE_BUILD_DIR}/arrow
${OFFLINE_BUILD_DIR}/arrow/cpp/thirdparty/download_dependencies.sh ${OFFLINE_BUILD_DIR}/arrow/cpp/arrow-thirdparty
wget https://github.com/substrait-io/substrait/archive/v${SUBSTRAIT_VERSION}.tar.gz -O ${OFFLINE_BUILD_DIR}/arrow/cpp/arrow-thirdparty/substrait-${SUBSTRAIT_VERSION}.tar.gz

alias cachi2='docker run --rm -ti -v "$PWD:$PWD:z" -w "$PWD" quay.io/konflux-ci/cachi2:f7a61b067f4446e4982d0e3b9545ce4aa0d8284f'
cachi2 fetch-deps \
  --output ${OFFLINE_BUILD_DIR}/cachi2-output \
  '{
  "type": "pip",
  "path": ".",
  "requirements_files": [
"sdk/python/feast/infra/feature_servers/multicloud/requirements.txt"
],
  "requirements_build_files": [
"sdk/python/feast/infra/feature_servers/multicloud/offline/pyarrow17-wheel-build-requirements.txt",
"sdk/python/feast/infra/feature_servers/multicloud/offline/psycopg3.2.5-wheel-build-requirements.txt",
"sdk/python/requirements/py3.11-sdist-requirements.txt",
"sdk/python/requirements/py3.11-pandas-requirements.txt",
"sdk/python/requirements/py3.11-addtl-sources-requirements.txt"
],
  "allow_binary": "false"
}'
cachi2 generate-env ${OFFLINE_BUILD_DIR}/cachi2-output -o ${OFFLINE_BUILD_DIR}/cachi2.env --for-output-dir /tmp/cachi2-output

# arrow OFFLINE builder - version 17.0.0
rm -f ${OFFLINE_BUILD_DIR}/arrow/.dockerignore
docker build \
  --network none \
  --volume ${OFFLINE_BUILD_DIR}/arrow:/tmp/arrow:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-output:/tmp/cachi2-output:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2.env:/tmp/cachi2.env:Z \
  --volume ${PROJECT_ROOT_DIR}/sdk/python/feast/infra/feature_servers/multicloud/offline:/tmp/offline:ro \
  --tag arrow-builder \
  -f sdk/python/feast/infra/feature_servers/multicloud/offline/Dockerfile.builder.arrow \
  --load offline_build/arrow

# maturin OFFLINE builder
git clone --branch v1.8.3 https://github.com/PyO3/maturin ${OFFLINE_BUILD_DIR}/maturin
cachi2 fetch-deps cargo --source ${OFFLINE_BUILD_DIR}/maturin --output ${OFFLINE_BUILD_DIR}/cachi2-maturin
cachi2 inject-files --for-output-dir /tmp/cachi2-maturin ${OFFLINE_BUILD_DIR}/cachi2-maturin

git clone --branch v2.27.2 https://github.com/pydantic/pydantic-core ${OFFLINE_BUILD_DIR}/pydantic-core
cachi2 fetch-deps cargo --source ${OFFLINE_BUILD_DIR}/pydantic-core --output ${OFFLINE_BUILD_DIR}/cachi2-pydantic-core
cachi2 inject-files --for-output-dir /tmp/cachi2-pydantic-core ${OFFLINE_BUILD_DIR}/cachi2-pydantic-core

git clone --branch v1.0.5 https://github.com/samuelcolvin/watchfiles ${OFFLINE_BUILD_DIR}/watchfiles
cachi2 fetch-deps cargo --source ${OFFLINE_BUILD_DIR}/watchfiles --output ${OFFLINE_BUILD_DIR}/cachi2-watchfiles
cachi2 inject-files --for-output-dir /tmp/cachi2-watchfiles ${OFFLINE_BUILD_DIR}/cachi2-watchfiles

git clone --branch v0.24.0 https://github.com/crate-py/rpds ${OFFLINE_BUILD_DIR}/rpds
cachi2 fetch-deps cargo --source ${OFFLINE_BUILD_DIR}/rpds --output ${OFFLINE_BUILD_DIR}/cachi2-rpds
cachi2 inject-files --for-output-dir /tmp/cachi2-rpds ${OFFLINE_BUILD_DIR}/cachi2-rpds

git clone --branch 44.0.2 https://github.com/pyca/cryptography ${OFFLINE_BUILD_DIR}/cryptography
cachi2 fetch-deps cargo --source ${OFFLINE_BUILD_DIR}/cryptography --output ${OFFLINE_BUILD_DIR}/cachi2-cryptography
cachi2 inject-files --for-output-dir /tmp/cachi2-cryptography ${OFFLINE_BUILD_DIR}/cachi2-cryptography

docker build \
  --network none \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-maturin:/tmp/cachi2-maturin:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-pydantic-core:/tmp/cachi2-pydantic-core:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-watchfiles:/tmp/cachi2-watchfiles:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-rpds:/tmp/cachi2-rpds:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-cryptography:/tmp/cachi2-cryptography:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-output:/tmp/cachi2-output:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2.env:/tmp/cachi2.env:Z \
  --tag maturin-builder \
  -f sdk/python/feast/infra/feature_servers/multicloud/offline/Dockerfile.builder.maturin \
  --load ${OFFLINE_BUILD_DIR}

# ibis OFFLINE builder
docker build \
  --network none \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-output:/tmp/cachi2-output:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2.env:/tmp/cachi2.env:Z \
  --tag ibis-builder \
  -f sdk/python/feast/infra/feature_servers/multicloud/offline/Dockerfile.builder.ibis \
  --load sdk/python/feast/infra/feature_servers/multicloud/offline

# feast OFFLINE builder
docker build \
  --network none \
  --volume ${OFFLINE_BUILD_DIR}/cachi2-output:/tmp/cachi2-output:Z \
  --volume ${OFFLINE_BUILD_DIR}/cachi2.env:/tmp/cachi2.env:Z \
  --tag feature-server:sdist-build \
  -f sdk/python/feast/infra/feature_servers/multicloud/offline/Dockerfile.sdist \
  --load sdk/python/feast/infra/feature_servers/multicloud

docker run --rm -ti feature-server:sdist-build feast version
