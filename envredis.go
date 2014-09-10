package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

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

// Transform an environment variable name to follow the POSIX standard.
func makePOSIXCompatible(envVar string) (newEnvVar string) {
	// Regular expression to match invalid characters in environment variable
	// names.
	// See: http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap08.html
	invalidRegexp := regexp.MustCompile(`(^[0-9]|[^A-Z0-9_])`)
	newEnvVar = strings.ToUpper(envVar)
	newEnvVar = invalidRegexp.ReplaceAllString(newEnvVar, "_")
	return
}

// Wrap Redis commands to automatically open and close the connection to the
// Redis instance.
func redisCommand(config *Config) (reply *redis.Reply, err error) {
	u, err := url.Parse(config.RedisURL)
	if err != nil {
		return
	}
	client, err := redis.Dial("tcp", u.Host)
	if err != nil {
		return
	}
	defer client.Close()
	args := []string{config.Key}
	args = append(args, config.Args...)
	reply = client.Cmd(config.Command, args)
	return
}

func RunWithEnvVars(config *Config, command string, args ...string) (ret int, err error) {
	// Load application configuration from Redis.
	envConfig, err := GetEnvVarsMap(config)
	// Concatenate the current environment with the configuration in Redis.
	currentEnv := os.Environ()
	childEnv := make([]string, len(currentEnv), len(currentEnv)+len(envConfig))
	copy(childEnv, currentEnv)
	for k, v := range envConfig {
		if config.POSIX {
			k = makePOSIXCompatible(k)
		}
		childEnv = append(childEnv, fmt.Sprintf("%s=%s", k, v))
	}
	// exec() the child process.
	var child *exec.Cmd
	child = exec.Command(command, args...)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.Env = childEnv
	err = child.Start()
	if err != nil {
		ret = 111
	}
	return
}

func GetEnvVarsMap(config *Config) (env map[string]string, err error) {
	reply, err := redisCommand(config)
	if err != nil {
		return
	}
	env, err = reply.Hash()
	return
}

func GetEnvVarsArray(config *Config) (envVars []string, err error) {
	// Load application configuration from Redis.
	env, err := GetEnvVarsMap(config)
	if err != nil {
		return
	}
	envVars = make([]string, 0, len(env))
	// Output environment variables as key=value. Surround the values in quotes
	// if they contain whitespace.
	for k, v := range env {
		if config.POSIX {
			k = makePOSIXCompatible(k)
		}
		if len(strings.Fields(v)) >= 2 {
			v = fmt.Sprintf("'%s'", v)
		}
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}
	return
}

func GetEnvVar(config *Config) (envVar string, err error) {
	reply, err := redisCommand(config)
	if err != nil {
		return
	}
	// Field does not exist.
	if reply.Type == redis.NilReply {
		err = fmt.Errorf("variable '%v' does not exist for key %v", config.Args[0], config.Key)
		return
	}
	envVar, err = reply.Str()
	return
}

func SetEnvVar(config *Config) (ret int, err error) {
	if config.POSIX {
		config.Args[0] = makePOSIXCompatible(config.Args[0])
	}
	reply, err := redisCommand(config)
	if err != nil {
		return
	}
	ret, err = reply.Int()
	return
}

func DeleteEnvVar(config *Config) (ret int, err error) {
	reply, err := redisCommand(config)
	if err != nil {
		return
	}
	ret, err = reply.Int()
	return
}

func ClearEnvVars(config *Config) (ret int, err error) {
	reply, err := redisCommand(config)
	if err != nil {
		return
	}
	ret, err = reply.Int()
	return
}
