package repo

import "fmt"

type Repo struct {
	Name     string
	ReadOnly bool
}

func (r *Repo) String() string {
	return fmt.Sprintf("Name: %q, ReadOnly: %t", r.Name, r.ReadOnly)
}
