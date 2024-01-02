// az deployment group create --name redeploy --resource-group <resourceGroupName> --template-file gecholog-container-only.bicep 

@description('URL of the AI Service API')
param aiServiceApiBase string

@description('OPTIONAL: ingress API Key for the /restricted/ router')
@secure()
param gechologApiKey string

@description('OPTIONAL: outbound API Key for /restricted/ router')
@secure()
param aiServiceApiKey string 

@description('Storage Account Name')
param storageAccountName string

@description('Storage Account Key')
@secure()
param storageAccountAccessKey string

@description('Log Analytics Workspace ID')
param logAnalyticsWorkspaceID string

@description('Log Analytics Key')
@secure()
param logAnalyticsWorkspaceSharedKeys string

@description('Config file share')
param confFileShareName string = 'conf'

@description('Log file share')
param logFileShareName string = 'log'

@description('Name for the container group')
param containerGroupName string = 'gecholog'

@description('Location for all resources.')
param location string = resourceGroup().location

@description('Container image to deploy. Should be of the form repoName/imagename:tag for images stored in public Docker Hub, or a fully qualified URI for other registries. Images from private registries require additional registry credentials.')
//param image string = 'gecholog/gechologpreview:latest'
param image string = 'gecholog/gecholog:latest'

@description('Set to .logger to activate nats2log')
param nats2logLoggerSubTopic string = '.logger'

@description('Set to .logger to activate nats2file')
param nats2fileLoggerSubTopic string = '.logger'

@description('DNS name prefix label for used to create FQDN')
param dnsLabel string = 'gecholog'
var uniqueStr = uniqueString(resourceGroup().id)
var dnsNameLabelUnique = '${dnsLabel}-${uniqueStr}'

@description('Port to open on the container and the public IP address.')
param port int = 5380

@description('The number of CPU cores to allocate to the container.')
param cpuCores int = 1

@description('The amount of memory to allocate to the container in gigabytes.')
param memoryInGb int = 1

@description('The behavior of Azure runtime if container has stopped.')
@allowed([
  'Always'
  'Never'
  'OnFailure'
])
param restartPolicy string = 'Always'

// Setup containers

resource containerGroup 'Microsoft.ContainerInstance/containerGroups@2021-09-01' = {
  name: containerGroupName
  location: location
  properties: {
    containers: [
      {
        name: containerGroupName
        properties: {
          image: image
          ports: [
            {
              port: port
              protocol: 'TCP'
            }
            {
              port: 4222
              protocol: 'TCP'
            }
          ]
          resources: {
            requests: {
              cpu: cpuCores 
              memoryInGB: memoryInGb
            }
          }
          environmentVariables: [
            {
              name: 'AISERVICE_API_BASE'
              value: aiServiceApiBase
            }
            {
              name: 'NATS2LOG_LOGGER_SUBTOPIC'
              value: nats2logLoggerSubTopic
            }
            {
              name: 'NATS2FILE_LOGGER_SUBTOPIC'
              value: nats2fileLoggerSubTopic
            }
            {
              name: 'AZURE_LOG_ANALYTICS_WORKSPACE_ID'
              value: logAnalyticsWorkspaceID
            }
            {
              name: 'AZURE_LOG_ANALYTICS_SHARED_KEY'
              secureValue: logAnalyticsWorkspaceSharedKeys
            }
            {
              name: 'AZURE_STORAGE_ACCESS_KEY'
              secureValue: storageAccountAccessKey
            }
            {
              name: 'GECHOLOG_API_KEY'
              secureValue: gechologApiKey
            }
            {
              name: 'AISERVICE_API_KEY'
              secureValue: aiServiceApiKey
            }
          ]
          volumeMounts: [
            {
              name: 'conf'
              mountPath: '/app/conf'
            }
            {
              name: 'log'
              mountPath: '/app/log'
            }
          ]
        }
      }
    ]
    osType: 'Linux'
    restartPolicy: restartPolicy
    ipAddress: {
      type: 'Public'
      dnsNameLabel: dnsNameLabelUnique
      ports: [
        {
          port: port
          protocol: 'TCP'
        }
        {
          port: 4222
          protocol: 'TCP'
        }
      ]
    }
    volumes: [
      {
        name: 'conf'
        azureFile: {
          shareName: confFileShareName
          storageAccountName: storageAccountName
          storageAccountKey: storageAccountAccessKey
          readOnly: false
        }
      }
      {
        name: 'log'
        azureFile: {
          shareName: logFileShareName
          storageAccountName: storageAccountName
          storageAccountKey: storageAccountAccessKey
          readOnly: false
        }
      }
    ]
  }
}

output containerGroupFqdn string = containerGroup.properties.ipAddress.fqdn
output containerIPv4Address string = containerGroup.properties.ipAddress.ip
