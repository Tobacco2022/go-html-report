# report

## Install

```bash
go install .
```


## Usage

```bash


# usage: convert go test output to html report
 go test -v ./... |report .
 go test -v ./... |report index.hmtl
```

## Refer 
- [go-junit-report](https://github.com/jstemmer/go-junit-report)
