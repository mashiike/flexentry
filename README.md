# flexentry
![Latest GitHub release](https://img.shields.io/github/release/mashiike/flexentry.svg)
![Github Actions test](https://github.com/mashiike/flexentry/workflows/Test/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/mashiike/flexentry)](https://goreportcard.com/report/mashiike/flexentry) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/mashiike/flexentry/blob/master/LICENSE)

Flexible entry point for Amazon ECS Task and Amazon Lambda container images

## Usage 

```Dockerfile
FROM golang:1.17-buster

RUN apt-get update && \
    apt-get install -y unzip && \
    apt-get clean

RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -R aws && \
    rm awscliv2.zip

ARG FLEXENTRY_VERSION=0.0.0
RUN curl -L https://github.com/mashiike/flexentry/releases/download/v${FLEXENTRY_VERSION}/flexentry_v${FLEXENTRY_VERSION}_linux_amd64.tar.gz | tar zxvf - && \
    install flexentry_v${FLEXENTRY_VERSION}_linux_amd64/flexentry /usr/local/bin/

ENTRYPOINT ["flexentry"]
```

Basically, all you have to do is specify the entry point of the container image.

### Run on ECS Task

```json
{
   "containerDefinitions": [ 
      { 
         "command": [
            "aws --version"
         ],
         "essential": true,
         "image": "<ecr image path>",
         "name": "sample-app"
      }
   ],
   "cpu": "256",
   "executionRoleArn": "arn:aws:iam::012345678910:role/ecsTaskExecutionRole",
   "family": "sample-task-definition",
   "memory": "512",
   "networkMode": "awsvpc",
   "requiresCompatibilities": [ 
       "FARGATE" 
    ]
}
```

Decide what to execute with `command`, as in the task definition above.

### Run on Lambda with container image

If the environment variable `FLEXENTRY_COMMAND` is specified, the command will be executed.
Otherwise, the command to be executed will be determined according to the payload of the event.

```json
{
    "cmd": "aws --version",
    "description": "this is sample"
}
```

If the event payload is a string, it will be interpreted as a command.
Otherwise, by default, it looks at the `cmd` key to decide which command to execute.
To change the key, specify a jq expression for reference with `FLEXENTRY_COMMAND_JQ_EXPR`.
For example `export FLEXENTRY_COMMAND_JQ_EXPR=".command"` 

When executed by Amazon Lambda, the event payload is passed directly to the standard input as JSON data.

## LICENSE

MIT

