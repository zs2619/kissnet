package kissnet

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestStack(t *testing.T) {
	logrus.Error(string(stack(3)))
}
