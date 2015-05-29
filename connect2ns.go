package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const prefix = "/var/run/netns"

var once sync.Once

func CreateBasePath() {
	err := os.MkdirAll(prefix, 0644)
	if err != nil && !os.IsExist(err) {
		fmt.Printf("%v", err)
	}
}

func CreateNamespaceFile(path string) (err error) {
	var f *os.File

	once.Do(CreateBasePath)
	if f, err = os.Create(path); err == nil {
		f.Close()
	}
	return err

}

func LoopbackUp() error {
	iface, _ := netlink.LinkByName("lo")
	return netlink.LinkSetUp(iface)
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	//get root network namespace
	origns, _ := netns.Get()
	defer origns.Close()

	// path name
	path1 := prefix + "ns1"
	path2 := prefix + "ns2"

	CreateNamespaceFile(path1)
	CreateNamespaceFile(path2)

	//create ns1, do something
	newns1, _ := netns.New()
	defer newns1.Close()

	// loopback up
	LoopbackUp()

	//get ns1 namespace file
	procNet := fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), syscall.Gettid())
	if err := syscall.Mount(procNet, path1, "bind", syscall.MS_BIND, ""); err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	newns2, _ := netns.New()
	defer newns2.Close()

	// loopback up
	LoopbackUp()

	//get ns1 namespace file
	procNet := fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), syscall.Gettid())
	if err := syscall.Mount(procNet, path2, "bind", syscall.MS_BIND, ""); err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	// back root namespace，create veth pair
	netns.Setns(origns)

	vethdev := &netlink.Veth{netlink.LinkAttrs{Name: veth1}, veth2}
	_ := netlink.LinkAdd(vethdev)

	veth1, _ := netlink.LinkByName("veth1")
	netlink.LinkSetNsFd(veth1, newns1)

	veth2, _ := netlink.LinkByName("veth2")
	netlink.LinkSetNsFd(veth2, newns2)

	// join in ns1 ,configure veth1
	netns.Setns(newns1)
	veth1, _ = netlink.LinkByName("veth1")
	netlink.LinkSetName(veth1, "eth0")
	addr, _ := netlink.ParseAddr("192.168.200.1/24")
	netlink.AddrAdd(veth1, addr)
	netlink.LinkSetUp(veth1)

	//join in ns2 ,configure veth2
	netns.Setns(newns2)
	veth1, _ = netlink.LinkByName("veth2")
	netlink.LinkSetName(veth2, "eth0")
	addr, _ := netlink.ParseAddr("192.168.200.2/24")
	netlink.AddrAdd(veth2, addr)
	netlink.LinkSetUp(veth2)
}
