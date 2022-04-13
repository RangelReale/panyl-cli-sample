# panyl-cli-sample

Complete sample for [panyl-cli](https://github.com/RangelReale/panyl-cli).

## Install

```shell
go install github.com/RangelReale/panyl-cli-sample/cmd/panyl-cli-sample
```

## Usage

### Using a file

```shell
panyl-cli-sample log --with-ansiescape --with-json --with-rubylog --output=ansi file.log
```

```shell
panyl-cli-sample preset all --output=ansi file.log
```

### Using stdin

```shell
cmd 2>&1 >/dev/null | panyl-cli-sample preset all --output=ansi -
```

```shell
cmd 2>&1 >/dev/null | panyl-cli-sample preset all --output=ecapplog --ecappname=cmd -
```

### Author

Rangel Reale (rangelreale@gmail.com)
