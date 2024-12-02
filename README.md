# ffs

Fuzzy search your Firefox history. 

_This is a simple PoC, thrown together in sub 1h and currently only supporting Linux and the default Firefox profile._

## Usage

```sh
ffs <query>

# e.g.
ffs "linkedin.com/in"
ffs "github*poc"
```

## Install/Build

```sh
CGO_ENABLED=1 go install -ldflags="-s -w" github.com/rtfmkiesel/ffs@latest
```

```sh
git clone https://github.com/rtfmkiesel/ffs
cd fss

# assuming you have go/bin in your path
CGO_ENABLED=1 go install -ldflags="-s -w" .
fss

# else 
CGO_ENABLED=1 go build -ldflags="-s -w" .
./ffs
```
