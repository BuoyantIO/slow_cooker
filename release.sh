GOOS=linux GOARCH=amd64  go build -o slow_cooker_linux_amd64 github.com/buoyantio/slow_cooker
GOOS=linux GOARCH=arm    go build -o slow_cooker_linux_arm   github.com/buoyantio/slow_cooker
GOOS=linux GOARCH=386    go build -o slow_cooker_linux_i386  github.com/buoyantio/slow_cooker
GOOS=darwin GOARCH=amd64 go build -o slow_cooker_darwin      github.com/buoyantio/slow_cooker
echo "releases built:"
ls slow_cooker*
