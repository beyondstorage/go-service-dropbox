# go-service-dropbox

[Dropbox](https://www.dropbox.com) service support for [go-storage](https://github.com/beyondstorage/go-storage).

## Notes

**This package has been moved to [go-storage](https://github.com/beyondstorage/go-storage/tree/master/services/dropbox).**

```shell
go get go.beyondstorage.io/services/dropbox/v3
```


## Install

```go
go get github.com/beyondstorage/go-service-dropbox/v2
```

## Usage

```go
import (
	"log"

	_ "github.com/beyondstorage/go-service-dropbox/v2"
	"github.com/beyondstorage/go-storage/v4/services"
)

func main() {
	store, err := services.NewStoragerFromString("dropbox:///path/to/workdir?credential=apikey:<apikey>")
	if err != nil {
		log.Fatal(err)
	}
	
	// Write data from io.Reader into hello.txt
	n, err := store.Write("hello.txt", r, length)
}
```

- See more examples in [go-storage-example](https://github.com/beyondstorage/go-storage-example).
- Read [more docs](https://beyondstorage.io/docs/go-storage/services/dropbox) about go-service-dropbox.
