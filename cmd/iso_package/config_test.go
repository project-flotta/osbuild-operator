package main_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kdomanski/iso9660/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	iso_package "github.com/project-flotta/osbuild-operator/cmd/iso_package"
)

var _ = Describe("Config", Ordered, func() {
	var (
		config *iso_package.Config
		server *http.Server
		port   int
	)

	BeforeAll(func() {
		server, port = IsoServer()
	})

	AfterAll(func() {
		_ = server.Shutdown(context.TODO())
	})

	BeforeEach(func() {
		config = &iso_package.Config{
			Source:    fmt.Sprintf("http://127.0.0.1:%d/Fedora.iso", port),
			Kickstart: "../../testdata/test.ks",
			Arch:      "x86",
		}
	})

	AfterEach(func() {
		config.CleanAll()
	})

	checkIsoImage := func(path string) {
		dirName, err := ioutil.TempDir("/tmp/", "isounittest")
		Expect(err).NotTo(HaveOccurred())
		f, err := os.Open(path)
		Expect(err).NotTo(HaveOccurred())

		defer func() {
			os.RemoveAll(dirName)
			f.Close()
		}()

		err = util.ExtractImageToDirectory(f, dirName)
		Expect(err).NotTo(HaveOccurred())

		isolinux := filepath.Join(dirName, "ISOLINUX/ISOLINUX.CFG")
		Expect(isolinux).Should(BeAnExistingFile())

		content, err := ioutil.ReadFile(isolinux)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(ContainSubstring("cdrom:/init.ks"))

		grub := filepath.Join(dirName, "EFI/BOOT/GRUB.CFG")
		content, err = ioutil.ReadFile(grub)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(ContainSubstring("cdrom:/init.ks"))
	}

	It("Got x86 iso working correctly", func() {

		// given

		// when
		path, err := config.Run()

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(path).Should(BeAnExistingFile())

		checkIsoImage(path)
	})

	It("Got arm iso working correctly", func() {
		// given
		config.Arch = "aarch"

		// when
		path, err := config.Run()

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(path).Should(BeAnExistingFile())

		checkIsoImage(path)
	})

})

func IsoServer() (*http.Server, int) {
	var isoFile string
	rootdir := "../../testdata/"

	files, err := ioutil.ReadDir(rootdir)
	Expect(err).NotTo(HaveOccurred())
	for _, x := range files {
		parts := strings.Split(x.Name(), ".")
		if len(parts) == 0 {
			continue
		}
		if strings.ToLower(parts[len(parts)-1]) == "iso" {
			isoFile = filepath.Join(rootdir, x.Name())
		}
	}

	Expect(isoFile).NotTo(Equal(""), "Iso file should be located on testdata")

	mux := http.NewServeMux()
	handler := http.FileServer(http.Dir("../../testdata/"))
	mux.Handle("/", handler)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	srv := &http.Server{
		Handler: mux,
	}
	go func() {
		err = srv.Serve(l)
	}()

	return srv, l.Addr().(*net.TCPAddr).Port
}
