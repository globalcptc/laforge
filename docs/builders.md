# LaForge Builders

## Configuration and Builder Definition

The main `conf.json` file should include a `builders` section. This is a map of "friendly" builder names to builders and config files. This friendly name is what should be specified in the laforge environment configuration file under the `builder` attribute.

If the friendly name specified in the laforge environment configuration does not exist in `conf.json`, then the environment will fail to build.

The `builder` attribute in the `conf.json` entry does not match the `ID` attribute of a given builder, any environment referencing this entry will fail to build.

### Example files

#### `conf.json`

```json
{
  // ...
  "builders": {
    "friendly_builder_name_here": {
      "builder": "builder_slug",
      "config": "/path/to/config/directory/friendly_builder_name_here.json"
    }
  }
  // ...
}
```

#### `envs/example.laforge`

```
// ...
environment "/envs/example" {
  // ...
  builder = "friendly_builder_name_here"
  // ...
```