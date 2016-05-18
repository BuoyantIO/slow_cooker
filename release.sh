GOOS=linux GOARCH=amd64 go build -o slow_cooker_amd64 github.com/buoyantio/slow_cooker
GOOS=linux GOARCH=arm   go build -o slow_cooker_arm   github.com/buoyantio/slow_cooker
GOOS=linux GOARCH=386   go build -o slow_cooker_i386  github.com/buoyantio/slow_cooker
echo "releases built:"
ls slow_cooker*
