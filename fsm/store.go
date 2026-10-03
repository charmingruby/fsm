package fsm

type memStateStore struct{}

func newMemStateStore() *memStateStore {
	return &memStateStore{}
}

func (s *memStateStore) GetState() {}

func (s *memStateStore) SetState() {}
