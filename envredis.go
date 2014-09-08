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

// Config holds the envredis configuration.
type Config struct {
	RedisURL string   // Redis connection URL
	Command  string   // Redis command
	Key      string   // Redis key
	Args     []string // Redis command arguments
	POSIX    bool     // whether POSIX compatibility is on or off
}

func NewConfig() *Config {
	return &Config{}
}

func NewConfigFromCLIContext(ctx *cli.Context) *Config {
	var command string
	switch ctx.Command.Name {
	case "list", "run":
		command = "HGETALL"
	case "get":
		command = "HGET"
	case "set":
		command = "HSET"
	case "delete":
		command = "HDEL"
	case "clear":
		command = "DEL"
	}
	return &Config{
		RedisURL: ctx.GlobalString("url"),
		Command:  command,
		Key:      ctx.GlobalString("key"),
		Args:     ctx.Args(),
		POSIX:    ctx.GlobalBool("posix"),
	}
}

// Transform an environment variable name to follow the POSIX standard.
func makePOSIXCompatible(envvar string) string {
	// Regular expression to match invalid characters in environment variable
	// names.
	// See: http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap08.html
	var InvalidRegexp = regexp.MustCompile(`(^[0-9]|[^A-Z0-9_])`)
	envvar = strings.ToUpper(envvar)
	envvar = InvalidRegexp.ReplaceAllString(envvar, "_")
	return envvar
}

// Wrap Redis commands to automatically open and close the connection to the
// Redis instance.
func redisCommand(config *Config) *redis.Reply {
	u, err := url.Parse(config.RedisURL)
	if err != nil {
		log.Fatal(err)
	}
	client, err := redis.Dial("tcp", u.Host)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	args := []string{config.Key}
	args = append(args, config.Args...)
	return client.Cmd(config.Command, args)
}

// Start a process with environment variables from Redis.
func runCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		log.Fatal("you must provide a command name")
	}
	config := NewConfigFromCLIContext(ctx)
	// Remove arguments before passing to redis.Client, otherwise HGETALL will
	// throw an error.
	config.Args = []string{}
	// Load application configuration from Redis.
	envConfig, err := redisCommand(config).Hash()
	if err != nil {
		log.Fatal(err)
	}
	// Concatenate the current environment with the configuration in Redis.
	currentEnv := os.Environ()
	configEnv := []string{}
	childEnv := make([]string, len(currentEnv), len(currentEnv)+len(configEnv))
	copy(childEnv, currentEnv)
	for k, v := range envConfig {
		if config.POSIX {
			k = makePOSIXCompatible(k)
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
func listCommand(ctx *cli.Context) (ret int, err error) {
	config := NewConfigFromCLIContext(ctx)
	// Load application configuration from Redis.
	envConfig, err := redisCommand(config).Hash()
	// Output environment variables as key=value. Surround the values in quotes
	// if they contain whitespace.
	for k, v := range envConfig {
		if config.POSIX {
			k = makePOSIXCompatible(k)
		}
		if len(strings.Fields(v)) >= 2 {
			v = fmt.Sprintf("'%s'", v)
		}
		fmt.Printf("%s=%s\n", k, v)
	}
	return ret, err
}

// Retrieve a specific environment variable from Redis.
func getCommand(ctx *cli.Context) (ret int, err error) {
	config := NewConfigFromCLIContext(ctx)
	if len(ctx.Args()) == 0 {
		log.Fatal("you must provide a variable name")
	}
	reply, err := redisCommand(config).Str()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply)
	return ret, err
}

// Set a specific environment variable in Redis.
func setCommand(ctx *cli.Context) (ret int, err error) {
	var (
		envvar string
		value  string
	)
	if len(ctx.Args()) == 1 {
		// set FOO=bar
		args := strings.Split(ctx.Args()[0], "=")
		if len(args) == 2 {
			envvar, value = args[0], args[1]
		} else {
			log.Fatal("you must provide a variable name and value")
		}
	} else if len(ctx.Args()) >= 2 {
		// set FOO bar
		envvar, value = ctx.Args()[0], ctx.Args()[1]
	} else {
		log.Fatal("you must provide a variable name and value")
	}
	config := NewConfigFromCLIContext(ctx)
	if config.POSIX {
		envvar = makePOSIXCompatible(envvar)
	}
	config.Args = []string{envvar, value}
	_, err = redisCommand(config).Int()
	if err != nil {
		log.Fatal(err)
	}
	return ret, err
}

// Delete an environment variable from Redis.
func deleteCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		log.Fatal("you must provide a variable name")
	}
	config := NewConfigFromCLIContext(ctx)
	_, err = redisCommand(config).Int()
	if err != nil {
		log.Fatal(err)
	}
	return ret, err
}

// Clear an application's environment variables from Redis.
func clearCommand(ctx *cli.Context) (ret int, err error) {
	config := NewConfigFromCLIContext(ctx)
	ret, err = redisCommand(config).Int()
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
	app.Version = "0.1.0"
	app.Usage = "Load process environments from Redis."
	app.Action = func(ctx *cli.Context) {
		runCommand(ctx)
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "url, u",
			Value:  "redis://localhost:6379",
			Usage:  "Redis connection URL",
			EnvVar: "ENVREDIS_REDIS_URL",
		},
		cli.StringFlag{
			Name:   "key, k",
			Value:  filepath.Base(pwd),
			Usage:  "name of Redis hash storing configuration",
			EnvVar: "ENVREDIS_REDIS_KEY",
		},
		cli.BoolFlag{
			Name:   "posix",
			Usage:  "make all variable names follow the POSIX standard",
			EnvVar: "ENVREDIS_POSIX",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "run a command",
			Action: func(ctx *cli.Context) {
				runCommand(ctx)
			},
		},
		{
			Name:  "list",
			Usage: "list environment variables",
			Action: func(ctx *cli.Context) {
				listCommand(ctx)
			},
		},
		{
			Name:  "get",
			Usage: "get an environment variable",
			Action: func(ctx *cli.Context) {
				getCommand(ctx)
			},
		},
		{
			Name:  "set",
			Usage: "set an environment variable",
			Action: func(ctx *cli.Context) {
				setCommand(ctx)
			},
		},
		{
			Name:  "delete",
			Usage: "delete an environment variable",
			Action: func(ctx *cli.Context) {
				deleteCommand(ctx)
			},
		},
		{
			Name:  "clear",
			Usage: "clear all environment variables",
			Action: func(ctx *cli.Context) {
				clearCommand(ctx)
			},
		},
	}
	app.Run(os.Args)
}
