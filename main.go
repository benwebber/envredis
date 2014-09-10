package main

import (
	"errors"
	"fmt"
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

// Output to standard error and exit.
func die(ret int, messages ...interface{}) {
	fmt.Fprintln(os.Stderr, messages...)
	os.Exit(ret)
}

// Start a process with environment variables from Redis.
func runCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		ret = 1
		err = errors.New("you must provide a command name")
		return
	}
	config := NewConfigFromCLIContext(ctx)
	ret, err = RunWithEnvVars(config, ctx.Args()[0], ctx.Args()[1:]...)
	return
}

// List all environment variables stored in Redis.
func listCommand(ctx *cli.Context) (ret int, err error) {
	config := NewConfigFromCLIContext(ctx)
	envVars, err := GetEnvVarsArray(config)
	for _, v := range envVars {
		fmt.Println(v)
	}
	return
}

// Retrieve a specific environment variable from Redis.
func getCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		ret = 1
		err = errors.New("you must provide a variable name")
		return
	}
	config := NewConfigFromCLIContext(ctx)
	envVar, err := GetEnvVar(config)
	if envVar != "" {
		fmt.Println(envVar)
	}
	return
}

// Set a specific environment variable in Redis.
func setCommand(ctx *cli.Context) (ret int, err error) {
	var (
		envVar string
		value  string
	)
	if len(ctx.Args()) == 1 {
		// set FOO=bar
		args := strings.Split(ctx.Args()[0], "=")
		if len(args) == 2 {
			envVar, value = args[0], args[1]
		} else {
			ret = 1
			err = errors.New("you must provide a variable name and value")
			return
		}
	} else if len(ctx.Args()) >= 2 {
		// set FOO bar
		envVar, value = ctx.Args()[0], ctx.Args()[1]
	} else {
		ret = 1
		err = errors.New("you must provide a variable name and value")
		return
	}
	config := NewConfigFromCLIContext(ctx)
	config.Args = []string{envVar, value}
	n, err := SetEnvVar(config)
	if n == 1 {
		fmt.Printf("set new variable %v=%v\n", envVar, value)
	} else {
		fmt.Printf("set existing variable %v=%v\n", envVar, value)
	}
	return
}

// Delete an environment variable from Redis.
func deleteCommand(ctx *cli.Context) (ret int, err error) {
	if len(ctx.Args()) == 0 {
		err = errors.New("you must provide a variable name")
		return
	}
	config := NewConfigFromCLIContext(ctx)
	n, err := DeleteEnvVar(config)
	fmt.Printf("deleted %v variable(s) from key %v\n", n, config.Key)
	return
}

// Clear an application's environment variables from Redis.
func clearCommand(ctx *cli.Context) (ret int, err error) {
	config := NewConfigFromCLIContext(ctx)
	n, err := ClearEnvVars(config)
	fmt.Printf("deleted %v key(s)\n", n)
	return ret, err
}

func main() {
	ret, err := realMain()
	if err != nil {
		die(ret, err)
	}
	os.Exit(ret)
}

func realMain() (ret int, err error) {
	pwd, err := os.Getwd()
	if err != nil {
		ret = 1
		return
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
				ret, err = runCommand(ctx)
				if err != nil {
					die(ret, err.Error())
				}
			},
		},
		{
			Name:  "list",
			Usage: "list environment variables",
			Action: func(ctx *cli.Context) {
				ret, err = listCommand(ctx)
				if err != nil {
					die(ret, err.Error())
				}
			},
		},
		{
			Name:  "get",
			Usage: "get an environment variable",
			Action: func(ctx *cli.Context) {
				ret, err = getCommand(ctx)
				if err != nil {
					die(ret, err.Error())
				}
			},
		},
		{
			Name:  "set",
			Usage: "set an environment variable",
			Action: func(ctx *cli.Context) {
				ret, err = setCommand(ctx)
				if err != nil {
					die(ret, err.Error())
				}
			},
		},
		{
			Name:  "delete",
			Usage: "delete an environment variable",
			Action: func(ctx *cli.Context) {
				ret, err = deleteCommand(ctx)
				if err != nil {
					die(ret, err.Error())
				}
			},
		},
		{
			Name:  "clear",
			Usage: "clear all environment variables",
			Action: func(ctx *cli.Context) {
				ret, err = clearCommand(ctx)
				if err != nil {
					die(ret, err.Error())
				}
			},
		},
	}
	err = app.Run(os.Args)
	return
}
