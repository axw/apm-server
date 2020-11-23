set -xe

DOCKER_PROTOBUF=otel/build-protobuf:0.2.1
PROTOC="docker run --rm -u $(id -u) -v$PWD:$PWD -v$PWD:/src/github.com/elastic/apm-server -w$PWD \
	$DOCKER_PROTOBUF -I/usr/include/github.com/gogo/protobuf -I/src/github.com/elastic"

GOGO_OPTIONS=\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types

$PROTOC --gogofaster_out=$GOGO_OPTIONS:/src /src/github.com/elastic/apm-server/model/proto/*.proto
