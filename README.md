## sqliteutil

> A simple Golang library for creating and migrating sqlite databases.

### Usage

```go
import (
	"github.com/LQR471814/sqliteutil"

	_ "modernc.org/sqlite"
)

func main() {
	// this will call "atlas" (https://atlasgo.io/) and automatically run migrations on the database
	sqliteutil.OpenAndMigrateSqlite("create table ...", "path/to/some.db")
}
```

