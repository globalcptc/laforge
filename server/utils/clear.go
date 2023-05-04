package utils

import (
	"context"
	"fmt"

	"github.com/gen0cide/laforge/ent"
)

// rollback calls to tx.Rollback and wraps the given error
// with the rollback error if occurred.
func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		err = fmt.Errorf("%w: %v", err, rerr)
	}
	return err
}

func ClearDB(ctx context.Context, client *ent.Client, laforgeConfig *ServerConfig) (map[string]int, error) {
	results := map[string]int{}
	txClient, err := client.Tx(ctx)
	if err != nil {
		return results, fmt.Errorf("failed to create transactional client: %v", err)
	}

	deletedCount, err := txClient.AdhocPlan.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["AdhocPlan"] = int(deletedCount)
	deletedCount, err = txClient.AgentTask.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["AgentTask"] = int(deletedCount)
	deletedCount, err = txClient.AgentStatus.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["AgentStatus"] = int(deletedCount)

	deletedCount, err = txClient.Status.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Status"] = int(deletedCount)

	deletedCount, err = txClient.PlanDiff.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["PlanDiff"] = int(deletedCount)

	deletedCount, err = client.GinFileMiddleware.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["GinFileMiddleware"] = int(deletedCount)
	deletedCount, err = client.ProvisioningStep.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["ProvisioningStep"] = int(deletedCount)
	deletedCount, err = client.ProvisioningScheduledStep.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["ProvisioningScheduledStep"] = int(deletedCount)
	deletedCount, err = client.ProvisionedHost.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["ProvisionedHost"] = int(deletedCount)
	deletedCount, err = client.ProvisionedNetwork.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["ProvisionedNetwork"] = int(deletedCount)
	deletedCount, err = client.Team.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Team"] = int(deletedCount)
	deletedCount, err = client.BuildCommit.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["BuildCommit"] = int(deletedCount)
	deletedCount, err = client.Plan.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Plan"] = int(deletedCount)
	deletedCount, err = client.Build.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Build"] = int(deletedCount)

	//
	deletedCount, err = client.DNSRecord.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}

	results["DNSRecord"] = deletedCount
	deletedCount, err = client.Command.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Command"] = deletedCount
	deletedCount, err = client.Script.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Script"] = deletedCount
	deletedCount, err = client.FileDownload.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["FileDownload"] = deletedCount
	deletedCount, err = client.FileExtract.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["FileExtract"] = deletedCount
	deletedCount, err = client.FileDelete.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["FileDelete"] = deletedCount
	//

	deletedCount, err = client.HostDependency.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["HostDependency"] = deletedCount
	deletedCount, err = client.Finding.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Finding"] = deletedCount
	deletedCount, err = client.Disk.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Disk"] = deletedCount
	deletedCount, err = client.Host.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Host"] = deletedCount

	deletedCount, err = client.IncludedNetwork.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["IncludedNetwork"] = deletedCount
	deletedCount, err = client.Network.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Network"] = deletedCount

	deletedCount, err = client.Identity.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Identity"] = deletedCount
	deletedCount, err = client.DNS.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["DNS"] = deletedCount
	deletedCount, err = client.Validation.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Validation"] = deletedCount

	deletedCount, err = client.Competition.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Competition"] = deletedCount

	deletedCount, err = client.Environment.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Environment"] = deletedCount

	deletedCount, err = client.RepoCommit.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["RepoCommit"] = deletedCount
	deletedCount, err = client.Repository.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["Repository"] = deletedCount

	deletedCount, err = client.ServerTask.Delete().Exec(ctx)
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}
	results["ServerTask"] = deletedCount

	err = txClient.Commit()
	if err != nil {
		err = rollback(txClient, err)
		return results, err
	}

	return results, nil
}
