package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/dns"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Hatch for first-time use",
	Long:  `Sets up the Hatch config directory, generates a root CA, trusts it in Keychain, and installs the DNS resolver. Does not start the daemon.`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()

	fmt.Printf("Initializing %s...\n\n", cyan("Hatch"))

	anyCreated := false

	// Step 1: Config directory
	dirExists := dirExistsAt(config.Dir())
	if err := config.EnsureConfigDir(); err != nil {
		fmt.Printf("  %s Failed to create config directory: %v\n", red("✗"), err)
		os.Exit(1)
	}
	if dirExists {
		fmt.Printf("  %s Config directory exists\n", green("✓"))
	} else {
		fmt.Printf("  %s Config directory created (~/.hatch)\n", green("✓"))
		anyCreated = true
	}

	// Step 2: Config file
	fileExists := fileExistsAt(config.ConfigFile())
	if err := config.EnsureConfigFile(); err != nil {
		fmt.Printf("  %s Failed to write default config: %v\n", red("✗"), err)
		os.Exit(1)
	}
	if fileExists {
		fmt.Printf("  %s Config file exists\n", green("✓"))
	} else {
		fmt.Printf("  %s Default config written (~/.hatch/config.yml)\n", green("✓"))
		anyCreated = true
	}

	// Load config to get TLD for later steps
	cfg, err := config.Load()
	if err != nil {
		var ve *config.ValidationErrors
		if errors.As(err, &ve) {
			for i, e := range ve.Errs {
				fmt.Printf("  %s Config error %d: %s\n", red("✗"), i+1, e)
			}
		} else {
			fmt.Printf("  %s Failed to load config: %v\n", red("✗"), err)
		}
		os.Exit(1)
	}

	// Step 3: Root CA
	caPaths := certs.NewCAPaths(config.CertsDir())
	if certs.CAExists(caPaths) {
		fmt.Printf("  %s Root CA exists\n", green("✓"))
	} else {
		if err := certs.GenerateCA(caPaths); err != nil {
			fmt.Printf("  %s Failed to generate root CA: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Root CA generated\n", green("✓"))
		anyCreated = true
	}

	// Step 3.5: Intermediate CA
	if certs.IntermediateCAExists(caPaths) {
		fmt.Printf("  %s Intermediate CA exists\n", green("✓"))
	} else {
		if err := certs.GenerateIntermediateCA(caPaths); err != nil {
			fmt.Printf("  %s Failed to generate intermediate CA: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Intermediate CA generated\n", green("✓"))
		anyCreated = true
	}

	// Step 4: Trust CA
	if certs.IsCATrusted(caPaths.Cert) {
		fmt.Printf("  %s Root CA already trusted\n", green("✓"))
	} else {
		if err := certs.TrustCA(&sudoRunner{}, caPaths.Cert); err != nil {
			fmt.Printf("  %s Failed to trust root CA: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Root CA trusted in Keychain\n", green("✓"))
		anyCreated = true
	}

	// Step 5: DNS resolver
	tld := cfg.Settings.TLD
	if dns.IsResolverInstalled(tld) {
		fmt.Printf("  %s DNS resolver already installed\n", green("✓"))
	} else {
		if err := dns.InstallResolverFile(&sudoRunner{}, tld, dns.DefaultListenIP, dns.DefaultPort); err != nil {
			fmt.Printf("  %s Failed to install DNS resolver: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s DNS resolver installed for .%s\n", green("✓"), tld)
		anyCreated = true
	}

	fmt.Println()
	if anyCreated {
		fmt.Printf("%s initialized! Next steps:\n", cyan("Hatch"))
		fmt.Println("  hatch up       Start the daemon")
		fmt.Println("  hatch version  Check installed version")
	} else {
		fmt.Printf("Already initialized. Run '%s' to start the daemon.\n", color.New(color.Bold).Sprint("hatch up"))
	}

	return nil
}

func dirExistsAt(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExistsAt(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
