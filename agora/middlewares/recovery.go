package middlewares

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/op/go-logging"
)

// Recovery is a Negroni middleware that recovers from any panics and writes a 500 if there was one.
type Recovery struct {
	Logger     *logging.Logger
	PrintStack bool
}

// NewRecovery returns a new instance of Recovery
func NewRecovery() *Recovery {
	return &Recovery{
		Logger:     logging.MustGetLogger("agora.request.errors"),
		PrintStack: true,
	}
}

func (rec *Recovery) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	defer func() {
		if err := recover(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			stack := debug.Stack()

			rec.Logger.Critical(fmt.Sprintf("%s\n%s", err, stack))

			if rec.PrintStack {
				fmtStr := `{"message":"%s","stack":"%s","type":"error"}`
				fmt.Fprintf(rw, fmtStr, err, stack)
			}
		}
	}()

	next(rw, r)
}
