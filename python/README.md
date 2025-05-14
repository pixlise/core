Developing on ubuntu in WSL2 in Windows 11:
- Install https://github.com/protocolbuffers/protobuf/releases/download/v25.7/protoc-25.7-linux-x86_64.zip
  - Download, unzip with: `unzip protoc-25.7-linux-x86_64.zip -d $HOME/.local`
  - The python protobuf lib will be compatible with 25.7 but latest seems too new
- Run ./genproto.sh
- Build the Go library using `core/client/lib/build.sh`
- Run `python3 lib-cgo.py`
