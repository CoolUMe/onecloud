package hostman

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/log"
)

// timer utils

func AddTimeout(second time.Duration, callback func()) {
	go func() {
		<-time.NewTimer(second).C
		callback()
	}()
}

func CommandWithTimeout(timeout int, cmds ...string) *exec.Cmd {
	if timeout > 0 {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", timeout)}, cmds...)
	}
	return exec.Command(cmds[0], cmds[1:]...)
}

// file utils

// TODO: test
func Cleandir(sPath string, keepdir bool) error {
	if f, _ := os.Lstat(sPath); f == nil || f.Mode()&os.ModeSymlink == os.ModeSymlink {
		return nil
	}
	files, _ := ioutil.ReadDir(sPath)
	for _, file := range files {
		fp := path.Join(sPath, file.Name())
		if f, _ := os.Lstat(fp); f.Mode()&os.ModeSymlink == os.ModeSymlink {
			if !keepdir {
				if err := os.Remove(fp); err != nil {
					return err
				}
			}
		} else if f.IsDir() {
			Cleandir(fp, keepdir)
			if !keepdir {
				if err := os.Remove(fp); err != nil {
					return err
				}
			}
		} else {
			if err := os.Remove(fp); err != nil {
				return err
			}
		}
	}
	return nil
}

func FilePutContents(filename string, content string, modAppend bool) error {
	var mode = os.O_WRONLY | os.O_CREATE
	if modAppend {
		mode = mode | os.O_APPEND
	}
	fd, err := os.OpenFile(filename, mode, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(content)
	return err
}

func IsBlockDevMounted(dev string) bool {
	devPath := "/dev/" + dev
	mounts, err := exec.Command("mount").Output()
	if err != nil {
		return false
	}
	for _, s := range strings.Split(string(mounts), "\n") {
		if strings.HasPrefix(s, devPath) {
			return true
		}
	}
	return false
}

func IsBlockDeviceUsed(dev string) bool {
	if strings.HasPrefix(dev, "/dev/") {
		dev = dev[strings.LastIndex(dev, "/")+1:]
	}
	devStr := fmt.Sprint(" %s\n", dev)
	devs, _ := exec.Command("cat", "/proc/partitions").Output()
	if idx := strings.Index(string(devs), devStr); idx > 0 {
		return false
	}
	return true
}

func ChangeAllBlkdevsParams(params map[string]string) {
	if _, err := os.Stat("/sys/block"); !os.IsNotExist(err) {
		blockDevs, err := ioutil.ReadDir("/sys/block")
		if err != nil {
			log.Errorln(err)
			return
		}
		for _, b := range blockDevs {
			if IsBlockDevMounted(b.Name()) {
				for k, v := range params {
					ChangeBlkdevParameter(b.Name(), k, v)
				}
			}
		}
	}
}

func ChangeBlkdevParameter(dev, key, value string) {
	p := path.Join("/sys/block", dev, key)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		err = FilePutContents(p, value, false)
		if err != nil {
			log.Errorf("Fail to set %s of %s to %s:%s", key, dev, value, err)
		}
		log.Infof("Set %s of %s to %s", key, dev, value)
	}
}

/*
func PathNotExists(path string) bool {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return true
    }
    return false
}

func PathExists(path string) bool {
    if _, err := os.Stat(path); !os.IsNotExist(err) {
        return true
    }
    return false
}
*/

func FileGetContents(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetFsFormat(diskPath string) string {
	ret, err := exec.Command("blkid", "-o", "value", "-s", "TYPE", diskPath).Output()
	if err != nil {
		return ""
	}
	var res string
	for _, line := range strings.Split(string(ret), "\n") {
		res += line
	}
	return res
}

func CleanFailedMountpoints() {
	var mtfile = "/etc/mtab"
	if _, err := os.Stat(mtfile); os.IsNotExist(err) {
		mtfile = "/proc/mounts"
	}
	f, err := os.Open(mtfile)
	if err != nil {
		log.Errorf("CleanFailedMountpoints error: %s", err)
	}
	reader := bufio.NewReader(f)
	line, _, err := reader.ReadLine()
	for err != nil {
		m := strings.Split(string(line), " ")
		if len(m) > 1 {
			mp := m[1]
			if _, err := os.Stat(mp); os.IsNotExist(err) {
				log.Warningf("Mount point %s not exists", mp)
			}
			exec.Command("umount", mp).Run()
		}
	}
}

type HostsFile map[string][]string

func (hf HostsFile) Parse(content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		data := regexp.MustCompile(`\s+`).Split(line, -1)
		for len(data) > 0 && data[len(data)-1] == "" {
			data = data[:len(data)-1]
		}
		if len(data) > 1 {
			hf[data[0]] = data[1:]
		}
	}
}

func (hf HostsFile) Add(name string, value ...string) {
	hf[name] = value
}

func (hf HostsFile) String() string {
	var ret = ""
	for k, v := range hf {
		if len(v) > 0 {
			ret += fmt.Sprintf("%s\t%s\n", k, strings.Join(v, "\t"))
		}
	}
	return ret
}
