# API JSON Compare

`apijc` fetches and compares the status codes and json responses of user-defined paths on two domains and
reports any differences.

# Table of Contents

- [API JSON Compare](#api-json-compare)
- [Table of Contents](#table-of-contents)
  - [Features](#features)
  - [Installation](#installation)
    - [Binary](#binary)
    - [Golang](#golang)
  - [Usage](#usage)
  - [Example output](#example-output)
  - [Configuration](#configuration)
    - [CLI Flags](#cli-flags)
    - [urlFile](#urlfile)
      - [Structure](#structure)
      - [Path expansion](#path-expansion)
      - [Example](#example)
    - [rateLimit](#ratelimit)
    - [headerFile](#headerfile)
      - [Example](#example-1)
      - [Precedence](#precedence)
    - [Output](#output)
      - [stdout](#stdout)
      - [outputFile](#outputfile)
        - [Example](#example-2)
  - [Exit codes](#exit-codes)

## Features

- diff comparison of response bodies from both domains
- Path expansion of
  - Lists (example: `/foo/{1,2,3}/bar`)
  - Numerical ranges (example: `/foo/{1-100}/bar`)
  - Mixed list and ranges (example: `/foo/{1,3-5,99,200-400}`)
  - See [Path expansion](#path-expansion) below
- Rate limiting
- Load headers from file
- Write errors/mismatches to stdout or file

## Installation

### Binary

1. Download the binary for your architecture from the [Releases](/releases) page.
2. Put it into a directory in your `$PATH`

### Golang

```sh
go install github.com/phux/apijc@latest
```

## Usage

```sh
apijc \
  --baseDomain "<http://first.domain>"  \
  --newDomain "<http://second.domain>" \
  --urlFile <path/to/a/url.json> \
  --headerFile <path/to/a/header.json>  \ # optional
  --rateLimit 100 \ # optional
  --outputFile <path/to/an/output.json> # optional
```

## Example output

```sh
$ apijc --baseDomain http://localhost:8080 \
  --newDomain http://localhost:8081 \
  --urlFile ./urlfile_example.json \
  --rateLimit 1000

Starting with rate limit: 1000.000000/second

2023/11/30 20:48:21 Checking GET /v1/example
2023/11/30 20:48:21 Success: GET /v1/example (checked 1 of 1 paths)

2023/11/30 20:48:21 Checking GET /v1/{1-100}
2023/11/30 20:48:21 Success: GET /v1/{1-100} (checked 100 of 100 paths)

2023/11/30 20:48:21 Checking GET /v1/@1-3@
2023/11/30 20:48:21 Success: GET /v1/@1-3@ (checked 3 of 3 paths)

2023/11/30 20:48:21 Checking GET /v1/expected_jsonmissmatch
2023/11/30 20:48:21 ERROR: GET /v1/expected_jsonmissmatch (checked 1 of 1 paths)

2023/11/30 20:48:21 Checking POST /v1/example
2023/11/30 20:48:21 Success: POST /v1/example (checked 1 of 1 paths)

2023/11/30 20:48:21 Done. Checked 106 of 106 paths

2023/11/30 20:48:21 Findings:
2023/11/30 20:48:21 /v1/expected_jsonmissmatch
Error: JSON mismatch
Diff: @ ["foo"]
- "baz"
+ "bar"
2023/11/30 20:48:21 Finished - 1 findings
exit status 1
```

## Configuration

### CLI Flags

| Flag       | Required | Description                                                                                                        | Default |
| ---------- | -------- | ------------------------------------------------------------------------------------------------------------------ | ------- |
| baseDomain | yes      | The first domain to make all requests to                                                                           | -       |
| newDomain  | yes      | The second domain to make all requests to                                                                          | -       |
| urlFile    | yes      | JSON file containing target URL paths, HTTP method, ...<br />See [urlFile](#urlfile)                               | -       |
| headerFile | no       | JSON file containing global header key-value pairs that will be set on each request. See [headerFile](#headerfile) | -       |
| rateLimit  | no       | Requests per second (float).<br /> See [rateLimit](#ratelimit)                                                     | 1       |
| outputFile | no       | Store findings in a JSON file. See [outputFile](#outputfile)                                                       | -       |

### urlFile

The `urlFile` defines the relative paths that will be requested and compared on both domains.

#### Structure

```json
{
  "targets": {
    "<HTTP method (GET|POST|...)>": [
      {
        "relativePath": "<required string, /a/relative/path/to/check/on/both/domains>",
        "expectedStatusCode": <required int; checked on both domains>,
        "requestBody": "<optional string, body to send to relativePath>",
        "requestHeaders": { // optional
          "<string, header key>": "<string, header value>"
        },
        "patternPrefix": "<optional string, a character to start exansion; default {>",
        "patternSuffix": "<optional string, a character to stop exansion; default }>"
      }
    ]

}
```

#### Path expansion

`relativePath` can contain lists and/or ranges to quickly define multiple
targets at once.<br />
Expansions are triggered for everything between a configurable `patternPrefix`
(default: `{`) and a `patternSuffix` (default: `}`) on each target.<br />
List items are separated by `,` (comma).<br />
Numerical ranges can be defined by `-` (dash).

Example:

```json
"relativePath": "/foo/{bar,3-5}"
```

This will translate to requesting and checking 4 paths:

- `/foo/bar` <-- from list item `bar`
- `/foo/3` <-- from range `3-5`
- `/foo/4` <-- from range `3-5`
- `/foo/5` <-- from range `3-5`

Note: a path can also define multiple expansions, like `/foo/{1-2}/bar/{a,b}`.
This path will result in 4 paths in total:

- `/foo/1/bar/a`
- `/foo/1/bar/b`
- `/foo/2/bar/a`
- `/foo/2/bar/b`

#### Example

```json
{
  "targets": {
    "GET": [
      {
        "relativePath": "/v1/example",
        "expectedStatusCode": 200
      },
      {
        "relativePath": "/v1/{1-100}",
        "expectedStatusCode": 200
      }
    ],
    "POST": [
      {
        "relativePath": "/v1/example",
        "expectedStatusCode": 201,
        "requestBody": "{\"a\":\"b\"}",
        "requestHeaders": {
          "Content-Type": "application/json"
        },
        "patternPrefix": "{",
        "patternSuffix": "}"
      }
    ]
  }
}
```

### rateLimit

Sometimes it's necessary to limit the rate with which the tool makes requests to the configured domains.
The flag `--rateLimit` allows to configure the rate.<br/>
Must be `int` or float.<br/>
The number defines the allowed requests per seconds.

Examples:

- `--rateLimit=1`: 1 request per second
- `--rateLimit=0.5`: 1 request per 2 seconds
- `--rateLimit=10`: 10 requests per second

### headerFile

The `headerFile` allows to define key-value pairs that will be set on each
request in addition to the static `requestHeaders` defined on each target in
the `urlFile`.<br/>
Format: flat JSON object

#### Example

```sh
# header.json
{
    "Authorization": "Bearer [...]"
}
```

#### Precedence

If the `headerFile` and a target's `requestHeaders` have duplicate header keys,
the target's `requestHeaders` value takes precedence.

### Output

#### stdout

If the flag `--outputFile` is not passed, the findings are written to
stdout, see [Example output](#example-output)

#### outputFile

Via `--outputFile` a path to a file can be passed. The findings will be written
to this file instead of stdout.

##### Example

```sh
# findings.json
[
  {
    "url": "/v1/expected_jsonmissmatch",
    "error": "JSON mismatch",
    "diff": "@ [\"foo\"]\n- \"baz\"\n+ \"bar\"\n"
  }
]%
```

## Exit codes

On successful execution `apijc` exits with code `0`.
On any issue the exit code will be `> 0`
