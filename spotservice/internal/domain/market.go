package domain

type Market struct {
	id     string
	active bool
}

func NewMarket(id string, active bool) *Market {
	return &Market{id: id, active: active}
}

func (s Market) ID() string {
	return s.id
}
func (s Market) Active() bool {
	return s.active
}
