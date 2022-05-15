package builder

import (
	"context"
	"fmt"
	"net/http"
	"time"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gen0cide/laforge/builder/aws"
	"github.com/gen0cide/laforge/builder/generic"
	lfopenstack "github.com/gen0cide/laforge/builder/openstack"
	"github.com/gen0cide/laforge/builder/vspherensxt"
	"github.com/gen0cide/laforge/builder/vspherensxt/nsxt"
	"github.com/gen0cide/laforge/builder/vspherensxt/vsphere"
	"github.com/gen0cide/laforge/configs"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

type Builder interface {
	ID() string
	Name() string
	Description() string
	Author() string
	Version() string
	DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error)
	DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error)
	DeployTeam(ctx context.Context, entTeam *ent.Team) (err error)
	TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error)
	TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error)
	TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error)
}

func BuilderFromEnvironment(buildersMap map[string]utils.BuilderConfig, environment *ent.Environment, logger *logging.Logger) (genericBuilder Builder, err error) {
	builderConfig, exists := buildersMap[environment.Builder]
	if !exists {
		return nil, fmt.Errorf("builder \"%s\" not found in builders config", environment.Builder)
	}

	switch builderConfig.Builder {
	case "vsphere-nsxt":
		genericBuilder, err = NewVSphereNSXTBuilder(builderConfig.ConfigFile, environment, logger)
		if err != nil {
			logrus.Errorf("Failed to make vSphere NSX-T builder. Err: %v", err)
			return
		}
		return
	case "aws":
		genericBuilder, err = NewAWSBuilder(builderConfig.ConfigFile, environment, logger)
		if err != nil {
			logrus.Errorf("Failed to make AWS builder. Err: %v", err)
			return
		}
		return
	case "openstack":
		genericBuilder, err = NewOpenstackBuilder(builderConfig.ConfigFile, environment, logger)
		if err != nil {
			logrus.Errorf("Failed to make openstack builder. Err: %v", err)
			return
		}
		return
	case "generic":
		genericBuilder, err = NewGenericBuilder(environment, logger)
		if err != nil {
			logrus.Errorf("Failed to make generic builder. Err: %v", err)
			return
		}
		return
	}

	err = fmt.Errorf("error: builder not found")
	logrus.Error(err)
	return
}

func NewAWSBuilder(configFilePath string, env *ent.Environment, logger *logging.Logger) (builder aws.AWSBuilder, err error) {
	var builderConfig aws.AWSBuilderConfig
	err = configs.LoadBuilderConfig(configFilePath, &builderConfig)
	if err != nil {
		return
	}

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithSharedCredentialsFiles([]string{builderConfig.AWSConfigFile}),
		awsConfig.WithRegion(builderConfig.Region))
	if err != nil {
		return
	}

	client := ec2.NewFromConfig(cfg)
	builder = aws.AWSBuilder{
		Logger:    logger,
		AMIConfig: builderConfig.AMIConfig,
		Config:    builderConfig,
		Client:    client,
	}

	return
}

// NewGenericBuilder creates a builder instance to deploy environments to NoWhere
func NewGenericBuilder(environment *ent.Environment, logger *logging.Logger) (builder generic.GenericBuilder, err error) {
	builder = generic.GenericBuilder{
		Logger: logger,
	}
	return
}

// NewVSphereNSXTBuilder creates a builder instance to deploy environments to VSphere and NSX-T
func NewVSphereNSXTBuilder(configFilePath string, env *ent.Environment, logger *logging.Logger) (builder vspherensxt.VSphereNSXTBuilder, err error) {
	var builderConfig vspherensxt.VSphereNSXTBuilderConfig
	err = configs.LoadBuilderConfig(configFilePath, &builderConfig)
	if err != nil {
		return
	}

	httpClient := http.Client{
		Timeout: 5 * time.Minute,
	}

	nsxtHttpClient, err := nsxt.NewPrincipalIdentityClient(builderConfig.Nsxt.CertPath, builderConfig.Nsxt.KeyPath, builderConfig.Nsxt.CaCertPath)
	if err != nil {
		return
	}

	nsxtClient := nsxt.NSXTClient{
		HttpClient:      nsxtHttpClient,
		BaseUrl:         builderConfig.Nsxt.BaseUrl,
		IpPoolName:      builderConfig.Nsxt.IpPoolName,
		EdgeClusterPath: builderConfig.Nsxt.EdgeClusterPath,
		MaxRetries:      10,
		Logger:          logger,
	}

	vsphereClient := vsphere.VSphere{
		HttpClient: httpClient,
		ServerUrl:  builderConfig.LaForgeServerUrl,
		BaseUrl:    builderConfig.Vsphere.BaseUrl,
		Username:   builderConfig.Vsphere.Username,
		Password:   builderConfig.Vsphere.Password,
		MaxRetries: 10,
		Logger:     logger,
	}

	vsphere.InitializeGovmomi(&vsphereClient, builderConfig.Vsphere.BaseUrl, builderConfig.Vsphere.Username, builderConfig.Vsphere.Password)

	ctx := context.Background()

	datastore, exists, err := vsphereClient.GetDatastoreSummaryByName(ctx, builderConfig.Vsphere.Datastore)
	if err != nil {
		return
	}
	if !exists {
		err = fmt.Errorf("error datastore \"%s\" doesn't exist", builderConfig.Vsphere.Datastore)
		logrus.Error(err)
		return
	}

	folder, err := vsphereClient.GetFolderSummaryByName(ctx, builderConfig.Vsphere.Folder)
	if err != nil {
		err = fmt.Errorf("error folder \"%s\" doesn't exist: %v", builderConfig.Vsphere.Folder, err)
		logrus.Error(err)
		return
	}

	resourcePool, err := vsphereClient.Finder.ResourcePool(ctx, builderConfig.Vsphere.ResourcePool)
	if err != nil {
		err = fmt.Errorf("error resource pool \"%s\" doesn't exist: %v", builderConfig.Vsphere.ResourcePool, err)
		logrus.Error(err)
		return
	}

	deployWorkerPool := semaphore.NewWeighted(int64(builderConfig.MaxBuildWorkers))
	teardownWorkerPool := semaphore.NewWeighted(int64(builderConfig.MaxTeardownWorkers))

	builder = vspherensxt.VSphereNSXTBuilder{
		Config:              builderConfig,
		HttpClient:          httpClient,
		NsxtClient:          nsxtClient,
		VSphereClient:       vsphereClient,
		VSphereDatastore:    datastore,
		VSphereResourcePool: resourcePool,
		VSphereFolder:       folder,
		Logger:              logger,
		DeployWorkerPool:    deployWorkerPool,
		TeardownWorkerPool:  teardownWorkerPool,
	}
	return
}

func NewOpenstackBuilder(configFilePath string, env *ent.Environment, logger *logging.Logger) (builder lfopenstack.OpenstackBuilder, err error) {
	var builderConfig lfopenstack.OpenstackBuilderConfig
	err = configs.LoadBuilderConfig(configFilePath, &builderConfig)
	if err != nil {
		return
	}

	deployWorkerPool := semaphore.NewWeighted(int64(builderConfig.MaxBuildWorkers))
	teardownWorkerPool := semaphore.NewWeighted(int64(builderConfig.MaxTeardownWorkers))

	builder = lfopenstack.OpenstackBuilder{
		Config: builderConfig,
		HttpClient: http.Client{
			Timeout: 5 * time.Minute,
		},
		Logger:             logger,
		DeployWorkerPool:   deployWorkerPool,
		TeardownWorkerPool: teardownWorkerPool,
	}
	return
}
