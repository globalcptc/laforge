import { gql } from 'apollo-angular';
import { DocumentNode } from 'graphql';

const getEnvConfigQuery = (id: string): DocumentNode => gql`
  {
    environment(envUUID: "${id}") {
      id
      CompetitionID
      Name
      Description
      Builder
      TeamCount
      AdminCIDRs
      ExposedVDIPorts
      tags {
        id
        name
        description
      }
      config {
        key
        value
      }
      maintainer {
        id
        name
        uuid
        email
      }
      build {
        id
        revision
        tags {
          id
          name
          description
        }
        config {
          key
          value
        }
        maintainer {
          id
          name
          uuid
          email
        }
        teams {
          id
          teamNumber
          provisionedNetworks {
            id
            name
            cidr
            network {
              id
              vdiVisible
            }
            provisionedHosts {
              id
              subnetIP
              status {
                state
                startedAt
                endedAt
                error
              }
              host {
                id
                hostname
                OS
                allowMacChanges
                exposedTCPPorts
                exposedUDPPorts
                userGroups
                overridePassword
                maintainer {
                  name
                  email
                }
                vars {
                  key
                  value
                }
                tags {
                  name
                  description
                }
              }
            }
            status {
              state
              startedAt
              endedAt
              error
            }
          }
        }
      }
    }
  }
`;

export { getEnvConfigQuery };