alias cachi2='podman run --rm -ti -v "$PWD:$PWD:z" -w "$PWD" quay.io/konflux-ci/cachi2:latest'
rm -rf ./cachi2-output
cachi2 fetch-deps \
  --output ./cachi2-output \
  '{
  "type": "pip",
  "path": ".",
  "requirements_files": ["sdk/python/feast/infra/feature_servers/multicloud/requirements.txt"],
  "requirements_build_files": ["sdk/python/requirements/py3.11-build-requirements.txt", "sdk/python/requirements/py3.11-pandas-requirements.txt", "sdk/python/requirements/py3.11-sdist-requirements.txt"],
  "allow_binary": "false"
}'

podman build \
  --tag sdist-builder:latest \
  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.sdist.builder \
  sdk/python/feast/infra/feature_servers/multicloud

rm -f cachi2.env
cachi2 generate-env ./cachi2-output -o ./cachi2.env --for-output-dir /tmp/cachi2-output

podman build \
  --volume "$(realpath ./cachi2-output)":/tmp/cachi2-output:Z \
  --volume "$(realpath ./cachi2.env)":/tmp/cachi2.env:Z \
  --network none \
  --tag foo \
  -f sdk/python/feast/infra/feature_servers/multicloud/Dockerfile.sdist \
  sdk/python/feast/infra/feature_servers/multicloud
