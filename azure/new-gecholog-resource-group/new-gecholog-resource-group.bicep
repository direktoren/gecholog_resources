@description('URL of the AI Service API')
param aiServiceApiBase string

@description('OPTIONAL: ingress API Key for the /restricted/ router')
@secure()
param gechologApiKey string = ''

@description('OPTIONAL: outbound API Key for /restricted/ router')
@secure()
param aiServiceApiKey string = ''

@description('Storage Account Name')
param storageAccountName string = 'gechologstorage'

@description('Config file share')
param confFileShareName string = 'conf'

@description('Log file share')
param logFileShareName string = 'log'

@description('Log Space name')
param logSpaceName string = 'gechologspace'

@description('sku')
param sku string = 'PerGB2018'

@description('Name for the container group')
param containerGroupName string = 'gecholog'

@description('Location for all resources.')
param location string = resourceGroup().location

@description('Container image to deploy. Should be of the form repoName/imagename:tag for images stored in public Docker Hub, or a fully qualified URI for other registries. Images from private registries require additional registry credentials.')
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

@description('The unique guid for the workbook instance')
param workbookId string = newGuid()

@description('The behavior of Azure runtime if container has stopped.')
@allowed([
  'Always'
  'Never'
  'OnFailure'
])
param restartPolicy string = 'Always'

// Storage account and file share for config and log files

resource storageAccount 'Microsoft.Storage/storageAccounts@2022-09-01' = {
  name: storageAccountName
  location: location
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {}
}

var storageAccountAccessKey = listKeys(storageAccount.id, '2021-08-01').keys[0].value

resource fileServices 'Microsoft.Storage/storageAccounts/fileServices@2022-09-01' = {
  name: 'default'
  parent: storageAccount
  // Properties for fileServices if any...
}


resource confFileShare 'Microsoft.Storage/storageAccounts/fileServices/shares@2021-04-01' = {
  name: confFileShareName
  parent: fileServices
  properties: {
    // Properties for confFileShare...
  }
}
//  name: '${storageAccount.name}/default/${confFileShareName}'
//}

resource logFileShare 'Microsoft.Storage/storageAccounts/fileServices/shares@2021-04-01' = {
  name: logFileShareName
  parent: fileServices
  properties: {
    // Properties for confFileShare...
  }
}
//  name: '${storageAccount.name}/default/${logFileShareName}'
//}

// Log Analytics Workspace

resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2021-06-01' = {
  name: logSpaceName
  location: location
  properties: {
    sku: {
      name: sku
    }
    retentionInDays: 30
  }
}
#disable-next-line use-resource-symbol-reference
var logAnalyticsWorkspaceSharedKeys = listKeys(logAnalyticsWorkspace.id, '2021-06-01').primarySharedKey

// Create the gecholog Dashboard DevOps Workbook

//@description('The friendly name for the workbook that is used in the Gallery or Saved List.  This name must be unique within a resource group.')
var workbookDisplayName = 'gecholog Dashboard DevOps'

//@description('The gallery that the workbook will been shown under. Supported values include workbook, tsg, etc. Usually, this is \'workbook\'')
var workbookType = 'workbook'

// We do this replace since the ARM template export includes absolut references in the serializedDataString
var rawSerializedDataStr = '{"version":"Notebook/1.0","items":[{"type":1,"content":{"json":"# Gecholog Analytics Workbook - Visit our docs site [docs.gecholog.ai](https://docs.gecholog.ai)","style":"upsell"},"name":"text - 8"},{"type":9,"content":{"version":"KqlParameterItem/1.0","parameters":[{"id":"95510b98-7196-47c1-b082-8fdbcea7a135","version":"KqlParameterItem/1.0","name":"columnFilter","type":2,"isRequired":true,"query":"gecholog_CL\\n| getschema\\n| distinct  ColumnName","typeSettings":{"additionalResourceOptions":["value::1"],"showDefault":false},"timeContext":{"durationMs":86400000},"defaultValue":"value::1","queryType":0,"resourceType":"microsoft.operationalinsights/workspaces","value":"request_gl_path_s"},{"id":"6a5f1484-a964-4597-9390-9b9f8dadd238","version":"KqlParameterItem/1.0","name":"ingressStartTime","type":4,"isRequired":true,"typeSettings":{"selectableValues":[{"durationMs":300000},{"durationMs":900000},{"durationMs":1800000},{"durationMs":3600000},{"durationMs":14400000},{"durationMs":43200000},{"durationMs":86400000},{"durationMs":172800000},{"durationMs":259200000},{"durationMs":604800000},{"durationMs":1209600000},{"durationMs":2419200000},{"durationMs":2592000000},{"durationMs":5184000000},{"durationMs":7776000000}],"allowCustom":true},"timeContext":{"durationMs":86400000},"value":{"durationMs":14400000}}],"style":"pills","queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Gecholog Parameters"},{"type":1,"content":{"json":"# Traffic Analysis","style":"info"},"name":"text - 18"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| summarize Count = count() by Bin = bin(ingressStartTimeStamp, (end_time - start_time) / 50), DynamicColumn\\n| order by Bin asc, DynamicColumn\\n| render columnchart with (series = DynamicColumn)\\n\\n","size":0,"title":"Count Histogram","timeContextFromParameter":"ingressStartTime","queryType":0,"resourceType":"microsoft.operationalinsights/workspaces","graphSettings":{"type":0}},"name":"Count Histogram"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| summarize Count = count(),\\n            TotalTokens = round(sum(response_token_count_total_tokens_d), 1), \\n            AverageTokens = round(avg(response_token_count_total_tokens_d), 1),\\n            PromptTokens = round(sum(response_token_count_prompt_tokens_d), 1), \\n            CompletionTokens = round(sum(response_token_count_completion_tokens_d), 1), \\n            AverageDuration = round(avg(ingress_egress_timer_duration_d), 1),\\n            StatusCodeNotOK = countif(response_egress_status_code_d != 200)\\n            by DynamicColumn\\n| order by DynamicColumn\\n","size":1,"title":"Table of Statistics","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"customWidth":"75","name":"Table of Statistics"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| summarize Count = count() by DynamicColumn\\n| render piechart\\n","size":3,"title":"Share of API Calls","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"customWidth":"25","name":"query - 5"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| summarize arg_max(ingressStartTimeStamp, *) by DynamicColumn\\n| project \\n    DynamicColumn,\\n    LastTransactionId = transaction_id_s, \\n    LastSessionId = session_id_s,\\n    LastTime = ingressStartTimeStamp,\\n    LastTotalTokens = response_token_count_total_tokens_d,\\n    LastDuration = ingress_egress_timer_duration_d\\n| order by DynamicColumn\\n","size":1,"title":"Last Transaction","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Last Transaction"},{"type":1,"content":{"json":"# Performance","style":"info"},"name":"text - 22"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = column_ifexists(columnName, \\"\\")\\n| summarize TotalTokenConsumption = sum(response_token_count_total_tokens_d) by Bin = bin(ingressStartTimeStamp, (end_time - start_time) / 50), DynamicColumn\\n| order by Bin asc, DynamicColumn\\n| render columnchart with (series = DynamicColumn)\\n","size":0,"title":"Total Token Histogram","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Total Token Histogram"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| summarize AverageDuration = avg(ingress_egress_timer_duration_d) by Bin = bin(ingressStartTimeStamp, (end_time - start_time) / 50), DynamicColumn\\n| order by Bin asc, DynamicColumn\\n| render linechart with (series = DynamicColumn)\\n","size":1,"aggregation":3,"title":"Average Duration Histogram (ms)","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces","tileSettings":{"showBorder":false}},"name":"Average Duration Histogram (ms)"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Replace with the column name you want to analyze\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\nlet min_duration = toscalar(gecholog_CL | extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s) | where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time | summarize min(ingress_egress_timer_duration_d));\\nlet max_duration = toscalar(gecholog_CL | extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s) | where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time | summarize max(ingress_egress_timer_duration_d));\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| summarize Count = count() by Bin = bin(ingress_egress_timer_duration_d, (max_duration - min_duration) / 50), DynamicColumn\\n| order by Bin asc, DynamicColumn\\n| render columnchart with (series = DynamicColumn)","size":0,"title":"Duration Distribution (ms)","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Duration Distribution (ms)"},{"type":1,"content":{"json":"# Processors","style":"info"},"name":"text - 16"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| project request_processors_s, DynamicColumn\\n| extend ParsedJson = parse_json(request_processors_s)\\n| mv-expand Processor = ParsedJson // Expanding the array\\n| extend processorName = tostring(Processor.name),\\n          completed = tobool(Processor.details.completed),\\n          duration = todouble(Processor.details.timestamp.duration)\\n| summarize countCompleted = sum(iif(completed == true, 1, 0)), \\n             countNotCompleted = sum(iif(completed == false, 1, 0)), \\n             averageDuration = round(avg(duration),2) by DynamicColumn, processorName\\n| order by DynamicColumn, processorName\\n","size":4,"title":"Request Processor Execution","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces","sortBy":[]},"name":"Request Processor Execution"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| project response_processors_s, DynamicColumn\\n| extend ParsedJson = parse_json(response_processors_s)\\n| mv-expand Processor = ParsedJson // Expanding the array\\n| extend processorName = tostring(Processor.name),\\n          completed = tobool(Processor.details.completed),\\n          duration = todouble(Processor.details.timestamp.duration)\\n| summarize countCompleted = sum(iif(completed == true, 1, 0)), \\n             countNotCompleted = sum(iif(completed == false, 1, 0)), \\n             averageDuration = round(avg(duration),2) by DynamicColumn, processorName\\n| order by DynamicColumn, processorName\\n","size":4,"title":"Response Processors Execution","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Response Processors Execution"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| project request_processors_async_s, DynamicColumn\\n| extend ParsedJson = parse_json(request_processors_async_s)\\n| mv-expand Processor = ParsedJson // Expanding the array\\n| extend processorName = tostring(Processor.name),\\n          completed = tobool(Processor.details.completed),\\n          duration = todouble(Processor.details.timestamp.duration)\\n| summarize countCompleted = sum(iif(completed == true, 1, 0)), \\n             countNotCompleted = sum(iif(completed == false, 1, 0)), \\n             averageDuration = round(avg(duration),2) by DynamicColumn, processorName\\n| order by DynamicColumn, processorName\\n","size":4,"title":"Request Processors Async Execution","noDataMessage":"The query returned no result","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Request Processors Async Execution"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| project response_processors_async_s, DynamicColumn\\n| extend ParsedJson = parse_json(response_processors_async_s)\\n| mv-expand Processor = ParsedJson // Expanding the array\\n| extend processorName = tostring(Processor.name),\\n          completed = tobool(Processor.details.completed),\\n          duration = todouble(Processor.details.timestamp.duration)\\n| summarize countCompleted = sum(iif(completed == true, 1, 0)), \\n             countNotCompleted = sum(iif(completed == false, 1, 0)), \\n             averageDuration = round(avg(duration),2) by DynamicColumn, processorName\\n| order by DynamicColumn, processorName\\n","size":4,"title":"Response Processors Async Execution","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Response Processors Async Execution"},{"type":1,"content":{"json":"# Fields","style":"info"},"name":"text - 17"},{"type":3,"content":{"version":"KqlItem/1.0","query":"gecholog_CL\\n| getschema\\n| extend Prefix = coalesce(\\n    extract(\\"^(\\\\\\\\w+?_\\\\\\\\w+?_\\\\\\\\w+?_)\\", 0, ColumnName),\\n    extract(\\"^(\\\\\\\\w+?_\\\\\\\\w+?_)\\", 0, ColumnName),\\n    extract(\\"^(\\\\\\\\w+?_)\\", 0, ColumnName)\\n  )\\n| summarize Columns = make_list(ColumnName) by Prefix\\n| order by Prefix asc  \\n| project Prefix, ColumnNames = strcat_array(Columns, \\", \\")\\n\\n\\n\\n","size":3,"title":"Gecholog - Log Analytics Schema","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"query - 2"},{"type":1,"content":{"json":"# Transaction Finder","style":"info"},"name":"text - 15"},{"type":9,"content":{"version":"KqlParameterItem/1.0","parameters":[{"id":"8a6141d4-2614-4db2-8155-7451ab7c2f47","version":"KqlParameterItem/1.0","name":"statusCode","type":2,"isRequired":true,"query":"gecholog_CL\\n| distinct response_egress_status_code_d","typeSettings":{"additionalResourceOptions":["value::1"],"showDefault":false},"timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces","value":"200"},{"id":"aca428e9-0a73-4312-aae9-99454631a7a5","version":"KqlParameterItem/1.0","name":"transactionIDRegex","type":1,"timeContext":{"durationMs":86400000},"value":""},{"id":"d8c9ca24-0b1a-4bda-a790-384dcefc5877","version":"KqlParameterItem/1.0","name":"minDuration","type":1,"timeContext":{"durationMs":86400000},"value":""},{"id":"0dffa56c-2860-4f45-8b90-9fa932ea6f5c","version":"KqlParameterItem/1.0","name":"maxDuration","type":1,"timeContext":{"durationMs":86400000},"value":""},{"id":"b41b51a5-1c1c-4123-94b3-e41f9d4464c5","version":"KqlParameterItem/1.0","name":"minTokens","type":1,"timeContext":{"durationMs":86400000},"value":""},{"id":"c0087e66-fc43-44ba-bd7c-c5882b8382ec","version":"KqlParameterItem/1.0","name":"maxTokens","type":1,"timeContext":{"durationMs":86400000},"value":""}],"style":"pills","queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"parameters - 21"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let columnName = \'{columnFilter}\'; // Use the workbook parameter\\nlet start_time = todatetime(\'{ingressStartTime:startISO}\'); // Use the workbook parameter\\nlet end_time = todatetime(\'{ingressStartTime:endISO}\'); // Use the workbook parameter\\nlet status_code = \'{statusCode}\';\\nlet findIngressStartTimeStamp = (t:datetime, s:string) { // We don\'t trust the log analytics indexing\\n    coalesce(t, todatetime(s))\\n};\\nlet transaction_id_pattern = coalesce(\'{transactionIDRegex}\', \'.\');\\nlet minDurationValue = toint(coalesce(\'{minDuration}\', \'0\')); // Convert to integer and set default to 0 if null\\nlet duration_min = iif(minDurationValue > 0, minDurationValue, 0);\\nlet maxDurationValue = toint(coalesce(\'{maxDuration}\', \'0\')); // Convert to integer and set default to 0 if null\\nlet duration_max = iif(maxDurationValue <> 0, maxDurationValue, 360000000);\\nlet minTokenValue = toint(coalesce(\'{minTokens}\', \'0\')); // Convert to integer and set default to 0 if null\\nlet token_min = iif(minTokenValue > 0, minTokenValue, 0);\\nlet maxTokenValue = toint(coalesce(\'{maxTokens}\', \'0\')); // Convert to integer and set default to 0 if null\\nlet token_max = iif(maxTokenValue <> 0, maxTokenValue, 360000000000000);\\ngecholog_CL\\n| extend ingressStartTimeStamp = findIngressStartTimeStamp(column_ifexists(\'ingress_egress_timer_start_t\', datetime(null)), ingress_egress_timer_start_s)\\n| where ingressStartTimeStamp >= start_time and ingressStartTimeStamp <= end_time\\n| extend DynamicColumn = coalesce(column_ifexists(columnName, \\"\\"), \\"NaN\\")\\n| where transaction_id_s matches regex transaction_id_pattern\\n| where ingress_egress_timer_duration_d >= duration_min\\n| where ingress_egress_timer_duration_d <= duration_max\\n| where response_token_count_total_tokens_d >= token_min\\n| where response_token_count_total_tokens_d <= token_max\\n| where response_egress_status_code_d == status_code\\n| project DynamicColumn, ingressStartTimeStamp, transaction_id_s, session_id_s, ingress_egress_timer_duration_d, response_token_count_total_tokens_d, response_egress_status_code_d\\n| order by ingressStartTimeStamp desc\\n","size":0,"title":"Transaction Finder","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Transaction Finder"},{"type":9,"content":{"version":"KqlParameterItem/1.0","parameters":[{"id":"1f78c634-7b32-42e9-ac01-5584410f30ca","version":"KqlParameterItem/1.0","name":"transactionID","type":1,"timeContext":{"durationMs":86400000},"value":""}],"style":"pills","queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"parameters - 20"},{"type":3,"content":{"version":"KqlItem/1.0","query":"let transaction_id = \'{transactionID}\';\\ngecholog_CL\\n| where transaction_id_s == transaction_id\\n| project clickOnThisToReviewContent = pack_all()\\n","size":4,"title":"Open Transaction","timeContext":{"durationMs":86400000},"queryType":0,"resourceType":"microsoft.operationalinsights/workspaces"},"name":"Open Transaction"}],"isLocked":false,"fallbackResourceIds":["/subscriptions/aa1fa7e2-6d38-5333-bf49-8025e5f723e9/resourcegroups/gecholog/providers/microsoft.operationalinsights/workspaces/gechologspace"]}'
var serializedDataString = replace(rawSerializedDataStr, '/subscriptions/aa1fa7e2-6d38-5333-bf49-8025e5f723e9/resourcegroups/gecholog/providers/microsoft.operationalinsights/workspaces/gechologspace', logAnalyticsWorkspace.id)


resource workbookId_resource 'microsoft.insights/workbooks@2022-04-01' = {
  #disable-next-line use-stable-resource-identifiers
  name: workbookId
  location: location
  kind: 'shared'
  properties: {
    displayName: workbookDisplayName
    serializedData: serializedDataString
    version: '1.0'
    sourceId: logAnalyticsWorkspace.id
    category: workbookType
  }
  dependsOn: []
}

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
              value: logAnalyticsWorkspace.properties.customerId
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
