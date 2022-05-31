package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kdomanski/iso9660/util"
)

const (
	tmpFolderPrefix = "iso_package_tmp"
)

type Builder struct {
	Source    string
	Kickstart string
	Arch      string

	tmpFolder      string
	isoSourceLabel string
	dstFolderPath  string
	finalIsoPath   string
}

// CleanAll remove all files from the temporal folder
func (c *Builder) CleanAll() {
	os.RemoveAll(c.tmpFolder)
}

// Run add the given Kickstart file to  given iso
func (c *Builder) Run() (string, error) {
	err := c.CheckConfig()
	if err != nil {
		return "", err
	}

	dirName, err := ioutil.TempDir("/tmp/", tmpFolderPrefix)
	if err != nil {
		return "", err
	}
	InfoLogger.Printf("Creating temp folder: %s", dirName)
	c.tmpFolder = dirName

	err = c.GetIso()
	if err != nil {
		return "", err
	}

	err = c.MountIso()
	if err != nil {
		return "", err
	}

	err = c.UpdateFiles()
	if err != nil {
		return "", err
	}

	path, err := c.CreateFinalIso()
	if err != nil {
		return "", err
	}
	return path, nil
}

func (c *Builder) Upload(creds *S3Config, targetOutput string) error {
	sess, err := creds.GetAwsSession()
	if err != nil {
		return fmt.Errorf("cannot get aws session: %v", err)
	}
	if c.finalIsoPath == "" {
		return errors.New("Final iso cannot be found")
	}

	err = creds.UploadFile(sess, targetOutput, c.finalIsoPath)
	if err != nil {
		return fmt.Errorf("cannot upload filename to S3: %v", err)
	}
	return err
}

// CheckConfig validates that current config struct is valid
func (c *Builder) CheckConfig() error {

	if c.Source == "" {
		return errors.New("Source file cannot be nil")
	}

	if c.Kickstart == "" {
		return errors.New("kickstart file cannot be nil")
	}

	_, err := os.Stat(c.Kickstart)
	if err != nil {
		return fmt.Errorf("kickstart file cannot be found: %v", err)
	}

	// @todo dependencies programs here geniso, xorriso, blkid
	return nil
}

// Mount the iso in the source_iso folder
func (c *Builder) MountIso() error {
	cmd := execCommand("blkid -s LABEL -o value %s", c.getIsoFilepath())
	if cmd.Failed() {
		return fmt.Errorf("cannot get iso label: %v", cmd.GetErrorMessage())
	}
	c.isoSourceLabel = strings.Trim(cmd.GetStdout(), "\n")

	sourceFolderPath := filepath.Join(c.tmpFolder, "source_iso")
	err := os.Mkdir(sourceFolderPath, 0775)
	if err != nil {
		return fmt.Errorf("cannot create source iso folder: %v", err)
	}

	c.dstFolderPath = filepath.Join(c.tmpFolder, "dest_iso")
	err = os.Mkdir(c.dstFolderPath, 0775)
	if err != nil {
		return fmt.Errorf("cannot create dest iso folder: %v", err)
	}

	f, err := os.Open(c.getIsoFilepath())
	if err != nil {
		return fmt.Errorf("cannot open source iso file: %v", err)
	}
	defer f.Close()

	if err = util.ExtractImageToDirectory(f, sourceFolderPath); err != nil {
		return fmt.Errorf("cannot mount source iso file: %v", err)
	}

	err = Dir(sourceFolderPath, c.dstFolderPath)
	if err != nil {
		return fmt.Errorf("cannot copy source iso files: %v", err)
	}
	return nil
}

// UpdateFiles copy the Kickstart file and update all isolinux and grub to
// parse it.
func (c *Builder) UpdateFiles() error {
	err := File(c.Kickstart, filepath.Join(c.dstFolderPath, "init.ks"))
	if err != nil {
		return fmt.Errorf("cannot copy kickstart file, %v", err)
	}

	grubPath := filepath.Join(c.dstFolderPath, "EFI/BOOT/GRUB.CFG")
	grub, err := AppendKickstartTogrub(grubPath, "init.ks")
	if err != nil {
		return fmt.Errorf("cannot append kickstart file to grub, %v", err)
	}

	err = os.WriteFile(grubPath, grub.Bytes(), 0600)
	if err != nil {
		// This error message is clear enough
		return err
	}

	isolinuxPath := filepath.Join(c.dstFolderPath, "/ISOLINUX/ISOLINUX.CFG")
	isolinux, err := AppendKickstartToIsoLinux(isolinuxPath, "init.ks")
	if err != nil {
		return fmt.Errorf("cannot append kickstart file to isolinux, %v", err)
	}
	err = os.WriteFile(isolinuxPath, isolinux.Bytes(), 0600)
	// This error message is clear enough
	return err
}

func (c *Builder) CreateFinalIso() (string, error) {
	finalPath := filepath.Join(c.tmpFolder, "final.iso")
	switch c.Arch {
	case x86ArchValue:
		err := c.genIso(finalPath, c.isoSourceLabel)
		c.finalIsoPath = finalPath
		return finalPath, err
	case armArchValue:
		err := c.XorrisogenIso(finalPath, c.isoSourceLabel)
		c.finalIsoPath = finalPath
		return finalPath, err
	default:
		return "", fmt.Errorf("Not valid arch to build the image: %s", c.Arch)
	}
}

func (c *Builder) XorrisogenIso(output string, label string) error {
	command := `xorriso \
    -as mkisofs \
    -V %s \
    -r -J -joliet-long -cache-inodes -efi-boot-part  \
    --efi-boot-image \
    -e IMAGES/EFIBOOT.IMG \
    -no-emul-boot \
    %s > %s`

	cmd := execCommand(command, label, c.dstFolderPath, output)
	if cmd.Failed() {
		return fmt.Errorf("Failed to create the iso %s", cmd.GetErrorMessage())
	}
	return nil
}

func (c *Builder) genIso(output string, label string) error {
	command := `cd %[1]s && \
    genisoimage -U -r -v -T -J \
      -joliet-long \
      -V %[2]s \
      -b ISOLINUX/ISOLINUX.BIN \
      -c ISOLINUX/BOOT.CAT \
      -no-emul-boot \
      -boot-load-size 4 \
      -boot-info-table \
      -eltorito-alt-boot \
      -e IMAGES/EFIBOOT.IMG \
      -no-emul-boot \
      -o %[3]s .
    `
	cmd := execCommand(fmt.Sprintf(command, c.dstFolderPath, label, output))
	if cmd.Failed() {
		return fmt.Errorf("Failed to create the iso %s", cmd.GetErrorMessage())
	}
	InfoLogger.Printf("Iso created correctly at %s", output)
	return nil
}

// Return the path for the original iso.
func (c *Builder) getIsoFilepath() string {
	return filepath.Join(c.tmpFolder, "original.iso")
}

// GetIso donwloads the image from the given url and store it on
// c.getIsoFilepath()
func (c *Builder) GetIso() error {
	out, err := os.Create(c.getIsoFilepath())
	if err != nil {
		return fmt.Errorf("cannot create temporal original iso file: %v", err)
	}
	defer out.Close()

	resp, err := http.Get(c.Source)
	if err != nil {
		return fmt.Errorf("cannot download the iso file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Cannot get source ISO image from url '%s' status %d", c.Source, resp.StatusCode)
	}
	InfoLogger.Printf("Writing iso image to '%s'", c.getIsoFilepath())
	_, err = io.Copy(out, resp.Body)
	return err
}
