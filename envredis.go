package main

import (
	"fmt"
	"log"
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

func RunWithEnvVars(config *Config, command string, args ...string) (ret int, err error) {
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
	child = exec.Command(command, args...)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.Env = childEnv
	err = child.Start()
	if err != nil {
		ret = 111
	}
	return ret, err
}

func ListEnvVar(config *Config) (ret int, err error) {
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

func GetEnvVar(config *Config) (ret int, err error) {
	reply, err := redisCommand(config).Str()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply)
	return ret, err
}

func SetEnvVar(config *Config) (ret int, err error) {
	if config.POSIX {
		config.Args[0] = makePOSIXCompatible(config.Args[0])
	}
	_, err = redisCommand(config).Int()
	if err != nil {
		log.Fatal(err)
	}
	return ret, err
}

func DeleteEnvVar(config *Config) (ret int, err error) {
	_, err = redisCommand(config).Int()
	if err != nil {
		log.Fatal(err)
	}
	return ret, err
}

func ClearEnvVar(config *Config) (ret int, err error) {
	ret, err = redisCommand(config).Int()
	if err != nil {
		log.Fatal(err)
	}
	return ret, err
}
