## Redeploy/deploy gecholog container into existin resource group

```sh
az deployment group create --name redeploy --resource-group <resourceGroupName> --template-file gecholog-container-only.bicep 
```