package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akamensky/argparse"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-json/sanitize"
)

const VERSION = "0.4.3"

var TRUE = true

//go:embed ui/dist
var frontend embed.FS

type arrayFlags []string

func (i arrayFlags) String() string {
	var ts []string
	for _, el := range i {
		ts = append(ts, el)
	}
	return strings.Join(ts, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type rover struct {
	Name             string
	WorkingDir       string
	TfPath           string
	TfVarsFiles      []string
	TfVars           []string
	TfBackendConfigs []string
	PlanPath         string
	PlanJSONPath     string
	WorkspaceName    string
	TFCOrgName       string
	TFCWorkspaceName string
	ShowSensitive    bool
	GenImage         bool
	TFCNewRun        bool
	Plan             *tfjson.Plan
	RSO              *ResourcesOverview
	Map              *Map
	Graph            Graph
}

func main() {
	var tfPath, workingDir, name, zipFileName, ipPort, planPathPtr, planJSONPathPtr, workspaceName, tfcOrgName, tfcWorkspaceName *string
	var standalone, genImage, showSensitive, getVersion, tfcNewRun *bool
	var tfVarsFiles, tfVars, tfBackendConfigs arrayFlags

	parser := argparse.NewParser("rover", "Rover is a Terraform visualizer")
	tfPath = parser.String("", "tfPath", &argparse.Options{
		Required: false,
		Help:     "Path to Terraform binary",
		Default:  "/bin/terraform",
	})
	workingDir = parser.String("", "workingDir", &argparse.Options{
		Required: false,
		Help:     "Path to Terraform configuration",
		Default:  ".",
	})
	name = parser.String("", "name", &argparse.Options{
		Required: false,
		Help:     "Configuration name",
		Default:  "rover",
	})
	zipFileName = parser.String("", "zipFileName", &argparse.Options{
		Required: false,
		Help:     "Standalone zip file name",
		Default:  "rover",
	})
	ipPort = parser.String("", "ipPort", &argparse.Options{
		Required: false,
		Help:     "IP and port for Rover server",
		Default:  "0.0.0.0:9000",
	})
	planPathPtr = parser.String("", "planPath", &argparse.Options{
		Required: false,
		Help:     "Plan file path",
		Default:  "",
	})
	planJSONPathPtr = parser.String("", "planJSONPath", &argparse.Options{
		Required: false,
		Help:     "Plan JSON file path",
		Default:  "",
	})
	workspaceName = parser.String("", "workspaceName", &argparse.Options{
		Required: false,
		Help:     "Workspace name",
		Default:  "",
	})
	tfcOrgName = parser.String("", "tfcOrg", &argparse.Options{
		Required: false,
		Help:     "Terraform Cloud Organization name",
		Default:  "",
	})
	tfcWorkspaceName = parser.String("", "tfcWorkspace", &argparse.Options{
		Required: false,
		Help:     "Terraform Cloud Workspace name",
		Default:  "",
	})
	standalone = parser.Flag("", "standalone", &argparse.Options{
		Required: false,
		Help:     "Generate standalone HTML files",
		Default:  false,
	})
	showSensitive = parser.Flag("", "showSensitive", &argparse.Options{
		Required: false,
		Help:     "Display sensitive values",
		Default:  false,
	})
	tfcNewRun = parser.Flag("", "tfcNewRun", &argparse.Options{
		Required: false,
		Help:     "Create new Terraform Cloud run",
		Default:  false,
	})
	getVersion = parser.Flag("", "version", &argparse.Options{
		Required: false,
		Help:     "Get current version",
		Default:  false,
	})
	genImage = parser.Flag("", "genImage", &argparse.Options{
		Required: false,
		Help:     "Generate graph image",
		Default:  false,
	})
	tfVarsFilesTmp := parser.StringList("", "tfVarsFile", &argparse.Options{
		Required: false,
		Help:     "Path to *.tfvars files",
		Default:  []string{},
	})
	tfVarsTmp := parser.StringList("", "tfVar", &argparse.Options{
		Required: false,
		Help:     "Terraform variable (key=value)",
		Default:  []string{},
	})
	tfBackendConfigsTmp := parser.StringList("", "tfBackendConfig", &argparse.Options{
		Required: false,
		Help:     "Path to *.tfbackend files",
		Default:  []string{},
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	if *getVersion {
		fmt.Printf("Rover v%s\n", VERSION)
		return
	}

	for _, tfVarFile := range *tfVarsFilesTmp {
		tfVarsFiles.Set(tfVarFile)
	}
	for _, tfVar := range *tfVarsTmp {
		tfVars.Set(tfVar)
	}
	for _, tfBackendConfig := range *tfBackendConfigsTmp {
		tfBackendConfigs.Set(tfBackendConfig)
	}

	log.Println("Starting Rover...")

	parsedTfVarsFiles := strings.Split(tfVarsFiles.String(), ",")
	parsedTfVars := strings.Split(tfVars.String(), ",")
	parsedTfBackendConfigs := strings.Split(tfBackendConfigs.String(), ",")

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.New("unable to get current working directory"))
	}

	planPath := *planPathPtr
	if planPath != "" {
		if !strings.HasPrefix(planPath, "/") {
			planPath = filepath.Join(path, planPath)
		}
	}

	planJSONPath := *planJSONPathPtr
	if planJSONPath != "" {
		if !strings.HasPrefix(planJSONPath, "/") {
			planJSONPath = filepath.Join(path, planJSONPath)
		}
	}

	r := rover{
		Name:             *name,
		WorkingDir:       *workingDir,
		TfPath:           *tfPath,
		PlanPath:         planPath,
		PlanJSONPath:     planJSONPath,
		ShowSensitive:    *showSensitive,
		GenImage:         *genImage,
		TfVarsFiles:      parsedTfVarsFiles,
		TfVars:           parsedTfVars,
		TfBackendConfigs: parsedTfBackendConfigs,
		WorkspaceName:    *workspaceName,
		TFCOrgName:       *tfcOrgName,
		TFCWorkspaceName: *tfcWorkspaceName,
		TFCNewRun:        *tfcNewRun,
	}

	// Generate assets
	err = r.generateAssets()
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Println("Done generating assets.")

	// Save to file (debug)
	// saveJSONToFile(name, "plan", "output", r.Plan)
	// saveJSONToFile(name, "rso", "output", r.Plan)
	// saveJSONToFile(name, "map", "output", r.Map)
	// saveJSONToFile(name, "graph", "output", r.Graph)

	// Embed frontend
	fe, err := fs.Sub(frontend, "ui/dist")
	if err != nil {
		log.Fatalln(err)
	}
	frontendFS := http.FileServer(http.FS(fe))

	if *standalone {
		err = r.generateZip(fe, fmt.Sprintf("%s.zip", *zipFileName))
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Generated zip file: %s.zip\n", *zipFileName)
		return
	}

	err = r.startServer(*ipPort, frontendFS)
	if err != nil {
		// http.Serve() returns error on shutdown
		if *genImage {
			log.Println("Server shut down.")
		} else {
			log.Fatalf("Could not start server: %s\n", err.Error())
		}
	}

}

func (r *rover) generateAssets() error {
	// Get Plan
	err := r.getPlan()
	if err != nil {
		return fmt.Errorf("unable to parse Plan: %s", err)
	}

	// Generate RSO, Map, Graph
	err = r.GenerateResourceOverview()
	if err != nil {
		return err
	}

	err = r.GenerateMap()
	if err != nil {
		return err
	}

	err = r.GenerateGraph()
	if err != nil {
		return err
	}

	return nil
}

func (r *rover) getPlan() error {
	tmpDir, err := os.MkdirTemp("", "rover")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tf, err := tfexec.NewTerraform(r.WorkingDir, r.TfPath)
	if err != nil {
		return err
	}

	planSanitizer := func(r *rover) {
		if r.ShowSensitive || r.Plan == nil {
			return
		}

		tmp, err := sanitize.SanitizePlan(r.Plan)
		if err != nil {
			log.Println("Failed to sanitize plan file!")
			return
		} else {
			log.Println("Sanitized plan file")
		}
		r.Plan = tmp
	}
	defer planSanitizer(r)

	// If user provided path to plan file
	if r.PlanPath != "" {
		log.Println("Using provided plan...")
		r.Plan, err = tf.ShowPlanFile(context.Background(), r.PlanPath)
		if err != nil {
			return fmt.Errorf("unable to read Plan (%s): %s", r.PlanPath, err)
		}
		return nil
	}

	// If user provided path to plan JSON file
	if r.PlanJSONPath != "" {
		log.Println("Using provided JSON plan...")

		planJsonFile, err := os.Open(r.PlanJSONPath)
		if err != nil {
			return fmt.Errorf("unable to read Plan (%s): %s", r.PlanJSONPath, err)
		}
		defer planJsonFile.Close()

		planJson, err := io.ReadAll(planJsonFile)
		if err != nil {
			return fmt.Errorf("unable to read Plan (%s): %s", r.PlanJSONPath, err)
		}

		if err := json.Unmarshal(planJson, &r.Plan); err != nil {
			return fmt.Errorf("unable to read Plan (%s): %s", r.PlanJSONPath, err)
		}

		return nil
	}

	// If user specified TFC workspace
	if r.TFCWorkspaceName != "" {
		tfcToken := os.Getenv("TFC_TOKEN")

		if tfcToken == "" {
			return errors.New("TFC_TOKEN environment variable not set")
		}

		if r.TFCOrgName == "" {
			return errors.New("must specify Terraform Cloud organization to retrieve plan from Terraform Cloud")
		}

		config := &tfe.Config{
			Token: tfcToken,
		}

		client, err := tfe.NewClient(config)
		if err != nil {
			return fmt.Errorf("unable to connect to Terraform Cloud. %s", err)
		}

		// Get TFC Workspace
		ws, err := client.Workspaces.Read(context.Background(), r.TFCOrgName, r.TFCWorkspaceName)
		if err != nil {
			return fmt.Errorf("unable to list workspace %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err)
		}

		// Retrieve all runs from specified TFC workspace
		runs, err := client.Runs.List(context.Background(), ws.ID, &tfe.RunListOptions{})
		if err != nil {
			return fmt.Errorf("unable to retrieve plan from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err)
		}

		run := runs.Items[0]

		// Get most recent plan item
		planID := runs.Items[0].Plan.ID

		// Run hasn't been applied or discarded, therefore is still "actionable" by user
		runIsActionable := run.StatusTimestamps.AppliedAt.IsZero() && run.StatusTimestamps.DiscardedAt.IsZero()

		if runIsActionable && r.TFCNewRun {
			return fmt.Errorf("did not create new run. %s in %s in %s is still active", run.ID, r.TFCWorkspaceName, r.TFCOrgName)
		}

		// If latest run is not actionable, rover will create new run
		if r.TFCNewRun {
			// Create new run in specified TFC workspace
			newRun, err := client.Runs.Create(context.Background(), tfe.RunCreateOptions{
				Refresh:   &TRUE,
				Workspace: ws,
			})
			if err != nil {
				return fmt.Errorf("unable to generate new run from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err)
			}

			run = newRun

			log.Printf("Starting new Terraform Cloud run in %s workspace...", r.TFCWorkspaceName)

			// Wait maximum of 5 mins
			for i := 0; i < 30; i++ {
				run, err := client.Runs.Read(context.Background(), newRun.ID)
				if err != nil {
					return fmt.Errorf("unable to retrieve run from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err)
				}

				if run.Plan != nil {
					planID = run.Plan.ID
					// Add 20 second timeout so plan JSON becomes available
					time.Sleep(20 * time.Second)
					log.Printf("Run %s to completed!", newRun.ID)
					break
				}

				time.Sleep(10 * time.Second)
				log.Printf("Waiting for run %s to complete (%ds)...", newRun.ID, 10*(i+1))
			}

			if planID == "" {
				return fmt.Errorf("timeout waiting for plan to complete in %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err)
			}
		}

		// Get most recent plan file
		planBytes, err := client.Plans.ReadJSONOutput(context.Background(), planID)
		if err != nil {
			return fmt.Errorf("unable to retrieve plan from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err)
		}
		// If empty plan file
		if string(planBytes) == "" {
			return fmt.Errorf("empty plan, check run %s in %s in %s is not pending", run.ID, r.TFCWorkspaceName, r.TFCOrgName)
		}

		if err := json.Unmarshal(planBytes, &r.Plan); err != nil {
			return fmt.Errorf("unable to parse plan (ID: %s) from %s in %s organization.: %s", planID, r.TFCWorkspaceName, r.TFCOrgName, err)
		}

		return nil
	}

	log.Println("Initializing Terraform...")

	// Create TF Init options
	var tfInitOptions []tfexec.InitOption
	tfInitOptions = append(tfInitOptions, tfexec.Upgrade(true))

	// Add *.tfbackend files
	for _, tfBackendConfig := range r.TfBackendConfigs {
		if tfBackendConfig != "" {
			tfInitOptions = append(tfInitOptions, tfexec.BackendConfig(tfBackendConfig))
		}
	}

	// tfInitOptions = append(tfInitOptions, tfexec.LockTimeout("60s"))

	err = tf.Init(context.Background(), tfInitOptions...)
	if err != nil {
		return fmt.Errorf("unable to initialize Terraform Plan: %s", err)
	}

	if r.WorkspaceName != "" {
		log.Printf("Running in %s workspace...", r.WorkspaceName)
		err = tf.WorkspaceSelect(context.Background(), r.WorkspaceName)
		if err != nil {
			return fmt.Errorf("unable to select workspace (%s): %s", r.WorkspaceName, err)
		}
	}

	log.Println("Generating plan...")
	planPath := fmt.Sprintf("%s/%s-%v", tmpDir, "roverplan", time.Now().Unix())

	// Create TF Plan options
	var tfPlanOptions []tfexec.PlanOption
	tfPlanOptions = append(tfPlanOptions, tfexec.Out(planPath))

	// Add *.tfvars files
	for _, tfVarsFile := range r.TfVarsFiles {
		if tfVarsFile != "" {
			tfPlanOptions = append(tfPlanOptions, tfexec.VarFile(tfVarsFile))
		}
	}

	// Add Terraform variables
	for _, tfVar := range r.TfVars {
		if tfVar != "" {
			tfPlanOptions = append(tfPlanOptions, tfexec.Var(tfVar))
		}
	}

	_, err = tf.Plan(context.Background(), tfPlanOptions...)
	if err != nil {
		return fmt.Errorf("unable to run Plan: %s", err)
	}

	r.Plan, err = tf.ShowPlanFile(context.Background(), planPath)
	if err != nil {
		return fmt.Errorf("unable to read Plan: %s", err)
	}

	return nil
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
