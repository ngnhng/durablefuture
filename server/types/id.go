package types

import "github.com/gofrs/uuid/v5"

type EventID uuid.UUID

func (id EventID) String() string {
	return id.String()
}
