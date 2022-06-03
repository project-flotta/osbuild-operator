package iso_packaging_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIsoPackaging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IsoPackaging Suite")
}
