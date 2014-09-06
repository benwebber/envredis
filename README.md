# envredis

envredis runs an application using environment variables stored in a Redis database. It also provides commands to manage and export environment variables.

## Usage

envredis stores configuration in a Redis hash. Choose a name for the hash and set some initial variables:

```
redis-cli HSET app ENVIRONMENT staging
redis-cli HSET app RATE_LIMIT 0
```

Run the child process using the new configuration.

```
envredis --name app run env
...
ENVIRONMENT=staging
RATE_LIMIT=0
```
