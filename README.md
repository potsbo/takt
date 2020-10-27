# takt

Task runner with cancel

## NOTE

This tool is now under implementation. Many planned features are missing.

## Features

`takt` includes several notable features
- Dependency Resolution
- Immediate Termination

### Example

```yaml
tasks:
  bundle:
    steps:
      - run: bundle install
  yarn:
    steps:
      - run: yarn install
  rails:
    depends:
      - bundle
    steps:
      - run: bin/rails s
  dev-server:
    depends:
      - yarn
    steps:
      - run: yarn start
```

### Dependency Resolution

This configuration above, `takt` does as follows
- run `bundle` and `yarn` concurrently
- once `bundle` finishes, run `rails`
- once `yarn` finishes, run `dev-server`

So that you can start multiple processes with a single command.

### Immediate Termination

Unlikee `make`, when a task exited with errors, `takt` immediately sends `SIGTERM` to safely stop all other tasks.
In the example above, when `yarn start` failed to start `rails` process would be terminated.
