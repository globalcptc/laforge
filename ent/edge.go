// Code generated by entc, DO NOT EDIT.

package ent

import "context"

func (as *AgentStatus) AgentStatusToTag(ctx context.Context) ([]*Tag, error) {
	result, err := as.Edges.AgentStatusToTagOrErr()
	if IsNotLoaded(err) {
		result, err = as.QueryAgentStatusToTag().All(ctx)
	}
	return result, err
}

func (as *AgentStatus) AgentStatusToProvisionedHost(ctx context.Context) ([]*ProvisionedHost, error) {
	result, err := as.Edges.AgentStatusToProvisionedHostOrErr()
	if IsNotLoaded(err) {
		result, err = as.QueryAgentStatusToProvisionedHost().All(ctx)
	}
	return result, err
}

func (b *Build) BuildToStatus(ctx context.Context) (*Status, error) {
	result, err := b.Edges.BuildToStatusOrErr()
	if IsNotLoaded(err) {
		result, err = b.QueryBuildToStatus().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (b *Build) BuildToEnvironment(ctx context.Context) (*Environment, error) {
	result, err := b.Edges.BuildToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = b.QueryBuildToEnvironment().Only(ctx)
	}
	return result, err
}

func (b *Build) BuildToCompetition(ctx context.Context) (*Competition, error) {
	result, err := b.Edges.BuildToCompetitionOrErr()
	if IsNotLoaded(err) {
		result, err = b.QueryBuildToCompetition().Only(ctx)
	}
	return result, err
}

func (b *Build) BuildToProvisionedNetwork(ctx context.Context) ([]*ProvisionedNetwork, error) {
	result, err := b.Edges.BuildToProvisionedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = b.QueryBuildToProvisionedNetwork().All(ctx)
	}
	return result, err
}

func (b *Build) BuildToTeam(ctx context.Context) ([]*Team, error) {
	result, err := b.Edges.BuildToTeamOrErr()
	if IsNotLoaded(err) {
		result, err = b.QueryBuildToTeam().All(ctx)
	}
	return result, err
}

func (b *Build) BuildToPlan(ctx context.Context) ([]*Plan, error) {
	result, err := b.Edges.BuildToPlanOrErr()
	if IsNotLoaded(err) {
		result, err = b.QueryBuildToPlan().All(ctx)
	}
	return result, err
}

func (c *Command) CommandToUser(ctx context.Context) ([]*User, error) {
	result, err := c.Edges.CommandToUserOrErr()
	if IsNotLoaded(err) {
		result, err = c.QueryCommandToUser().All(ctx)
	}
	return result, err
}

func (c *Command) CommandToTag(ctx context.Context) ([]*Tag, error) {
	result, err := c.Edges.CommandToTagOrErr()
	if IsNotLoaded(err) {
		result, err = c.QueryCommandToTag().All(ctx)
	}
	return result, err
}

func (c *Command) CommandToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := c.Edges.CommandToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = c.QueryCommandToEnvironment().All(ctx)
	}
	return result, err
}

func (c *Competition) CompetitionToTag(ctx context.Context) ([]*Tag, error) {
	result, err := c.Edges.CompetitionToTagOrErr()
	if IsNotLoaded(err) {
		result, err = c.QueryCompetitionToTag().All(ctx)
	}
	return result, err
}

func (c *Competition) CompetitionToDNS(ctx context.Context) ([]*DNS, error) {
	result, err := c.Edges.CompetitionToDNSOrErr()
	if IsNotLoaded(err) {
		result, err = c.QueryCompetitionToDNS().All(ctx)
	}
	return result, err
}

func (c *Competition) CompetitionToEnvironment(ctx context.Context) (*Environment, error) {
	result, err := c.Edges.CompetitionToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = c.QueryCompetitionToEnvironment().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (c *Competition) CompetitionToBuild(ctx context.Context) ([]*Build, error) {
	result, err := c.Edges.CompetitionToBuildOrErr()
	if IsNotLoaded(err) {
		result, err = c.QueryCompetitionToBuild().All(ctx)
	}
	return result, err
}

func (d *DNS) DNSToTag(ctx context.Context) ([]*Tag, error) {
	result, err := d.Edges.DNSToTagOrErr()
	if IsNotLoaded(err) {
		result, err = d.QueryDNSToTag().All(ctx)
	}
	return result, err
}

func (d *DNS) DNSToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := d.Edges.DNSToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = d.QueryDNSToEnvironment().All(ctx)
	}
	return result, err
}

func (d *DNS) DNSToCompetition(ctx context.Context) ([]*Competition, error) {
	result, err := d.Edges.DNSToCompetitionOrErr()
	if IsNotLoaded(err) {
		result, err = d.QueryDNSToCompetition().All(ctx)
	}
	return result, err
}

func (dr *DNSRecord) DNSRecordToTag(ctx context.Context) ([]*Tag, error) {
	result, err := dr.Edges.DNSRecordToTagOrErr()
	if IsNotLoaded(err) {
		result, err = dr.QueryDNSRecordToTag().All(ctx)
	}
	return result, err
}

func (dr *DNSRecord) DNSRecordToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := dr.Edges.DNSRecordToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = dr.QueryDNSRecordToEnvironment().All(ctx)
	}
	return result, err
}

func (d *Disk) DiskToTag(ctx context.Context) ([]*Tag, error) {
	result, err := d.Edges.DiskToTagOrErr()
	if IsNotLoaded(err) {
		result, err = d.QueryDiskToTag().All(ctx)
	}
	return result, err
}

func (d *Disk) DiskToHost(ctx context.Context) ([]*Host, error) {
	result, err := d.Edges.DiskToHostOrErr()
	if IsNotLoaded(err) {
		result, err = d.QueryDiskToHost().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToTag(ctx context.Context) ([]*Tag, error) {
	result, err := e.Edges.EnvironmentToTagOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToTag().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToUser(ctx context.Context) ([]*User, error) {
	result, err := e.Edges.EnvironmentToUserOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToUser().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToHost(ctx context.Context) ([]*Host, error) {
	result, err := e.Edges.EnvironmentToHostOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToHost().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToCompetition(ctx context.Context) ([]*Competition, error) {
	result, err := e.Edges.EnvironmentToCompetitionOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToCompetition().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToIdentity(ctx context.Context) ([]*Identity, error) {
	result, err := e.Edges.EnvironmentToIdentityOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToIdentity().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToCommand(ctx context.Context) ([]*Command, error) {
	result, err := e.Edges.EnvironmentToCommandOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToCommand().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToScript(ctx context.Context) ([]*Script, error) {
	result, err := e.Edges.EnvironmentToScriptOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToScript().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToFileDownload(ctx context.Context) ([]*FileDownload, error) {
	result, err := e.Edges.EnvironmentToFileDownloadOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToFileDownload().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToFileDelete(ctx context.Context) ([]*FileDelete, error) {
	result, err := e.Edges.EnvironmentToFileDeleteOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToFileDelete().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToFileExtract(ctx context.Context) ([]*FileExtract, error) {
	result, err := e.Edges.EnvironmentToFileExtractOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToFileExtract().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToIncludedNetwork(ctx context.Context) ([]*IncludedNetwork, error) {
	result, err := e.Edges.EnvironmentToIncludedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToIncludedNetwork().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToFinding(ctx context.Context) ([]*Finding, error) {
	result, err := e.Edges.EnvironmentToFindingOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToFinding().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToDNSRecord(ctx context.Context) ([]*DNSRecord, error) {
	result, err := e.Edges.EnvironmentToDNSRecordOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToDNSRecord().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToDNS(ctx context.Context) ([]*DNS, error) {
	result, err := e.Edges.EnvironmentToDNSOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToDNS().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToNetwork(ctx context.Context) ([]*Network, error) {
	result, err := e.Edges.EnvironmentToNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToNetwork().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToHostDependency(ctx context.Context) ([]*HostDependency, error) {
	result, err := e.Edges.EnvironmentToHostDependencyOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToHostDependency().All(ctx)
	}
	return result, err
}

func (e *Environment) EnvironmentToBuild(ctx context.Context) ([]*Build, error) {
	result, err := e.Edges.EnvironmentToBuildOrErr()
	if IsNotLoaded(err) {
		result, err = e.QueryEnvironmentToBuild().All(ctx)
	}
	return result, err
}

func (fd *FileDelete) FileDeleteToTag(ctx context.Context) ([]*Tag, error) {
	result, err := fd.Edges.FileDeleteToTagOrErr()
	if IsNotLoaded(err) {
		result, err = fd.QueryFileDeleteToTag().All(ctx)
	}
	return result, err
}

func (fd *FileDelete) FileDeleteToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := fd.Edges.FileDeleteToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = fd.QueryFileDeleteToEnvironment().All(ctx)
	}
	return result, err
}

func (fd *FileDownload) FileDownloadToTag(ctx context.Context) ([]*Tag, error) {
	result, err := fd.Edges.FileDownloadToTagOrErr()
	if IsNotLoaded(err) {
		result, err = fd.QueryFileDownloadToTag().All(ctx)
	}
	return result, err
}

func (fd *FileDownload) FileDownloadToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := fd.Edges.FileDownloadToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = fd.QueryFileDownloadToEnvironment().All(ctx)
	}
	return result, err
}

func (fe *FileExtract) FileExtractToTag(ctx context.Context) ([]*Tag, error) {
	result, err := fe.Edges.FileExtractToTagOrErr()
	if IsNotLoaded(err) {
		result, err = fe.QueryFileExtractToTag().All(ctx)
	}
	return result, err
}

func (fe *FileExtract) FileExtractToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := fe.Edges.FileExtractToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = fe.QueryFileExtractToEnvironment().All(ctx)
	}
	return result, err
}

func (f *Finding) FindingToUser(ctx context.Context) ([]*User, error) {
	result, err := f.Edges.FindingToUserOrErr()
	if IsNotLoaded(err) {
		result, err = f.QueryFindingToUser().All(ctx)
	}
	return result, err
}

func (f *Finding) FindingToTag(ctx context.Context) ([]*Tag, error) {
	result, err := f.Edges.FindingToTagOrErr()
	if IsNotLoaded(err) {
		result, err = f.QueryFindingToTag().All(ctx)
	}
	return result, err
}

func (f *Finding) FindingToHost(ctx context.Context) ([]*Host, error) {
	result, err := f.Edges.FindingToHostOrErr()
	if IsNotLoaded(err) {
		result, err = f.QueryFindingToHost().All(ctx)
	}
	return result, err
}

func (f *Finding) FindingToScript(ctx context.Context) ([]*Script, error) {
	result, err := f.Edges.FindingToScriptOrErr()
	if IsNotLoaded(err) {
		result, err = f.QueryFindingToScript().All(ctx)
	}
	return result, err
}

func (f *Finding) FindingToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := f.Edges.FindingToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = f.QueryFindingToEnvironment().All(ctx)
	}
	return result, err
}

func (gfm *GinFileMiddleware) GinFileMiddlewareToProvisionedHost(ctx context.Context) (*ProvisionedHost, error) {
	result, err := gfm.Edges.GinFileMiddlewareToProvisionedHostOrErr()
	if IsNotLoaded(err) {
		result, err = gfm.QueryGinFileMiddlewareToProvisionedHost().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (gfm *GinFileMiddleware) GinFileMiddlewareToProvisioningStep(ctx context.Context) (*ProvisioningStep, error) {
	result, err := gfm.Edges.GinFileMiddlewareToProvisioningStepOrErr()
	if IsNotLoaded(err) {
		result, err = gfm.QueryGinFileMiddlewareToProvisioningStep().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (h *Host) HostToDisk(ctx context.Context) ([]*Disk, error) {
	result, err := h.Edges.HostToDiskOrErr()
	if IsNotLoaded(err) {
		result, err = h.QueryHostToDisk().All(ctx)
	}
	return result, err
}

func (h *Host) HostToUser(ctx context.Context) ([]*User, error) {
	result, err := h.Edges.HostToUserOrErr()
	if IsNotLoaded(err) {
		result, err = h.QueryHostToUser().All(ctx)
	}
	return result, err
}

func (h *Host) HostToTag(ctx context.Context) ([]*Tag, error) {
	result, err := h.Edges.HostToTagOrErr()
	if IsNotLoaded(err) {
		result, err = h.QueryHostToTag().All(ctx)
	}
	return result, err
}

func (h *Host) HostToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := h.Edges.HostToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = h.QueryHostToEnvironment().All(ctx)
	}
	return result, err
}

func (h *Host) HostToIncludedNetwork(ctx context.Context) ([]*IncludedNetwork, error) {
	result, err := h.Edges.HostToIncludedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = h.QueryHostToIncludedNetwork().All(ctx)
	}
	return result, err
}

func (h *Host) DependOnHostToHostDependency(ctx context.Context) ([]*HostDependency, error) {
	result, err := h.Edges.DependOnHostToHostDependencyOrErr()
	if IsNotLoaded(err) {
		result, err = h.QueryDependOnHostToHostDependency().All(ctx)
	}
	return result, err
}

func (h *Host) DependByHostToHostDependency(ctx context.Context) ([]*HostDependency, error) {
	result, err := h.Edges.DependByHostToHostDependencyOrErr()
	if IsNotLoaded(err) {
		result, err = h.QueryDependByHostToHostDependency().All(ctx)
	}
	return result, err
}

func (hd *HostDependency) HostDependencyToDependOnHost(ctx context.Context) (*Host, error) {
	result, err := hd.Edges.HostDependencyToDependOnHostOrErr()
	if IsNotLoaded(err) {
		result, err = hd.QueryHostDependencyToDependOnHost().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (hd *HostDependency) HostDependencyToDependByHost(ctx context.Context) (*Host, error) {
	result, err := hd.Edges.HostDependencyToDependByHostOrErr()
	if IsNotLoaded(err) {
		result, err = hd.QueryHostDependencyToDependByHost().Only(ctx)
	}
	return result, err
}

func (hd *HostDependency) HostDependencyToNetwork(ctx context.Context) (*Network, error) {
	result, err := hd.Edges.HostDependencyToNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = hd.QueryHostDependencyToNetwork().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (hd *HostDependency) HostDependencyToEnvironment(ctx context.Context) (*Environment, error) {
	result, err := hd.Edges.HostDependencyToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = hd.QueryHostDependencyToEnvironment().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (i *Identity) IdentityToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := i.Edges.IdentityToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = i.QueryIdentityToEnvironment().All(ctx)
	}
	return result, err
}

func (in *IncludedNetwork) IncludedNetworkToTag(ctx context.Context) ([]*Tag, error) {
	result, err := in.Edges.IncludedNetworkToTagOrErr()
	if IsNotLoaded(err) {
		result, err = in.QueryIncludedNetworkToTag().All(ctx)
	}
	return result, err
}

func (in *IncludedNetwork) IncludedNetworkToHost(ctx context.Context) ([]*Host, error) {
	result, err := in.Edges.IncludedNetworkToHostOrErr()
	if IsNotLoaded(err) {
		result, err = in.QueryIncludedNetworkToHost().All(ctx)
	}
	return result, err
}

func (in *IncludedNetwork) IncludedNetworkToNetwork(ctx context.Context) ([]*Network, error) {
	result, err := in.Edges.IncludedNetworkToNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = in.QueryIncludedNetworkToNetwork().All(ctx)
	}
	return result, err
}

func (in *IncludedNetwork) IncludedNetworkToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := in.Edges.IncludedNetworkToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = in.QueryIncludedNetworkToEnvironment().All(ctx)
	}
	return result, err
}

func (n *Network) NetworkToTag(ctx context.Context) ([]*Tag, error) {
	result, err := n.Edges.NetworkToTagOrErr()
	if IsNotLoaded(err) {
		result, err = n.QueryNetworkToTag().All(ctx)
	}
	return result, err
}

func (n *Network) NetworkToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := n.Edges.NetworkToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = n.QueryNetworkToEnvironment().All(ctx)
	}
	return result, err
}

func (n *Network) NetworkToHostDependency(ctx context.Context) ([]*HostDependency, error) {
	result, err := n.Edges.NetworkToHostDependencyOrErr()
	if IsNotLoaded(err) {
		result, err = n.QueryNetworkToHostDependency().All(ctx)
	}
	return result, err
}

func (n *Network) NetworkToIncludedNetwork(ctx context.Context) ([]*IncludedNetwork, error) {
	result, err := n.Edges.NetworkToIncludedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = n.QueryNetworkToIncludedNetwork().All(ctx)
	}
	return result, err
}

func (pl *Plan) PrevPlan(ctx context.Context) ([]*Plan, error) {
	result, err := pl.Edges.PrevPlanOrErr()
	if IsNotLoaded(err) {
		result, err = pl.QueryPrevPlan().All(ctx)
	}
	return result, err
}

func (pl *Plan) NextPlan(ctx context.Context) ([]*Plan, error) {
	result, err := pl.Edges.NextPlanOrErr()
	if IsNotLoaded(err) {
		result, err = pl.QueryNextPlan().All(ctx)
	}
	return result, err
}

func (pl *Plan) PlanToBuild(ctx context.Context) (*Build, error) {
	result, err := pl.Edges.PlanToBuildOrErr()
	if IsNotLoaded(err) {
		result, err = pl.QueryPlanToBuild().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pl *Plan) PlanToTeam(ctx context.Context) (*Team, error) {
	result, err := pl.Edges.PlanToTeamOrErr()
	if IsNotLoaded(err) {
		result, err = pl.QueryPlanToTeam().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pl *Plan) PlanToProvisionedNetwork(ctx context.Context) (*ProvisionedNetwork, error) {
	result, err := pl.Edges.PlanToProvisionedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = pl.QueryPlanToProvisionedNetwork().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pl *Plan) PlanToProvisionedHost(ctx context.Context) (*ProvisionedHost, error) {
	result, err := pl.Edges.PlanToProvisionedHostOrErr()
	if IsNotLoaded(err) {
		result, err = pl.QueryPlanToProvisionedHost().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pl *Plan) PlanToProvisioningStep(ctx context.Context) (*ProvisioningStep, error) {
	result, err := pl.Edges.PlanToProvisioningStepOrErr()
	if IsNotLoaded(err) {
		result, err = pl.QueryPlanToProvisioningStep().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ph *ProvisionedHost) ProvisionedHostToStatus(ctx context.Context) (*Status, error) {
	result, err := ph.Edges.ProvisionedHostToStatusOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToStatus().Only(ctx)
	}
	return result, err
}

func (ph *ProvisionedHost) ProvisionedHostToProvisionedNetwork(ctx context.Context) (*ProvisionedNetwork, error) {
	result, err := ph.Edges.ProvisionedHostToProvisionedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToProvisionedNetwork().Only(ctx)
	}
	return result, err
}

func (ph *ProvisionedHost) ProvisionedHostToHost(ctx context.Context) (*Host, error) {
	result, err := ph.Edges.ProvisionedHostToHostOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToHost().Only(ctx)
	}
	return result, err
}

func (ph *ProvisionedHost) ProvisionedHostToEndStepPlan(ctx context.Context) (*Plan, error) {
	result, err := ph.Edges.ProvisionedHostToEndStepPlanOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToEndStepPlan().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ph *ProvisionedHost) ProvisionedHostToProvisioningStep(ctx context.Context) ([]*ProvisioningStep, error) {
	result, err := ph.Edges.ProvisionedHostToProvisioningStepOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToProvisioningStep().All(ctx)
	}
	return result, err
}

func (ph *ProvisionedHost) ProvisionedHostToAgentStatus(ctx context.Context) ([]*AgentStatus, error) {
	result, err := ph.Edges.ProvisionedHostToAgentStatusOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToAgentStatus().All(ctx)
	}
	return result, err
}

func (ph *ProvisionedHost) ProvisionedHostToPlan(ctx context.Context) ([]*Plan, error) {
	result, err := ph.Edges.ProvisionedHostToPlanOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToPlan().All(ctx)
	}
	return result, err
}

func (ph *ProvisionedHost) ProvisionedHostToGinFileMiddleware(ctx context.Context) (*GinFileMiddleware, error) {
	result, err := ph.Edges.ProvisionedHostToGinFileMiddlewareOrErr()
	if IsNotLoaded(err) {
		result, err = ph.QueryProvisionedHostToGinFileMiddleware().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pn *ProvisionedNetwork) ProvisionedNetworkToStatus(ctx context.Context) (*Status, error) {
	result, err := pn.Edges.ProvisionedNetworkToStatusOrErr()
	if IsNotLoaded(err) {
		result, err = pn.QueryProvisionedNetworkToStatus().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pn *ProvisionedNetwork) ProvisionedNetworkToNetwork(ctx context.Context) (*Network, error) {
	result, err := pn.Edges.ProvisionedNetworkToNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = pn.QueryProvisionedNetworkToNetwork().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pn *ProvisionedNetwork) ProvisionedNetworkToBuild(ctx context.Context) (*Build, error) {
	result, err := pn.Edges.ProvisionedNetworkToBuildOrErr()
	if IsNotLoaded(err) {
		result, err = pn.QueryProvisionedNetworkToBuild().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pn *ProvisionedNetwork) ProvisionedNetworkToTeam(ctx context.Context) (*Team, error) {
	result, err := pn.Edges.ProvisionedNetworkToTeamOrErr()
	if IsNotLoaded(err) {
		result, err = pn.QueryProvisionedNetworkToTeam().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (pn *ProvisionedNetwork) ProvisionedNetworkToProvisionedHost(ctx context.Context) ([]*ProvisionedHost, error) {
	result, err := pn.Edges.ProvisionedNetworkToProvisionedHostOrErr()
	if IsNotLoaded(err) {
		result, err = pn.QueryProvisionedNetworkToProvisionedHost().All(ctx)
	}
	return result, err
}

func (pn *ProvisionedNetwork) ProvisionedNetworkToPlan(ctx context.Context) (*Plan, error) {
	result, err := pn.Edges.ProvisionedNetworkToPlanOrErr()
	if IsNotLoaded(err) {
		result, err = pn.QueryProvisionedNetworkToPlan().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToStatus(ctx context.Context) (*Status, error) {
	result, err := ps.Edges.ProvisioningStepToStatusOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToStatus().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToProvisionedHost(ctx context.Context) (*ProvisionedHost, error) {
	result, err := ps.Edges.ProvisioningStepToProvisionedHostOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToProvisionedHost().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToScript(ctx context.Context) (*Script, error) {
	result, err := ps.Edges.ProvisioningStepToScriptOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToScript().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToCommand(ctx context.Context) (*Command, error) {
	result, err := ps.Edges.ProvisioningStepToCommandOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToCommand().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToDNSRecord(ctx context.Context) (*DNSRecord, error) {
	result, err := ps.Edges.ProvisioningStepToDNSRecordOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToDNSRecord().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToFileDelete(ctx context.Context) (*FileDelete, error) {
	result, err := ps.Edges.ProvisioningStepToFileDeleteOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToFileDelete().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToFileDownload(ctx context.Context) (*FileDownload, error) {
	result, err := ps.Edges.ProvisioningStepToFileDownloadOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToFileDownload().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToFileExtract(ctx context.Context) (*FileExtract, error) {
	result, err := ps.Edges.ProvisioningStepToFileExtractOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToFileExtract().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToPlan(ctx context.Context) (*Plan, error) {
	result, err := ps.Edges.ProvisioningStepToPlanOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToPlan().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (ps *ProvisioningStep) ProvisioningStepToGinFileMiddleware(ctx context.Context) (*GinFileMiddleware, error) {
	result, err := ps.Edges.ProvisioningStepToGinFileMiddlewareOrErr()
	if IsNotLoaded(err) {
		result, err = ps.QueryProvisioningStepToGinFileMiddleware().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (s *Script) ScriptToTag(ctx context.Context) ([]*Tag, error) {
	result, err := s.Edges.ScriptToTagOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryScriptToTag().All(ctx)
	}
	return result, err
}

func (s *Script) ScriptToUser(ctx context.Context) ([]*User, error) {
	result, err := s.Edges.ScriptToUserOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryScriptToUser().All(ctx)
	}
	return result, err
}

func (s *Script) ScriptToFinding(ctx context.Context) ([]*Finding, error) {
	result, err := s.Edges.ScriptToFindingOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryScriptToFinding().All(ctx)
	}
	return result, err
}

func (s *Script) ScriptToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := s.Edges.ScriptToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryScriptToEnvironment().All(ctx)
	}
	return result, err
}

func (s *Status) StatusToBuild(ctx context.Context) (*Build, error) {
	result, err := s.Edges.StatusToBuildOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryStatusToBuild().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (s *Status) StatusToProvisionedNetwork(ctx context.Context) (*ProvisionedNetwork, error) {
	result, err := s.Edges.StatusToProvisionedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryStatusToProvisionedNetwork().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (s *Status) StatusToProvisionedHost(ctx context.Context) (*ProvisionedHost, error) {
	result, err := s.Edges.StatusToProvisionedHostOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryStatusToProvisionedHost().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (s *Status) StatusToProvisioningStep(ctx context.Context) (*ProvisioningStep, error) {
	result, err := s.Edges.StatusToProvisioningStepOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryStatusToProvisioningStep().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (s *Status) StatusToTeam(ctx context.Context) (*Team, error) {
	result, err := s.Edges.StatusToTeamOrErr()
	if IsNotLoaded(err) {
		result, err = s.QueryStatusToTeam().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (t *Team) TeamToBuild(ctx context.Context) (*Build, error) {
	result, err := t.Edges.TeamToBuildOrErr()
	if IsNotLoaded(err) {
		result, err = t.QueryTeamToBuild().Only(ctx)
	}
	return result, err
}

func (t *Team) TeamToStatus(ctx context.Context) (*Status, error) {
	result, err := t.Edges.TeamToStatusOrErr()
	if IsNotLoaded(err) {
		result, err = t.QueryTeamToStatus().Only(ctx)
	}
	return result, MaskNotFound(err)
}

func (t *Team) TeamToProvisionedNetwork(ctx context.Context) ([]*ProvisionedNetwork, error) {
	result, err := t.Edges.TeamToProvisionedNetworkOrErr()
	if IsNotLoaded(err) {
		result, err = t.QueryTeamToProvisionedNetwork().All(ctx)
	}
	return result, err
}

func (t *Team) TeamToPlan(ctx context.Context) ([]*Plan, error) {
	result, err := t.Edges.TeamToPlanOrErr()
	if IsNotLoaded(err) {
		result, err = t.QueryTeamToPlan().All(ctx)
	}
	return result, err
}

func (u *User) UserToTag(ctx context.Context) ([]*Tag, error) {
	result, err := u.Edges.UserToTagOrErr()
	if IsNotLoaded(err) {
		result, err = u.QueryUserToTag().All(ctx)
	}
	return result, err
}

func (u *User) UserToEnvironment(ctx context.Context) ([]*Environment, error) {
	result, err := u.Edges.UserToEnvironmentOrErr()
	if IsNotLoaded(err) {
		result, err = u.QueryUserToEnvironment().All(ctx)
	}
	return result, err
}
