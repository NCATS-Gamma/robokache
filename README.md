# Robokache

The Q&A store for qgraph.

Workflow:

1. authenticate via JWT (Google, GitHub, etc.)
2. push/get your files

## Getting started

### Prebuilt Images

Prebuilt container images can be found in the following registry: [github.com/orgs/NCATS-Gamma/packages/container/package/robokache](https://github.com/orgs/NCATS-Gamma/packages/container/package/robokache). 

You can pull the latest image using the following command:

```
docker pull ghcr.io/ncats-gamma/robokache
```

### Local Docker Build

Build the image:

```bash
>> docker build -t robokache .
```

Run the image: 


```bash
>> docker run -it --name robokache -p 8080:80 robokache
```

### Native

Install:

```bash
>> go get -t ./...
```

Run:

```bash
>> go run ./cmd
```

* Go to <http://localhost:8080/>
* Sign in via the buttons at the top of the page
* Copy ID token into authentication field
* Have fun

## Testing

Set up testing certificate:

```bash
>> openssl req -new -newkey rsa:1024 -days 365 -nodes -x509 -keyout test/certs/test.key -out test/certs/test.cert
```

Run tests and print coverage:

```bash
>> go test ./internal/robokache -coverprofile=cover.out
>> go tool cover -func=cover.out
```

## How it works

### Security

* Auth0 Sign-in
* document visibility levels:
  * private (1) - only the owner
  * shareable (2) - anyone with the link
  * public (3) - anyone
* visibility is assigned to both questions and answers
  * the effective visibility of an answer is min(answer.visibility, question.visibility)
