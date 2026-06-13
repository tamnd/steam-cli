package cli

import (
	"errors"

	"github.com/tamnd/steam-cli/steam"
)

func isNotFound(err error) bool {
	return errors.Is(err, steam.ErrNotFound)
}
