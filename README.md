# concron

**Con**tainerized **cron**. Golang native scheduler to run repeated command inside containers (Docker, k8s)

## command-line

examples:
```
# run with config file 'tasks.yaml' located in current directory
concron -c tasks.yaml
# show options
concron -h
```

arguments:
```
-c <config file> : config file, YAML format
-h               : show help
-p <http port>   : http port for http server (default: 8080)
-debug           : show debug logs
```

## http server endpoint

- `/healthz` - health endpoint, returns code `200` with text `OK`. Useful for kubernetes pods ready/live probes.

## config format

example:

see [tasks.yaml](tasks.yaml)

description:

_in work..._