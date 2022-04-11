# Emojictl - manage your emojis!

[![Actions Test Workflow Widget]][Actions Test Workflow Status]
[![GoReport Widget]][GoReport Status]
[![GoDocWidget]][GoDocReference]

[Actions Test Workflow Status]: https://github.com/gkze/emojictl/actions?query=workflow%3Aci
[Actions Test Workflow Widget]: https://github.com/gkze/emojictl/workflows/ci/badge.svg

[GoReport Status]: https://goreportcard.com/report/github.com/gkze/emojictl
[GoReport Widget]: https://goreportcard.com/badge/github.com/gkze/emojictl

[GoDocWidget]: https://godoc.org/github.com/gkze/emojictl?status.svg
[GoDocReference]:https://godoc.org/github.com/gkze/emojictl

Emojictl (pronounced emoji control) is a package (& CLI wrapper) for managing
Emojis on platforms that support custom emojis. Currently only Slack support
is implemented.

This project is under active development, so expect it to change a lot & quickly!

## Getting Started

### Install

`emojictl` is available via three channels:

* **Homebrew**

  ```
  brew install gkze/gkze/emojictl
  ```

* **Go**

  ```
  go get github.com/gkze/emojictl/cmd/emojictl
  ```

* **Binaries**

  See the [releases page](https://github.com/gkze/emojictl/releases).

### Configure

`emojictl` the CLI is configured via environment variables:
* `SLACK_WORKSPACE`: name of the target Slack workspace
* `SLACK_HEADER_TOKEN`: the Slack API token that is included in the cookie. It can be found by inspecting the request for the `/customize/emojis` homepage. It's in the `Cookie` header and its section starts like this: `d=xoxd-....`. The string is URL escaped, but we store it unescaped. For this you can do:

  `$ python3 -c 'import urllib.parse; print(urllib.parse.unquote("[INSERT COPIED TOKEN HERE"))'`

  and store the output value in the environment variable.
* `SLACK_BODY_TOKEN`: In the multipart form body of the request. Can also be found by inspecting the same request and specifically looking at the multipart form body. The field name is `token` and it will look something like this:

  ```
  ------WebKitFormBoundaryhsp7MJOeVbNItf5c
  Content-Disposition: form-data; name="token"

  xoxc-[...]
  ```

### Use

```
$ emojictl
Emoji control center

Usage:
  emojictl [flags]
  emojictl [command]

Available Commands:
  add         Upload emoji
  alias       Alias emoji
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        List emojis
  remove      Remove emoji

Flags:
  -h, --help   help for emojictl

Use "emojictl [command] --help" for more information about a command.
```

## License

[MIT](LICENSE)
