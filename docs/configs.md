# LaForge Config Files

## Main Server Config - `conf.*.json`

This is the main configuration file for the LaForge server.

### Overrides

You can have multiple levels of file overrides. The order of precedence is as follows:

```
1) conf.prod.json
2) conf.dev.json
3) conf.json
```

> *Note: These files will not be "merged" and one will completely replace the other*

### Builders

Read more [here](./builders.md)