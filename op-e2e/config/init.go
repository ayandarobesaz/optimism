package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	// All of the following variables are set in the init function
	// and read from JSON files on disk that are generated by the
	// foundry deploy script. The are globally exported to be used
	// in end to end tests.

	// L1Allocs represents the L1 genesis block state.
	L1Allocs *state.Dump
	// L1Deployments maps contract names to accounts in the L1
	// genesis block state.
	L1Deployments *genesis.L1Deployments
	// DeployConfig represents the deploy config used by the system.
	DeployConfig *genesis.DeployConfig
	// ExternalL2Nodes is the shim to use if external ethereum client testing is
	// enabled
	ExternalL2Nodes string
	// EthNodeVerbosity is the level of verbosity to output
	EthNodeVerbosity int
)

// Init testing to enable test flags
var _ = func() bool {
	testing.Init()
	return true
}()

func init() {
	var l1AllocsPath, l1DeploymentsPath, deployConfigPath string

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, err := findMonorepoRoot(cwd)
	if err != nil {
		panic(err)
	}

	defaultL1AllocsPath := filepath.Join(root, ".devnet", "allocs-l1.json")
	defaultL1DeploymentsPath := filepath.Join(root, ".devnet", "addresses.json")
	defaultDeployConfigPath := filepath.Join(root, "packages", "contracts-bedrock", "deploy-config", "devnetL1.json")

	flag.StringVar(&l1AllocsPath, "l1-allocs", defaultL1AllocsPath, "")
	flag.StringVar(&l1DeploymentsPath, "l1-deployments", defaultL1DeploymentsPath, "")
	flag.StringVar(&deployConfigPath, "deploy-config", defaultDeployConfigPath, "")
	flag.StringVar(&ExternalL2Nodes, "externalL2", "", "Enable tests with external L2")
	flag.IntVar(&EthNodeVerbosity, "ethLogVerbosity", 3, "The level of verbosity to use for the eth node logs")
	flag.Parse()

	if err := allExist(l1AllocsPath, l1DeploymentsPath, deployConfigPath); err != nil {
		return
	}

	L1Allocs, err = genesis.NewStateDump(l1AllocsPath)
	if err != nil {
		panic(err)
	}
	L1Deployments, err = genesis.NewL1Deployments(l1DeploymentsPath)
	if err != nil {
		panic(err)
	}
	DeployConfig, err = genesis.NewDeployConfig(deployConfigPath)
	if err != nil {
		panic(err)
	}

	// Do not use clique in the in memory tests. Otherwise block building
	// would be much more complex.
	DeployConfig.L1UseClique = false
	// Set the L1 genesis block timestamp to now
	DeployConfig.L1GenesisBlockTimestamp = hexutil.Uint64(time.Now().Unix())
	DeployConfig.FundDevAccounts = true
	// Speed up the in memory tests
	DeployConfig.L1BlockTime = 2
	DeployConfig.L2BlockTime = 1

	if L1Deployments != nil {
		DeployConfig.SetDeployments(L1Deployments)
	}
}

func allExist(filenames ...string) error {
	for _, filename := range filenames {
		if _, err := os.Stat(filename); err != nil {
			fmt.Printf("file %s does not exist, skipping genesis generation\n", filename)
			return err
		}
	}
	return nil
}

// findMonorepoRoot will recursively search upwards for a go.mod file.
// This depends on the structure of the monorepo having a go.mod file at the root.
func findMonorepoRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		modulePath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modulePath); err == nil {
			return dir, nil
		}
		parentDir := filepath.Dir(dir)
		// Check if we reached the filesystem root
		if parentDir == dir {
			break
		}
		dir = parentDir
	}
	return "", fmt.Errorf("monorepo root not found")
}
