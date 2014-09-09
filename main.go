package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
)

func NewConfigFromCLIContext(ctx *cli.Context) *Config {
	var command string
	args := ctx.Args()
	switch ctx.Command.Name {
	case "list":
		command = "HGETALL"
	case "get":
		command = "HGET"
	case "set":
		command = "HSET"
	case "delete":
		command = "HDEL"
	case "clear":
		command = "DEL"
	default:
		command = "HGETALL"
	}
	if command == "HGETALL" {
		// Remove arguments before passing to redis.Client, otherwise HGETALL
		// will throw an error.
		args = []string{}
	}
	return &Config{
		RedisURL: ctx.GlobalString("url"),
		Command:  command,
		Key:      ctx.GlobalString("key"),
		Args:     args,
		POSIX:    ctx.GlobalBool("posix"),
	}
}

// Start a process with environment variables from Redis.
func runCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		log.Fatal("you must provide a command name")
	}
	config := NewConfigFromCLIContext(ctx)
	return RunWithEnvVars(config, ctx.Args()[0], ctx.Args()[1:]...)
}

// List all environment variables stored in Redis.
func listCommand(ctx *cli.Context) (ret int, err error) {
	config := NewConfigFromCLIContext(ctx)
	return ListEnvVar(config)
}

// Retrieve a specific environment variable from Redis.
func getCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		log.Fatal("you must provide a variable name")
	}
	config := NewConfigFromCLIContext(ctx)
	return GetEnvVar(config)
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
	config.Args = []string{envvar, value}
	return SetEnvVar(config)
}

// Delete an environment variable from Redis.
func deleteCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		log.Fatal("you must provide a variable name")
	}
	config := NewConfigFromCLIContext(ctx)
	return DeleteEnvVar(config)
}

// Clear an application's environment variables from Redis.
func clearCommand(ctx *cli.Context) (ret int, err error) {
	config := NewConfigFromCLIContext(ctx)
	return ClearEnvVar(config)
}

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	app := cli.NewApp()
	app.Name = "envredis"
	app.Version = Version
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
