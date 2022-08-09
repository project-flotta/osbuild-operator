package main_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	iso_package "github.com/project-flotta/osbuild-operator/cmd/iso_package"
)

var _ = Describe("Grub utilities", func() {
	format.TruncatedDiff = false
	var (
		tempPath string
	)

	BeforeEach(func() {
		file, err := os.CreateTemp("/tmp/", "grub.cfg")
		Expect(err).NotTo(HaveOccurred())
		tempPath = file.Name()
	})

	AfterEach(func() {
		os.Remove(tempPath)
	})

	writeFile := func(content string) {
		err := os.WriteFile(tempPath, []byte(content), 0600)
		Expect(err).NotTo(HaveOccurred())
	}

	Context("Grub parsing", func() {
		It("Fedora 36 example", func() {

			// given
			original := `
set default="1"

function load_video {
  insmod efi_gop
  insmod efi_uga
  insmod video_bochs
  insmod video_cirrus
  insmod all_video
}

load_video
set gfxpayload=keep
insmod gzio
insmod part_gpt
insmod ext2

set timeout=60
### END /etc/grub.d/00_header ###

search --no-floppy --set=root -l 'Fedora-S-dvd-x86_64-36'

### BEGIN /etc/grub.d/10_linux ###
menuentry 'Install Fedora 36' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 quiet
	initrdefi /images/pxeboot/initrd.img
}
menuentry 'Test this media & install Fedora 36' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 rd.live.check quiet
	initrdefi /images/pxeboot/initrd.img
}
submenu 'Troubleshooting -->' {
	menuentry 'Install Fedora 36 in basic graphics mode' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 nomodeset quiet
		initrdefi /images/pxeboot/initrd.img
	}
	menuentry 'Rescue a Fedora system' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 inst.rescue quiet
		initrdefi /images/pxeboot/initrd.img
	}
}
`
			expected := `
set default="1"

function load_video {
  insmod efi_gop
  insmod efi_uga
  insmod video_bochs
  insmod video_cirrus
  insmod all_video
}

load_video
set gfxpayload=keep
insmod gzio
insmod part_gpt
insmod ext2

set timeout=60
### END /etc/grub.d/00_header ###

search --no-floppy --set=root -l 'Fedora-S-dvd-x86_64-36'

### BEGIN /etc/grub.d/10_linux ###
menuentry 'Install Fedora 36' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 quiet inst.ks=cdrom:/test.ks
	initrdefi /images/pxeboot/initrd.img
}
menuentry 'Test this media & install Fedora 36' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 rd.live.check quiet inst.ks=cdrom:/test.ks
	initrdefi /images/pxeboot/initrd.img
}
submenu 'Troubleshooting -->' {
	menuentry 'Install Fedora 36 in basic graphics mode' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 nomodeset quiet inst.ks=cdrom:/test.ks
		initrdefi /images/pxeboot/initrd.img
	}
	menuentry 'Rescue a Fedora system' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 inst.rescue quiet inst.ks=cdrom:/test.ks
		initrdefi /images/pxeboot/initrd.img
	}
}
`
			writeFile(original)

			// // when
			output, err := iso_package.AppendKickstartTogrub(tempPath, "test.ks")
			// then

			Expect(output.String()).To(Equal(expected))
			Expect(err).NotTo(HaveOccurred())

		})

		It("RHEL 9.0", func() {
			// given
			original := `
set default="1"

function load_video {
  insmod efi_gop
  insmod efi_uga
  insmod video_bochs
  insmod video_cirrus
  insmod all_video
}

load_video
set gfxpayload=keep
insmod gzio
insmod part_gpt
insmod ext2

set timeout=60
### END /etc/grub.d/00_header ###

search --no-floppy --set=root -l 'RHEL-9-0-0-BaseOS-x86_64'

### BEGIN /etc/grub.d/10_linux ###
menuentry 'Install Red Hat Enterprise Linux 9.0' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 quiet
	initrdefi /images/pxeboot/initrd.img
}
menuentry 'Test this media & install Red Hat Enterprise Linux 9.0' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 rd.live.check quiet
	initrdefi /images/pxeboot/initrd.img
}
submenu 'Troubleshooting -->' {
	menuentry 'Install Red Hat Enterprise Linux 9.0 in text mode' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 inst.text quiet
		initrdefi /images/pxeboot/initrd.img
	}
	menuentry 'Rescue a Red Hat Enterprise Linux system' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 inst.rescue quiet
		initrdefi /images/pxeboot/initrd.img
	}
}
`
			expected := `
set default="1"

function load_video {
  insmod efi_gop
  insmod efi_uga
  insmod video_bochs
  insmod video_cirrus
  insmod all_video
}

load_video
set gfxpayload=keep
insmod gzio
insmod part_gpt
insmod ext2

set timeout=60
### END /etc/grub.d/00_header ###

search --no-floppy --set=root -l 'RHEL-9-0-0-BaseOS-x86_64'

### BEGIN /etc/grub.d/10_linux ###
menuentry 'Install Red Hat Enterprise Linux 9.0' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 quiet inst.ks=cdrom:/test.ks
	initrdefi /images/pxeboot/initrd.img
}
menuentry 'Test this media & install Red Hat Enterprise Linux 9.0' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 rd.live.check quiet inst.ks=cdrom:/test.ks
	initrdefi /images/pxeboot/initrd.img
}
submenu 'Troubleshooting -->' {
	menuentry 'Install Red Hat Enterprise Linux 9.0 in text mode' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 inst.text quiet inst.ks=cdrom:/test.ks
		initrdefi /images/pxeboot/initrd.img
	}
	menuentry 'Rescue a Red Hat Enterprise Linux system' --class fedora --class gnu-linux --class gnu --class os {
		linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=RHEL-9-0-0-BaseOS-x86_64 inst.rescue quiet inst.ks=cdrom:/test.ks
		initrdefi /images/pxeboot/initrd.img
	}
}
`
			writeFile(original)

			// when
			output, err := iso_package.AppendKickstartTogrub(tempPath, "test.ks")

			// then
			Expect(output.String()).To(Equal(expected))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Isolinux parsging", func() {
		It("Parsed config", func() {

			// given
			original := `
default vesamenu.c32
timeout 600

display boot.msg

# Clear the screen when exiting the menu, instead of leaving the menu displayed.
# For vesamenu, this means the graphical background is still displayed without
# the menu itself for as long as the screen remains in graphics mode.
menu clear
menu background splash.png
menu title Fedora 36
menu vshift 8
menu rows 18
menu margin 8
#menu hidden
menu helpmsgrow 15
menu tabmsgrow 13

# Border Area
menu color border * #00000000 #00000000 none

# Selected item
menu color sel 0 #ffffffff #00000000 none

# Title bar
menu color title 0 #ff7ba3d0 #00000000 none

# Press [Tab] message
menu color tabmsg 0 #ff3a6496 #00000000 none

# Unselected menu item
menu color unsel 0 #84b8ffff #00000000 none

# Selected hotkey
menu color hotsel 0 #84b8ffff #00000000 none

# Unselected hotkey
menu color hotkey 0 #ffffffff #00000000 none

# Help text
menu color help 0 #ffffffff #00000000 none

# A scrollbar of some type? Not sure.
menu color scrollbar 0 #ffffffff #ff355594 none

# Timeout msg
menu color timeout 0 #ffffffff #00000000 none
menu color timeout_msg 0 #ffffffff #00000000 none

# Command prompt text
menu color cmdmark 0 #84b8ffff #00000000 none
menu color cmdline 0 #ffffffff #00000000 none

# Do not display the actual menu unless the user presses a key. All that is displayed is a timeout message.

menu tabmsg Press Tab for full configuration options on menu items.

menu separator # insert an empty line
menu separator # insert an empty line

label linux
  menu label ^Install Fedora 36
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 quiet

label check
  menu label Test this ^media & install Fedora 36
  menu default
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 rd.live.check quiet

menu separator # insert an empty line

# utilities submenu
menu begin ^Troubleshooting
  menu title Troubleshooting Fedora 36

label basic
  menu indent count 5
  menu label Install using ^basic graphics mode
  text help
	Try this option out if you're having trouble installing
	Fedora 36.
  endtext
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 nomodeset quiet

label rescue
  menu indent count 5
  menu label ^Rescue a Fedora system
  text help
	If the system will not boot, this lets you access files
	and edit config files to try to get it booting again.
  endtext
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 inst.rescue quiet

menu separator # insert an empty line

label local
  menu label Boot from ^local drive
  localboot 0xffff

menu separator # insert an empty line
menu separator # insert an empty line

label returntomain
  menu label Return to ^main menu
  menu exit

menu end
`
			expected := `
default vesamenu.c32
timeout 600

display boot.msg

# Clear the screen when exiting the menu, instead of leaving the menu displayed.
# For vesamenu, this means the graphical background is still displayed without
# the menu itself for as long as the screen remains in graphics mode.
menu clear
menu background splash.png
menu title Fedora 36
menu vshift 8
menu rows 18
menu margin 8
#menu hidden
menu helpmsgrow 15
menu tabmsgrow 13

# Border Area
menu color border * #00000000 #00000000 none

# Selected item
menu color sel 0 #ffffffff #00000000 none

# Title bar
menu color title 0 #ff7ba3d0 #00000000 none

# Press [Tab] message
menu color tabmsg 0 #ff3a6496 #00000000 none

# Unselected menu item
menu color unsel 0 #84b8ffff #00000000 none

# Selected hotkey
menu color hotsel 0 #84b8ffff #00000000 none

# Unselected hotkey
menu color hotkey 0 #ffffffff #00000000 none

# Help text
menu color help 0 #ffffffff #00000000 none

# A scrollbar of some type? Not sure.
menu color scrollbar 0 #ffffffff #ff355594 none

# Timeout msg
menu color timeout 0 #ffffffff #00000000 none
menu color timeout_msg 0 #ffffffff #00000000 none

# Command prompt text
menu color cmdmark 0 #84b8ffff #00000000 none
menu color cmdline 0 #ffffffff #00000000 none

# Do not display the actual menu unless the user presses a key. All that is displayed is a timeout message.

menu tabmsg Press Tab for full configuration options on menu items.

menu separator # insert an empty line
menu separator # insert an empty line

label linux
  menu label ^Install Fedora 36
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 quiet inst.ks=cdrom:/test.ks

label check
  menu label Test this ^media & install Fedora 36
  menu default
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 rd.live.check quiet inst.ks=cdrom:/test.ks

menu separator # insert an empty line

# utilities submenu
menu begin ^Troubleshooting
  menu title Troubleshooting Fedora 36

label basic
  menu indent count 5
  menu label Install using ^basic graphics mode
  text help
	Try this option out if you're having trouble installing
	Fedora 36.
  endtext
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 nomodeset quiet inst.ks=cdrom:/test.ks

label rescue
  menu indent count 5
  menu label ^Rescue a Fedora system
  text help
	If the system will not boot, this lets you access files
	and edit config files to try to get it booting again.
  endtext
  kernel vmlinuz
  append initrd=initrd.img inst.stage2=hd:LABEL=Fedora-S-dvd-x86_64-36 inst.rescue quiet inst.ks=cdrom:/test.ks

menu separator # insert an empty line

label local
  menu label Boot from ^local drive
  localboot 0xffff

menu separator # insert an empty line
menu separator # insert an empty line

label returntomain
  menu label Return to ^main menu
  menu exit

menu end
`
			writeFile(original)

			// when
			output, err := iso_package.AppendKickstartToIsoLinux(tempPath, "test.ks")

			// then
			Expect(output.String()).To(Equal(expected))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
