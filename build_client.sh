# Could also use: https://github.com/crazy-max/docker-osxcross
# And/or:         https://github.com/elastic/golang-crossbuild/tree/main

echo "Linux build..."
# CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o pixlise-linux-amd64.so main.go
# CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -v -buildmode=c-shared -o pixlise-linux-arm64.so main.go

docker run -it --rm \
  -v $PWD:/usr/src/app \
  -w /usr/src/app \
  -e CGO_ENABLED=1 \
  docker.elastic.co/beats-dev/golang-crossbuild:1.24.2-main \
  --build-cmd "go build -buildmode=c-shared -o ./_out/client/pixlise-linux-amd64.so ./core/client/lib" \
  -p "linux/amd64"

docker run -it --rm \
  -v $PWD:/usr/src/app \
  -w /usr/src/app \
  -e CGO_ENABLED=1 \
  docker.elastic.co/beats-dev/golang-crossbuild:1.24.2-armm \
  --build-cmd "go build -buildmode=c-shared -o ./_out/client/pixlise-linux-arm64.so ./core/client/lib" \
   -p "linux/arm64"


echo ""
echo "Windows build..."
#GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -v -buildmode=c-shared -o pixlise-windows.dll main.go

docker run -it --rm \
  -v $PWD:/usr/src/app \
  -w /usr/src/app \
  -e CGO_ENABLED=1 \
  docker.elastic.co/beats-dev/golang-crossbuild:1.24.2-main \
  --build-cmd "go build -buildmode=c-shared -o ./_out/client/pixlise-windows-amd64.dll ./core/client/lib" \
  -p "windows/amd64"


exit


echo ""
echo "Darwin build..."
#CC=o64-clang CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -o pixlise-darwin-amd64.so main.go
#CC=o64-clang CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -buildmode=c-shared -o pixlise-darwin-arm64.so main.go

docker run -it \
  -v $PWD:/usr/src/app \
  -w /usr/src/app \
  -e CGO_ENABLED=1 \
  -e CC=o64-clang \
  -e GOOS=darwin \
  -e GOARCH=amd64 \
  docker.elastic.co/beats-dev/golang-crossbuild:1.24.2-darwin \
  --build-cmd "go build -work -buildmode=c-shared -o ./_out/client/pixlise-darwin-amd64.so ./core/client/lib" \
  -p "darwin/amd64"

  # -v -work -x

# docker run -it --rm \
#   -v $PWD:/usr/src/app \
#   -w /usr/src/app \
#   -e CGO_ENABLED=1 \
#   -e CC=o64-clang \
#   -e GOOS=darwin \
#   -e GOARCH=arm64 \
#   docker.elastic.co/beats-dev/golang-crossbuild:1.24.2-darwin \
#   --build-cmd "go build -buildmode=c-shared -o ./_out/client/pixlise-darwin-arm64.so ./core/client/lib" \
#   -p "darwin/arm64"
