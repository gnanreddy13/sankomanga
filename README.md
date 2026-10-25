# sankomanga

[![Tests](https://github.com/gnanreddy13/sankomanga/actions/workflows/test.yml/badge.svg)](https://github.com/gnanreddy13/sankomanga/actions/workflows/test.yml)

A minimal site for looking up Japanese manga: search a title, browse the
results, see the key facts. サンコー (sankō) means "reference".

![The sankomanga home page: the name in the middle of a black page, with popular manga covers floating around it](docs/screenshot.jpg)

Go (standard library only) and plain HTML, CSS and JavaScript, in a single
program.

## Run

Needs [Go](https://go.dev/dl/) 1.27+.

```bash
go run .
```

Open <http://localhost:8080>. Restart after changing anything in `web/`.

## Test

```bash
gofmt -l . && go vet ./... && go test -race ./...
```

## Docs

- [Design](docs/design.md): every design decision
- [API](docs/api-contract.md): endpoints and responses
- [Deploying](docs/deploying.md): hosting, settings and AniList's terms

Data from [AniList](https://anilist.co). Not affiliated with AniList.
