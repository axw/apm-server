set -xe

DOCKER_IMAGE=elastic/apm-server-protoc
docker build -t $DOCKER_IMAGE $PWD/script/genproto

PROTOC="docker run --rm -u $(id -u) -v$PWD:$PWD -v$PWD:/src/github.com/elastic/apm-server -w$PWD \
	$DOCKER_IMAGE protoc -I/src/github.com/elastic"

for x in model/proto/*.proto; do
  $PROTOC \
    --go_out=. \
    --go_opt=module=github.com/elastic/apm-server \
    --go_vtproto_out=. \
    --go_vtproto_opt=features=marshal+unmarshal+size \
    --go_vtproto_opt=module=github.com/elastic/apm-server \
    -I /src/github.com/elastic/apm-server/model/proto \
    /src/github.com/elastic/apm-server/$x
done
