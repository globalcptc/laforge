package loader

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/ansible"
	"github.com/gen0cide/laforge/ent/command"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/disk"
	"github.com/gen0cide/laforge/ent/dns"
	"github.com/gen0cide/laforge/ent/dnsrecord"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/filedelete"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/gen0cide/laforge/ent/finding"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/hostdependency"
	"github.com/gen0cide/laforge/ent/identity"
	"github.com/gen0cide/laforge/ent/includednetwork"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/scheduledstep"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/validation"
	"github.com/gen0cide/laforge/loader/include"
	"github.com/gen0cide/laforge/logging"
	"github.com/google/uuid"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/transform"
	gohcl2 "github.com/hashicorp/hcl/v2/gohcl"
	hcl2parse "github.com/hashicorp/hcl/v2/hclparse"
	_ "github.com/mattn/go-sqlite3"
	zglob "github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

// Include defines a named include type
type Include struct {
	Path string `hcl:"path,attr"`
}

type fileGlobResolver struct {
	BaseDir string
	Parser  *hcl2parse.Parser
}

// DefinedConfigs is the stuct to hold in all the loading for hcl
type DefinedConfigs struct {
	Filename              string
	BaseDir               string                        `hcl:"base_dir,optional" json:"base_dir,omitempty"`
	IncludePaths          []*Include                    `hcl:"include,block" json:"include_paths,omitempty"`
	DefinedCompetitions   []*ent.Competition            `hcl:"competition,block" json:"competitions,omitempty"`
	DefinedHosts          []*ent.Host                   `hcl:"host,block" json:"hosts,omitempty"`
	DefinedNetworks       []*ent.Network                `hcl:"network,block" json:"networks,omitempty"`
	DefinedScripts        []*ent.Script                 `hcl:"script,block" json:"scripts,omitempty"`
	DefinedCommands       []*ent.Command                `hcl:"command,block" json:"defined_commands,omitempty"`
	DefinedDNSRecords     []*ent.DNSRecord              `hcl:"dns_record,block" json:"defined_dns_records,omitempty"`
	DefinedValidations    []*ent.Validation             `hcl:"validation,block" json:"defined_validations,omitempty"`
	DefinedEnvironments   []*ent.Environment            `hcl:"environment,block" json:"environments,omitempty"`
	DefinedFileDownload   []*ent.FileDownload           `hcl:"file_download,block" json:"file_download,omitempty"`
	DefinedFileDelete     []*ent.FileDelete             `hcl:"file_delete,block" json:"file_delete,omitempty"`
	DefinedFileExtract    []*ent.FileExtract            `hcl:"file_extract,block" json:"file_extract,omitempty"`
	DefinedIdentities     []*ent.Identity               `hcl:"identity,block" json:"identities,omitempty"`
	DefinedAnsible        []*ent.Ansible                `hcl:"ansible,block" json:"ansible,omitempty"`
	DefinedScheduledSteps []*ent.ScheduledStep          `hcl:"schedule_step,block" json:"schedule_step,omitempty"`
	Competitions          map[string]*ent.Competition   `json:"-"`
	Hosts                 map[string]*ent.Host          `json:"-"`
	Networks              map[string]*ent.Network       `json:"-"`
	Scripts               map[string]*ent.Script        `json:"-"`
	Commands              map[string]*ent.Command       `json:"-"`
	DNSRecords            map[string]*ent.DNSRecord     `json:"-"`
	Validations           map[string]*ent.Validation    `json:"-"`
	Environments          map[string]*ent.Environment   `json:"-"`
	FileDownload          map[string]*ent.FileDownload  `json:"-"`
	FileDelete            map[string]*ent.FileDelete    `json:"-"`
	FileExtract           map[string]*ent.FileExtract   `json:"-"`
	Identities            map[string]*ent.Identity      `json:"-"`
	Ansible               map[string]*ent.Ansible       `json:"-"`
	ScheduledSteps        map[string]*ent.ScheduledStep `json:"-"`
}

// Loader defines the Laforge configuration loader object
type Loader struct {
	// Parser is the actual HCLv2 parser
	Parser *hcl2parse.Parser

	// SourceFile is the location of the first file loaded
	SourceFile string

	// ConfigMap contains all the configuration steps
	ConfigMap map[string]*DefinedConfigs
}

// FileGlobResolver is a modified FileResolver in the HCLv2 include extension that accounts for globbed
// includes:
//
//	include {
//		path = "./foo/*.laforge"
//	}
func FileGlobResolver(baseDir string, parser *hcl2parse.Parser) include.Resolver {
	return &fileGlobResolver{
		BaseDir: baseDir,
		Parser:  parser,
	}
}

func (r fileGlobResolver) ResolveBodyPath(path string, refRange hcl2.Range) (hcl2.Body, hcl2.Diagnostics) {
	callerFile := filepath.Join(refRange.Filename)
	callerDir := filepath.Dir(callerFile)
	targetFile := filepath.Join(callerDir, path)
	body := hcl2.EmptyBody()
	var diags hcl2.Diagnostics
	if strings.Contains(targetFile, `*`) {
		matches, err := zglob.Glob(targetFile)
		if err != nil {
			return body, hcl2.Diagnostics{&hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "directory glob error",
				Detail:   fmt.Sprintf("could not glob on %s: %v", targetFile, err),
			}}
		}
		for _, m := range matches {
			switch {
			case strings.HasSuffix(m, ".json"):
				_, newDiags := r.Parser.ParseJSONFile(m)
				diags = diags.Extend(newDiags)
			case strings.HasSuffix(m, ".laforge"):
				_, newDiags := r.Parser.ParseHCLFile(m)
				diags = diags.Extend(newDiags)
			default:
				newDiag := &hcl2.Diagnostic{
					Severity: hcl2.DiagWarning,
					Summary:  "invalid file in glob",
					Detail:   fmt.Sprintf("%s is not a valid JSON or Laforge file (glob=%s)", m, targetFile),
				}
				diags = diags.Append(newDiag)
			}
		}
	} else {
		if strings.HasSuffix(targetFile, ".json") {
			_, diags = r.Parser.ParseJSONFile(targetFile)
		} else {
			_, diags = r.Parser.ParseHCLFile(targetFile)
		}
		if diags.HasErrors() {
			for _, e := range diags.Errs() {
				ne, ok := e.(*hcl2.Diagnostic)
				if ok {
					logrus.Errorf("Laforge failed to parse a config file:\n Location: %v\n    Issue: %v\n   Detail: %v", ne.Subject, ne.Summary, ne.Detail)
				}
			}
		}
		return nil, diags
	}
	if diags.HasErrors() {
		for _, e := range diags.Errs() {
			ne, ok := e.(*hcl2.Diagnostic)
			if ok {
				logrus.Errorf("Laforge failed to parse a config file:\n Location: %v\n    Issue: %v\n   Detail: %v", ne.Subject, ne.Summary, ne.Detail)
			}
		}
	}
	return body, diags
}

// ParseConfigFile loads a root file into Loader
func (l *Loader) ParseConfigFile(log *logging.Logger, filename string) error {
	var diags hcl2.Diagnostics
	if strings.HasSuffix(filename, ".json") {
		_, diags = l.Parser.ParseJSONFile(filename)
	} else {
		_, diags = l.Parser.ParseHCLFile(filename)
	}
	if diags.HasErrors() {
		for _, e := range diags.Errs() {
			ne, ok := e.(*hcl2.Diagnostic)
			if ok {
				log.Log.Errorf("Laforge failed to parse a config file:\n Location: %v\n    Issue: %v\n   Detail: %v", ne.Subject, ne.Summary, ne.Detail)
			}
		}
		return diags
	}
	l.SourceFile = filename
	return nil
}

// Bind enumerates the Loader's original file, performing recursive include loads to the
// Loader, generating ASTs for each dependency. Bind finishes with a call to Deconflict().
func (l *Loader) Bind(log *logging.Logger) (*DefinedConfigs, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	transformer := include.Transformer("include", nil, FileGlobResolver(root, l.Parser))
	filenames := []string{}
	for name := range l.Parser.Files() {
		filenames = append([]string{name}, filenames...)
	}
	currLen := len(l.Parser.Files())
	for {
		for name, f := range l.Parser.Files() {
			transform.Deep(f.Body, transformer)
			exists := false
			for _, i := range filenames {
				if i == name {
					exists = true
				}
			}
			if !exists {
				filenames = append([]string{name}, filenames...)
			}
			newLF := &DefinedConfigs{}
			diags := gohcl2.DecodeBody(f.Body, nil, newLF)
			if diags.HasErrors() {
				for _, e := range diags.Errs() {
					ne, ok := e.(*hcl2.Diagnostic)
					if ok {
						log.Log.Errorf("Laforge failed to parse a config file:\n Location: %v\n    Issue: %v\n   Detail: %v", ne.Subject, ne.Summary, ne.Detail)
					}
				}
				return nil, diags
			}
			newLF.Filename = name
			l.ConfigMap[name] = newLF
		}
		newLen := len(l.Parser.Files())
		if currLen == newLen {
			break
		}
		currLen = newLen
	}
	return l.merger(filenames)
}

// NewLoader returns a default Loader type
func NewLoader() *Loader {
	return &Loader{
		Parser:     hcl2parse.NewParser(),
		ConfigMap:  map[string]*DefinedConfigs{},
		SourceFile: "",
	}
}

func (l *Loader) merger(filenames []string) (*DefinedConfigs, error) {
	combinedConfigs := &DefinedConfigs{
		Filename:       l.SourceFile,
		Competitions:   map[string]*ent.Competition{},
		Hosts:          map[string]*ent.Host{},
		Networks:       map[string]*ent.Network{},
		Scripts:        map[string]*ent.Script{},
		Commands:       map[string]*ent.Command{},
		DNSRecords:     map[string]*ent.DNSRecord{},
		Validations:    map[string]*ent.Validation{},
		Environments:   map[string]*ent.Environment{},
		FileDownload:   map[string]*ent.FileDownload{},
		FileDelete:     map[string]*ent.FileDelete{},
		FileExtract:    map[string]*ent.FileExtract{},
		Identities:     map[string]*ent.Identity{},
		Ansible:        map[string]*ent.Ansible{},
		ScheduledSteps: map[string]*ent.ScheduledStep{},
	}
	for _, filename := range filenames {
		element := l.ConfigMap[filename]
		for _, x := range element.DefinedCompetitions {
			obj, found := combinedConfigs.Competitions[x.HclID]
			if !found {
				combinedConfigs.Competitions[x.HclID] = x
				continue
			}
			if x.RootPassword != "" {
				obj.RootPassword = x.RootPassword
			}
			if x.Config != nil {
				obj.Config = x.Config
			}
			if x.Tags != nil {
				obj.Tags = x.Tags
			}
			if x.HCLCompetitionToDNS != nil {
				obj.HCLCompetitionToDNS = x.HCLCompetitionToDNS
			}
			combinedConfigs.Competitions[x.HclID] = obj
		}
		for _, x := range element.DefinedHosts {
			_, found := combinedConfigs.Hosts[x.HclID]
			if !found {
				combinedConfigs.Hosts[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedNetworks {
			_, found := combinedConfigs.Networks[x.HclID]
			if !found {
				combinedConfigs.Networks[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedScripts {
			if x.SourceType == "local" {
				dir := path.Dir(element.Filename)
				absPath := path.Join(dir, x.Source)
				x.AbsPath = absPath
			}
			_, found := combinedConfigs.Scripts[x.HclID]
			if !found {
				combinedConfigs.Scripts[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedCommands {
			_, found := combinedConfigs.Commands[x.HclID]
			if !found {
				combinedConfigs.Commands[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedDNSRecords {
			_, found := combinedConfigs.DNSRecords[x.HclID]
			if !found {
				combinedConfigs.DNSRecords[x.HclID] = x
				continue
			}
		}
		for _, x := range element.Validations {
			_, found := combinedConfigs.Environments[x.HclID]
			if !found {
				combinedConfigs.Validations[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedEnvironments {
			_, found := combinedConfigs.Environments[x.HclID]
			if !found {
				combinedConfigs.Environments[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedFileDownload {
			_, found := combinedConfigs.FileDownload[x.HclID]
			dir := path.Dir(element.Filename)
			absPath := path.Join(dir, x.Source)
			x.AbsPath = absPath
			if !found {
				combinedConfigs.FileDownload[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedFileDelete {
			element.FileDelete[x.HclID] = x
		}
		for _, x := range element.DefinedFileExtract {
			element.FileExtract[x.HclID] = x
		}
		for _, x := range element.DefinedIdentities {
			_, found := combinedConfigs.Identities[x.HclID]
			if !found {
				combinedConfigs.Identities[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedAnsible {
			dir := path.Dir(element.Filename)
			absPath := path.Join(dir, x.Source)
			x.AbsPath = absPath
			_, found := combinedConfigs.Ansible[x.HclID]
			if !found {
				combinedConfigs.Ansible[x.HclID] = x
				continue
			}
		}
		for _, x := range element.DefinedScheduledSteps {
			_, found := combinedConfigs.ScheduledSteps[x.HclID]
			if !found {
				combinedConfigs.ScheduledSteps[x.HclID] = x
				continue
			}
		}
	}
	return combinedConfigs, nil
}

// LoadEnvironment Loads in enviroment at specified filepath
func LoadEnvironment(ctx context.Context, client *ent.Client, log *logging.Logger, filePath string) ([]*ent.Environment, error) {
	tloader := NewLoader()
	tloader.ParseConfigFile(log, filePath)
	loadedConfig, err := tloader.Bind(log)
	if err != nil {
		log.Log.Errorf("Unable to Load ENV Config: %v Err: %v", filePath, err)
		return nil, err
	}
	log.Log.Infof("Loading environment from: %s", filePath)
	return createEnviroments(ctx, client, log, loadedConfig.Environments, loadedConfig)
}

// rollback calls to tx.Rollback and wraps the given error
// with the rollback error if occurred.
func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		err = fmt.Errorf("%w: %v", err, rerr)
	}
	return err
}

// getHostIDs gets Host IDs from array of Host objects and returns a array of Host IDs
func getHostIDs(hosts []*ent.Host) []uuid.UUID {
	var hostIDs []uuid.UUID
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}
	return hostIDs
}

// getNetworkIDs gets Network IDs from array of Network objects and returns a array of Network IDs
func getNetworkIDs(networks []*ent.Network) []uuid.UUID {
	var networkIDs []uuid.UUID
	for _, n := range networks {
		networkIDs = append(networkIDs, n.ID)
	}
	return networkIDs
}

// Need to combine everything here
func createEnviroments(ctx context.Context, client *ent.Client, log *logging.Logger, configEnvs map[string]*ent.Environment, loadedConfig *DefinedConfigs) ([]*ent.Environment, error) {
	txClient, err := client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting a transaction: %w", err)
	}
	returnedEnvironment := []*ent.Environment{}
	for _, cEnviroment := range configEnvs {
		log.Log.Debugf("Creating ENV: %v", cEnviroment.HclID)

		environmentHosts := []string{}
		for _, cIncludedNetwork := range cEnviroment.HCLEnvironmentToIncludedNetwork {
			environmentHosts = append(environmentHosts, cIncludedNetwork.Hosts...)
		}
		returnedCompetitions, returnedDNS, err := createCompetitions(txClient, ctx, log, loadedConfig.Competitions, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in competition into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedValidations, err := createValidations(txClient, ctx, log, loadedConfig.Validations, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in validations into env: %v, Err: %v", cEnviroment.HclID, err)
		}
		returnedScripts, returnedFindings, err := createScripts(txClient, ctx, log, loadedConfig.Scripts, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in findings into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedCommands, err := createCommands(txClient, ctx, log, loadedConfig.Commands, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in commands into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedDNSRecords, err := createDNSRecords(txClient, ctx, log, loadedConfig.DNSRecords, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in dns_records into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedFileDownloads, err := createFileDownload(txClient, ctx, log, loadedConfig.FileDownload, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in file_downloads into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedFileDeletes, err := createFileDelete(txClient, ctx, log, loadedConfig.FileDelete, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in file_delets into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedFileExtracts, err := createFileExtract(txClient, ctx, log, loadedConfig.FileExtract, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in file_extracts into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedIdentities, err := createIdentities(txClient, ctx, log, loadedConfig.Identities, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in identities into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedNetworks, err := createNetworks(txClient, ctx, log, loadedConfig.Networks, cEnviroment.HCLEnvironmentToIncludedNetwork, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in competition into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedNetworkIDs := getNetworkIDs(returnedNetworks)
		returnedAnsible, err := createAnsible(txClient, ctx, log, loadedConfig.Ansible, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in Ansible into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedScheduledSteps, err := createScheduledStep(txClient, ctx, log, loadedConfig.ScheduledSteps, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in Scheduled Step into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		// returnedHostDependencies is empty if ran once but ok when ran multiple times
		returnedHosts, returnedHostDependencies, err := createHosts(txClient, ctx, log, loadedConfig.Hosts, cEnviroment.HclID, environmentHosts)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in Hosts into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedHostIDs := getHostIDs(returnedHosts)
		returnedIncludedNetworks, err := createIncludedNetwork(txClient, ctx, log, cEnviroment.HCLEnvironmentToIncludedNetwork, cEnviroment.HclID, returnedHostIDs, returnedNetworkIDs)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Error loading in included_networks into env: %v, Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		entEnvironment, err := txClient.Environment.
			Query().
			Where(environment.HclIDEQ(cEnviroment.HclID)).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				newEnvironment, err := txClient.Environment.Create().
					SetHclID(cEnviroment.HclID).
					SetAdminCidrs(cEnviroment.AdminCidrs).
					SetBuilder(cEnviroment.Builder).
					SetCompetitionID(cEnviroment.CompetitionID).
					SetConfig(cEnviroment.Config).
					SetDescription(cEnviroment.Description).
					SetExposedVdiPorts(cEnviroment.ExposedVdiPorts).
					SetName(cEnviroment.Name).
					SetRevision(cEnviroment.Revision).
					SetTags(cEnviroment.Tags).
					SetTeamCount(cEnviroment.TeamCount).
					AddEnvironmentToCompetition(returnedCompetitions...).
					AddEnvironmentToScript(returnedScripts...).
					AddEnvironmentToFinding(returnedFindings...).
					AddEnvironmentToCommand(returnedCommands...).
					AddEnvironmentToDNSRecord(returnedDNSRecords...).
					AddEnvironmentToValidation(returnedValidations...).
					AddEnvironmentToFileDownload(returnedFileDownloads...).
					AddEnvironmentToFileDelete(returnedFileDeletes...).
					AddEnvironmentToFileExtract(returnedFileExtracts...).
					AddEnvironmentToIdentity(returnedIdentities...).
					AddEnvironmentToNetwork(returnedNetworks...).
					AddEnvironmentToHost(returnedHosts...).
					AddEnvironmentToHostDependency(returnedHostDependencies...).
					AddEnvironmentToIncludedNetwork(returnedIncludedNetworks...).
					AddEnvironmentToDNS(returnedDNS...).
					AddEnvironmentToAnsible(returnedAnsible...).
					AddEnvironmentToScheduledStep(returnedScheduledSteps...).
					Save(ctx)
				if err != nil {
					err = rollback(txClient, err)
					log.Log.Errorf("Failed to Create Environment %v. Err: %v", cEnviroment.HclID, err)
					return nil, err
				}
				_, err = validateHostDependencies(txClient, ctx, log, returnedHostDependencies, cEnviroment.HclID)
				if err != nil {
					err = rollback(txClient, err)
					log.Log.Errorf("Failed to Validate Host Dependencies in Environment %v. Err: %v", cEnviroment.HclID, err)
					return nil, err
				}
				returnedEnvironment = append(returnedEnvironment, newEnvironment)
				continue
			}
		}
		entEnvironment, err = entEnvironment.Update().
			SetHclID(cEnviroment.HclID).
			SetAdminCidrs(cEnviroment.AdminCidrs).
			SetBuilder(cEnviroment.Builder).
			SetCompetitionID(cEnviroment.CompetitionID).
			SetConfig(cEnviroment.Config).
			SetDescription(cEnviroment.Description).
			SetExposedVdiPorts(cEnviroment.ExposedVdiPorts).
			SetName(cEnviroment.Name).
			SetRevision(entEnvironment.Revision + 1).
			SetTags(cEnviroment.Tags).
			SetTeamCount(cEnviroment.TeamCount).
			ClearEnvironmentToCompetition().
			ClearEnvironmentToScript().
			ClearEnvironmentToFinding().
			ClearEnvironmentToCommand().
			ClearEnvironmentToDNSRecord().
			ClearEnvironmentToValidation().
			ClearEnvironmentToFileDownload().
			ClearEnvironmentToFileDelete().
			ClearEnvironmentToFileExtract().
			ClearEnvironmentToIdentity().
			ClearEnvironmentToNetwork().
			ClearEnvironmentToHostDependency().
			ClearEnvironmentToIncludedNetwork().
			ClearEnvironmentToDNS().
			ClearEnvironmentToHost().
			ClearEnvironmentToAnsible().
			ClearEnvironmentToScheduledStep().
			Save(ctx)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Failed to Update Environment %v. Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		entEnvironment, err = entEnvironment.Update().
			AddEnvironmentToCompetition(returnedCompetitions...).
			AddEnvironmentToScript(returnedScripts...).
			AddEnvironmentToFinding(returnedFindings...).
			AddEnvironmentToCommand(returnedCommands...).
			AddEnvironmentToDNSRecord(returnedDNSRecords...).
			AddEnvironmentToFileDownload(returnedFileDownloads...).
			AddEnvironmentToFileDelete(returnedFileDeletes...).
			AddEnvironmentToFileExtract(returnedFileExtracts...).
			AddEnvironmentToIdentity(returnedIdentities...).
			AddEnvironmentToNetwork(returnedNetworks...).
			AddEnvironmentToHost(returnedHosts...).
			AddEnvironmentToHostDependency(returnedHostDependencies...).
			AddEnvironmentToIncludedNetwork(returnedIncludedNetworks...).
			AddEnvironmentToDNS(returnedDNS...).
			AddEnvironmentToValidation(returnedValidations...).
			AddEnvironmentToAnsible(returnedAnsible...).
			AddEnvironmentToAnsible(returnedAnsible...).
			Save(ctx)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Failed to Update Environment %v with it's edges. Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		_, err = validateHostDependencies(txClient, ctx, log, returnedHostDependencies, cEnviroment.HclID)
		if err != nil {
			err = rollback(txClient, err)
			log.Log.Errorf("Failed to Validate Host Dependencies in Environment %v. Err: %v", cEnviroment.HclID, err)
			return nil, err
		}
		returnedEnvironment = append(returnedEnvironment, entEnvironment)
	}
	err = txClient.Commit()
	if err != nil {
		err = rollback(txClient, err)
		log.Log.Errorf("Failed Commit Environments. Err: %v", err)
		return nil, err
	}
	log.Log.Info("Sucessfully imported ENV")
	return returnedEnvironment, nil
}

func createCompetitions(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configCompetitions map[string]*ent.Competition, envHclID string) ([]*ent.Competition, []*ent.DNS, error) {
	bulk := []*ent.CompetitionCreate{}
	returnedCompetitions := []*ent.Competition{}
	returnedAllDNS := []*ent.DNS{}
	for _, cCompetition := range configCompetitions {
		log.Log.Debugf("Creating Competition: %v for Env: %v", cCompetition.HclID, envHclID)
		returnedDNS, err := createDNS(txClient, ctx, log, cCompetition.HCLCompetitionToDNS, envHclID)
		if err != nil {
			return nil, nil, err
		}
		entCompetition, err := txClient.Competition.
			Query().
			Where(
				competition.And(
					competition.HclIDEQ(cCompetition.HclID),
					competition.HasCompetitionToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Competition.Create().
					SetConfig(cCompetition.Config).
					SetHclID(cCompetition.HclID).
					SetRootPassword(cCompetition.RootPassword).
					SetTags(cCompetition.Tags).
					AddCompetitionToDNS(returnedDNS...)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entCompetition, err = entCompetition.Update().
			SetConfig(cCompetition.Config).
			SetHclID(cCompetition.HclID).
			SetRootPassword(cCompetition.RootPassword).
			SetTags(cCompetition.Tags).
			ClearCompetitionToDNS().
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Competition %v. Err: %v", cCompetition.HclID, err)
			return nil, nil, err
		}
		_, err = entCompetition.Update().AddCompetitionToDNS(returnedDNS...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Competition %v with DNS. Err: %v", cCompetition.HclID, err)
			return nil, nil, err
		}
		returnedAllDNS = append(returnedAllDNS, returnedDNS...)
		returnedCompetitions = append(returnedCompetitions, entCompetition)
	}
	if len(bulk) > 0 {
		dbCompetitions, err := txClient.Competition.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Competitions. Err: %v", err)
			return nil, nil, err
		}
		returnedCompetitions = append(returnedCompetitions, dbCompetitions...)
	}
	return returnedCompetitions, returnedAllDNS, nil
}

func removeDuplicateValues(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func createHosts(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configHosts map[string]*ent.Host, envHclID string, environmentHosts []string) ([]*ent.Host, []*ent.HostDependency, error) {
	returnedHosts := []*ent.Host{}
	returnedAllHostDependencies := []*ent.HostDependency{}
	environmentHosts = removeDuplicateValues(environmentHosts)
	for _, cHostID := range environmentHosts {
		log.Log.Debugf("Creating Host: %v for Env: %v", cHostID, envHclID)
		cHost, ok := configHosts[cHostID]
		if !ok {
			log.Log.Errorf("Host %v was not defined in the Enviroment %v", cHostID, envHclID)
			return nil, nil, fmt.Errorf("err: Host %v was not defined in the Enviroment %v", cHostID, envHclID)
		}
		returnedDisk, err := createDisk(txClient, ctx, log, cHost.HCLHostToDisk, cHost.HclID, envHclID)
		if err != nil {
			return nil, nil, err
		}
		entHost, err := txClient.Host.
			Query().
			Where(
				host.And(
					host.HclIDEQ(cHost.HclID),
					host.HasHostToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				entHost, err = txClient.Host.Create().
					SetAllowMACChanges(cHost.AllowMACChanges).
					SetDescription(cHost.Description).
					SetExposedTCPPorts(cHost.ExposedTCPPorts).
					SetExposedUDPPorts(cHost.ExposedUDPPorts).
					SetHclID(cHost.HclID).
					SetHostname(cHost.Hostname).
					SetInstanceSize(cHost.InstanceSize).
					SetLastOctet(cHost.LastOctet).
					SetOS(cHost.OS).
					SetOverridePassword(cHost.OverridePassword).
					SetProvisionSteps(cHost.ProvisionSteps).
					SetScheduledSteps(cHost.ScheduledSteps).
					SetTags(cHost.Tags).
					SetUserGroups(cHost.UserGroups).
					SetVars(cHost.Vars).
					SetHostToDisk(returnedDisk).
					Save(ctx)
				if err != nil {
					log.Log.Errorf("Failed to Create Host %v. Err: %v", cHost.HclID, err)
					return nil, nil, err
				}
			} else {
				return nil, nil, err
			}
		} else {
			entHost, err = entHost.Update().
				SetAllowMACChanges(cHost.AllowMACChanges).
				SetDescription(cHost.Description).
				SetExposedTCPPorts(cHost.ExposedTCPPorts).
				SetExposedUDPPorts(cHost.ExposedUDPPorts).
				SetHclID(cHost.HclID).
				SetHostname(cHost.Hostname).
				SetInstanceSize(cHost.InstanceSize).
				SetLastOctet(cHost.LastOctet).
				SetOS(cHost.OS).
				SetOverridePassword(cHost.OverridePassword).
				SetProvisionSteps(cHost.ProvisionSteps).
				SetScheduledSteps(cHost.ScheduledSteps).
				SetTags(cHost.Tags).
				SetUserGroups(cHost.UserGroups).
				SetVars(cHost.Vars).
				ClearHostToDisk().
				Save(ctx)
			if err != nil {
				log.Log.Errorf("Failed to Update Host %v. Err: %v", cHost.HclID, err)
				return nil, nil, err
			}
			_, err = entHost.Update().SetHostToDisk(returnedDisk).Save(ctx)
			if err != nil {
				log.Log.Errorf("Failed to Update Disk to Host %v. Err: %v", cHost.HclID, err)
				return nil, nil, err
			}
		}

		returnedHosts = append(returnedHosts, entHost)
		returnedHostDependencies, err := createHostDependencies(txClient, ctx, log, cHost.HCLDependOnHostToHostDependency, envHclID, entHost)
		if err != nil {
			return nil, nil, err
		}
		returnedAllHostDependencies = append(returnedAllHostDependencies, returnedHostDependencies...)
	}
	return returnedHosts, returnedAllHostDependencies, nil
}

func createNetworks(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configNetworks map[string]*ent.Network, configIncludedNetworks []*ent.IncludedNetwork, envHclID string) ([]*ent.Network, error) {
	bulk := []*ent.NetworkCreate{}
	returnedNetworks := []*ent.Network{}
	for _, cNetwork := range configNetworks {
		log.Log.Debugf("Creating Network: %v for Env: %v", cNetwork.HclID, envHclID)
		included := false
		for _, cIncludedNetwork := range configIncludedNetworks {
			if cIncludedNetwork.Name == cNetwork.HclID {
				included = true
				break
			}
		}
		if !included {
			continue
		}
		entNetwork, err := txClient.Network.
			Query().
			Where(
				network.And(
					network.HclIDEQ(cNetwork.HclID),
					network.HasNetworkToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Network.Create().
					SetCidr(cNetwork.Cidr).
					SetHclID(cNetwork.HclID).
					SetName(cNetwork.Name).
					SetTags(cNetwork.Tags).
					SetVars(cNetwork.Vars).
					SetVdiVisible(cNetwork.VdiVisible)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entNetwork, err = entNetwork.Update().
			SetCidr(cNetwork.Cidr).
			SetHclID(cNetwork.HclID).
			SetName(cNetwork.Name).
			SetTags(cNetwork.Tags).
			SetVars(cNetwork.Vars).
			SetVdiVisible(cNetwork.VdiVisible).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Network %v. Err: %v", cNetwork.HclID, err)
			return nil, err
		}
		returnedNetworks = append(returnedNetworks, entNetwork)
	}
	if len(bulk) > 0 {
		dbNetwork, err := txClient.Network.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Networks. Err: %v", err)
			return nil, err
		}
		returnedNetworks = append(returnedNetworks, dbNetwork...)
	}
	return returnedNetworks, nil
}

func createScripts(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configScript map[string]*ent.Script, envHclID string) ([]*ent.Script, []*ent.Finding, error) {
	bulk := []*ent.ScriptCreate{}
	returnedScripts := []*ent.Script{}
	returnedAllFindings := []*ent.Finding{}
	for _, cScript := range configScript {
		log.Log.Debugf("Creating Script: %v for Env: %v", cScript.HclID, envHclID)
		returnedFindings, err := createFindings(txClient, ctx, log, cScript.HCLScriptToFinding, envHclID, cScript.HclID)
		if err != nil {
			return nil, nil, err
		}
		for _, validationHCLID := range cScript.Validations {
			exist, err := txClient.Validation.Query().
				Where(
					validation.HclIDEQ(validationHCLID),
				).Exist(ctx)
			if err != nil {
				return nil, nil, err
			}

			if !exist {
				return nil, nil, fmt.Errorf("validation \"%s\" does not exist for script \"%s\"", validationHCLID, cScript.HclID)
			}
		}

		entScript, err := txClient.Script.
			Query().
			Where(
				script.And(
					script.HclIDEQ(cScript.HclID),
					script.HasScriptToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Script.Create().
					SetHclID(cScript.HclID).
					SetName(cScript.Name).
					SetLanguage(cScript.Language).
					SetDescription(cScript.Description).
					SetSource(cScript.Source).
					SetSourceType(cScript.SourceType).
					SetCooldown(cScript.Cooldown).
					SetTimeout(cScript.Timeout).
					SetIgnoreErrors(cScript.IgnoreErrors).
					SetArgs(cScript.Args).
					SetDisabled(cScript.Disabled).
					SetVars(cScript.Vars).
					SetTags(cScript.Tags).
					SetAbsPath(cScript.AbsPath).
					SetValidations(cScript.Validations).
					AddScriptToFinding(returnedFindings...)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entScript, err = entScript.Update().
			SetHclID(cScript.HclID).
			SetName(cScript.Name).
			SetLanguage(cScript.Language).
			SetDescription(cScript.Description).
			SetSource(cScript.Source).
			SetSourceType(cScript.SourceType).
			SetCooldown(cScript.Cooldown).
			SetTimeout(cScript.Timeout).
			SetIgnoreErrors(cScript.IgnoreErrors).
			SetArgs(cScript.Args).
			SetDisabled(cScript.Disabled).
			SetVars(cScript.Vars).
			SetTags(cScript.Tags).
			SetAbsPath(cScript.AbsPath).
			SetValidations(cScript.Validations).
			ClearScriptToFinding().
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Script %v. Err: %v", cScript.HclID, err)
			return nil, nil, err
		}
		_, err = entScript.Update().AddScriptToFinding(returnedFindings...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Script %v with it's Findings. Err: %v", cScript.HclID, err)
			return nil, nil, err
		}
		returnedAllFindings = append(returnedAllFindings, returnedFindings...)
		returnedScripts = append(returnedScripts, entScript)
	}
	if len(bulk) > 0 {
		dbScript, err := txClient.Script.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Scripts. Err: %v", err)
			return nil, nil, err
		}
		returnedScripts = append(returnedScripts, dbScript...)
	}
	return returnedScripts, returnedAllFindings, nil
}

func createAnsible(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configAnsible map[string]*ent.Ansible, envHclID string) ([]*ent.Ansible, error) {
	bulk := []*ent.AnsibleCreate{}
	returnedAnsible := []*ent.Ansible{}
	for _, cAnsible := range configAnsible {
		log.Log.Debugf("Creating Ansible: %v for Env: %v", cAnsible.HclID, envHclID)

		for _, validationHCLID := range cAnsible.Validations {
			exist, err := txClient.Validation.Query().
				Where(
					validation.HclIDEQ(validationHCLID),
				).Exist(ctx)
			if err != nil {
				return nil, err
			}

			if !exist {
				return nil, fmt.Errorf("validation \"%s\" does not exist for ansible \"%s\"", validationHCLID, cAnsible.HclID)
			}
		}

		entAnsible, err := txClient.Ansible.
			Query().
			Where(
				ansible.And(
					ansible.HclIDEQ(cAnsible.HclID),
					ansible.HasAnsibleFromEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Ansible.Create().
					SetName(cAnsible.Name).
					SetHclID(cAnsible.HclID).
					SetDescription(cAnsible.Description).
					SetSource(cAnsible.Source).
					SetPlaybookName(cAnsible.PlaybookName).
					SetMethod(cAnsible.Method).
					SetInventory(cAnsible.Inventory).
					SetTags(cAnsible.Tags).
					SetAbsPath(cAnsible.AbsPath).
					SetValidations(cAnsible.Validations)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entAnsible, err = entAnsible.Update().
			SetName(cAnsible.Name).
			SetHclID(cAnsible.HclID).
			SetDescription(cAnsible.Description).
			SetSource(cAnsible.Source).
			SetPlaybookName(cAnsible.PlaybookName).
			SetMethod(cAnsible.Method).
			SetInventory(cAnsible.Inventory).
			SetTags(cAnsible.Tags).
			SetAbsPath(cAnsible.AbsPath).
			SetValidations(cAnsible.Validations).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Ansible %v. Err: %v", cAnsible.HclID, err)
			return nil, err
		}
		returnedAnsible = append(returnedAnsible, entAnsible)
	}
	if len(bulk) > 0 {
		dbAnsible, err := txClient.Ansible.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Ansible. Err: %v", err)
			return nil, err
		}
		returnedAnsible = append(returnedAnsible, dbAnsible...)
	}
	return returnedAnsible, nil
}

func createScheduledStep(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configScheduledSteps map[string]*ent.ScheduledStep, envHclID string) ([]*ent.ScheduledStep, error) {
	bulk := []*ent.ScheduledStepCreate{}
	returnedScheduledSteps := []*ent.ScheduledStep{}
	for _, cScheduledStep := range configScheduledSteps {
		log.Log.Debugf("Creating Scheduled Step: %v for Env: %v", cScheduledStep.HclID, envHclID)
		entScheduledStep, err := txClient.ScheduledStep.
			Query().
			Where(
				scheduledstep.And(
					scheduledstep.HclIDEQ(cScheduledStep.HclID),
					scheduledstep.HasScheduledStepToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.ScheduledStep.Create().
					SetName(cScheduledStep.Name).
					SetHclID(cScheduledStep.HclID).
					SetDescription(cScheduledStep.Description).
					SetStep(cScheduledStep.Step).
					SetType(cScheduledStep.Type).
					SetSchedule(cScheduledStep.Schedule).
					SetRunAt(cScheduledStep.RunAt)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entScheduledStep, err = entScheduledStep.Update().
			SetName(cScheduledStep.Name).
			SetHclID(cScheduledStep.HclID).
			SetDescription(cScheduledStep.Description).
			SetStep(cScheduledStep.Step).
			SetType(cScheduledStep.Type).
			SetSchedule(cScheduledStep.Schedule).
			SetRunAt(cScheduledStep.RunAt).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Scheduled Step %v. Err: %v", cScheduledStep.HclID, err)
			return nil, err
		}
		returnedScheduledSteps = append(returnedScheduledSteps, entScheduledStep)
	}
	if len(bulk) > 0 {
		dbAnsible, err := txClient.ScheduledStep.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Scheduled Step. Err: %v", err)
			return nil, err
		}
		returnedScheduledSteps = append(returnedScheduledSteps, dbAnsible...)
	}
	return returnedScheduledSteps, nil
}

func createCommands(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configCommands map[string]*ent.Command, envHclID string) ([]*ent.Command, error) {
	bulk := []*ent.CommandCreate{}
	returnedCommands := []*ent.Command{}
	for _, cCommand := range configCommands {
		log.Log.Debugf("Creating Command: %v for Env: %v", cCommand.HclID, envHclID)

		for _, validationHCLID := range cCommand.Validations {
			exist, err := txClient.Validation.Query().
				Where(
					validation.HclIDEQ(validationHCLID),
				).Exist(ctx)
			if err != nil {
				return nil, err
			}

			if !exist {
				return nil, fmt.Errorf("validation \"%s\" does not exist for command \"%s\"", validationHCLID, cCommand.HclID)
			}
		}

		entCommand, err := txClient.Command.
			Query().
			Where(
				command.And(
					command.HclIDEQ(cCommand.HclID),
					command.HasCommandToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Command.Create().
					SetArgs(cCommand.Args).
					SetCooldown(cCommand.Cooldown).
					SetDescription(cCommand.Description).
					SetDisabled(cCommand.Disabled).
					SetHclID(cCommand.HclID).
					SetIgnoreErrors(cCommand.IgnoreErrors).
					SetName(cCommand.Name).
					SetProgram(cCommand.Program).
					SetTags(cCommand.Tags).
					SetTimeout(cCommand.Timeout).
					SetVars(cCommand.Vars).
					SetValidations(cCommand.Validations)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entCommand, err = entCommand.Update().
			SetArgs(cCommand.Args).
			SetCooldown(cCommand.Cooldown).
			SetDescription(cCommand.Description).
			SetDisabled(cCommand.Disabled).
			SetHclID(cCommand.HclID).
			SetIgnoreErrors(cCommand.IgnoreErrors).
			SetName(cCommand.Name).
			SetProgram(cCommand.Program).
			SetTags(cCommand.Tags).
			SetTimeout(cCommand.Timeout).
			SetVars(cCommand.Vars).
			SetValidations(cCommand.Validations).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Command %v. Err: %v", cCommand.HclID, err)
			return nil, err
		}
		returnedCommands = append(returnedCommands, entCommand)
	}
	if len(bulk) > 0 {
		dbCommand, err := txClient.Command.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Command. Err: %v", err)
			return nil, err
		}
		returnedCommands = append(returnedCommands, dbCommand...)
	}
	return returnedCommands, nil
}

func createDNSRecords(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configDNSRecords map[string]*ent.DNSRecord, envHclID string) ([]*ent.DNSRecord, error) {
	bulk := []*ent.DNSRecordCreate{}
	returnedDNSRecords := []*ent.DNSRecord{}
	for _, cDNSRecord := range configDNSRecords {
		log.Log.Debugf("Creating DNSRecord: %v for Env: %v", cDNSRecord.HclID, envHclID)

		for _, validationHCLID := range cDNSRecord.Validations {
			exist, err := txClient.Validation.Query().
				Where(
					validation.HclIDEQ(validationHCLID),
				).Exist(ctx)
			if err != nil {
				return nil, err
			}

			if !exist {
				return nil, fmt.Errorf("validation \"%s\" does not exist for DNSRecord \"%s\"", validationHCLID, cDNSRecord.HclID)
			}
		}

		entDNSRecord, err := txClient.DNSRecord.
			Query().
			Where(
				dnsrecord.And(
					dnsrecord.HclIDEQ(cDNSRecord.HclID),
					dnsrecord.HasDNSRecordToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.DNSRecord.Create().
					SetDisabled(cDNSRecord.Disabled).
					SetHclID(cDNSRecord.HclID).
					SetName(cDNSRecord.Name).
					SetTags(cDNSRecord.Tags).
					SetType(cDNSRecord.Type).
					SetValues(cDNSRecord.Values).
					SetVars(cDNSRecord.Vars).
					SetZone(cDNSRecord.Zone).
					SetValidations(cDNSRecord.Validations)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entDNSRecord, err = entDNSRecord.Update().
			SetDisabled(cDNSRecord.Disabled).
			SetHclID(cDNSRecord.HclID).
			SetName(cDNSRecord.Name).
			SetTags(cDNSRecord.Tags).
			SetType(cDNSRecord.Type).
			SetValues(cDNSRecord.Values).
			SetVars(cDNSRecord.Vars).
			SetZone(cDNSRecord.Zone).
			SetValidations(cDNSRecord.Validations).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update DNS Record %v. Err: %v", cDNSRecord.HclID, err)
			return nil, err
		}
		returnedDNSRecords = append(returnedDNSRecords, entDNSRecord)
	}
	if len(bulk) > 0 {
		dbDNSRecords, err := txClient.DNSRecord.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk DNS Records. Err: %v", err)
			return nil, err
		}
		returnedDNSRecords = append(returnedDNSRecords, dbDNSRecords...)
	}
	return returnedDNSRecords, nil
}

func createValidations(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configValidations map[string]*ent.Validation, envHclID string) ([]*ent.Validation, error) {
	bulk := []*ent.ValidationCreate{}
	returnedValidations := []*ent.Validation{}
	for _, cValidation := range configValidations {
		log.Log.Debugf("Creating Validations: %v for Env: %v", cValidation.HclID, envHclID)

		entValidation, err := txClient.Validation.
			Query().
			Where(
				validation.And(
					validation.HclIDEQ(cValidation.HclID),
					validation.HasValidationToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Validation.Create().
					SetHclID(cValidation.HclID).
					SetValidationType(cValidation.ValidationType).
					SetOutput(cValidation.Output).
					SetState(cValidation.State).
					SetErrorMessage(cValidation.ErrorMessage).
					SetHash(cValidation.Hash).
					SetRegex(cValidation.Regex).
					SetIP(cValidation.IP).
					SetPort(cValidation.Port).
					SetHostname(cValidation.Hostname).
					SetNameservers(cValidation.Nameservers).
					SetPackageName(cValidation.PackageName).
					SetUsername(cValidation.Username).
					SetGroupName(cValidation.GroupName).
					SetFilePath(cValidation.FilePath).
					SetSearchString(cValidation.SearchString).
					SetServiceName(cValidation.ServiceName).
					SetServiceStatus(cValidation.ServiceStatus).
					SetProcessName(cValidation.ProcessName)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entValidation, err = entValidation.Update().
			SetHclID(cValidation.HclID).
			SetValidationType(cValidation.ValidationType).
			SetOutput(cValidation.Output).
			SetState(cValidation.State).
			SetErrorMessage(cValidation.ErrorMessage).
			SetHash(cValidation.Hash).
			SetRegex(cValidation.Regex).
			SetIP(cValidation.IP).
			SetPort(cValidation.Port).
			SetHostname(cValidation.Hostname).
			SetNameservers(cValidation.Nameservers).
			SetPackageName(cValidation.PackageName).
			SetUsername(cValidation.Username).
			SetGroupName(cValidation.GroupName).
			SetFilePath(cValidation.FilePath).
			SetSearchString(cValidation.SearchString).
			SetServiceName(cValidation.ServiceName).
			SetServiceStatus(cValidation.ServiceStatus).
			SetProcessName(cValidation.ProcessName).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Validation &v. Err: %v", cValidation, err)
			return nil, err
		}
		returnedValidations = append(returnedValidations, entValidation)
	}
	if len(bulk) > 0 {
		dbValidators, err := txClient.Validation.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("failed to create bulk Validators. Err: %v", err)
			return nil, err
		}
		returnedValidations = append(returnedValidations, dbValidators...)
	}

	return returnedValidations, nil
}

func createFileDownload(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configFileDownloads map[string]*ent.FileDownload, envHclID string) ([]*ent.FileDownload, error) {
	bulk := []*ent.FileDownloadCreate{}
	returnedFileDownloads := []*ent.FileDownload{}
	for _, cFileDownload := range configFileDownloads {
		log.Log.Debugf("Creating FileDownload: %v for Env: %v", cFileDownload.HclID, envHclID)

		for _, validationHCLID := range cFileDownload.Validations {
			exist, err := txClient.Validation.Query().
				Where(
					validation.HclIDEQ(validationHCLID),
				).Exist(ctx)
			if err != nil {
				return nil, err
			}

			if !exist {
				return nil, fmt.Errorf("validation \"%s\" does not exist for FileDownload \"%s\"", validationHCLID, cFileDownload.HclID)
			}
		}

		entFileDownload, err := txClient.FileDownload.
			Query().
			Where(
				filedownload.And(
					filedownload.HclIDEQ(cFileDownload.HclID),
					filedownload.HasFileDownloadToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.FileDownload.Create().
					SetHclID(cFileDownload.HclID).
					SetSourceType(cFileDownload.SourceType).
					SetSource(cFileDownload.Source).
					SetDestination(cFileDownload.Destination).
					SetTemplate(cFileDownload.Template).
					SetPerms(cFileDownload.Perms).
					SetDisabled(cFileDownload.Disabled).
					SetMd5(cFileDownload.Md5).
					SetAbsPath(cFileDownload.AbsPath).
					SetTags(cFileDownload.Tags).
					SetIsTxt(cFileDownload.IsTxt).
					SetValidations(cFileDownload.Validations)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entFileDownload, err = entFileDownload.Update().
			SetHclID(cFileDownload.HclID).
			SetSourceType(cFileDownload.SourceType).
			SetSource(cFileDownload.Source).
			SetDestination(cFileDownload.Destination).
			SetTemplate(cFileDownload.Template).
			SetPerms(cFileDownload.Perms).
			SetDisabled(cFileDownload.Disabled).
			SetMd5(cFileDownload.Md5).
			SetAbsPath(cFileDownload.AbsPath).
			SetTags(cFileDownload.Tags).
			SetIsTxt(cFileDownload.IsTxt).
			SetValidations(cFileDownload.Validations).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update File Download %v. Err: %v", cFileDownload.HclID, err)
			return nil, err
		}
		returnedFileDownloads = append(returnedFileDownloads, entFileDownload)
	}
	if len(bulk) > 0 {
		dbFileDownloads, err := txClient.FileDownload.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk File Downloads. Err: %v", err)
			return nil, err
		}
		returnedFileDownloads = append(returnedFileDownloads, dbFileDownloads...)
	}
	return returnedFileDownloads, nil
}

func createFileDelete(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configFileDeletes map[string]*ent.FileDelete, envHclID string) ([]*ent.FileDelete, error) {
	bulk := []*ent.FileDeleteCreate{}
	returnedFileDeletes := []*ent.FileDelete{}
	for _, cFileDelete := range configFileDeletes {
		log.Log.Debugf("Creating FileDelete: %v for Env: %v", cFileDelete.HclID, envHclID)

		for _, validationHCLID := range cFileDelete.Validations {
			exist, err := txClient.Validation.Query().
				Where(
					validation.HclIDEQ(validationHCLID),
				).Exist(ctx)
			if err != nil {
				return nil, err
			}

			if !exist {
				return nil, fmt.Errorf("validation \"%s\" does not exist for FileDelete \"%s\"", validationHCLID, cFileDelete.HclID)
			}
		}

		entFileDelete, err := txClient.FileDelete.
			Query().
			Where(
				filedelete.And(
					filedelete.HclIDEQ(cFileDelete.HclID),
					filedelete.HasFileDeleteToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.FileDelete.Create().
					SetHclID(cFileDelete.HclID).
					SetPath(cFileDelete.Path).
					SetTags(cFileDelete.Tags).
					SetValidations(cFileDelete.Validations)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entFileDelete, err = entFileDelete.Update().
			SetHclID(cFileDelete.HclID).
			SetPath(cFileDelete.Path).
			SetTags(cFileDelete.Tags).
			SetValidations(cFileDelete.Validations).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update File Delete %v. Err: %v", cFileDelete.HclID, err)
			return nil, err
		}
		returnedFileDeletes = append(returnedFileDeletes, entFileDelete)
	}
	if len(bulk) > 0 {
		dbFileDelete, err := txClient.FileDelete.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk File Delete. Err: %v", err)
			return nil, err
		}
		returnedFileDeletes = append(returnedFileDeletes, dbFileDelete...)
	}
	return returnedFileDeletes, nil
}

func createFileExtract(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configFileExtracts map[string]*ent.FileExtract, envHclID string) ([]*ent.FileExtract, error) {
	bulk := []*ent.FileExtractCreate{}
	returnedFileExtracts := []*ent.FileExtract{}
	for _, cFileExtract := range configFileExtracts {
		log.Log.Debugf("Creating FileExtract: %v for Env: %v", cFileExtract.HclID, envHclID)

		for _, validationHCLID := range cFileExtract.Validations {
			exist, err := txClient.Validation.Query().
				Where(
					validation.HclIDEQ(validationHCLID),
				).Exist(ctx)
			if err != nil {
				return nil, err
			}

			if !exist {
				return nil, fmt.Errorf("validation \"%s\" does not exist for script \"%s\"", validationHCLID, cFileExtract.HclID)
			}
		}

		entFileExtract, err := txClient.FileExtract.
			Query().
			Where(
				fileextract.And(
					fileextract.HclIDEQ(cFileExtract.HclID),
					fileextract.HasFileExtractToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.FileExtract.Create().
					SetDestination(cFileExtract.Destination).
					SetHclID(cFileExtract.HclID).
					SetSource(cFileExtract.Source).
					SetTags(cFileExtract.Tags).
					SetType(cFileExtract.Type).
					SetValidations(cFileExtract.Validations)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entFileExtract, err = entFileExtract.Update().
			SetDestination(cFileExtract.Destination).
			SetHclID(cFileExtract.HclID).
			SetSource(cFileExtract.Source).
			SetTags(cFileExtract.Tags).
			SetType(cFileExtract.Type).
			SetValidations(cFileExtract.Validations).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update File Extract %v. Err: %v", cFileExtract.HclID, err)
			return nil, err
		}
		returnedFileExtracts = append(returnedFileExtracts, entFileExtract)
	}
	if len(bulk) > 0 {
		dbFileExtracts, err := txClient.FileExtract.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk File Extract. Err: %v", err)
			return nil, err
		}
		returnedFileExtracts = append(returnedFileExtracts, dbFileExtracts...)
	}
	return returnedFileExtracts, nil
}

func createIdentities(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configIdentities map[string]*ent.Identity, envHclID string) ([]*ent.Identity, error) {
	bulk := []*ent.IdentityCreate{}
	returnedIdentities := []*ent.Identity{}
	for _, cIdentity := range configIdentities {
		log.Log.Debugf("Creating Identity: %v for Env: %v", cIdentity.HclID, envHclID)

		entIdentity, err := txClient.Identity.
			Query().
			Where(
				identity.And(
					identity.HclIDEQ(cIdentity.HclID),
					identity.HasIdentityToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Identity.Create().
					SetAvatarFile(cIdentity.AvatarFile).
					SetDescription(cIdentity.Description).
					SetEmail(cIdentity.Email).
					SetFirstName(cIdentity.FirstName).
					SetHclID(cIdentity.HclID).
					SetLastName(cIdentity.LastName).
					SetPassword(cIdentity.Password).
					SetVars(cIdentity.Vars).
					SetTags(cIdentity.Tags)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entIdentity, err = entIdentity.Update().
			SetAvatarFile(cIdentity.AvatarFile).
			SetDescription(cIdentity.Description).
			SetEmail(cIdentity.Email).
			SetFirstName(cIdentity.FirstName).
			SetHclID(cIdentity.HclID).
			SetLastName(cIdentity.LastName).
			SetPassword(cIdentity.Password).
			SetVars(cIdentity.Vars).
			SetTags(cIdentity.Tags).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Identity %v. Err: %v", cIdentity.HclID, err)
			return nil, err
		}
		returnedIdentities = append(returnedIdentities, entIdentity)
	}
	if len(bulk) > 0 {
		dbIdentities, err := txClient.Identity.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Identities. Err: %v", err)
			return nil, err
		}
		returnedIdentities = append(returnedIdentities, dbIdentities...)
	}
	return returnedIdentities, nil
}

func createFindings(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configFindings []*ent.Finding, envHclID string, entScriptID string) ([]*ent.Finding, error) {
	bulk := []*ent.FindingCreate{}
	returnedFindings := []*ent.Finding{}
	for _, cFinding := range configFindings {
		log.Log.Debugf("Creating Finding: %v for Env: %v", cFinding.Name, envHclID)

		entFinding, err := txClient.Finding.
			Query().
			Where(
				finding.And(
					finding.Name(cFinding.Name),
					finding.HasFindingToEnvironmentWith(environment.HclIDEQ(envHclID)),
					finding.HasFindingToScriptWith(script.HclID(entScriptID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.Finding.Create().
					SetDescription(cFinding.Description).
					SetDifficulty(cFinding.Difficulty).
					SetName(cFinding.Name).
					SetSeverity(cFinding.Severity).
					SetTags(cFinding.Tags)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entFinding, err = entFinding.Update().
			SetDescription(cFinding.Description).
			SetDifficulty(cFinding.Difficulty).
			SetName(cFinding.Name).
			SetSeverity(cFinding.Severity).
			SetTags(cFinding.Tags).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Finding %v for Script %v in Enviroment %v. Err: %v", cFinding.Name, entScriptID, envHclID, err)
			return nil, err
		}
		returnedFindings = append(returnedFindings, entFinding)
	}
	if len(bulk) > 0 {
		dbFinding, err := txClient.Finding.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Findings. Err: %v", err)
			return nil, err
		}
		returnedFindings = append(returnedFindings, dbFinding...)
	}
	return returnedFindings, nil
}

func createHostDependencies(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configHostDependencies []*ent.HostDependency, envHclID string, dependByHost *ent.Host) ([]*ent.HostDependency, error) {
	bulk := []*ent.HostDependencyCreate{}
	returnedHostDependencies := []*ent.HostDependency{}
	for _, cHostDependency := range configHostDependencies {
		log.Log.Debugf("Creating HostDependency for host %v to host %v in network %v for Env: %v", dependByHost.HclID, cHostDependency.HostID, cHostDependency.NetworkID, envHclID)

		entHostDependency, err := txClient.HostDependency.
			Query().
			Where(
				hostdependency.And(
					hostdependency.HasHostDependencyToDependByHostWith(host.HclIDEQ(dependByHost.HclID)),
					hostdependency.HostIDEQ(cHostDependency.HostID),
					hostdependency.NetworkIDEQ(cHostDependency.NetworkID),
					hostdependency.HasHostDependencyToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.HostDependency.Create().
					SetHostID(cHostDependency.HostID).
					SetNetworkID(cHostDependency.NetworkID).
					SetHostDependencyToDependByHost(dependByHost)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entHostDependency, err = entHostDependency.Update().
			ClearHostDependencyToDependByHost().
			ClearHostDependencyToDependOnHost().
			ClearHostDependencyToNetwork().
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Clear Host Dependency by %v on Host %v Err: %v", dependByHost.HclID, cHostDependency.HostID, err)
			return nil, err
		}
		entHostDependency, err = entHostDependency.Update().
			SetHostDependencyToDependByHost(dependByHost).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update Host Dependency by %v on Host %v Err: %v", dependByHost.HclID, cHostDependency.HostID, err)
			return nil, err
		}
		returnedHostDependencies = append(returnedHostDependencies, entHostDependency)
	}
	if len(bulk) > 0 {
		dbHostDependency, err := txClient.HostDependency.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Host Dependencies. Err: %v", err)
			return nil, err
		}
		returnedHostDependencies = append(returnedHostDependencies, dbHostDependency...)
	}
	return returnedHostDependencies, nil
}

func createDNS(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configDNS []*ent.DNS, envHclID string) ([]*ent.DNS, error) {
	bulk := []*ent.DNSCreate{}
	returnedDNS := []*ent.DNS{}
	for _, cDNS := range configDNS {
		log.Log.Debugf("Creating DNS: %v for Env: %v", cDNS.HclID, envHclID)
		entDNS, err := txClient.DNS.
			Query().
			Where(
				dns.And(
					dns.HclIDEQ(cDNS.HclID),
					dns.HasDNSToEnvironmentWith(environment.HclIDEQ(envHclID)),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.DNS.Create().
					SetConfig(cDNS.Config).
					SetDNSServers(cDNS.DNSServers).
					SetHclID(cDNS.HclID).
					SetNtpServers(cDNS.NtpServers).
					SetRootDomain(cDNS.RootDomain).
					SetType(cDNS.Type)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entDNS, err = entDNS.Update().
			SetConfig(cDNS.Config).
			SetDNSServers(cDNS.DNSServers).
			SetHclID(cDNS.HclID).
			SetNtpServers(cDNS.NtpServers).
			SetRootDomain(cDNS.RootDomain).
			SetType(cDNS.Type).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to Update DNS %v. Err: %v", cDNS.HclID, err)
			return nil, err
		}
		returnedDNS = append(returnedDNS, entDNS)
	}
	if len(bulk) > 0 {
		dbDNS, err := txClient.DNS.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk DNS. Err: %v", err)
			return nil, err
		}
		returnedDNS = append(returnedDNS, dbDNS...)
	}
	return returnedDNS, nil
}

func createDisk(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configDisk *ent.Disk, hostHclID string, envHclID string) (*ent.Disk, error) {
	entDisk, err := txClient.Disk.
		Query().
		Where(
			disk.And(
				disk.HasDiskToHostWith(
					host.And(
						host.HclIDEQ(hostHclID),
						host.HasHostToEnvironmentWith(environment.HclIDEQ(envHclID)),
					),
				),
			),
		).
		Only(ctx)
	if err != nil {
		if err == err.(*ent.NotFoundError) {
			entDisk, err = txClient.Disk.Create().
				SetSize(configDisk.Size).
				Save(ctx)
			if err != nil {
				log.Log.Errorf("Failed to create Disk for Host %v. Err: %v", hostHclID, err)
				return nil, err
			}
		}
	}
	entDisk, err = entDisk.Update().
		SetSize(configDisk.Size).
		Save(ctx)
	if err != nil {
		log.Log.Errorf("Failed to update Disk Size for Host %v. Err: %v", hostHclID, err)
		return nil, err
	}
	return entDisk, nil
}

func createIncludedNetwork(txClient *ent.Tx, ctx context.Context, log *logging.Logger, configIncludedNetworks []*ent.IncludedNetwork, envHclID string, returnedHostIDs, returnedNetworkIDS []uuid.UUID) ([]*ent.IncludedNetwork, error) {
	bulk := []*ent.IncludedNetworkCreate{}
	returnedIncludedNetworks := []*ent.IncludedNetwork{}
	for _, cIncludedNetwork := range configIncludedNetworks {
		entNetwork, err := txClient.Network.Query().Where(
			network.And(
				network.HclIDEQ(cIncludedNetwork.Name),
				network.IDIn(returnedNetworkIDS...),
				// network.Or(
				// 	network.Not(network.HasNetworkToEnvironment()),
				// 	network.HasNetworkToEnvironmentWith(environment.HclIDEQ(envHclID)),
				// ),
			),
		).Only(ctx)
		if err != nil {
			log.Log.Errorf("Unable to Query %v network in %v enviroment. Err: %v", cIncludedNetwork.Name, envHclID, err)
			return nil, err
		}
		entHosts := []*ent.Host{}
		for _, cHostHclID := range cIncludedNetwork.Hosts {
			entHost, err := txClient.Host.Query().Where(
				host.And(
					host.HclIDEQ(cHostHclID),
					host.IDIn(returnedHostIDs...),
					// host.Or(
					// 	host.Not(host.HasHostToEnvironment()),
					// 	host.HasHostToEnvironmentWith(environment.HclIDEQ(envHclID)),
					// ),
				),
			).Only(ctx)
			if err != nil {
				log.Log.Errorf("Unable to Query %v host in %v enviroment. Err: %v", cHostHclID, envHclID, err)
				return nil, err
			}
			entHosts = append(entHosts, entHost)
		}
		entIncludedNetwork, err := txClient.IncludedNetwork.
			Query().
			Where(
				includednetwork.And(
					includednetwork.HasIncludedNetworkToEnvironmentWith(environment.HclIDEQ(envHclID)),
					includednetwork.NameEQ(cIncludedNetwork.Name),
				),
			).
			Only(ctx)
		if err != nil {
			if err == err.(*ent.NotFoundError) {
				createdQuery := txClient.IncludedNetwork.Create().
					SetName(cIncludedNetwork.Name).
					SetHosts(cIncludedNetwork.Hosts).
					SetIncludedNetworkToNetwork(entNetwork).
					AddIncludedNetworkToHost(entHosts...)
				bulk = append(bulk, createdQuery)
				continue
			}
		}
		entIncludedNetwork, err = entIncludedNetwork.Update().
			SetName(cIncludedNetwork.Name).
			SetHosts(cIncludedNetwork.Hosts).
			ClearIncludedNetworkToHost().
			ClearIncludedNetworkToNetwork().
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to update the Included Network %v with Hosts %v. Err: %v", cIncludedNetwork.Name, cIncludedNetwork.Hosts, err)
			return nil, err
		}
		entIncludedNetwork, err = entIncludedNetwork.Update().
			AddIncludedNetworkToHost(entHosts...).
			SetIncludedNetworkToNetwork(entNetwork).
			Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to update the Included Network %v Edges with Hosts %v. Err: %v", cIncludedNetwork.Name, cIncludedNetwork.Hosts, err)
			return nil, err
		}
		returnedIncludedNetworks = append(returnedIncludedNetworks, entIncludedNetwork)
	}
	if len(bulk) > 0 {
		dbIncludedNetwork, err := txClient.IncludedNetwork.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			log.Log.Errorf("Failed to create bulk Included Network. Err: %v", err)
			return nil, err
		}
		returnedIncludedNetworks = append(returnedIncludedNetworks, dbIncludedNetwork...)
	}
	return returnedIncludedNetworks, nil
}

func validateHostDependencies(txClient *ent.Tx, ctx context.Context, log *logging.Logger, uncheckedHostDependencies []*ent.HostDependency, envHclID string) ([]*ent.HostDependency, error) {
	checkedHostDependencies := []*ent.HostDependency{}
	for _, uncheckedHostDependency := range uncheckedHostDependencies {
		entNetwork, err := txClient.Network.Query().Where(
			network.And(
				network.HclIDEQ(uncheckedHostDependency.NetworkID),
				network.HasNetworkToEnvironmentWith(environment.HclIDEQ(envHclID)),
			),
		).Only(ctx)
		if err != nil {
			log.Log.Errorf("Unable to Query %v network in %v enviroment. Err: %v", uncheckedHostDependency.NetworkID, envHclID, err)
			return nil, err
		}
		entHost, err := txClient.Host.Query().Where(
			host.And(
				host.HasHostToEnvironmentWith(environment.HclIDEQ(envHclID)),
				host.HclIDEQ(uncheckedHostDependency.HostID),
			),
		).Only(ctx)
		if err != nil {
			log.Log.Errorf("Unable to Query %v host in %v enviroment. Err: %v", uncheckedHostDependency.HostID, envHclID, err)
			return nil, err
		}
		_, err = txClient.IncludedNetwork.Query().Where(
			includednetwork.And(
				includednetwork.HasIncludedNetworkToEnvironmentWith(environment.HclIDEQ(envHclID)),
				includednetwork.HasIncludedNetworkToHostWith(host.HclIDEQ(uncheckedHostDependency.HostID)),
				includednetwork.HasIncludedNetworkToNetworkWith(network.HclIDEQ(uncheckedHostDependency.NetworkID)),
			),
		).Only(ctx)
		if err != nil {
			log.Log.Errorf("Unable to Verify %v host in %v network while loading %v enviroment. Err: %v", uncheckedHostDependency.HostID, uncheckedHostDependency.NetworkID, envHclID, err)
			return nil, err
		}
		uncheckedHostDependency, err := uncheckedHostDependency.Update().
			ClearHostDependencyToDependOnHost().
			ClearHostDependencyToNetwork().
			Save(ctx)
		if err != nil {
			dependedByHost, queryErr := uncheckedHostDependency.QueryHostDependencyToDependByHost().Only(ctx)
			if queryErr != nil {
				log.Log.Errorf("Unable to find the host depended by %v Err: %v", uncheckedHostDependency.HostID, queryErr)
				return nil, queryErr
			}
			log.Log.Errorf("Failed to clear the Host dependency of %v which relies on %v host in %v network. Err: %v", dependedByHost.HclID, uncheckedHostDependency.HostID, uncheckedHostDependency.NetworkID, err)
			return nil, err
		}
		entHostDependency, err := uncheckedHostDependency.Update().
			SetHostDependencyToDependOnHost(entHost).
			SetHostDependencyToNetwork(entNetwork).
			Save(ctx)
		if err != nil {
			dependedByHost, queryErr := uncheckedHostDependency.QueryHostDependencyToDependByHost().Only(ctx)
			if queryErr != nil {
				log.Log.Errorf("Unable to find the host depended by %v Err: %v", uncheckedHostDependency.HostID, queryErr)
				return nil, queryErr
			}
			log.Log.Errorf("Failed to update the Host dependency of %v which relies on %v host in %v network. Err: %v", dependedByHost.HclID, uncheckedHostDependency.HostID, uncheckedHostDependency.NetworkID, err)
			return nil, err
		}
		checkedHostDependencies = append(checkedHostDependencies, entHostDependency)

	}
	return checkedHostDependencies, nil
}
