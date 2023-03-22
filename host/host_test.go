package host

import (
	"testing"
)

func TestLocal(t *testing.T) {
	host := Local{}

	testHost(t, host)
}
