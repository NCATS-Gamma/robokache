# Robokache

The Q&A store for ROBOKOP.

Workflow:

1. authenticate via JWT (Google, Facebook, etc.)
2. push/get your files

## Getting started

### Docker

Build the image:

```bash
>> docker build -t robokache .
```

Run the image: 


```bash
>> docker run -it --name robokache -p 8080:8080 robokache
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

* Got to <http://lvh.me:8080/>
* Copy ID token from developer tools into authentication field
* Have fun

## Testing

Set up testing certificate (to emulate Google Auth):

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

* Google Sign-in
* document visibility levels:
  * private (1) - only the owner
  * shareable (2) - anyone with the link
  * public (3) - anyone
* visibility is assigned to both questions and answers
  * the effective visibility of an answer is min(answer.visibility, question.visibility)
