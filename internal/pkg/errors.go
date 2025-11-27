package pkg

import "fmt"

type ErrDBProcedure struct {
	Cause string
	Info  string
	Err   error
}

func (e ErrDBProcedure) Error() string {
	return fmt.Sprintf("%s; got error: %s; info: %s", e.Cause, e.Err, e.Info)
}
