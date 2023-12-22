package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/agentstatus"
	"github.com/gen0cide/laforge/ent/agenttask"
	"github.com/gen0cide/laforge/ent/authuser"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/repository"
	"github.com/gen0cide/laforge/ent/servertask"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/gen0cide/laforge/graphql/auth"
	"github.com/gen0cide/laforge/graphql/graph/generated"
	"github.com/gen0cide/laforge/graphql/graph/model"
	"github.com/gen0cide/laforge/loader"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/planner"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func (r *adhocPlanResolver) ID(ctx context.Context, obj *ent.AdhocPlan) (string, error) {
	return obj.ID.String(), nil
}

func (r *agentTaskResolver) ID(ctx context.Context, obj *ent.AgentTask) (string, error) {
	return obj.ID.String(), nil
}

func (r *agentTaskResolver) Command(ctx context.Context, obj *ent.AgentTask) (model.AgentCommand, error) {
	return model.AgentCommand(obj.Command), nil
}

func (r *agentTaskResolver) State(ctx context.Context, obj *ent.AgentTask) (model.AgentTaskState, error) {
	return model.AgentTaskState(obj.State), nil
}

func (r *ansibleResolver) ID(ctx context.Context, obj *ent.Ansible) (string, error) {
	return obj.ID.String(), nil
}

func (r *ansibleResolver) Method(ctx context.Context, obj *ent.Ansible) (model.AnsibleMethod, error) {
	return model.AnsibleMethod(obj.Method), nil
}

func (r *ansibleResolver) Tags(ctx context.Context, obj *ent.Ansible) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *authUserResolver) ID(ctx context.Context, obj *ent.AuthUser) (string, error) {
	return obj.ID.String(), nil
}

func (r *authUserResolver) PublicKey(ctx context.Context, obj *ent.AuthUser) (string, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return "", err
	}
	// Read entire file content, giving us little control but
	// making it very simple. No need to close the file.
	content, err := ioutil.ReadFile(currentUser.PrivateKeyPath + ".pub")
	if err != nil {
		return "", err
	}

	// Convert []byte to string and print to screen
	text := string(content)
	return text, nil
}

func (r *authUserResolver) Role(ctx context.Context, obj *ent.AuthUser) (model.RoleLevel, error) {
	return model.RoleLevel(obj.Role), nil
}

func (r *authUserResolver) Provider(ctx context.Context, obj *ent.AuthUser) (model.ProviderType, error) {
	return model.ProviderType(obj.Provider), nil
}

func (r *buildResolver) ID(ctx context.Context, obj *ent.Build) (string, error) {
	return obj.ID.String(), nil
}

func (r *buildCommitResolver) ID(ctx context.Context, obj *ent.BuildCommit) (string, error) {
	return obj.ID.String(), nil
}

func (r *buildCommitResolver) Type(ctx context.Context, obj *ent.BuildCommit) (model.BuildCommitType, error) {
	return model.BuildCommitType(obj.Type), nil
}

func (r *buildCommitResolver) State(ctx context.Context, obj *ent.BuildCommit) (model.BuildCommitState, error) {
	return model.BuildCommitState(obj.State), nil
}

func (r *commandResolver) ID(ctx context.Context, obj *ent.Command) (string, error) {
	return obj.ID.String(), nil
}

func (r *commandResolver) Vars(ctx context.Context, obj *ent.Command) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *commandResolver) Tags(ctx context.Context, obj *ent.Command) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *competitionResolver) ID(ctx context.Context, obj *ent.Competition) (string, error) {
	return obj.ID.String(), nil
}

func (r *competitionResolver) Config(ctx context.Context, obj *ent.Competition) ([]*model.ConfigMap, error) {
	results := make([]*model.ConfigMap, 0)
	for configKey, configValue := range obj.Config {
		configTag := &model.ConfigMap{
			Key:   configKey,
			Value: configValue,
		}
		results = append(results, configTag)
	}
	return results, nil
}

func (r *competitionResolver) Tags(ctx context.Context, obj *ent.Competition) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *dNSResolver) ID(ctx context.Context, obj *ent.DNS) (string, error) {
	return obj.ID.String(), nil
}

func (r *dNSResolver) Config(ctx context.Context, obj *ent.DNS) ([]*model.ConfigMap, error) {
	results := make([]*model.ConfigMap, 0)
	for configKey, configValue := range obj.Config {
		configTag := &model.ConfigMap{
			Key:   configKey,
			Value: configValue,
		}
		results = append(results, configTag)
	}
	return results, nil
}

func (r *dNSRecordResolver) ID(ctx context.Context, obj *ent.DNSRecord) (string, error) {
	return obj.ID.String(), nil
}

func (r *dNSRecordResolver) Vars(ctx context.Context, obj *ent.DNSRecord) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *dNSRecordResolver) Tags(ctx context.Context, obj *ent.DNSRecord) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *diskResolver) ID(ctx context.Context, obj *ent.Disk) (string, error) {
	return obj.ID.String(), nil
}

func (r *environmentResolver) ID(ctx context.Context, obj *ent.Environment) (string, error) {
	return obj.ID.String(), nil
}

func (r *environmentResolver) Config(ctx context.Context, obj *ent.Environment) ([]*model.ConfigMap, error) {
	results := make([]*model.ConfigMap, 0)
	for configKey, configValue := range obj.Config {
		configTag := &model.ConfigMap{
			Key:   configKey,
			Value: configValue,
		}
		results = append(results, configTag)
	}
	return results, nil
}

func (r *environmentResolver) Tags(ctx context.Context, obj *ent.Environment) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *fileDeleteResolver) ID(ctx context.Context, obj *ent.FileDelete) (string, error) {
	return obj.ID.String(), nil
}

func (r *fileDeleteResolver) Tags(ctx context.Context, obj *ent.FileDelete) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *fileDownloadResolver) ID(ctx context.Context, obj *ent.FileDownload) (string, error) {
	return obj.ID.String(), nil
}

func (r *fileDownloadResolver) Tags(ctx context.Context, obj *ent.FileDownload) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *fileExtractResolver) ID(ctx context.Context, obj *ent.FileExtract) (string, error) {
	return obj.ID.String(), nil
}

func (r *fileExtractResolver) Tags(ctx context.Context, obj *ent.FileExtract) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *findingResolver) ID(ctx context.Context, obj *ent.Finding) (string, error) {
	return obj.ID.String(), nil
}

func (r *findingResolver) Severity(ctx context.Context, obj *ent.Finding) (model.FindingSeverity, error) {
	return model.FindingSeverity(obj.Severity), nil
}

func (r *findingResolver) Difficulty(ctx context.Context, obj *ent.Finding) (model.FindingDifficulty, error) {
	return model.FindingDifficulty(obj.Difficulty), nil
}

func (r *findingResolver) Tags(ctx context.Context, obj *ent.Finding) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *ginFileMiddlewareResolver) ID(ctx context.Context, obj *ent.GinFileMiddleware) (string, error) {
	return obj.ID.String(), nil
}

func (r *hostResolver) ID(ctx context.Context, obj *ent.Host) (string, error) {
	return obj.ID.String(), nil
}

func (r *hostResolver) Vars(ctx context.Context, obj *ent.Host) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *hostResolver) Tags(ctx context.Context, obj *ent.Host) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *hostDependencyResolver) ID(ctx context.Context, obj *ent.HostDependency) (string, error) {
	return obj.ID.String(), nil
}

func (r *identityResolver) ID(ctx context.Context, obj *ent.Identity) (string, error) {
	return obj.ID.String(), nil
}

func (r *identityResolver) Vars(ctx context.Context, obj *ent.Identity) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *identityResolver) Tags(ctx context.Context, obj *ent.Identity) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *includedNetworkResolver) ID(ctx context.Context, obj *ent.IncludedNetwork) (string, error) {
	return obj.ID.String(), nil
}

func (r *mutationResolver) LoadEnvironment(ctx context.Context, envFilePath string) ([]*ent.Environment, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting auth user from context: %v", err)
	}
	taskStatus, serverTask, err := utils.CreateServerTask(ctx, r.client, r.rdb, currentUser, servertask.TypeLOADENV)
	if err != nil {
		return nil, fmt.Errorf("error creating server task: %v", err)
	}
	log, err := logging.CreateLoggerForServerTask(r.laforgeConfig, serverTask)
	if err != nil {
		return nil, err
	}
	results, err := loader.LoadEnvironment(ctx, r.client, log, envFilePath)
	if err != nil {
		_, _, taskErr := utils.FailServerTask(ctx, r.client, r.rdb, taskStatus, serverTask, err)
		if taskErr != nil {
			return nil, fmt.Errorf("error failing server task: %v", err)
		}
		return nil, err
	}
	taskStatus, serverTask, err = utils.CompleteServerTask(ctx, r.client, r.rdb, taskStatus, serverTask)
	if err != nil {
		return nil, fmt.Errorf("error completing server task: %v", err)
	}
	serverTask, err = r.client.ServerTask.UpdateOne(serverTask).SetEnvironment(results[0]).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting environment on server task: %v", err)
	}
	r.rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())
	return results, nil
}

func (r *mutationResolver) CreateBuild(ctx context.Context, envUUID string, renderFiles bool) (*ent.Build, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return nil, err
	}

	uuid, err := uuid.Parse(envUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	entEnvironment, err := r.client.Environment.Query().Where(environment.IDEQ(uuid)).WithBuilds().Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying Environment: %v", err)
	}
	planner.ShouldRenderFiles = renderFiles
	if renderFiles {
		planner.RenderFilesTaskStatus, planner.RenderFilesTask, err = utils.CreateServerTask(ctx, r.client, r.rdb, currentUser, servertask.TypeRENDERFILES)
	} else {
		planner.RenderFilesTask = nil
		planner.RenderFilesTaskStatus = nil
	}

	return planner.CreateBuild(ctx, r.client, r.rdb, r.laforgeConfig, currentUser, entEnvironment)
}

func (r *mutationResolver) DeleteUser(ctx context.Context, userUUID string) (bool, error) {
	uuid, err := uuid.Parse(userUUID)

	if err != nil {
		return false, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	err = r.client.AuthUser.DeleteOneID(uuid).Exec(ctx)

	if err != nil {
		return false, err
	}
	return true, err
}

func (r *mutationResolver) ExecutePlan(ctx context.Context, buildUUID string) (*ent.Build, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return nil, err
	}

	uuid, err := uuid.Parse(buildUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	b, err := r.client.Build.Query().Where(build.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying Build: %v", err)
	}

	entEnvironment, err := b.QueryEnvironment().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query environment from build: %v", err)
	}

	taskStatus, serverTask, err := utils.CreateServerTask(ctx, r.client, r.rdb, currentUser, servertask.TypeEXECUTEBUILD)
	if err != nil {
		return nil, fmt.Errorf("error creating server task: %v", err)
	}
	serverTask, err = r.client.ServerTask.UpdateOne(serverTask).SetBuild(b).SetEnvironment(entEnvironment).Save(ctx)
	if err != nil {
		taskStatus, serverTask, err = utils.FailServerTask(ctx, r.client, r.rdb, taskStatus, serverTask)
		if err != nil {
			return nil, fmt.Errorf("error failing execute build server task: %v", err)
		}
		return nil, fmt.Errorf("error assigning environment and build to execute build server task: %v", err)
	}
	r.rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())

	logger, err := logging.CreateLoggerForServerTask(r.laforgeConfig, serverTask)
	if err != nil {
		return nil, err
	}

	go planner.StartBuild(r.client, r.laforgeConfig, logger, currentUser, serverTask, taskStatus, b)

	return b, nil
}

func (r *mutationResolver) DeleteBuild(ctx context.Context, buildUUID string) (string, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return "", err
	}

	uuid, err := uuid.Parse(buildUUID)

	if err != nil {
		return "", fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	b, err := r.client.Build.Query().Where(build.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return "", fmt.Errorf("failed querying Build: %v", err)
	}

	entEnvironment, err := b.QueryEnvironment().Only(ctx)
	if err != nil {
		logrus.Errorf("failed to query environment from build: %v", err)
		return "", err
	}

	taskStatus, serverTask, err := utils.CreateServerTask(ctx, r.client, r.rdb, currentUser, servertask.TypeDELETEBUILD)
	if err != nil {
		return "", fmt.Errorf("error creating server task: %v", err)
	}
	serverTask, err = r.client.ServerTask.UpdateOne(serverTask).SetBuild(b).SetEnvironment(entEnvironment).Save(ctx)
	if err != nil {
		taskStatus, serverTask, err = utils.FailServerTask(ctx, r.client, r.rdb, taskStatus, serverTask)
		if err != nil {
			return "", fmt.Errorf("error failing execute build server task: %v", err)
		}
		return "", fmt.Errorf("error assigning environment and build to execute build server task: %v", err)
	}
	r.rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())
	log, err := logging.CreateLoggerForServerTask(r.laforgeConfig, serverTask)
	if err != nil {
		return "", fmt.Errorf("error creating logger for build delete: %v", err)
	}

	spawnedDelete := make(chan bool, 1)
	go planner.DeleteBuild(r.client, r.rdb, r.laforgeConfig, log, currentUser, serverTask, taskStatus, b, spawnedDelete)

	deleteIsSuccess := <-spawnedDelete
	if deleteIsSuccess {
		entBuildCommit, err := b.QueryLatestBuildCommit().Only(ctx)
		if err != nil {
			return "", nil
		}
		return entBuildCommit.ID.String(), nil
	}
	taskStatus, serverTask, err = utils.FailServerTask(ctx, r.client, r.rdb, taskStatus, serverTask)
	if err != nil {
		return "", fmt.Errorf("error failing execute build server task: %v", err)
	}
	return "", fmt.Errorf("unknown error occurred")
}

func (r *mutationResolver) CreateTask(ctx context.Context, proHostUUID string, command model.AgentCommand, args string) (bool, error) {
	uuid, err := uuid.Parse(proHostUUID)

	if err != nil {
		return false, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	ph, err := r.client.ProvisionedHost.Query().Where(provisionedhost.IDEQ(uuid)).Only(ctx)
	if err != nil {
		return false, fmt.Errorf("failed querying Provisioned Host %v: %v", proHostUUID, err)
	}
	taskCount, err := ph.QueryAgentTasks().Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed querying Number of Tasks: %v", err)
	}
	_, err = r.client.AgentTask.Create().
		SetCommand(agenttask.Command(command.String())).
		SetArgs(args).
		SetNumber(taskCount).
		SetState(agenttask.StateAWAITING).
		SetProvisionedHost(ph).
		Save(ctx)
	if err != nil {
		return false, fmt.Errorf("failed Creating Agent Task: %v", err)
	}
	return true, nil
}

func (r *mutationResolver) DumpBuild(ctx context.Context, buildUUID string) (string, error) {
	uuid, err := uuid.Parse(buildUUID)

	if err != nil {
		return "", fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	entBuild, err := r.client.Build.Query().Where(build.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return "", fmt.Errorf("failed querying Build: %v", err)
	}

	return utils.GenerateBuildConf(ctx, r.client, entBuild)
}

func (r *mutationResolver) Rebuild(ctx context.Context, rootPlans []*string) (bool, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return false, err
	}

	uuids := make([]uuid.UUID, len(rootPlans))
	for _, rootPlanId := range rootPlans {
		uuid, err := uuid.Parse(*rootPlanId)
		if err != nil {
			return false, err
		}
		uuids = append(uuids, uuid)
	}

	entPlans, err := r.client.Plan.Query().Where(plan.IDIn(uuids...)).All(ctx)
	if err != nil {
		return false, err
	}
	b, err := entPlans[0].QueryBuild().First(ctx)
	if err != nil {
		return false, err
	}
	env, err := b.QueryEnvironment().First(ctx)
	if err != nil {
		return false, err
	}

	taskStatus, serverTask, err := utils.CreateServerTask(ctx, r.client, r.rdb, currentUser, servertask.TypeREBUILD)
	if err != nil {
		return false, fmt.Errorf("error creating server task: %v", err)
	}
	serverTask, err = r.client.ServerTask.UpdateOne(serverTask).SetBuild(b).SetEnvironment(env).Save(ctx)
	if err != nil {
		taskStatus, serverTask, err = utils.FailServerTask(ctx, r.client, r.rdb, taskStatus, serverTask)
		if err != nil {
			return false, fmt.Errorf("error failing execute rebuild server task: %v", err)
		}
		return false, fmt.Errorf("error assigning environment and build to execute rebuild server task: %v", err)
	}
	r.rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())

	logger, err := logging.CreateLoggerForServerTask(r.laforgeConfig, serverTask)
	if err != nil {
		return false, err
	}

	spawnedRebuild := make(chan bool, 1)
	go planner.Rebuild(r.client, r.rdb, r.laforgeConfig, logger, currentUser, serverTask, taskStatus, entPlans, spawnedRebuild)

	rebuildStartedSuccess := <-spawnedRebuild
	if rebuildStartedSuccess {
		return true, nil
	}
	return false, nil
}

func (r *mutationResolver) ApproveCommit(ctx context.Context, commitUUID string) (bool, error) {
	uuid, err := uuid.Parse(commitUUID)
	if err != nil {
		return false, err
	}
	err = r.client.BuildCommit.UpdateOneID(uuid).SetState(buildcommit.StateAPPROVED).Exec(ctx)
	if err != nil {
		return false, err
	}
	r.rdb.Publish(ctx, "updatedBuildCommit", commitUUID)
	return true, nil
}

func (r *mutationResolver) CancelCommit(ctx context.Context, commitUUID string) (bool, error) {
	uuid, err := uuid.Parse(commitUUID)
	if err != nil {
		return false, fmt.Errorf("failed casting commit uuid to uuid: %v", err)
	}
	entBuildCommit, err := r.client.BuildCommit.UpdateOneID(uuid).SetState(buildcommit.StateCANCELLED).Save(ctx)
	if err != nil {
		return false, fmt.Errorf("failed setting build commit state to cancelled: %v", err)
	}
	if entBuildCommit.Type == buildcommit.TypeROOT {
		entBuild, err := entBuildCommit.QueryBuild().Only(ctx)
		if err != nil {
			return false, fmt.Errorf("failed querying build from build commit: %v", err)
		}
		err = r.client.Status.Update().Where(status.HasBuildWith(build.IDEQ(entBuild.ID))).SetState(status.StateCANCELLED).Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("failed setting build status to cancelled: %v", err)
		}
	}
	r.rdb.Publish(ctx, "updatedBuildCommit", commitUUID)
	entServerTasks, err := entBuildCommit.QueryServerTasks().WithStatus().All(ctx)
	if err != nil {
		return false, fmt.Errorf("failed querying server tasks from build commit: %v", err)
	}
	for _, serverTask := range entServerTasks {
		if serverTask.Edges.Status.State == status.StateINPROGRESS {
			err := serverTask.Edges.Status.Update().SetEndedAt(time.Now()).SetState(status.StateCANCELLED).Exec(ctx)
			if err != nil {
				return false, fmt.Errorf("failed to cancel server task(s) associated with build commit: %v", err)
			}
			r.rdb.Publish(ctx, "updatedServerTask", serverTask.ID)
		}
	}
	return true, nil
}

func (r *mutationResolver) CreateAgentTasks(ctx context.Context, hostHclid string, command model.AgentCommand, buildUUID string, args []string, teams []int) ([]*ent.AgentTask, error) {
	uuid, err := uuid.Parse(buildUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	entBuild, err := r.client.Build.Get(ctx, uuid)

	if err != nil {
		return nil, fmt.Errorf("failed querying build: %v", err)
	}

	agentTasksReturn := []*ent.AgentTask{}

	for _, team_number := range teams {
		entTeam, err := entBuild.QueryTeams().Where(team.TeamNumberEQ(team_number)).Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed querying team: %v", err)
		}
		entProvisionedHost, err := entTeam.QueryProvisionedNetworks().QueryProvisionedHosts().Where(provisionedhost.HasHostWith(host.HclIDEQ(hostHclid))).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed querying provisoned hosts: %v", err)
		}
		for _, pHost := range entProvisionedHost {
			taskCount, err := pHost.QueryAgentTasks().Count(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed querying Number of Tasks: %v", err)
			}
			createdAgentTask, err := r.client.AgentTask.Create().
				SetCommand(agenttask.Command(command.String())).
				SetArgs(strings.Join(args, "💔")).
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetProvisionedHost(pHost).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create agent task: %v", err)
			}
			agentTasksReturn = append(agentTasksReturn, createdAgentTask)
		}

	}

	return agentTasksReturn, nil
}

func (r *mutationResolver) CreateBatchAgentTasks(ctx context.Context, proHostUUIDs []string, command model.AgentCommand, args []string) ([]*ent.AgentTask, error) {
	agentTasksReturn := []*ent.AgentTask{}

	for _, proHostUUID := range proHostUUIDs {

		uuid, err := uuid.Parse(proHostUUID)

		if err != nil {
			return agentTasksReturn, fmt.Errorf("failed casting UUID to UUID: %v", err)
		}

		entProvisionedHost, err := r.client.ProvisionedHost.Get(ctx, uuid)
		if err != nil {
			return agentTasksReturn, fmt.Errorf("failed querying provisoned host: %v", err)
		}
		taskCount, err := entProvisionedHost.QueryAgentTasks().Count(ctx)
		if err != nil {
			return agentTasksReturn, fmt.Errorf("failed querying Number of Tasks: %v", err)
		}
		createdAgentTask, err := r.client.AgentTask.Create().
			SetCommand(agenttask.Command(command.String())).
			SetArgs(strings.Join(args, "💔")).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			Save(ctx)
		if err != nil {
			return agentTasksReturn, fmt.Errorf("failed to create agent task: %v", err)
		}
		agentTasksReturn = append(agentTasksReturn, createdAgentTask)
	}

	return agentTasksReturn, nil
}

func (r *mutationResolver) CreateEnviromentFromRepo(ctx context.Context, repoURL string, branchName string, envFilePath string) ([]*ent.Environment, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return nil, err
	}

	foundRepo, _ := r.client.Repository.Query().Where(
		repository.And(
			repository.BranchName(branchName),
			repository.EnviromentFilepath(envFilePath),
			repository.RepoURL(repoURL),
		),
	).WithRepoCommits().First(ctx)

	if foundRepo != nil {
		return r.UpdateEnviromentViaPull(ctx, foundRepo.ID.String())
	}

	entRepo, err := r.client.Repository.Create().
		SetRepoURL(repoURL).
		SetBranchName(branchName).
		SetEnviromentFilepath(envFilePath).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	repoFolderPath := fmt.Sprintf(utils.RepoPath, entRepo.ID.String(), branchName)

	commit_info, err := utils.CloneGit(repoURL, repoFolderPath, currentUser.PrivateKeyPath, branchName)
	if err != nil {
		r.client.Repository.DeleteOne(entRepo).Exec(ctx)
		return nil, err
	}

	p_hashes := make([]string, len(commit_info.ParentHashes))
	for i := 0; i < len(commit_info.ParentHashes); i++ {
		p_hashes = append(p_hashes, commit_info.ParentHashes[i].String())
	}

	entRepoCommit, err := r.client.RepoCommit.Create().
		SetRevision(0).
		SetHash(commit_info.Hash.String()).
		SetAuthor(commit_info.Author).
		SetCommitter(commit_info.Committer).
		SetPgpSignature(commit_info.PGPSignature).
		SetMessage(commit_info.Message).
		SetTreeHash(commit_info.TreeHash.String()).
		SetParentHashes(p_hashes).
		Save(ctx)

	if err != nil {
		r.client.Repository.DeleteOne(entRepo).Exec(ctx)
		return nil, fmt.Errorf("couldn't create entRepoCommit: %v", err)
	}

	err = entRepo.Update().AddRepoCommits(entRepoCommit).Exec(ctx)
	if err != nil {
		r.client.Repository.DeleteOne(entRepo).Exec(ctx)
		r.client.RepoCommit.DeleteOne(entRepoCommit).Exec(ctx)
		return nil, fmt.Errorf("couldn't add RepoCommit to Repository: %v", err)
	}

	envPath := path.Join(repoFolderPath, envFilePath)

	loadedEnviroments, err := r.LoadEnvironment(ctx, envPath)
	if err != nil {
		r.client.Repository.DeleteOne(entRepo).Exec(ctx)
		r.client.RepoCommit.DeleteOne(entRepoCommit).Exec(ctx)
		return nil, err
	}

	_, err = entRepo.Update().
		SetFolderPath(repoFolderPath).
		AddEnvironments(loadedEnviroments...).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return loadedEnviroments, nil
}

func (r *mutationResolver) UpdateEnviromentViaPull(ctx context.Context, envUUID string) ([]*ent.Environment, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return nil, err
	}
	uuid, err := uuid.Parse(envUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	entEnvironment, err := r.client.Environment.Get(ctx, uuid)
	if err != nil {
		return nil, err
	}

	entRepo, err := entEnvironment.QueryRepositories().WithRepoCommits().Only(ctx)
	if err != nil {
		return nil, err
	}

	commit_info, err := utils.PullGit(entRepo.FolderPath, currentUser.PrivateKeyPath, entRepo.BranchName)
	if err != nil {
		return nil, err
	}

	p_hashes := make([]string, len(commit_info.ParentHashes))
	for i := 0; i < len(commit_info.ParentHashes); i++ {
		p_hashes = append(p_hashes, commit_info.ParentHashes[i].String())
	}

	entRepoCommit, err := r.client.RepoCommit.Create().
		SetRevision(len(entRepo.Edges.RepoCommits)).
		SetHash(commit_info.Hash.String()).
		SetAuthor(commit_info.Author).
		SetCommitter(commit_info.Committer).
		SetPgpSignature(commit_info.PGPSignature).
		SetMessage(commit_info.Message).
		SetTreeHash(commit_info.TreeHash.String()).
		SetParentHashes(p_hashes).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("couldn't create entRepoCommit: %v", err)
	}

	err = entRepo.Update().AddRepoCommits(entRepoCommit).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't add RepoCommit to Repository: %v", err)
	}

	envPath := path.Join(entRepo.FolderPath, entRepo.EnviromentFilepath)

	return r.LoadEnvironment(ctx, envPath)
}

func (r *mutationResolver) CancelBuild(ctx context.Context, buildUUID string) (bool, error) {
	uuid, err := uuid.Parse(buildUUID)

	if err != nil {
		return false, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}
	return planner.CancelBuild(uuid), nil
}

func (r *mutationResolver) ModifySelfPassword(ctx context.Context, currentPassword string, newPassword string) (bool, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return false, err
	}
	if err = bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(currentPassword)); err == nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 8)
		if err != nil {
			return false, err
		}
		newPassword = string(hashedPassword[:])
		err = currentUser.Update().SetPassword(newPassword).Exec(ctx)
		if err != nil {
			return false, err
		}
		return true, nil
	} else {
		return false, fmt.Errorf("incorrect current password")
	}
}

func (r *mutationResolver) ModifySelfUserInfo(ctx context.Context, firstName *string, lastName *string, email *string, phone *string, company *string, occupation *string) (*ent.AuthUser, error) {
	currentUser, err := auth.ForContext(ctx)
	if err != nil {
		return nil, err
	}
	newFirstName := ""
	if firstName == nil {
		newFirstName = currentUser.FirstName
	} else {
		newFirstName = *firstName
	}
	newLastName := ""
	if lastName == nil {
		newLastName = currentUser.LastName
	} else {
		newLastName = *lastName
	}
	newEmail := ""
	if email == nil {
		newEmail = currentUser.Email
	} else {
		newEmail = *email
	}
	newPhone := ""
	if phone == nil {
		newPhone = currentUser.Phone
	} else {
		newPhone = *phone
	}
	newCompany := ""
	if company == nil {
		newCompany = currentUser.Company
	} else {
		newCompany = *company
	}
	newOccupation := ""
	if occupation == nil {
		newOccupation = currentUser.Occupation
	} else {
		newOccupation = *occupation
	}

	currentUser, err = currentUser.Update().
		SetFirstName(newFirstName).
		SetLastName(newLastName).
		SetEmail(newEmail).
		SetPhone(newPhone).
		SetCompany(newCompany).
		SetOccupation(newOccupation).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return currentUser, nil
}

func (r *mutationResolver) CreateUser(ctx context.Context, username string, password string, role model.RoleLevel, provider model.ProviderType) (*ent.AuthUser, error) {
	sshFolderPath := fmt.Sprintf(utils.UserKeyPath, strings.ToLower(authuser.ProviderLOCAL.String()), username)

	err := os.MkdirAll(sshFolderPath, os.ModeAppend|os.ModePerm)
	if err != nil {
		return nil, err
	}
	sshPrivateFile := fmt.Sprintf("%s/id_ed25519", sshFolderPath)
	err = utils.MakeED25519KeyPair(sshPrivateFile)
	if err != nil {
		return nil, err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return nil, err
	}
	password = string(hashedPassword[:])
	entAuthUser, err := r.client.AuthUser.Create().
		SetUsername(username).
		SetPassword(password).
		SetRole(authuser.Role(role)).
		SetProvider(authuser.Provider(provider)).
		SetPrivateKeyPath(sshPrivateFile).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return entAuthUser, nil
}

func (r *mutationResolver) ModifyAdminUserInfo(ctx context.Context, userID string, username *string, firstName *string, lastName *string, email *string, phone *string, company *string, occupation *string, role *model.RoleLevel, provider *model.ProviderType) (*ent.AuthUser, error) {
	uuid, err := uuid.Parse(userID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	entAuthUser, err := r.client.AuthUser.Get(ctx, uuid)
	if err != nil {
		return nil, err
	}
	newUsername := ""
	if username == nil {
		newUsername = entAuthUser.Username
	} else {
		newUsername = *username
	}
	newFirstName := ""
	if firstName == nil {
		newFirstName = entAuthUser.FirstName
	} else {
		newFirstName = *firstName
	}
	newLastName := ""
	if lastName == nil {
		newLastName = entAuthUser.LastName
	} else {
		newLastName = *lastName
	}
	newEmail := ""
	if email == nil {
		newEmail = entAuthUser.Email
	} else {
		newEmail = *email
	}
	newPhone := ""
	if phone == nil {
		newPhone = entAuthUser.Phone
	} else {
		newPhone = *phone
	}
	newCompany := ""
	if company == nil {
		newCompany = entAuthUser.Company
	} else {
		newCompany = *company
	}
	newOccupation := ""
	if occupation == nil {
		newOccupation = entAuthUser.Occupation
	} else {
		newOccupation = *occupation
	}
	newRole := authuser.RoleUSER
	if role == nil {
		newRole = entAuthUser.Role
	} else {
		newRole = authuser.Role(*role)
	}
	newProvider := authuser.ProviderLOCAL
	if provider == nil {
		newProvider = entAuthUser.Provider
	} else {
		newProvider = authuser.Provider(*provider)
	}

	entAuthUser, err = entAuthUser.Update().
		SetUsername(newUsername).
		SetFirstName(newFirstName).
		SetLastName(newLastName).
		SetEmail(newEmail).
		SetPhone(newPhone).
		SetCompany(newCompany).
		SetOccupation(newOccupation).
		SetRole(newRole).
		SetProvider(newProvider).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return entAuthUser, nil
}

func (r *mutationResolver) ModifyAdminPassword(ctx context.Context, userID string, newPassword string) (bool, error) {
	uuid, err := uuid.Parse(userID)

	if err != nil {
		return false, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	entAuthUser, err := r.client.AuthUser.Get(ctx, uuid)
	if err != nil {
		return false, err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 8)
	if err != nil {
		return false, err
	}
	newPassword = string(hashedPassword[:])
	err = entAuthUser.Update().SetPassword(newPassword).Exec(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *mutationResolver) NukeBackend(ctx context.Context) ([]*model.IntMap, error) {
	results := make([]*model.IntMap, 0)
	returnedResults, err := utils.ClearDB(ctx, r.client, r.laforgeConfig)
	for varKey, varValue := range returnedResults {
		tempVar := &model.IntMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	if err != nil {
		return results, err
	}
	return results, nil
}

func (r *networkResolver) ID(ctx context.Context, obj *ent.Network) (string, error) {
	return obj.ID.String(), nil
}

func (r *networkResolver) Vars(ctx context.Context, obj *ent.Network) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *networkResolver) Tags(ctx context.Context, obj *ent.Network) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *planResolver) ID(ctx context.Context, obj *ent.Plan) (string, error) {
	return obj.ID.String(), nil
}

func (r *planResolver) Type(ctx context.Context, obj *ent.Plan) (model.PlanType, error) {
	return model.PlanType(obj.Type), nil
}

func (r *planDiffResolver) ID(ctx context.Context, obj *ent.PlanDiff) (string, error) {
	return obj.ID.String(), nil
}

func (r *planDiffResolver) NewState(ctx context.Context, obj *ent.PlanDiff) (model.ProvisionStatus, error) {
	return model.ProvisionStatus(obj.NewState), nil
}

func (r *provisionedHostResolver) ID(ctx context.Context, obj *ent.ProvisionedHost) (string, error) {
	return obj.ID.String(), nil
}

func (r *provisionedHostResolver) AddonType(ctx context.Context, obj *ent.ProvisionedHost) (*model.ProvisionedHostAddonType, error) {
	var addonType model.ProvisionedHostAddonType
	switch *obj.AddonType {
	case provisionedhost.AddonTypeDNS:
		addonType = model.ProvisionedHostAddonTypeDNS
	}
	return &addonType, nil
}

func (r *provisionedHostResolver) Vars(ctx context.Context, obj *ent.ProvisionedHost) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *provisionedNetworkResolver) ID(ctx context.Context, obj *ent.ProvisionedNetwork) (string, error) {
	return obj.ID.String(), nil
}

func (r *provisionedNetworkResolver) Vars(ctx context.Context, obj *ent.ProvisionedNetwork) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *provisioningScheduledStepResolver) ID(ctx context.Context, obj *ent.ProvisioningScheduledStep) (string, error) {
	return obj.ID.String(), nil
}

func (r *provisioningScheduledStepResolver) Type(ctx context.Context, obj *ent.ProvisioningScheduledStep) (model.ProvisioningScheduledStepType, error) {
	return model.ProvisioningScheduledStepType(obj.Type), nil
}

func (r *provisioningStepResolver) ID(ctx context.Context, obj *ent.ProvisioningStep) (string, error) {
	return obj.ID.String(), nil
}

func (r *provisioningStepResolver) Type(ctx context.Context, obj *ent.ProvisioningStep) (model.ProvisioningStepType, error) {
	return model.ProvisioningStepType(obj.Type), nil
}

func (r *queryResolver) Environments(ctx context.Context) ([]*ent.Environment, error) {
	e, err := r.client.Environment.Query().Order(ent.Asc(environment.FieldID)).All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying Environment: %v", err)
	}

	return e, nil
}

func (r *queryResolver) Environment(ctx context.Context, envUUID string) (*ent.Environment, error) {
	uuid, err := uuid.Parse(envUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	e, err := r.client.Environment.Query().Where(environment.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying Environment: %v", err)
	}

	return e, nil
}

func (r *queryResolver) ProvisionedHost(ctx context.Context, proHostUUID string) (*ent.ProvisionedHost, error) {
	uuid, err := uuid.Parse(proHostUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	ph, err := r.client.ProvisionedHost.Query().Where(provisionedhost.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying ProvisionedHost: %v", err)
	}

	return ph, nil
}

func (r *queryResolver) ProvisionedNetwork(ctx context.Context, proNetUUID string) (*ent.ProvisionedNetwork, error) {
	uuid, err := uuid.Parse(proNetUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	pn, err := r.client.ProvisionedNetwork.Query().Where(provisionednetwork.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying ProvisionedNetwork: %v", err)
	}

	return pn, nil
}

func (r *queryResolver) ProvisionedStep(ctx context.Context, proStepUUID string) (*ent.ProvisioningStep, error) {
	uuid, err := uuid.Parse(proStepUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	ps, err := r.client.ProvisioningStep.Query().Where(provisioningstep.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying ProvisionedStep: %v", err)
	}

	return ps, nil
}

func (r *queryResolver) Plan(ctx context.Context, planUUID string) (*ent.Plan, error) {
	uuid, err := uuid.Parse(planUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	plan, err := r.client.Plan.Query().Where(plan.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying ProvisionedNetwork: %v", err)
	}

	return plan, nil
}

func (r *queryResolver) GetBuilds(ctx context.Context) ([]*ent.Build, error) {
	builds, err := r.client.Environment.Query().Order(ent.Asc(environment.FieldID)).QueryBuilds().All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying Builds: %v", err)
	}

	return builds, nil
}

func (r *queryResolver) Build(ctx context.Context, buildUUID string) (*ent.Build, error) {
	uuid, err := uuid.Parse(buildUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	build, err := r.client.Build.Query().Where(build.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying ProvisionedNetwork: %v", err)
	}

	return build, nil
}

func (r *queryResolver) GetBuildCommits(ctx context.Context, envUUID string) ([]*ent.BuildCommit, error) {
	uuid, err := uuid.Parse(envUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting envUUID to UUID: %v", err)
	}

	buildCommits, err := r.client.Environment.Query().Where(environment.IDEQ(uuid)).QueryBuilds().QueryBuildCommits().All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying BuildCommits from Environment: %v", err)
	}

	return buildCommits, nil
}

func (r *queryResolver) GetBuildCommit(ctx context.Context, buildCommitUUID string) (*ent.BuildCommit, error) {
	uuid, err := uuid.Parse(buildCommitUUID)
	if err != nil {
		return nil, fmt.Errorf("failed casing buildCommitUUID to UUID: %v", err)
	}

	buildCommit, err := r.client.BuildCommit.Query().Where(buildcommit.IDEQ(uuid)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying BuildCommit with ID %s: %v", buildCommitUUID, err)
	}

	return buildCommit, nil
}

func (r *queryResolver) Status(ctx context.Context, statusUUID string) (*ent.Status, error) {
	uuid, err := uuid.Parse(statusUUID)

	if err != nil {
		return nil, fmt.Errorf("failed casting statusUUID to UUID: %v", err)
	}

	status, err := r.client.Status.Query().Where(status.IDEQ(uuid)).Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying Status: %v", err)
	}

	return status, nil
}

func (r *queryResolver) AgentStatus(ctx context.Context, clientID string) (*ent.AgentStatus, error) {
	uuid, err := uuid.Parse(clientID)

	if err != nil {
		return nil, fmt.Errorf("failed casting clientID to UUID: %v", err)
	}

	status, err := r.client.AgentStatus.Query().
		Where(agentstatus.HasProvisionedHostWith(provisionedhost.IDEQ(uuid))).
		Order(ent.Desc(agentstatus.FieldTimestamp)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed querying Status: %v", err)
	}

	return status, nil
}

func (r *queryResolver) GetServerTasks(ctx context.Context) ([]*ent.ServerTask, error) {
	serverTasks, err := r.client.ServerTask.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error querying server tasks: %v", err)
	}
	return serverTasks, nil
}

func (r *queryResolver) CurrentUser(ctx context.Context) (*ent.AuthUser, error) {
	return auth.ForContext(ctx)
}

func (r *queryResolver) GetUserList(ctx context.Context) ([]*ent.AuthUser, error) {
	return r.client.AuthUser.Query().All(ctx)
}

func (r *queryResolver) GetCurrentUserTasks(ctx context.Context) ([]*ent.ServerTask, error) {
	return r.client.AuthUser.Query().QueryServerTasks().All(ctx)
}

func (r *queryResolver) GetAgentTasks(ctx context.Context, proStepUUID *string, proSchedStepUUID *string) ([]*ent.AgentTask, error) {
	var stepUuid uuid.UUID
	var err error
	if proStepUUID != nil {
		stepUuid, err = uuid.Parse(*proStepUUID)
		if err != nil {
			return nil, err
		}
	} else if proSchedStepUUID != nil {
		stepUuid, err = uuid.Parse(*proSchedStepUUID)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("either proStepUUID or proSchedStepUUID must be supplied")
	}

	var agentTasks []*ent.AgentTask
	if proStepUUID != nil {
		agentTasks, err = r.client.ProvisioningStep.Query().Where(provisioningstep.IDEQ(stepUuid)).QueryAgentTasks().All(ctx)
		if err != nil {
			return nil, err
		}
	} else if proSchedStepUUID != nil {
		agentTasks, err = r.client.ProvisioningScheduledStep.Query().Where(provisioningscheduledstep.IDEQ(stepUuid)).QueryAgentTasks().All(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("either proStepUUID or proSchedStepUUID must be supplied")
	}

	return agentTasks, err
}

func (r *queryResolver) ListAgentStatuses(ctx context.Context, buildUUID string) ([]*ent.AgentStatus, error) {
	uuid, err := uuid.Parse(buildUUID)
	if err != nil {
		return nil, fmt.Errorf("failed casting buildUUID to UUID")
	}

	agentStatuses, err := r.client.Build.Query().Where(build.IDEQ(uuid)).QueryAgentStatuses().All(ctx)
	if err != nil {
		return nil, err
	}

	return agentStatuses, nil
}

func (r *queryResolver) ListBuildStatuses(ctx context.Context, buildUUID string) ([]*ent.Status, error) {
	uuid, err := uuid.Parse(buildUUID)
	if err != nil {
		return nil, fmt.Errorf("failed casting buildUUID to UUID")
	}

	statuses := make([]*ent.Status, 0)

	buildStatus, err := r.client.Build.Query().Where(build.IDEQ(uuid)).QueryStatus().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying build status from build: %v", err)
	}
	statuses = append(statuses, buildStatus)

	planStatuses, err := r.client.Status.Query().Where(status.HasPlanWith(plan.HasBuildWith(build.IDEQ(uuid)))).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying plan statuses from build: %v", err)
	}
	statuses = append(statuses, planStatuses...)

	teamStatuses, err := r.client.Team.Query().Where(team.HasBuildWith(build.IDEQ(uuid))).QueryStatus().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying team statuses from build: %v", err)
	}
	statuses = append(statuses, teamStatuses...)

	provisionedNetworkStatuses, err := r.client.ProvisionedNetwork.Query().Where(provisionednetwork.HasBuildWith(build.IDEQ(uuid))).QueryStatus().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying provisioned network statuses from build: %v", err)
	}
	statuses = append(statuses, provisionedNetworkStatuses...)

	provisionedHostStatuses, err := r.client.ProvisionedHost.Query().Where(provisionedhost.HasBuildWith(build.IDEQ(uuid))).QueryStatus().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying provisioned host statuses from build: %v", err)
	}
	statuses = append(statuses, provisionedHostStatuses...)

	provisioningStepStatuses, err := r.client.ProvisioningStep.Query().Where(provisioningstep.HasProvisionedHostWith(provisionedhost.HasBuildWith(build.IDEQ(uuid)))).QueryStatus().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying provisioning step statuses from build: %v", err)
	}
	statuses = append(statuses, provisioningStepStatuses...)

	return statuses, nil
}

func (r *queryResolver) GetAllAgentStatus(ctx context.Context, buildUUID string, count int, offset int) (*model.AgentStatusBatch, error) {
	uuid, err := uuid.Parse(buildUUID)
	if err != nil {
		return nil, err
	}

	totalAgentStatuses, err := r.client.AgentStatus.Query().Where(agentstatus.HasBuildWith(build.IDEQ(uuid))).Count(ctx)
	if err != nil {
		return nil, err
	}

	agentStatuses, err := r.client.AgentStatus.Query().Where(agentstatus.HasBuildWith(build.IDEQ(uuid))).Order(ent.Asc(agentstatus.FieldTimestamp)).Limit(count).Offset(offset).All(ctx)
	if err != nil {
		return nil, err
	}

	return &model.AgentStatusBatch{
		AgentStatuses: agentStatuses,
		PageInfo: &model.LaForgePageInfo{
			Total:      totalAgentStatuses,
			NextOffset: offset + count,
		},
	}, nil
}

func (r *queryResolver) GetAllPlanStatus(ctx context.Context, buildUUID string, count int, offset int) (*model.StatusBatch, error) {
	uuid, err := uuid.Parse(buildUUID)
	if err != nil {
		return nil, err
	}

	totalStatuses, err := r.client.Status.Query().Where(status.HasPlanWith(plan.HasBuildWith(build.IDEQ(uuid)))).Count(ctx)
	if err != nil {
		return nil, err
	}

	statuses, err := r.client.Status.Query().Where(status.HasPlanWith(plan.HasBuildWith(build.IDEQ(uuid)))).Order(ent.Asc(status.FieldStartedAt)).Limit(count).Offset(offset).All(ctx)
	if err != nil {
		return nil, err
	}

	return &model.StatusBatch{
		Statuses: statuses,
		PageInfo: &model.LaForgePageInfo{
			Total:      totalStatuses,
			NextOffset: offset + count,
		},
	}, nil
}

func (r *queryResolver) GetPlanStatusCounts(ctx context.Context, buildUUID string) (*model.PlanCounts, error) {
	uuid, err := uuid.Parse(buildUUID)
	if err != nil {
		return nil, err
	}

	entBuild, err := r.client.Build.Query().Where(build.IDEQ(uuid)).Only(ctx)
	if err != nil {
		return nil, err
	}
	planningCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StatePLANNING))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all planning plans from build: %v", err)
	}
	awaitingCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateAWAITING))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all awaiting plans from build: %v", err)
	}
	parentAwaitingCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StatePARENTAWAITING))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all parent_awaiting plans from build: %v", err)
	}
	inProgressCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateINPROGRESS))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all in_progress plans from build: %v", err)
	}
	failedCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateFAILED))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all failed plans from build: %v", err)
	}
	completeCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateCOMPLETE))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all complete plans from build: %v", err)
	}
	taintedCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateTAINTED))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all tainted plans from build: %v", err)
	}
	toDeleteCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateTODELETE))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all to_delete plans from build: %v", err)
	}
	deleteInProgressCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateDELETEINPROGRESS))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all delete_in_progress plans from build: %v", err)
	}
	deletedCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateDELETED))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all deleted plans from build: %v", err)
	}
	toRebuildCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateTOREBUILD))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all to_rebuild plans from build: %v", err)
	}
	cancelledCount, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StateCANCELLED))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all cancelled plans from build: %v", err)
	}

	return &model.PlanCounts{
		Planning:         planningCount,
		Awaiting:         awaitingCount,
		ParentAwaiting:   parentAwaitingCount,
		InProgress:       inProgressCount,
		Failed:           failedCount,
		Complete:         completeCount,
		Tainted:          taintedCount,
		Undefined:        0,
		ToDelete:         toDeleteCount,
		DeleteInProgress: deleteInProgressCount,
		Deleted:          deletedCount,
		ToRebuild:        toRebuildCount,
		Cancelled:        cancelledCount,
	}, nil
}

func (r *queryResolver) ViewServerTaskLogs(ctx context.Context, taskID string) (string, error) {
	uuid, err := uuid.Parse(taskID)

	if err != nil {
		return "", fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	entServerTask, err := r.client.ServerTask.Get(ctx, uuid)
	if err != nil {
		return "", err
	}

	fileBytes, err := ioutil.ReadFile(entServerTask.LogFilePath)
	if err != nil {
		return "", err
	}

	fileString := string(fileBytes)
	return fileString, nil
}

func (r *queryResolver) ViewAgentTask(ctx context.Context, taskID string) (*ent.AgentTask, error) {
	uuid, err := uuid.Parse(taskID)

	if err != nil {
		return nil, fmt.Errorf("failed casting UUID to UUID: %v", err)
	}

	return r.client.AgentTask.Get(ctx, uuid)
}

func (r *queryResolver) ServerTasks(ctx context.Context, taskUUIDs []*string) ([]*ent.ServerTask, error) {
	uuids := make([]uuid.UUID, 0)
	for _, taskUUID := range taskUUIDs {
		uuid, err := uuid.Parse(*taskUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to cast one or more task UUID to UUID: %v", err)
		}
		uuids = append(uuids, uuid)
	}

	serverTasks, err := r.client.ServerTask.Query().Where(servertask.IDIn(uuids...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query server tasks: %v", err)
	}

	return serverTasks, nil
}

func (r *queryResolver) ServerConfig(ctx context.Context) (*model.ServerConfig, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *repoCommitResolver) ID(ctx context.Context, obj *ent.RepoCommit) (string, error) {
	return obj.ID.String(), nil
}

func (r *repoCommitResolver) Author(ctx context.Context, obj *ent.RepoCommit) (string, error) {
	return obj.Author.String(), nil
}

func (r *repoCommitResolver) Committer(ctx context.Context, obj *ent.RepoCommit) (string, error) {
	return obj.Committer.String(), nil
}

func (r *repositoryResolver) ID(ctx context.Context, obj *ent.Repository) (string, error) {
	return obj.ID.String(), nil
}

func (r *repositoryResolver) EnvironmentFilepath(ctx context.Context, obj *ent.Repository) (string, error) {
	return obj.EnviromentFilepath, nil
}

func (r *scheduledStepResolver) ID(ctx context.Context, obj *ent.ScheduledStep) (string, error) {
	return obj.ID.String(), nil
}

func (r *scheduledStepResolver) Type(ctx context.Context, obj *ent.ScheduledStep) (model.ScheduledStepType, error) {
	return model.ScheduledStepType(obj.Type), nil
}

func (r *scriptResolver) ID(ctx context.Context, obj *ent.Script) (string, error) {
	return obj.ID.String(), nil
}

func (r *scriptResolver) Vars(ctx context.Context, obj *ent.Script) ([]*model.VarsMap, error) {
	results := make([]*model.VarsMap, 0)
	for varKey, varValue := range obj.Vars {
		tempVar := &model.VarsMap{
			Key:   varKey,
			Value: varValue,
		}
		results = append(results, tempVar)
	}
	return results, nil
}

func (r *scriptResolver) Tags(ctx context.Context, obj *ent.Script) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Tags {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *serverTaskResolver) ID(ctx context.Context, obj *ent.ServerTask) (string, error) {
	return obj.ID.String(), nil
}

func (r *serverTaskResolver) Type(ctx context.Context, obj *ent.ServerTask) (model.ServerTaskType, error) {
	return model.ServerTaskType(obj.Type), nil
}

func (r *statusResolver) ID(ctx context.Context, obj *ent.Status) (string, error) {
	return obj.ID.String(), nil
}

func (r *statusResolver) State(ctx context.Context, obj *ent.Status) (model.ProvisionStatus, error) {
	return model.ProvisionStatus(obj.State), nil
}

func (r *statusResolver) StatusFor(ctx context.Context, obj *ent.Status) (model.ProvisionStatusFor, error) {
	return model.ProvisionStatusFor(obj.StatusFor), nil
}

func (r *statusResolver) StartedAt(ctx context.Context, obj *ent.Status) (string, error) {
	return obj.StartedAt.String(), nil
}

func (r *statusResolver) EndedAt(ctx context.Context, obj *ent.Status) (string, error) {
	return obj.EndedAt.String(), nil
}

func (r *subscriptionResolver) UpdatedAgentStatus(ctx context.Context) (<-chan *ent.AgentStatus, error) {
	newAgentStatus := make(chan *ent.AgentStatus, 1)
	go func() {
		sub := r.rdb.Subscribe(ctx, "newAgentStatus")
		_, err := sub.Receive(ctx)
		if err != nil {
			return
		}
		ch := sub.Channel()
		for {
			select {
			case message := <-ch:
				uuid, err := uuid.Parse(message.Payload)
				if err != nil {
					sub.Close()
					return
				}
				entAgentStatus, err := r.client.AgentStatus.Get(ctx, uuid)
				if err != nil {
					sub.Close()
					return
				}
				newAgentStatus <- entAgentStatus
			// close when context done
			case <-ctx.Done():
				sub.Close()
				return
			}
		}
	}()
	return newAgentStatus, nil
}

func (r *subscriptionResolver) UpdatedStatus(ctx context.Context) (<-chan *ent.Status, error) {
	newStatus := make(chan *ent.Status, 1)
	go func() {
		sub := r.rdb.Subscribe(ctx, "updatedStatus")
		_, err := sub.Receive(ctx)
		if err != nil {
			return
		}
		ch := sub.Channel()
		for {
			select {
			case message := <-ch:
				uuid, err := uuid.Parse(message.Payload)
				if err != nil {
					sub.Close()
					return
				}
				entStatus, err := r.client.Status.Get(ctx, uuid)
				if err != nil {
					sub.Close()
					return
				}
				newStatus <- entStatus
			// close when context done
			case <-ctx.Done():
				sub.Close()
				return
			}
		}
	}()
	return newStatus, nil
}

func (r *subscriptionResolver) UpdatedServerTask(ctx context.Context) (<-chan *ent.ServerTask, error) {
	newServerTask := make(chan *ent.ServerTask, 1)
	go func() {
		sub := r.rdb.Subscribe(ctx, "updatedServerTask")
		_, err := sub.Receive(ctx)
		if err != nil {
			return
		}
		ch := sub.Channel()
		for {
			select {
			case message := <-ch:
				uuid, err := uuid.Parse(message.Payload)
				if err != nil {
					sub.Close()
					return
				}
				entServerTask, err := r.client.ServerTask.Get(ctx, uuid)
				if err != nil {
					sub.Close()
					return
				}
				newServerTask <- entServerTask
			// close when context done
			case <-ctx.Done():
				sub.Close()
				return
			}
		}
	}()
	return newServerTask, nil
}

func (r *subscriptionResolver) UpdatedBuild(ctx context.Context) (<-chan *ent.Build, error) {
	newBuild := make(chan *ent.Build, 1)
	go func() {
		sub := r.rdb.Subscribe(ctx, "updatedBuild")
		_, err := sub.Receive(ctx)
		if err != nil {
			return
		}
		ch := sub.Channel()
		for {
			select {
			case message := <-ch:
				uuid, err := uuid.Parse(message.Payload)
				if err != nil {
					sub.Close()
					return
				}
				entBuild, err := r.client.Build.Get(ctx, uuid)
				if err != nil {
					sub.Close()
					return
				}
				newBuild <- entBuild
			// close when context done
			case <-ctx.Done():
				sub.Close()
				return
			}
		}
	}()
	return newBuild, nil
}

func (r *subscriptionResolver) UpdatedCommit(ctx context.Context) (<-chan *ent.BuildCommit, error) {
	newBuildCommit := make(chan *ent.BuildCommit, 1)
	go func() {
		sub := r.rdb.Subscribe(ctx, "updatedBuildCommit")
		_, err := sub.Receive(ctx)
		if err != nil {
			return
		}
		ch := sub.Channel()
		for {
			select {
			case message := <-ch:
				uuid, err := uuid.Parse(message.Payload)
				if err != nil {
					sub.Close()
					return
				}
				entBuildCommit, err := r.client.BuildCommit.Get(ctx, uuid)
				if err != nil {
					sub.Close()
					return
				}
				newBuildCommit <- entBuildCommit
			// close when context done
			case <-ctx.Done():
				sub.Close()
				return
			}
		}
	}()
	return newBuildCommit, nil
}

func (r *subscriptionResolver) UpdatedAgentTask(ctx context.Context) (<-chan *ent.AgentTask, error) {
	newAgentTask := make(chan *ent.AgentTask, 1)
	go func() {
		sub := r.rdb.Subscribe(ctx, "updatedAgentTask")
		_, err := sub.Receive(ctx)
		if err != nil {
			return
		}
		ch := sub.Channel()
		for {
			select {
			case message := <-ch:
				uuid, err := uuid.Parse(message.Payload)
				if err != nil {
					sub.Close()
					return
				}
				entAgentTask, err := r.client.AgentTask.Get(ctx, uuid)
				if err != nil {
					sub.Close()
					return
				}
				newAgentTask <- entAgentTask
			// close when context done
			case <-ctx.Done():
				sub.Close()
				return
			}
		}
	}()
	return newAgentTask, nil
}

func (r *subscriptionResolver) StreamServerTaskLog(ctx context.Context, taskID string) (<-chan string, error) {
	logStream := make(chan string, 1)
	go func(taskID string, logStream chan<- string) {
		filePeriod := 10 * time.Second

		uuid, err := uuid.Parse(taskID)
		if err != nil {
			return
		}
		entServerTask, err := r.client.ServerTask.Get(ctx, uuid)
		if err != nil {
			return
		}

		fileTicker := time.NewTicker(filePeriod)

		fi, err := os.Stat(entServerTask.LogFilePath)
		if err != nil {
			return
		}
		lastModified := fi.ModTime()
		fileBytes, err := ioutil.ReadFile(entServerTask.LogFilePath)
		if err != nil {
			return
		}

		logStream <- string(fileBytes)

		for {
			select {
			case <-fileTicker.C:
				fileBytes, lastModified, err = utils.ReadFileIfModified(lastModified, entServerTask.LogFilePath)
				if err != nil {
					logStream <- string(err.Error())
				}
				if fileBytes != nil {
					logStream <- string(fileBytes)
				}
			case <-ctx.Done():
				return
			}
		}

	}(taskID, logStream)
	return logStream, nil
}

func (r *tagResolver) ID(ctx context.Context, obj *ent.Tag) (string, error) {
	return obj.ID.String(), nil
}

func (r *tagResolver) UUID(ctx context.Context, obj *ent.Tag) (string, error) {
	return obj.ID.String(), nil
}

func (r *tagResolver) Description(ctx context.Context, obj *ent.Tag) ([]*model.TagMap, error) {
	results := make([]*model.TagMap, 0)
	for tagKey, tagValue := range obj.Description {
		tempTag := &model.TagMap{
			Key:   tagKey,
			Value: tagValue,
		}
		results = append(results, tempTag)
	}
	return results, nil
}

func (r *teamResolver) ID(ctx context.Context, obj *ent.Team) (string, error) {
	return obj.ID.String(), nil
}

func (r *userResolver) ID(ctx context.Context, obj *ent.User) (string, error) {
	return obj.ID.String(), nil
}

// AdhocPlan returns generated.AdhocPlanResolver implementation.
func (r *Resolver) AdhocPlan() generated.AdhocPlanResolver { return &adhocPlanResolver{r} }

// AgentTask returns generated.AgentTaskResolver implementation.
func (r *Resolver) AgentTask() generated.AgentTaskResolver { return &agentTaskResolver{r} }

// Ansible returns generated.AnsibleResolver implementation.
func (r *Resolver) Ansible() generated.AnsibleResolver { return &ansibleResolver{r} }

// AuthUser returns generated.AuthUserResolver implementation.
func (r *Resolver) AuthUser() generated.AuthUserResolver { return &authUserResolver{r} }

// Build returns generated.BuildResolver implementation.
func (r *Resolver) Build() generated.BuildResolver { return &buildResolver{r} }

// BuildCommit returns generated.BuildCommitResolver implementation.
func (r *Resolver) BuildCommit() generated.BuildCommitResolver { return &buildCommitResolver{r} }

// Command returns generated.CommandResolver implementation.
func (r *Resolver) Command() generated.CommandResolver { return &commandResolver{r} }

// Competition returns generated.CompetitionResolver implementation.
func (r *Resolver) Competition() generated.CompetitionResolver { return &competitionResolver{r} }

// DNS returns generated.DNSResolver implementation.
func (r *Resolver) DNS() generated.DNSResolver { return &dNSResolver{r} }

// DNSRecord returns generated.DNSRecordResolver implementation.
func (r *Resolver) DNSRecord() generated.DNSRecordResolver { return &dNSRecordResolver{r} }

// Disk returns generated.DiskResolver implementation.
func (r *Resolver) Disk() generated.DiskResolver { return &diskResolver{r} }

// Environment returns generated.EnvironmentResolver implementation.
func (r *Resolver) Environment() generated.EnvironmentResolver { return &environmentResolver{r} }

// FileDelete returns generated.FileDeleteResolver implementation.
func (r *Resolver) FileDelete() generated.FileDeleteResolver { return &fileDeleteResolver{r} }

// FileDownload returns generated.FileDownloadResolver implementation.
func (r *Resolver) FileDownload() generated.FileDownloadResolver { return &fileDownloadResolver{r} }

// FileExtract returns generated.FileExtractResolver implementation.
func (r *Resolver) FileExtract() generated.FileExtractResolver { return &fileExtractResolver{r} }

// Finding returns generated.FindingResolver implementation.
func (r *Resolver) Finding() generated.FindingResolver { return &findingResolver{r} }

// GinFileMiddleware returns generated.GinFileMiddlewareResolver implementation.
func (r *Resolver) GinFileMiddleware() generated.GinFileMiddlewareResolver {
	return &ginFileMiddlewareResolver{r}
}

// Host returns generated.HostResolver implementation.
func (r *Resolver) Host() generated.HostResolver { return &hostResolver{r} }

// HostDependency returns generated.HostDependencyResolver implementation.
func (r *Resolver) HostDependency() generated.HostDependencyResolver {
	return &hostDependencyResolver{r}
}

// Identity returns generated.IdentityResolver implementation.
func (r *Resolver) Identity() generated.IdentityResolver { return &identityResolver{r} }

// IncludedNetwork returns generated.IncludedNetworkResolver implementation.
func (r *Resolver) IncludedNetwork() generated.IncludedNetworkResolver {
	return &includedNetworkResolver{r}
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Network returns generated.NetworkResolver implementation.
func (r *Resolver) Network() generated.NetworkResolver { return &networkResolver{r} }

// Plan returns generated.PlanResolver implementation.
func (r *Resolver) Plan() generated.PlanResolver { return &planResolver{r} }

// PlanDiff returns generated.PlanDiffResolver implementation.
func (r *Resolver) PlanDiff() generated.PlanDiffResolver { return &planDiffResolver{r} }

// ProvisionedHost returns generated.ProvisionedHostResolver implementation.
func (r *Resolver) ProvisionedHost() generated.ProvisionedHostResolver {
	return &provisionedHostResolver{r}
}

// ProvisionedNetwork returns generated.ProvisionedNetworkResolver implementation.
func (r *Resolver) ProvisionedNetwork() generated.ProvisionedNetworkResolver {
	return &provisionedNetworkResolver{r}
}

// ProvisioningScheduledStep returns generated.ProvisioningScheduledStepResolver implementation.
func (r *Resolver) ProvisioningScheduledStep() generated.ProvisioningScheduledStepResolver {
	return &provisioningScheduledStepResolver{r}
}

// ProvisioningStep returns generated.ProvisioningStepResolver implementation.
func (r *Resolver) ProvisioningStep() generated.ProvisioningStepResolver {
	return &provisioningStepResolver{r}
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// RepoCommit returns generated.RepoCommitResolver implementation.
func (r *Resolver) RepoCommit() generated.RepoCommitResolver { return &repoCommitResolver{r} }

// Repository returns generated.RepositoryResolver implementation.
func (r *Resolver) Repository() generated.RepositoryResolver { return &repositoryResolver{r} }

// ScheduledStep returns generated.ScheduledStepResolver implementation.
func (r *Resolver) ScheduledStep() generated.ScheduledStepResolver { return &scheduledStepResolver{r} }

// Script returns generated.ScriptResolver implementation.
func (r *Resolver) Script() generated.ScriptResolver { return &scriptResolver{r} }

// ServerTask returns generated.ServerTaskResolver implementation.
func (r *Resolver) ServerTask() generated.ServerTaskResolver { return &serverTaskResolver{r} }

// Status returns generated.StatusResolver implementation.
func (r *Resolver) Status() generated.StatusResolver { return &statusResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

// Tag returns generated.TagResolver implementation.
func (r *Resolver) Tag() generated.TagResolver { return &tagResolver{r} }

// Team returns generated.TeamResolver implementation.
func (r *Resolver) Team() generated.TeamResolver { return &teamResolver{r} }

// User returns generated.UserResolver implementation.
func (r *Resolver) User() generated.UserResolver { return &userResolver{r} }

type adhocPlanResolver struct{ *Resolver }
type agentTaskResolver struct{ *Resolver }
type ansibleResolver struct{ *Resolver }
type authUserResolver struct{ *Resolver }
type buildResolver struct{ *Resolver }
type buildCommitResolver struct{ *Resolver }
type commandResolver struct{ *Resolver }
type competitionResolver struct{ *Resolver }
type dNSResolver struct{ *Resolver }
type dNSRecordResolver struct{ *Resolver }
type diskResolver struct{ *Resolver }
type environmentResolver struct{ *Resolver }
type fileDeleteResolver struct{ *Resolver }
type fileDownloadResolver struct{ *Resolver }
type fileExtractResolver struct{ *Resolver }
type findingResolver struct{ *Resolver }
type ginFileMiddlewareResolver struct{ *Resolver }
type hostResolver struct{ *Resolver }
type hostDependencyResolver struct{ *Resolver }
type identityResolver struct{ *Resolver }
type includedNetworkResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type networkResolver struct{ *Resolver }
type planResolver struct{ *Resolver }
type planDiffResolver struct{ *Resolver }
type provisionedHostResolver struct{ *Resolver }
type provisionedNetworkResolver struct{ *Resolver }
type provisioningScheduledStepResolver struct{ *Resolver }
type provisioningStepResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type repoCommitResolver struct{ *Resolver }
type repositoryResolver struct{ *Resolver }
type scheduledStepResolver struct{ *Resolver }
type scriptResolver struct{ *Resolver }
type serverTaskResolver struct{ *Resolver }
type statusResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
type tagResolver struct{ *Resolver }
type teamResolver struct{ *Resolver }
type userResolver struct{ *Resolver }
