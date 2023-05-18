package executer

import "context"

type GoMigrate struct {
	db DB
}

func NewGoMigrate(db DB) *GoMigrate {
	return &GoMigrate{
		db: db,
	}
}

//nolint:all
func (sm *GoMigrate) UpExec(ctx context.Context, path string) error {
	return nil
}

//nolint:all
func (sm *GoMigrate) DownExec(ctx context.Context, path string) error {
	return nil
}
