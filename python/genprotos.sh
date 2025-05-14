# To run this we need:
# pip install protoletariat

# This will generate the proto files, but then modify them to have relative
# imports. This is something Google hasn't bothered fixing for over 7 years
# See: https://github.com/cpcloud/protoletariat and https://github.com/protocolbuffers/protobuf/issues/1491#issuecomment-1648982084


#protoc --python_out=./python/pixlisemsgs.zip --proto_path=./data-formats/api-messages/ ./data-formats/api-messages/*.proto
mkdir -p ./pixlisemsgs

cd ..
protoc --python_out=./python/pixlisemsgs/ --proto_path=./data-formats/api-messages/ ./data-formats/api-messages/*.proto

#touch ./pixlisemsgs/__init__.py

protol \
  --create-package \
  --in-place \
  --python-out ./python/pixlisemsgs \
  protoc --proto-path=./data-formats/api-messages/ ./data-formats/api-messages/*.proto

cd python
