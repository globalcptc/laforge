import { environment } from 'src/environments/environment';
import { ID, Tag, configMap, User, Team, ProvisionStatus } from './common.model';
import { DNS } from './dns.model';
import { Host, ProvisionedHost } from './host.model';
import { Network, ProvisionedNetwork } from './network.model';

interface Build {
  id: ID;
  revision: number;
  tags: Tag[];
  config: configMap[];
  maintainer: User;
  teams: Team[];
}

interface Competition {
  id: ID;
  rootPassword: string;
  config: configMap[];
  dns: DNS;
}

interface Environment {
  id: ID;
  CompetitionID: string;
  Name: string;
  Description: string;
  Builder: string;
  TeamCount: number;
  AdminCIDRs: string[];
  ExposedVDIPorts: string[];
  tags: Tag[];
  config: configMap[];
  maintainer: User;
  networks: Network[];
  hosts: Host[];
  build: Build;
  competition: Competition;
}

function resolveStatuses(environment: any): any {
  return {
    ...environment,
    build: {
      ...environment.build,
      teams: [
        ...environment.build.teams.map((team: Team) => ({
          ...team,
          provisionedNetworks: team.provisionedNetworks.map((provisionedNetwork: ProvisionedNetwork) => ({
            ...provisionedNetwork,
            status: {
              ...provisionedNetwork.status,
              state: ProvisionStatus[provisionedNetwork.status.state]
            },
            provisionedHosts: provisionedNetwork.provisionedHosts.map((provisionedHost: ProvisionedHost) => ({
              ...provisionedHost,
              build: undefined,
              status: {
                ...provisionedHost.status,
                state: ProvisionStatus[provisionedHost.status.state]
              }
            }))
          }))
        }))
      ]
    }
  };
}

export { Environment, Build, Competition, resolveStatuses };