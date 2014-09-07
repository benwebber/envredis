# envredis

envredis runs processes in a modified environment by reading environment variables from a [Redis](http://redis.io/) database.

envredis is directly inspired by [envconsul](https://github.com/hashicorp/envconsul), and indirectly by [envdir](http://cr.yp.to/daemontools/envdir.html) and its [multiple ports](https://github.com/search?utf8=%E2%9C%93&q=envdir).

**envredis is not production-ready.**

## Installation

envredis is written in Go. Check out the [official documentation](https://golang.org/doc/install) for how to get started with the Go toolchain.

```
go get github.com/benwebber/envredis
go build github.com/benwebber/envredis
cp envredis /path/in/$PATH
```

## Usage

envredis stores configuration in a Redis [hash](http://redis.io/topics/data-types#hashes). Choose a name for the hash and set some initial variables:

```
$ envredis -k app set ENVIRONMENT=staging
$ envredis -k app set RATE_LIMIT=0
```

Run the process using the new configuration.

```
$ envredis -k app run env
...
ENVIRONMENT=staging
RATE_LIMIT=0
```

Of course, envredis doesn't care where the environment variables came from. We can configure the environment using any Redis client.


```
$ redis-cli HSET app AWS_ACCESS_KEY_ID AKIAIOSFODNN7EXAMPLE
```

```
$ envredis -k app list
ENVIRONMENT=staging
RATE_LIMIT=0
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
```

Finally, envredis assumes `run` is the default action, so the following works just as well:

```
$ envredis -k app env
...
ENVIRONMENT=staging
RATE_LIMIT=0
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
```

## Configuration

| Parameter | Environment Variable | Description | Example |
|------------|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------|
| `-u/--url` | `ENVREDIS_REDIS_URL` | URL of Redis instance | `redis://localhost:6379` (default) |
| `-k/--key` | `ENVREDIS_REDIS_KEY` | name of key storing application configuration | `next-big-thing-production` (default: current directory) |
| `--posix` | `ENVREDIS_POSIX` | transform variable names to adhere to the [POSIX standard](http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap08.html#tag_08) (valid for `set`, `list`, and `run`) | `--posix` or `ENVREDIS_POSIX=1` |

## Managing the environment

envredis provides a number of commands to manage an application's environment.

### set

Set an environment variable.

`set` accepts variables as `NAME=value` or `NAME value`.

```
$ envredis set FOO=bar
$ envredis set BAR baz
```

### get

Return the value of an environment variable.

```
$ envredis set BAZ=quux
$ envredis get BAZ
quux
```

### list

List environment variables.

```
$ envredis list
FOO=bar
BAR=baz
BAZ=quux
```

### delete

Delete an environment variable.

```
$ envredis delete FOO
$ envredis list
BAR=baz
BAZ=quux
```

### clear

Clear all environment variables.

```
$ envredis clear
$ envredis list
```

## Contributing

envredis is very rough around the edges. Feel free to open [issues](/benwebber/envredis/issues) for bugs or questions.

Pull requests are more than welcome.
