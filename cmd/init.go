package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/compose"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/dns"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize Hatch (system or project)",
	Long: `Without arguments: sets up the Hatch config directory, generates a root CA,
trusts it in Keychain, and installs the DNS resolver.

With a path argument: scans for docker-compose files, discovers services,
and generates a hatch.yml project config.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		return runProjectInit(cmd, args[0])
	}

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

func runProjectInit(cmd *cobra.Command, dir string) error {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	reader := bufio.NewReader(os.Stdin)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	hatchFile := filepath.Join(absDir, "hatch.yml")
	if fileExistsAt(hatchFile) {
		fmt.Printf("A hatch.yml already exists in %s. Overwrite? [y/N] ", dir)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Printf("Scanning %s for docker-compose files...\n\n", cyan(dir))

	services, err := compose.Discover(absDir)
	if err != nil {
		return fmt.Errorf("discover services: %w", err)
	}

	projectName := filepath.Base(absDir)
	tld := "test"
	cfg, loadErr := config.Load()
	if loadErr == nil {
		tld = cfg.Settings.TLD
	}
	defaultDomain := projectName + "." + tld

	if len(services) == 0 {
		fmt.Println("No docker-compose services with ports found.")
		fmt.Printf("Generating default config with a single %s service on port %s.\n\n",
			cyan("web"), cyan("3000"))
		services = []compose.DiscoveredService{
			{Name: "web", Port: 3000, Source: "(default)"},
		}
	}

	type accepted struct {
		name      string
		subdomain string
		port      int
	}
	var kept []accepted

	fmt.Println("Discovered services:")
	fmt.Println()

	for _, svc := range services {
		defaultSub := svc.Name
		fmt.Printf("  %s  port %d  %s\n", cyan(svc.Name), svc.Port, dim("from "+svc.Source))
		fmt.Printf("  Subdomain [%s] (enter=accept, s=skip): ", defaultSub)

		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		if strings.ToLower(line) == "s" {
			fmt.Println()
			continue
		}

		subdomain := defaultSub
		if line != "" {
			subdomain = line
		}

		kept = append(kept, accepted{
			name:      svc.Name,
			subdomain: subdomain,
			port:      svc.Port,
		})
		fmt.Println()
	}

	if len(kept) == 0 {
		fmt.Println("All services skipped. No hatch.yml written.")
		return nil
	}

	pc := config.ProjectConfig{
		Domain:   defaultDomain,
		Services: make(map[string]config.Service),
	}
	for _, k := range kept {
		svc := config.Service{
			Proxy: fmt.Sprintf("http://localhost:%d", k.port),
		}
		if k.subdomain != k.name || len(kept) > 1 {
			svc.Subdomain = k.subdomain
		}
		pc.Services[k.name] = svc
	}

	data, err := yaml.Marshal(pc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(hatchFile, data, 0o644); err != nil {
		return fmt.Errorf("write hatch.yml: %w", err)
	}

	fmt.Printf("%s Wrote %s\n", green("✓"), filepath.Join(dir, "hatch.yml"))
	fmt.Printf("\nYour project will be available at %s\n", cyan("https://"+defaultDomain))
	fmt.Printf("Run '%s' to register the project with Hatch.\n", color.New(color.Bold).Sprint("hatch link"))

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
