package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lox/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	macOsMinor          = 11
	vmDS                string
	vmdkDS              string
	vmdkPath            string
	hostIPPrefix        string
	vmNetwork           string
	vmMemoryMB          int64
	vmNumCPUs           int32
	vmNumCoresPerSocket int32
)

func ConfigureCreateVM(app *kingpin.Application) {
	cmd := app.Command("create-vm", "create a virtual machine")

	cmd.Flag("target-datastore", "name of datastore for new VM").
		Required().
		StringVar(&vmDS)

	cmd.Flag("source-datastore", "name of datastore holding source image").
		Required().
		StringVar(&vmdkDS)

	cmd.Flag("source-path", "path of source disk image").
		Required().
		StringVar(&vmdkPath)

	cmd.Flag("host-ip-prefix", "IP prefix of hosts to consider launching VMs on").
		Required().
		StringVar(&hostIPPrefix)

	cmd.Flag("vm-network-label", "name of network to connect VM to").
		Required().
		StringVar(&vmNetwork)

	cmd.Flag("vm-memory-mb", "Specify the memory size in MB of the new virtual machine").
		Required().
		Int64Var(&vmMemoryMB)

	cmd.Flag("vm-num-cpus", "Specify the number of the virtual CPUs of the new virtual machine").
		Required().
		Int32Var(&vmNumCPUs)

	cmd.Flag("vm-num-cores-per-socket", "Number of cores used to distribute virtual CPUs among sockets in this virtual machine").
		Required().
		Int32Var(&vmNumCoresPerSocket)

	cmd.Action(cmdCreateVM)
}

func cmdCreateVM(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, connectionParams)
	if err != nil {
		return err
	}

	st := &state{}

	if err = loadHostSystems(vs, st, clusterPath); err != nil {
		return err
	}

	if err = loadVirtualMachines(vs, st, vmPath); err != nil {
		return err
	}

	countManagedVMsPerHost(st, managedVMPrefix)

	err = createVM(vs, st)
	if err != nil {
		return err
	}

	return nil
}

func createVM(vs *vsphere.Session, st *state) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	hs, hostWhich, err := st.pickHost()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("vmkite-host-macOS_10_%d-%s-%s", macOsMinor, hs.IP, hostWhich)
	params := vsphere.VirtualMachineCreationParams{
		DatastoreName:     vmDS,
		HostSystem:        hs,
		MacOsMinorVersion: macOsMinor,
		MemoryMB:          vmMemoryMB,
		Name:              name,
		NetworkLabel:      vmNetwork,
		NumCPUs:           vmNumCPUs,
		NumCoresPerSocket: vmNumCoresPerSocket,
		SrcDiskDataStore:  vmdkDS,
		SrcDiskPath:       vmdkPath,
	}
	vm, err := vs.CreateVM(params)
	if err != nil {
		return err
	}
	if err := vm.PowerOn(); err != nil {
		return err
	}
	return nil
}

// st.mu must be held.
func (st *state) pickHost() (hs vsphere.HostSystem, hostWhich string, err error) {
	for hostID, inUse := range st.HostManagedVMCount {
		hs = st.HostSystems[hostID]
		ip := hs.IP
		if !strings.HasPrefix(ip, hostIPPrefix) {
			continue
		}
		if inUse >= 2 {
			// Apple virtualization license policy.
			continue
		}
		hostWhich = "a" // unless in use
		if st.whichAInUse(ip) {
			hostWhich = "b"
		}
		return
	}
	err = errors.New("no usable host found")
	return
}

// whichAInUse reports whether a VM is running on the provided hostIP named
// with suffix "a".
//
// st.mu must be held
func (st *state) whichAInUse(ip string) bool {
	suffix := fmt.Sprintf("-%s-a", ip) // vmkite-host-macOS_10_%d-%s-%s
	for _, vm := range st.VirtualMachines {
		if strings.HasSuffix(vm.Name, suffix) {
			return true
		}
	}
	return false
}