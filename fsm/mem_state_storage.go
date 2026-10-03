package fsm

type memStateStorage struct{}

func newMemStateStorage() *memStateStorage {
	return &memStateStorage{}
}

func (s *memStateStorage) GetState() {}

func (s *memStateStorage) SetState() {}
