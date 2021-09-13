package sync

type Syncer interface {
	Sync() error
}
