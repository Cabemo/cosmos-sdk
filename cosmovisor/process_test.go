// +build linux

package cosmovisor_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/cosmovisor"
)

type processTestSuite struct {
	suite.Suite
}

func TestProcessTestSuite(t *testing.T) {
	suite.Run(t, new(processTestSuite))
}

// TestLaunchProcess will try running the script a few times and watch upgrades work properly
// and args are passed through
func (s *processTestSuite) TestLaunchProcess() {
	home := copyTestData(s.T(), "validate")
	cfg := &cosmovisor.Config{Home: home, Name: "dummyd", PoolInterval: 20}
	upgradeFile := cfg.UpgradeInfoFilePath()

	// should run the genesis binary and produce expected output
	var stdout, stderr = NewBuffer(), NewBuffer()
	currentBin, err := cfg.CurrentBin()
	s.Require().NoError(err)
	s.Require().Equal(cfg.GenesisBin(), currentBin)

	launcher, err := cosmovisor.NewLauncher(cfg)
	s.Require().NoError(err)

	args := []string{"foo", "bar", "1234", upgradeFile}
	doUpgrade, err := launcher.Run(args, stdout, stderr)
	s.Require().NoError(err)
	s.Require().True(doUpgrade)
	s.Require().Equal("", stderr.String())
	s.Require().Equal(fmt.Sprintf("Genesis foo bar 1234 %s\nUPGRADE \"chain2\" NEEDED at height: 49: {}\n", upgradeFile),
		stdout.String())

	// ensure this is upgraded now and produces new output

	currentBin, err = cfg.CurrentBin()
	s.Require().NoError(err)
	s.Require().Equal(cfg.UpgradeBin("chain2"), currentBin)
	args = []string{"second", "run", "--verbose"}
	stdout.Reset()
	stderr.Reset()

	doUpgrade, err = launcher.Run(args, stdout, stderr)
	s.Require().NoError(err)
	s.Require().False(doUpgrade)
	s.Require().Equal("", stderr.String())
	s.Require().Equal("Chain 2 is live!\nArgs: second run --verbose\nFinished successfully\n", stdout.String())

	// ended without other upgrade
	s.Require().Equal(cfg.UpgradeBin("chain2"), currentBin)
}

// TestLaunchProcess will try running the script a few times and watch upgrades work properly
// and args are passed through
func (s *processTestSuite) TestLaunchProcessWithDownloads() {
	// test case upgrade path:
	// genesis -> "chain2" = zip_binary
	// zip_binary -> "chain3" = ref_zipped -> zip_directory
	// zip_directory no upgrade
	home := copyTestData(s.T(), "download")
	cfg := &cosmovisor.Config{Home: home, Name: "autod", AllowDownloadBinaries: true, PoolInterval: 100}

	// should run the genesis binary and produce expected output
	var stdout, stderr = NewBuffer(), NewBuffer()
	currentBin, err := cfg.CurrentBin()
	s.Require().NoError(err)

	s.Require().Equal(cfg.GenesisBin(), currentBin)
	args := []string{"some", "args", cfg.UpgradeInfoFilePath()}

	launcher, err := cosmovisor.NewLauncher(cfg)
	s.Require().NoError(err)

	doUpgrade, err := launcher.Run(args, stdout, stderr)
	fmt.Println(">>>>", stderr.String())
	fmt.Println(">>>>", stdout.String())

	s.Require().NoError(err)
	s.Require().True(doUpgrade)
	s.Require().Equal("", stderr.String())
	s.Require().Equal("Preparing auto-download some args "+cfg.UpgradeInfoFilePath()+"\n"+`ERROR: UPGRADE "chain2" NEEDED at height: 49: {"binaries":{"linux/amd64":"https://github.com/cosmos/cosmos-sdk/raw/robert/cosmvisor-file-watch/cosmovisor/testdata/repo/zip_binary/autod.zip?checksum=sha256:9428a1c135430d89243fa48e2ae67cb73530c636818a49bdc4ae6335a576370a"}} module=main`+"\n", stdout.String())

	// ensure this is upgraded now and produces new output
	currentBin, err = cfg.CurrentBin()
	s.Require().NoError(err)
	s.Require().Equal(cfg.UpgradeBin("chain2"), currentBin)
	args = []string{"run", "--fast"}
	stdout.Reset()
	stderr.Reset()

	doUpgrade, err = launcher.Run(args, stdout, stderr)
	s.Require().NoError(err)
	s.Require().True(doUpgrade)
	s.Require().Equal("", stderr.String())
	s.Require().Equal("Chain 2 from zipped binary link to referral\nArgs: run --fast\n"+`ERROR: UPGRADE "chain3" NEEDED at height: 936: ttps://github.com/cosmos/cosmos-sdk/raw/robert/cosmvisor-file-watch/cosmovisor/testdata/repo/zip_binary/autod.zip?checksum=sha256:9428a1c135430d89243fa48e2ae67cb73530c636818a49bdc4ae6335a576370a module=main`+"\n", stdout.String())

	// ended with one more upgrade
	currentBin, err = cfg.CurrentBin()
	s.Require().NoError(err)
	s.Require().Equal(cfg.UpgradeBin("chain3"), currentBin)
	// make sure this is the proper binary now....
	args = []string{"end", "--halt"}
	stdout.Reset()
	stderr.Reset()
	doUpgrade, err = launcher.Run(args, stdout, stderr)
	s.Require().NoError(err)
	s.Require().False(doUpgrade)
	s.Require().Equal("", stderr.String())
	s.Require().Equal("Chain 2 from zipped directory\nArgs: end --halt\n", stdout.String())

	// and this doesn't upgrade
	currentBin, err = cfg.CurrentBin()
	s.Require().NoError(err)
	s.Require().Equal(cfg.UpgradeBin("chain3"), currentBin)
}
