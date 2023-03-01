package repo

type Lister interface {
	List() ([]*Repo, error)
}
