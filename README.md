# Hook
simple github webhook server.

### Installation
```bash
go get github.com/igor-k/hook
cd $GOPATH/src/github.com/igor-k/hook
go install
```

### Configuration
configuration is in the following format; you can have multiple repositories.

```json
{
    "igor-k/hook": {
        "master": "/full/path/to/script.sh",
        "dev": "/full/path/to/another/script.sh"
    },
    "igor-k/repo": {
        "master": "/full/path/to/script.sh",
    }
}
```
or just
```json
{
    "master": "/full/path/to/script.sh",
    "dev": "/full/path/to/another/script.sh"
}
```
each script gets repo ssh url, branch name, and commit hash as arguments.

### Usage
```bash
hook --help # to see all available options
hook -config /path/to/config.json # start a server using a provided config file
```
