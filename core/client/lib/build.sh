echo "Linux build..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o pixlise-linux.so main.go

# docker run -it --rm \
#   -v $GOPATH/src/github.com/pixlise/core:/go/src/github.com/pixlise/core \
#   -w /go/src/github.com/pixlise/core \
#   -e CGO_ENABLED=1 \
#   docker.elastic.co/beats-dev/golang-crossbuild:1.24.2-main \
#   --build-cmd "make build" \
#   -p "linux/amd64"

echo ""
echo "Windows build..."
# set CGO_CFLAGS=-g -O2 -Wl,–kill-at
# set CGO_CXXFLAGS=-g -O2 -Wl,–kill-at
# set CGO_FFLAGS=-g -O2 -Wl,–kill-at
# set CGO_LDFLAGS=-g -O2 -Wl,–kill-at
# set GOOS windows
# set GOARCH 386

#GOOS=windows GOARCH=386 CGO_ENABLED=1 go build -v -buildmode=c-shared -o pixlise-win32.dll main.go
#GOOS=windows GOARCH=386   go build -buildmode=c-shared -o pixlise-win32.dll main.go
#CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -v -buildmode=c-shared -o pixlise-win32.dll main.go
#CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -v -buildmode=c-shared -o pixlise-win32.dll main.go
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -v -buildmode=c-shared -o pixlise-win32.dll main.go


# docker run -it --rm \
#   -v $GOPATH/src/github.com/pixlise/core:/go/src/github.com/pixlise/core \
#   -w /go/src/github.com/pixlise/core \
#   -e CGO_ENABLED=1 \
#   docker.elastic.co/beats-dev/golang-crossbuild:1.24.2-main \
#   --build-cmd "make build" \
#   -p "windows/amd64"

echo ""
#echo "Darwin build..."
#GOOS=darwin GOARCH=386 go build -buildmode=c-shared -o pixlise-darwin.so main.go
#GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o pixlise-darwin.so main.go
#GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -o pixlise-darwin.so main.go
