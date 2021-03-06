package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	vmClusterPath       string
	vmDS                string
	vmdkDS              string
	vmdkPath            string
	vmNetwork           string
	vmMemoryMB          int64
	vmNumCPUs           int32
	vmNumCoresPerSocket int32
	vmGuestId           string
)

var (
	vmGuestInfo = map[string]string{}
)

func ConfigureCreateVM(app *kingpin.Application) {
	cmd := app.Command("create-vm", "create a virtual machine")

	addCreateVMFlags(cmd)

	cmd.Flag("source-path", "path of source disk image").
		Required().
		StringVar(&vmdkPath)

	cmd.Flag("buildkite-agent-token", "Buildkite Agent Token").
		Required().
		StringVar(&buildkiteAgentToken)

	cmd.Action(cmdCreateVM)
}

func addCreateVMFlags(cmd *kingpin.CmdClause) {
	cmd.Flag("target-datastore", "name of datastore for new VM").
		Required().
		StringVar(&vmDS)

	cmd.Flag("source-datastore", "name of datastore holding source image").
		Required().
		StringVar(&vmdkDS)

	cmd.Flag("vm-cluster-path", "path to the cluster").
		Required().
		StringVar(&vmClusterPath)

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

	cmd.Flag("vm-guest-id", "The guestid of the vm").
		Default("darwin14_64Guest").
		StringVar(&vmGuestId)

	cmd.Flag("vm-guest-info", "A set of key=value params to pass to the vm").
		StringMapVar(&vmGuestInfo)
}

func cmdCreateVM(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, connectionParams)
	if err != nil {
		return err
	}

	params := vsphere.VirtualMachineCreationParams{
		BuildkiteAgentToken: buildkiteAgentToken,
		ClusterPath:         vmClusterPath,
		DatastoreName:       vmDS,
		MemoryMB:            vmMemoryMB,
		Name:                fmt.Sprintf("vmkite-%s", time.Now().Format("200612-150405")),
		NetworkLabel:        vmNetwork,
		NumCPUs:             vmNumCPUs,
		NumCoresPerSocket:   vmNumCoresPerSocket,
		SrcDiskDataStore:    vmdkDS,
		SrcDiskPath:         vmdkPath,
		GuestInfo:           vmGuestInfo,
	}

	_, err = creator.CreateVM(vs, params)
	if err != nil {
		return err
	}

	return nil
}
