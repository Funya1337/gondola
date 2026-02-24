# windows build usage:
``` yaml
build:
  entry: "." # build in current directory?
  output: "bin/myapp.exe" # build output
  goos: "windows"
  goarch: "amd64"
  ldflags: "-s -w"
  extra_env:
    - "CGO_ENABLED=0"
```