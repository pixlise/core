# Linux build:
go build -buildmode=c-shared -o pixlise-linux.so main.go

# Windows build:
# set CGO_CFLAGS=-g -O2 -Wl,–kill-at
# set CGO_CXXFLAGS=-g -O2 -Wl,–kill-at
# set CGO_FFLAGS=-g -O2 -Wl,–kill-at
# set CGO_LDFLAGS=-g -O2 -Wl,–kill-at
# set GOOS windows
# set GOARCH 386
#GOOS=windows GOARCH=386 go build -buildmode=c-shared -o pixlise-win32.dll main.go
