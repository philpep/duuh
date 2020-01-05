package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

func checkOutput(command string, args ...string) string {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(stdout)
}

func checkCall(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

var alpinePackageRegexp = regexp.MustCompile("^(.*) \\[upgradable from: (.*)$")
var debianPackageRegexp = alpinePackageRegexp
var centosPackageRegexp = regexp.MustCompile("^(.*) (updates|base)$")

type unattendedUpgrade struct {
	OsType   string
	Upgrades []string
}

func getUnattendedUpgrades() *unattendedUpgrade {
	uu := &unattendedUpgrade{}
	if _, err := os.Stat("/sbin/apk"); !os.IsNotExist(err) {
		uu.OsType = "alpine"
		stdout := checkOutput("/sbin/apk", "--no-cache", "list", "-u")
		scanner := bufio.NewScanner(strings.NewReader(stdout))
		for scanner.Scan() {
			line := scanner.Text()
			if alpinePackageRegexp.MatchString(line) {
				uu.Upgrades = append(uu.Upgrades, line)
			}
		}
	}
	if _, err := os.Stat("/usr/bin/apt-get"); !os.IsNotExist(err) {
		uu.OsType = "debian"
		checkCall("/usr/bin/apt-get", "update")
		stdout := checkOutput("/usr/bin/apt", "list", "--upgradable")
		scanner := bufio.NewScanner(strings.NewReader(stdout))
		for scanner.Scan() {
			line := scanner.Text()
			if debianPackageRegexp.MatchString(line) {
				uu.Upgrades = append(uu.Upgrades, line)
			}
		}
	}
	if _, err := os.Stat("/usr/bin/yum"); !os.IsNotExist(err) {
		uu.OsType = "centos"
		cmd := exec.Command("/usr/bin/yum", "check-update")
		cmd.Stderr = os.Stderr
		stdout, err := cmd.Output()
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() != 100 {
					log.Fatal(err)
				} else {
					scanner := bufio.NewScanner(strings.NewReader(string(stdout)))
					for scanner.Scan() {
						line := scanner.Text()
						if centosPackageRegexp.MatchString(line) {
							uu.Upgrades = append(uu.Upgrades, line)
						}
					}
				}
			}
		}
	}
	return uu
}

func imageGetUnattendedUpgrade(image string) *unattendedUpgrade {
	uu := &unattendedUpgrade{}
	self, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("checking unattended upgrades in %s", image)
	stdout := checkOutput("docker", "run", "--rm", "-u", "root", "--entrypoint", "/duuh", "-v", self+":/duuh", image, "-check")
	err = json.Unmarshal([]byte(stdout), uu)
	if err != nil {
		log.Fatal(err)
	}
	return uu
}

func buildUnattendedUpgradeImage(image string, ostype string, label string) error {
	dockerfile := fmt.Sprintf("FROM %s\n", image)
	dockerfile += fmt.Sprintf("LABEL duuh.upgrades=\"%s\"\n", label)
	user := checkOutput("docker", "image", "inspect", "-f", "{{.Config.User}}", image)
	user = strings.TrimSpace(user)
	if len(user) > 0 {
		dockerfile += "USER root\n"
	}
	switch ostype {
	case "alpine":
		dockerfile += "RUN apk --no-cache upgrade\n"
	case "debian":
		dockerfile += "RUN apt-get update && apt-get -y dist-upgrade && rm -rf /var/lib/apt/lists/*\n"
	case "centos":
		dockerfile += "RUN yum -y update"
	default:
		return fmt.Errorf("unhandled ostype %s", ostype)
	}
	if len(user) > 0 {
		dockerfile += fmt.Sprintf("USER %s\n", user)
	}
	fmt.Print(dockerfile)
	tempdir, err := ioutil.TempDir("", "duuh")
	if err != nil {
		return err
	}
	defer func() {
		err := os.RemoveAll(tempdir)
		if err != nil {
			log.Fatal(err)
		}
	}()
	err = ioutil.WriteFile(filepath.Join(tempdir, "Dockerfile"), []byte(dockerfile), 0644)
	if err != nil {
		return err
	}
	checkCall("docker", "build", "-t", image, tempdir)
	return nil
}

func main() {
	var image string
	var build bool
	var check bool
	var pull bool
	var push bool
	flag.BoolVar(&build, "build", false, "Build image with unattended upgrades")
	flag.BoolVar(&pull, "pull", false, "force pull image from registry before processing")
	flag.BoolVar(&push, "push", false, "push image to registry after processing")
	flag.BoolVar(&check, "check", false, "check current container and output json unattended upgrades (internal use)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of duuh: duuh <docker image>\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	image = flag.Arg(0)
	if !check && image == "" {
		log.Fatal("Please provide an image")
	}
	if check {
		uu := getUnattendedUpgrades()
		log.Printf("detected os type: %s", uu.OsType)
		for _, u := range uu.Upgrades {
			log.Printf("detected upgrade: %s", u)
		}
		uuJSON, err := json.Marshal(uu)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(uuJSON))
	} else {
		if pull {
			checkCall("docker", "pull", image)
		}
		uu := imageGetUnattendedUpgrade(image)
		if len(uu.Upgrades) > 0 {
			if build {
				if err := buildUnattendedUpgradeImage(image, uu.OsType, strings.Join(uu.Upgrades, "\\ \n")); err != nil {
					log.Fatal(err)
				}
				if push {
					checkCall("docker", "push", image)
				}
			} else {
				os.Exit(2)
			}
		} else {
			log.Print("image has no unattended upgrades")
		}
	}
}
