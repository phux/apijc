# API JSON Compare (apijc)

`apijc` fetches and compares the status codes and json responses of
user-defined paths on two domains and reports any differences.

<!--toc:start-->

- [API JSON Compare (apijc)](#api-json-compare-apijc)
  - [Features](#features)
  - [Installation](#installation)
    - [Binary](#binary)
    - [Golang](#golang)
  - [Usage](#usage)
    - [Quickstart](#quickstart)
  - [Example output](#example-output)
  - [Configuration](#configuration)
    - [CLI Flags](#cli-flags)
    - [urlFile](#urlfile)
      - [targets](#targets)
      - [sequentialTargets](#sequentialtargets)
      - [Structure](#structure)
      - [Path expansion](#path-expansion)
      - [urlFile Example](#urlfile-example)
    - [rateLimit](#ratelimit)
    - [headerFile](#headerfile)
      - [headerFile Example](#headerfile-example)
      - [Precedence](#precedence)
    - [Output](#output)
      - [stdout](#stdout)
      - [outputFile](#outputfile)
        - [outputFile Example](#outputfile-example)
  - [Exit codes](#exit-codes)
  <!--toc:end-->

## Features

- Diff comparison of response bodies from both domains
- Sequential request chains. See [sequentialTargets](#sequentialtargets) below
- Check for expected status codes
- Path expansion of
  - Lists (example: `/foo/{1,2,3}/bar`)
  - Numerical ranges (example: `/foo/{1-100}/bar`)
  - Mixed list and ranges (example: `/foo/{1,3-5,99,200-400}`)
  - See [Path expansion](#path-expansion) below
- Rate limiting
- Load headers from file
  - Specify header key-value pairs globally or per domain
- Custom headers per url target
- Write errors/mismatches to stdout or file

## Installation

### Binary

1. Download the binary for your architecture from the
   [Releases](https://github.com/phux/apijc/releases) page.
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

### Quickstart

1. Install - see [Installation](#installation)
2. Create a urlFile JSON - see [urlFile](#urlfile) - and setup up at least one target

Minimal `urlFile` example:

```json
{
  "targets": [
    {
      "relativePath": "/some/relative/path",
      "httpMethod": "GET",
      "expectedStatusCode": 200
    }
  ]
}
```

3. execute `apijc`

```sh
apijc \
  --baseDomain "<http://your-base.domain>" \
  --newDomain "<http://your-other.domain>" \
  --urlFile path/to/your/urlFile.json
```

Note: if `--rateLimit` is not passed to `apijc`, the default rate limit is 1 request per second.

## Example output

```sh
$ apijc --baseDomain http://localhost:8080 \
  --newDomain http://localhost:8081 \
  --urlFile ./urlfile_example.json \
  --rateLimit 1000

Starting with rate limit: 1000.000000/second

2023/12/06 22:09:35 Checking GET /v1/example
2023/12/06 22:09:35 Success: GET /v1/example (checked 1 of 1 paths)

2023/12/06 22:09:35 Checking GET /v1/{1-100}
2023/12/06 22:09:36 Success: GET /v1/{1-100} (checked 100 of 100 paths)

2023/12/06 22:09:36 Checking GET /v1/@1-3@
2023/12/06 22:09:36 Success: GET /v1/@1-3@ (checked 3 of 3 paths)

2023/12/06 22:09:36 Checking GET /v1/expected_jsonmissmatch
2023/12/06 22:09:36 ERROR: GET /v1/expected_jsonmissmatch (checked 1 of 1 paths)

2023/12/06 22:09:36 Checking POST /v1/example
2023/12/06 22:09:36 Success: POST /v1/example (checked 1 of 1 paths)

2023/12/06 22:09:36 Checking sequential group: First POST, then GET
2023/12/06 22:09:36 Success: POST /v1/sequential_post (checked 1 of 1 paths)

2023/12/06 22:09:36 Success: GET /v1/sequential_get (checked 1 of 1 paths)

2023/12/06 22:09:36 Done. Checked 108 of 108 paths

2023/12/06 22:09:36 Findings:
2023/12/06 22:09:36 /v1/expected_jsonmissmatch
Error: JSON mismatch
Diff: @ ["foo"]
- "baz"
+ "bar"
2023/12/06 22:09:36 Finished - 1 findings
exit status 1
```

## Configuration

### CLI Flags

| Flag       | Required | Description                                                                                                                                  | Default |
| ---------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| baseDomain | yes      | The first domain to make all requests to                                                                                                     | -       |
| newDomain  | yes      | The second domain to make all requests to                                                                                                    | -       |
| urlFile    | yes      | Path to JSON file containing target URL paths, HTTP method, ...<br />See [urlFile](#urlfile)                                                 | -       |
| headerFile | no       | Path to JSON file containing global and/or per-domain header key-value pairs that will be set on each request. See [headerFile](#headerfile) | -       |
| rateLimit  | no       | Requests per second (float).<br /> See [rateLimit](#ratelimit)                                                                               | 1       |
| outputFile | no       | Path to store findings in JSON format. See [outputFile](#outputfile)                                                                         | -       |

### urlFile

The `urlFile` defines the relative paths that will be requested and compared on
both domains. It contains `targets` and/or `sequentialTargets`.

#### targets

Standalone requests are defined in the `targets` key of the `urlFile`. Each
target will be requested on both, `baseDomain` and `newDomain`.

#### sequentialTargets

In the `sequentialTargets` key chains of consecutive calls can be defined.
Example: first make a POST request to create an entity, then
make a GET request to fetch the created entity.

The steps for a sequential target group are:

1. make request to target 1 on `baseDomain`
2. compare actual status code vs `expectedStatusCode`
3. make request to target 1 on `newDomain`
4. compare actual status code vs `expectedStatusCode`
5. compare response bodies
6. make request to target 2 on `baseDomain`
7. compare actual status code vs `expectedStatusCode`
8. make request to target 2 on `newDomain`
9. compare actual status code vs `expectedStatusCode`
10. compare response bodies

#### Structure

```json
{
  "targets": [
      {
        "relativePath": "<required string, /a/relative/path/to/check/on/both/domains>",
        "httpMethod": "<GET|POST|...>",
        "expectedStatusCode": <required int; checked on both domains>,
        "requestBody": "<optional string, body to send to relativePath>",
        "requestHeaders": { // optional
          "<string, header key>": "<string, header value>"
        },
        "patternPrefix": "<optional string, a character to start expansion; default {>",
        "patternSuffix": "<optional string, a character to stop expansion; default }>"
      }
    ],
  "sequentialTargets": {
    "Some name for the sequence, example: Create Order, then fetch Order": [
      {
        "relativePath": "/first/path",
        "httpMethod": "<GET|POST|...>",
        "expectedStatusCode": 201
      },
      {
        "relativePath": "/second/path",
        "httpMethod": "<GET|POST|...>",
        "expectedStatusCode": 200
      }
    ]
  }
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

#### urlFile Example

```json
{
  "targets": [
    {
      "relativePath": "/v1/example",
      "httpMethod": "GET",
      "expectedStatusCode": 200
    },
    {
      "relativePath": "/v1/{1-100}",
      "httpMethod": "GET",
      "expectedStatusCode": 200
    },
    {
      "relativePath": "/v1/example",
      "httpMethod": "POST",
      "expectedStatusCode": 201,
      "requestBody": "{\"a\":\"b\"}",
      "requestHeaders": {
        "Content-Type": "application/json"
      },
      "patternPrefix": "{",
      "patternSuffix": "}"
    }
  ],
  "sequentialTargets": {
    "First POST, then GET": [
      {
        "relativePath": "/v1/sequential_post",
        "httpMethod": "POST",
        "expectedStatusCode": 201,
        "requestBody": "{\"a\":\"b\"}"
      },
      {
        "relativePath": "/v1/sequential_get",
        "httpMethod": "GET",
        "expectedStatusCode": 200
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

- `--rateLimit=1`: 1 request per second (default)
- `--rateLimit=0.5`: 1 request per 2 seconds
- `--rateLimit=10`: 10 requests per second

### headerFile

The `headerFile` allows to define key-value pairs in the `global` key that will be set on each
request, in addition to the static `requestHeaders` defined on each target in
the `urlFile`.<br/>

Additionally, it is possible to set per-domain header key-value pairs that will
be set on each request to the particular domain (`baseDomain|newDomain`).
This is helpful for example if you need to set different `Authorization` headers per domain.

Note: The `global`, `baseDomain` and `newDomain` keys are all optional.

#### headerFile Example

```sh
# header.json
{
  "global": {
    "SomeHeaderName": "Value applied to all requests to both domains"
  },
  "baseDomain": {
    "SomeHeaderName": "Value applied to all requests to BaseDomain"
  },
  "newDomain": {
    "SomeHeaderName": "Value applied to all requests to NewDomain"
  }
}
```

#### Precedence

If the `headerFile` and a target's `requestHeaders` contain duplicate header keys,
the target's `requestHeaders` value takes precedence.

```
headerFile.global < headerFile.<new|base>Domain < target.requestHeaders
```

### Output

#### stdout

If the flag `--outputFile` is not passed, the findings are written to
stdout, see [Example output](#example-output)

#### outputFile

Via `--outputFile` a path to a file can be passed. The findings will be written
to this file instead of stdout.

##### outputFile Example

```sh
# findings.json
[
  {
    "url": "/v1/expected_jsonmissmatch",
    "error": "JSON mismatch",
    "diff": "@ [\"foo\"]\n- \"baz\"\n+ \"bar\"\n"
  }
]
```

## Exit codes

On successful execution `apijc` exits with code `0`.
On any issue the exit code will be `> 0`
