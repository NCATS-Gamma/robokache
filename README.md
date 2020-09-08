# Robokache

The Q&A store for ROBOKOP.

Workflow:

1. authenticate via JWT (Google, Facebook, etc.)
2. push/get your files

## Getting started

### Run via Docker

Build the image:

```bash
>> docker build -t robokache .
```

Run the image: 

```bash
>> docker run -it -p 8080:8080 robokache
```

* Got to <http://lvh.me:8080/>
* Copy ID token from developer tools into authentication field
* Have fun

### Test via Docker

Set up testing certificate:
```bash
>> openssl req -new -newkey rsa:1024 -days 365 -nodes -x509 -keyout test/certs/test.key -out test/certs/test.cert
```

Make sure to re-build the image to include your certificates.

Run tests and print coverage:
```bash
>> docker run robokache sh -c "go test ./internal/robokache -coverprofile=cover.out; go tool cover -func=cover.out"
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
