# LaForge `/configs` directory

This is where the builder configuration files are stored. These configuration files are up to the builder to determine and will be loaded in when the respective builder is specified in an environment.

> *Note: Each file here should be named to match the "friendly name" in the `/conf.json` file for ease of use.*

## Example

For example, if we have 2 separate VMWare vSphere/NSXT environments, we could name one `dev_vmware` and the other `prod_vmware`. These two files would be referenced from the `/conf.json` as:

```json
{
  "builders": {
    // ...
    "dev_vmware": {
      "builder": "vspherensxt",
      "config": "configs/dev_vmware.json"
    },
    "prod_vmware": {
      "builder": "vspherensxt",
      "config": "configs/prod_vmware.json"
    },
    // ...
  }
}
```
