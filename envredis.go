package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/fzzy/radix/redis"
)

type wrapped func(*redis.Client, *cli.Context) (int, error)

// Regular expression to match invalid characters in environment variable
// names.
// See: http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap08.html
var InvalidRegexp = regexp.MustCompile(`(^[0-9]|[^A-Z0-9_])`)

// Wrap Redis functions to automatically open and close the connection to the
// Redis instance.
func redisCommand(f wrapped, ctx *cli.Context) {
	u, err := url.Parse(ctx.GlobalString("url"))
	if err != nil {
		log.Fatal(err)
	}
	client, err := redis.Dial("tcp", u.Host)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	f(client, ctx)
}

// Start a child process with environment variables from Redis.
func run(client *redis.Client, ctx *cli.Context) (ret int, err error) {
	// Load application configuration from Redis.
	config, err := client.Cmd("HGETALL", ctx.GlobalString("key")).Hash()
	if err != nil {
		log.Fatal(err)
	}
	// Concatenate the current environment with the configuration in Redis.
	currentEnv := os.Environ()
	configEnv := []string{}
	childEnv := make([]string, len(currentEnv), len(currentEnv)+len(configEnv))
	copy(childEnv, currentEnv)
	for k, v := range config {
		if ctx.GlobalBool("posix") {
			k = strings.ToUpper(k)
			k = InvalidRegexp.ReplaceAllString(k, "_")
		}
		childEnv = append(childEnv, fmt.Sprintf("%s=%s", k, v))
	}
	// exec() the child process.
	var child *exec.Cmd
	child = exec.Command(ctx.Args()[0], ctx.Args()[1:]...)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.Env = childEnv
	err = child.Start()
	if err != nil {
		ret = 111
	}
	return ret, err
}

// List all environment variables stored in Redis.
func list(client *redis.Client, ctx *cli.Context) (ret int, err error) {
	// Load application configuration from Redis.
	config, err := client.Cmd("HGETALL", ctx.GlobalString("key")).Hash()
	if err != nil {
		log.Fatal(err)
	}
	// Output environment variables as key=value. Surround the values in quotes
	// if they contain whitespace.
	for k, v := range config {
		if len(strings.Fields(v)) >= 2 {
			v = fmt.Sprintf("'%s'", v)
		}
		fmt.Printf("%s=%s\n", k, v)
	}
	return ret, err
}

// Retrieve a specific environment variable from Redis.
func get(client *redis.Client, ctx *cli.Context) (ret int, err error) {
	envvar := ctx.Args()[0]
	reply, err := client.Cmd("HGET", ctx.GlobalString("key"), envvar).Str()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply)
	return ret, err
}

// Set a specific environment variable in Redis.
func set(client *redis.Client, ctx *cli.Context) (ret int, err error) {
	envvar, value := ctx.Args()[0], ctx.Args()[1]
	_, err = client.Cmd("HSET", ctx.GlobalString("key"), envvar, value).Int()
	if err != nil {
		log.Fatal(err)
	}
	return ret, err
}

// Delete an environment variable from Redis.
func del(client *redis.Client, ctx *cli.Context) (ret int, err error) {
	envvar := ctx.Args()[0]
	_, err = client.Cmd("HDEL", ctx.GlobalString("key"), envvar).Int()
	if err != nil {
		log.Fatal(err)
	}
	return ret, err
}

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	app := cli.NewApp()
	app.Name = "envredis"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url, u",
			Value: "redis://localhost:6379",
			Usage: "Redis connection URL",
		},
		cli.StringFlag{
			Name:  "key, k",
			Value: filepath.Base(pwd),
			Usage: "name of Redis hash storing configuration",
		},
		cli.BoolFlag{
			Name:  "posix",
			Usage: "convert all variable names to to follow the POSIX standard",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "run a command",
			Action: func(ctx *cli.Context) {
				redisCommand(run, ctx)
			},
		},
		{
			Name:  "list",
			Usage: "list environment variables",
			Action: func(ctx *cli.Context) {
				redisCommand(list, ctx)
			},
		},
		{
			Name:  "get",
			Usage: "get an environment variable",
			Action: func(ctx *cli.Context) {
				redisCommand(get, ctx)
			},
		},
		{
			Name:  "set",
			Usage: "set an environment variable",
			Action: func(ctx *cli.Context) {
				redisCommand(set, ctx)
			},
		},
		{
			Name:  "delete",
			Usage: "delete an environment variable",
			Action: func(ctx *cli.Context) {
				redisCommand(del, ctx)
			},
		},
	}
	app.Run(os.Args)
}
