package minecraft

import "github.com/openhost/cli/internal/gamestatus"

type Checker struct{}

func (c *Checker) GameName() string {
	return "minecraft"
}

func (c *Checker) Check(_ gamestatus.Target) (*gamestatus.Status, error) {
	return &gamestatus.Status{
		State:  gamestatus.StateUnknown,
		Detail: "minecraft status check not implemented yet",
	}, nil
}

func init() {
	gamestatus.Register(&Checker{})
}
